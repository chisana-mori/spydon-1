package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"strings"
	"time"

	"sync"

	"github.com/gin-gonic/gin"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SimpleDrainService 提供简化的 Drain 服务
type SimpleDrainService struct {
	clientFactory    sharedservices.K8sClientFactory
	stateManager     StateManager
	resourceManager  *DrainResourceManager
	operationManager *DrainOperationManager
	eventManager     *SimpleDrainEventManager
	retryExecutor    *RetryExecutor
	redisHandler     sharedservices.RedisClient
	versionCache     *sharedservices.ClusterVersionCache

	pdbInFlightPerRS map[string]bool
	pdbRetryStates   map[string]*pdbRetryState
	pdbMutex         sync.Mutex

	// 支持外部按 drain 级别取消的控制
	cancelMutex         sync.Mutex
	drainCancels        map[string]context.CancelFunc
	normalizeProbeMutex sync.Mutex
	lastNormalizeProbe  map[string]time.Time
}

// NewSimpleDrainService 创建 SimpleDrainService
func NewSimpleDrainService(clientFactory sharedservices.K8sClientFactory, redisHandler sharedservices.RedisClient) *SimpleDrainService {
	eventManager := NewSimpleDrainEventManager(redisHandler)
	versionCache := sharedservices.NewClusterVersionCache(5 * time.Minute)

	return &SimpleDrainService{
		clientFactory:   clientFactory,
		stateManager:    NewDrainStateManager(redisHandler),
		resourceManager: NewDrainResourceManager(clientFactory, redisHandler, eventManager, versionCache),
		// DrainOperationManager now depends on sharedservices.RedisClient interface
		operationManager:   NewDrainOperationManager(eventManager, redisHandler),
		eventManager:       eventManager,
		retryExecutor:      NewRetryExecutor(DefaultRetryConfig()),
		redisHandler:       redisHandler,
		versionCache:       versionCache,
		pdbInFlightPerRS:   make(map[string]bool),
		pdbRetryStates:     make(map[string]*pdbRetryState),
		drainCancels:       make(map[string]context.CancelFunc),
		lastNormalizeProbe: make(map[string]time.Time),
	}
}

// setDrainCancel 记录 drain 级别取消函数
func (sds *SimpleDrainService) setDrainCancel(drainID string, cancel context.CancelFunc) {
	sds.cancelMutex.Lock()
	sds.drainCancels[drainID] = cancel
	sds.cancelMutex.Unlock()
}

// popDrainCancel 取出并删除取消函数
func (sds *SimpleDrainService) popDrainCancel(drainID string) context.CancelFunc {
	sds.cancelMutex.Lock()
	defer sds.cancelMutex.Unlock()
	if cf, ok := sds.drainCancels[drainID]; ok {
		delete(sds.drainCancels, drainID)
		return cf
	}
	return nil
}

// pdbRetryState 记录因 PDB 阻塞时的重试状态
type pdbRetryState struct {
	migrationID string
	rsKey       string
	namespace   string
	podName     string
	clusterName string
	attempts    int
	nextRetryAt time.Time
}

func (sds *SimpleDrainService) checkNodeUnschedulable(ctx context.Context, clusterName, nodeName string) error {
	client, err := sds.clientFactory.GetClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client for cluster %s: %w", clusterName, err)
	}

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// 检查节点是否为不可调度状态
	if !node.Spec.Unschedulable {
		return fmt.Errorf("节点 %s 尚未被标记为不可调度状态，请先cordon节点 %s 后再进行drain操作", nodeName, nodeName)
	}

	return nil
}

// StartDrain 启动 Drain 操作
func (sds *SimpleDrainService) StartDrain(ctx context.Context, req *SimpleDrainRequest) (*SimpleDrainResponse, error) {
	// 校验节点调度状态
	if err := sds.checkNodeUnschedulable(ctx, req.ClusterName, req.NodeName); err != nil {
		return nil, fmt.Errorf("节点状态校验失败: %w", err)
	}

	drainID := generateDrainID()
	// No Redis lock required, using Lease mechanism instead

	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	sds.setDrainCancel(drainID, cancel)

	// Start a control-channel subscriber for cross-instance cancel
	go sds.subscribeControlChannel(drainCtx, drainID, cancel)

	snapshot, err := sds.resourceManager.CreateSnapshot(drainCtx, drainID, req.ClusterName, req.NodeName)
	if err != nil {
		cancel()
		_ = sds.popDrainCancel(drainID)

		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Initialize migration tracking synchronously so first query has data
	if err := sds.resourceManager.StartPodMigrationTracking(drainCtx, drainID, req.ClusterName, snapshot.Pods); err != nil {
		sds.eventManager.SendError(drainID, "启动Pod迁移跟踪失败", err)
	}

	sds.operationManager.StartDrain(drainID, len(snapshot.Pods))
	if err := sds.stateManager.SaveDrainState(drainID, &DrainState{
		DrainID: drainID, NodeName: req.NodeName, ClusterName: req.ClusterName,
		Status: DrainStatusRunning, StartTime: time.Now(), UserID: "system",
	}); err != nil {
		// record non-blocking error
		sds.eventManager.SendError(drainID, "failed to save drain state", err)
	}

	go func() {
		defer cancel()
		defer sds.popDrainCancel(drainID)
		sds.executeDrain(drainCtx, drainID, req, snapshot)
	}()

	return &SimpleDrainResponse{
		DrainID: drainID, Message: "Drain操作已开始", Status: "started",
	}, nil
}

// subscribeControlChannel 订阅指定 drainID 的控制通道（用于跨实例取消等指令）
func (sds *SimpleDrainService) subscribeControlChannel(ctx context.Context, drainID string, cancel context.CancelFunc) {
	if sds.redisHandler == nil {
		return
	}
	channel := fmt.Sprintf("drain:control:%s", drainID)
	ps := sds.redisHandler.Subscribe(channel)
	defer func() {
		if err := ps.Close(); err != nil {
			log.Printf("[subscribeControlChannel] failed to close pubsub: %v", err)
		}
	}()

	// 尝试在有限时间内建立订阅，避免后台阻塞
	setupCtx, cancelSetup := context.WithTimeout(ctx, 5*time.Second)
	if _, err := ps.Receive(setupCtx); err != nil {
		cancelSetup()
		return
	}
	cancelSetup()

	ch := ps.ChannelSize(256)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if strings.ToLower(msg.Payload) == "cancel" {
				if cf := sds.popDrainCancel(drainID); cf != nil {
					cf()
				} else if cancel != nil {
					cancel()
				}
				// 更新状态
				if err := sds.stateManager.UpdateDrainStatus(drainID, DrainStatusCanceled); err != nil {
					sds.eventManager.SendError(drainID, "failed to update drain status to canceled", err)
				}
				sds.operationManager.CompleteDrain(drainID, false, "Drain操作已取消(跨实例)")
				sds.cleanup(drainID)
				return
			}
		}
	}
}

// executeDrain 编排 Drain 全流程
func (sds *SimpleDrainService) executeDrain(ctx context.Context, drainID string, req *SimpleDrainRequest, snapshot *DrainSnapshotInfo) {
	defer sds.cleanup(drainID)

	client, err := sds.clientFactory.GetClient(req.ClusterName)
	if err != nil {
		sds.operationManager.CompleteDrain(drainID, false, fmt.Sprintf("Failed to get client: %v", err))
		return
	}

	sds.eventManager.SendPodSnapshotCreated(drainID, snapshot)

	appGroups := sds.groupPodsForEviction(snapshot.Pods)
	sds.createPDBsForAppGroups(ctx, drainID, req, appGroups)
	sds.evictPodsInGroups(ctx, drainID, req, client, appGroups, len(snapshot.Pods))

	sds.waitForMigrationsToComplete(ctx, drainID)
	sds.finalizeDrain(drainID)
}

// createPDBsForAppGroups 为应用分组创建 PDB
func (sds *SimpleDrainService) createPDBsForAppGroups(ctx context.Context, drainID string, req *SimpleDrainRequest, appGroups map[string][]DrainPodInfo) {
	for groupKey, pods := range appGroups {
		// 过滤掉被标记为 Ignored 的 Pod（如 DaemonSet / StatefulSet）
		eligible := make([]DrainPodInfo, 0, len(pods))
		for _, p := range pods {
			migrationID := fmt.Sprintf("%s-%s-%s", drainID, p.Namespace, p.Name)
			if mi := sds.resourceManager.GetPodMigrationInfo(migrationID); mi != nil && mi.Status == DrainMigrationIgnored {
				continue
			}
			eligible = append(eligible, p)
		}

		if len(eligible) > 1 && !req.DryRun {
			// Ensure Lease exists for this namespace/node combination
			ns := eligible[0].Namespace
			lease, err := sds.resourceManager.EnsureNodeLease(ctx, req.ClusterName, ns, req.NodeName, drainID)
			if err != nil {
				sds.eventManager.SendErrorWithPod(drainID, fmt.Sprintf("创建Lease失败: %s", ns), err, eligible[0].Name, ns)
				continue
			}

			pdbKey, err := sds.resourceManager.CreatePDB(ctx, drainID, req.ClusterName, ns, groupKey, len(eligible)-1, eligible, lease)
			if err != nil {
				sds.eventManager.SendErrorWithPod(drainID, fmt.Sprintf("创建PDB失败: %s", groupKey), err, eligible[0].Name, eligible[0].Namespace)
			} else {
				sds.eventManager.SendProgressWithPod(drainID, "pdb", fmt.Sprintf("为组 %s 创建PDB: %s", groupKey, pdbKey), 0, eligible[0].Name, eligible[0].Namespace)
			}
		}
	}
}

// evictPodsInGroups 分组驱逐 Pod
func (sds *SimpleDrainService) evictPodsInGroups(ctx context.Context, drainID string, req *SimpleDrainRequest, client kubernetes.Interface, appGroups map[string][]DrainPodInfo, totalPods int) {
	processedPods := 0
	failedPods := 0

	for groupKey, pods := range appGroups {
		select {
		case <-ctx.Done():
			sds.eventManager.SendMessage(drainID, DrainCanceled, "drain canceled", nil)
			return
		default:
		}
		for _, pod := range pods {
			select {
			case <-ctx.Done():
				sds.eventManager.SendMessage(drainID, DrainCanceled, "drain canceled", nil)
				return
			default:
			}
			migrationID := fmt.Sprintf("%s-%s-%s", drainID, pod.Namespace, pod.Name)
			if mi := sds.resourceManager.GetPodMigrationInfo(migrationID); mi != nil && mi.Status == DrainMigrationIgnored {
				sds.eventManager.SendMessage(drainID, DrainProgress, fmt.Sprintf("跳过不可驱逐Pod: %s/%s", pod.Namespace, pod.Name), map[string]interface{}{
					"step":     "ignored",
					"category": "pod",
					"level":    "INFO",
				})
				// 发送ignored状态事件，确保前端能正确显示
				sds.eventManager.SendPodMigrationIgnored(drainID, mi)
				processedPods++
				sds.operationManager.UpdateProgress(drainID, processedPods, failedPods, fmt.Sprintf("Processed %d/%d pods", processedPods, totalPods))
				continue
			}
			if req.DryRun {
				sds.simulateEviction(drainID, migrationID, &pod)
			} else {
				if err := sds.evictPodWithRetry(ctx, client, req.ClusterName, pod.Namespace, pod.Name); err != nil {
					if sds.isPDBBlockError(err) {
						sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationPending, "PDB限制，等待可中断窗口")
						sds.schedulePDBRetry(ctx, client, req.ClusterName, drainID, groupKey, migrationID, pod.Namespace, pod.Name)
					} else {
						failedPods++
						detailedError := sds.buildDetailedEvictionError(&pod, groupKey, err, failedPods, len(pods))
						sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationFailed, detailedError)
					}
				} else {
					sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationEvicting, "")
					sds.handleSuccessfulEviction(drainID, migrationID, &pod)
				}
			}
			processedPods++
			sds.operationManager.UpdateProgress(drainID, processedPods, failedPods, fmt.Sprintf("Processed %d/%d pods", processedPods, totalPods))
		}
	}
}

// simulateEviction 模拟驱逐（DryRun）
func (sds *SimpleDrainService) simulateEviction(drainID, migrationID string, pod *DrainPodInfo) {
	sds.eventManager.SendProgressWithPod(drainID, "evicting", fmt.Sprintf("模拟驱逐Pod: %s/%s", pod.Namespace, pod.Name), 0, pod.Name, pod.Namespace)
	sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationEvicting, "")
	time.Sleep(100 * time.Millisecond)
	sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationEvicted, "")
	time.Sleep(100 * time.Millisecond)
	sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationCompleted, "")
}

// handleSuccessfulEviction 处理驱逐成功后的逻辑
func (sds *SimpleDrainService) handleSuccessfulEviction(drainID, migrationID string, pod *DrainPodInfo) {
	sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationEvicted, "")
	logMessage := fmt.Sprintf("成功驱逐Pod: %s/%s", pod.Namespace, pod.Name)
	sds.eventManager.SendMessage(drainID, "log", logMessage, map[string]interface{}{"level": "INFO", "message": logMessage})

	if _, exists := pod.Labels["pod-template-hash"]; !exists {
		time.Sleep(100 * time.Millisecond)
		sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationCompleted, "")
	}
}

// waitForMigrationsToComplete 等待所有迁移结束或超时
func (sds *SimpleDrainService) waitForMigrationsToComplete(ctx context.Context, drainID string) {
	// 使用父context的剩余时间，不再嵌套超时
	ticker := time.NewTicker(2 * time.Second) // 减少轮询间隔到2秒
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sds.eventManager.SendMessage(drainID, "log", "Drain context timeout reached, marking remaining migrations as failed", map[string]interface{}{"level": "WARN"})
			sds.markRemainingAsFailed(drainID, "drain context timeout reached")
			return
		case <-ticker.C:
			// 每个tick对非终态迁移做一次轻量归一化检查，避免watcher漏事件导致卡住
			sds.normalizeNonTerminalMigrations(ctx, drainID)
			if sds.areAllMigrationsFinished(drainID) {
				sds.eventManager.SendMessage(drainID, "log", "All pod migrations completed successfully", map[string]interface{}{"level": "INFO"})
				return
			}
		}
	}
}

// normalizeNonTerminalMigrations 对非终态迁移做一次快速API归一化检查
func (sds *SimpleDrainService) normalizeNonTerminalMigrations(ctx context.Context, drainID string) {
	// 获取集群客户端
	var clusterName string
	if st, _ := sds.stateManager.GetDrainState(drainID); st != nil {
		clusterName = st.ClusterName
	} else {
		return
	}
	client, err := sds.clientFactory.GetClient(clusterName)
	if err != nil {
		return
	}

	migrations := sds.resourceManager.GetAllMigrations(drainID)
	if len(migrations) == 0 {
		return
	}

	current := make(map[string]struct{}, len(migrations))
	for _, m := range migrations {
		current[m.MigrationID] = struct{}{}
	}
	sds.normalizeProbeMutex.Lock()
	for id := range sds.lastNormalizeProbe {
		if _, ok := current[id]; !ok {
			delete(sds.lastNormalizeProbe, id)
		}
	}
	sds.normalizeProbeMutex.Unlock()

	apiCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const probeInterval = 10 * time.Second

	for _, m := range migrations {
		switch m.Status {
		case DrainMigrationCompleted, DrainMigrationTimeout, DrainMigrationIgnored:
			sds.normalizeProbeMutex.Lock()
			delete(sds.lastNormalizeProbe, m.MigrationID)
			sds.normalizeProbeMutex.Unlock()
			continue
		case DrainMigrationFailed:
			if m.EvictionTime == nil {
				// 等待evictionTime填充，避免误判
				continue
			}
			sds.normalizeProbeMutex.Lock()
			delete(sds.lastNormalizeProbe, m.MigrationID)
			sds.normalizeProbeMutex.Unlock()
			continue
		case DrainMigrationPending, DrainMigrationEvicting, DrainMigrationEvicted, DrainMigrationCreating:
			// proceed to check
		}

		if m.TargetPod == nil || m.TargetPod.Namespace == "" || m.TargetPod.Name == "" {
			continue
		}

		now := time.Now()
		sds.normalizeProbeMutex.Lock()
		lastProbe := sds.lastNormalizeProbe[m.MigrationID]
		if !lastProbe.IsZero() && now.Sub(lastProbe) < probeInterval {
			sds.normalizeProbeMutex.Unlock()
			continue
		}
		sds.lastNormalizeProbe[m.MigrationID] = now
		sds.normalizeProbeMutex.Unlock()

		// Perform the check sequentially
		select {
		case <-apiCtx.Done():
			return // Timeout for the whole normalization batch
		default:
			if p, err := client.CoreV1().Pods(m.TargetPod.Namespace).Get(apiCtx, m.TargetPod.Name, metav1.GetOptions{}); err == nil {
				if isPodStablyReady(p, 8*time.Second) {
					sds.resourceManager.UpdatePodMigrationStatus(m.MigrationID, DrainMigrationCompleted, "")
				}
			}
		}
	}
}

// areAllMigrationsFinished 判断是否全部进入终态
func (sds *SimpleDrainService) areAllMigrationsFinished(drainID string) bool {
	// Redis优先 + 本地兜底，保证多实例一致性与实时性
	migrations := sds.resourceManager.GetAllMigrations(drainID)
	if len(migrations) == 0 {
		if stats := sds.operationManager.GetStats(drainID); stats != nil {
			return stats.TotalPods == 0
		}
		return false
	}

	for _, m := range migrations {
		switch m.Status {
		case DrainMigrationCompleted, DrainMigrationTimeout, DrainMigrationIgnored, DrainMigrationFailed:
			continue
		case DrainMigrationPending, DrainMigrationEvicting, DrainMigrationEvicted, DrainMigrationCreating:
			return false
		default:
			return false
		}
	}
	return true
}

// finalizeDrain 收尾 Drain 并设置最终状态
func (sds *SimpleDrainService) finalizeDrain(drainID string) {
	// Normalize migrations into authoritative terminal states before computing summary
	finalFailedCount := 0
	migrations := sds.resourceManager.GetAllMigrations(drainID)

	// Try to best‑effort confirm new Pod status from API if necessary
	var clusterName string
	if st, _ := sds.stateManager.GetDrainState(drainID); st != nil {
		clusterName = st.ClusterName
	}
	var client kubernetes.Interface
	if clusterName != "" {
		if cs, err := sds.clientFactory.GetClient(clusterName); err == nil {
			client = cs
		}
	}

	// 智能检查Pod状态：只检查非终态的migration，减少API调用
	if client != nil && len(migrations) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, m := range migrations {
			// 跳过已经是终态的migration
			if m.Status == DrainMigrationCompleted || m.Status == DrainMigrationIgnored ||
				m.Status == DrainMigrationFailed || m.Status == DrainMigrationTimeout {
				continue
			}

			// Perform the check sequentially
			select {
			case <-ctx.Done():
				break // Break the loop if context is done
			default:
				// 快速API调用检查，3秒超时
				apiCtx, apiCancel := context.WithTimeout(ctx, 3*time.Second)

				// 情况A：已有 targetPod，直接校验其就绪
				if m.TargetPod != nil && m.TargetPod.Name != "" {
					if p, err := client.CoreV1().Pods(m.TargetPod.Namespace).Get(apiCtx, m.TargetPod.Name, metav1.GetOptions{}); err == nil {
						if isPodStablyReady(p, 8*time.Second) {
							sds.resourceManager.UpdatePodMigrationStatus(m.MigrationID, DrainMigrationCompleted, "")
							apiCancel()
							continue
						}
					}
				}

				// 情况B：滚动发布导致 RS/hash 变化，尝试基于稳定标签(app/…)
				if sds.detectRollingUpdate(apiCtx, client, m) {
					sds.resourceManager.UpdatePodMigrationStatus(m.MigrationID, DrainMigrationCompleted, "")
				}
				apiCancel()
			}
		}
	}

	// Re-read after normalization
	migrations = sds.resourceManager.GetAllMigrations(drainID)
	for _, m := range migrations {
		if m.Status == DrainMigrationFailed || m.Status == DrainMigrationTimeout {
			finalFailedCount++
		}
	}

	success := finalFailedCount == 0
	if success {
		sds.operationManager.CompleteDrain(drainID, true, "Drain操作成功完成")
		if err := sds.stateManager.UpdateDrainStatus(drainID, DrainStatusCompleted); err != nil {
			sds.eventManager.SendError(drainID, "failed to update drain status to completed", err)
		}
		// 延迟关闭连接，确保前端收到完成事件
		go func() {
			time.Sleep(1 * time.Second)
			sds.eventManager.CloseStreams(drainID, "completed")
		}()
	} else {
		sds.operationManager.CompleteDrain(drainID, false, fmt.Sprintf("Drain操作完成，但有%d个Pod迁移失败", finalFailedCount))
		if err := sds.stateManager.UpdateDrainStatus(drainID, DrainStatusFailed); err != nil {
			sds.eventManager.SendError(drainID, "failed to update drain status to failed", err)
		}
		// 延迟关闭连接，确保前端收到失败事件
		go func() {
			time.Sleep(1 * time.Second)
			sds.eventManager.CloseStreams(drainID, "failed")
		}()
	}
}

// detectRollingUpdate 在滚动发布导致 pod-template-hash 变化时，发现同应用但哈希已变化的 Pod 即视为迁移成功
func (sds *SimpleDrainService) detectRollingUpdate(ctx context.Context, client kubernetes.Interface, migration *DrainPodMigrationInfo) bool {
	if migration == nil {
		return false
	}
	ns := migration.SourcePod.Namespace
	if ns == "" {
		return false
	}

	originalHash := ""
	if migration.SourcePod.Labels != nil {
		originalHash = migration.SourcePod.Labels["pod-template-hash"]
	}

	stableKeys := []string{"app.kubernetes.io/instance", "app.kubernetes.io/name", "app"}
	for _, key := range stableKeys {
		val, ok := migration.SourcePod.Labels[key]
		if !ok || val == "" {
			continue
		}

		selector := fmt.Sprintf("%s=%s", key, val)
		list, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil || list == nil || len(list.Items) == 0 {
			continue
		}

		lower := migration.StartTime.Add(-2 * time.Second)
		for i := range list.Items {
			pod := &list.Items[i]
			if !pod.CreationTimestamp.Time.After(lower) {
				continue
			}
			if pod.Spec.NodeName == migration.SourcePod.NodeName {
				continue
			}
			if pod.Labels == nil {
				continue
			}
			newHash := pod.Labels["pod-template-hash"]
			if newHash != "" && newHash != originalHash {
				return true
			}
		}
	}
	return false
}

// markRemainingAsFailed 将未终态的迁移标记为超时失败
func (sds *SimpleDrainService) markRemainingAsFailed(drainID, reason string) {
	for _, m := range sds.resourceManager.GetAllMigrations(drainID) {
		if m.Status != DrainMigrationCompleted && m.Status != DrainMigrationFailed {
			sds.resourceManager.UpdatePodMigrationStatus(m.MigrationID, DrainMigrationTimeout, reason)
		}
	}
}

// evictPodWithRetry 尝试驱逐 Pod 并带重试
func (sds *SimpleDrainService) evictPodWithRetry(ctx context.Context, client kubernetes.Interface, clusterName, namespace, podName string) error {
	if _, ok := ctx.Deadline(); !ok {
		return fmt.Errorf("context has no deadline, cannot safely retry")
	}

	// 先判断集群版本，选择 v1 或 v1beta1 的 Eviction API
	useBeta := sds.versionCache.ShouldUsePolicyV1beta1(clusterName, client)

	err := sds.retryExecutor.Execute(ctx, func() error {
		if useBeta {
			// v1beta1 eviction
			eviction := &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: namespace}}
			err := client.PolicyV1beta1().Evictions(namespace).Evict(ctx, eviction)
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		// v1 eviction (default)
		eviction := &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: namespace}}
		err := client.PolicyV1().Evictions(namespace).Evict(ctx, eviction)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to evict pod %s/%s: %w", namespace, podName, err)
	}
	return nil
}

// groupPodsForEviction groups pods for eviction.
// It groups by 'pod-template-hash' for ReplicaSet pods and 'job-name' for Job pods.
// Pods without either label are grouped individually to ensure they are processed.
func (sds *SimpleDrainService) groupPodsForEviction(pods []DrainPodInfo) map[string][]DrainPodInfo {
	groups := make(map[string][]DrainPodInfo)
	for _, pod := range pods {
		if hash, exists := pod.Labels["pod-template-hash"]; exists {
			groupKey := fmt.Sprintf("rs-%s/%s", pod.Namespace, hash)
			groups[groupKey] = append(groups[groupKey], pod)
		} else if jobName, exists := pod.Labels["job-name"]; exists {
			groupKey := fmt.Sprintf("job-%s/%s", pod.Namespace, jobName)
			groups[groupKey] = append(groups[groupKey], pod)
		} else {
			// For pods without a hash or job name, group them individually.
			groupKey := fmt.Sprintf("individual-%s/%s", pod.Namespace, pod.Name)
			groups[groupKey] = append(groups[groupKey], pod)
		}
	}
	return groups
}

// cleanup 清理 Drain 过程中的资源
func (sds *SimpleDrainService) cleanup(drainID string) {
	if state, _ := sds.stateManager.GetDrainState(drainID); state != nil {
		if err := sds.resourceManager.CleanupPDB(context.Background(), drainID, state.ClusterName); err != nil {
			sds.eventManager.SendError(drainID, "failed to cleanup pdb", err)
		}
	}
	sds.resourceManager.CleanupMigrationTracking(drainID)
	sds.operationManager.Cleanup(drainID)

	sds.cleanupExpiredPDBs()
}

// HandleSSE 处理 SSE 连接
func (sds *SimpleDrainService) HandleSSE(c *gin.Context) {
	drainID := c.Param("drainId")
	if drainID == "" {
		drainID = c.Param("drain_id")
	}
	if drainID == "" {
		c.JSON(400, gin.H{"error": "drainId is required"})
		return
	}

	clientIP := c.ClientIP()
	connectionKey := fmt.Sprintf("%s:%s", drainID, clientIP)
	now := time.Now().Unix()

	go func(id, key string, ts int64) {
		if !sds.shouldSendBootstrap(id, key, ts) {
			return
		}
		time.Sleep(120 * time.Millisecond)
		infos := sds.resourceManager.getPDBInfos(id)
		if len(infos) > 0 {
			for _, pdb := range infos {
				sds.eventManager.SendMessage(id, PdbEvent, "pdb created (bootstrap)", map[string]interface{}{
					"action":    "created",
					"pdbKey":    pdb.Key,
					"pdbName":   pdb.Name,
					"namespace": pdb.Namespace,
					"bootstrap": true,
				})
			}
			sds.eventManager.SendMessage(id, DrainStatsEvt, "stats", map[string]interface{}{
				"pdbCreated": len(infos),
				"bootstrap":  true,
			})
		}

		// 无论是否有迁移数据，都尝试获取并发送，确保ignored状态的Pod也能被正确推送
		migs := sds.resourceManager.GetAllMigrations(id)

		// 获取快照数据并发送完整的Pod信息
		snapshot := sds.resourceManager.GetSnapshot(id)
		if snapshot != nil {
			podData := make([]map[string]interface{}, 0, len(snapshot.Pods))
			for _, pod := range snapshot.Pods {
				podInfo := map[string]interface{}{
					"name":            pod.Name,
					"namespace":       pod.Namespace,
					"nodeName":        pod.NodeName,
					"uid":             pod.UID,
					"phase":           pod.Phase,
					"createdAt":       pod.CreatedAt,
					"labels":          pod.Labels,
					"annotations":     pod.Annotations,
					"ownerReferences": pod.OwnerReferences,
					"shouldIgnore":    sds.shouldIgnorePod(&pod),
				}
				podData = append(podData, podInfo)
			}

			sds.eventManager.SendMessage(id, "pod_snapshot_created", "已创建Pod快照", map[string]interface{}{
				"snapshot": map[string]interface{}{
					"totalPods": len(snapshot.Pods),
					"pods":      podData,
				},
				"bootstrap": true,
			})
		} else if len(migs) == 0 {
			// 如果没有快照数据也没有迁移数据，发送一个空的快照事件
			sds.eventManager.SendMessage(id, "pod_snapshot_created", "已创建Pod快照", map[string]interface{}{
				"snapshot": map[string]interface{}{
					"totalPods": 0,
					"pods":      []interface{}{},
				},
				"bootstrap": true,
			})
		}

		// 发送当前迁移数据（注意：这里只发送，不修改状态！）
		// 之前的逻辑错误地将非终态迁移标记为 canceled，导致进行中的 drain 被意外取消
		if len(migs) > 0 {
			// 发送每个迁移的当前状态，让前端可以恢复/显示正确的状态
			for _, m := range migs {
				if m != nil {
					sds.eventManager.SendMessage(id, "migration_update", "migration status", map[string]interface{}{
						"migration": m,
						"bootstrap": true,
					})
				}
			}
		}
	}(drainID, connectionKey, now)

	sds.eventManager.HandleSSE(c, drainID)
}

// shouldSendBootstrap 判定是否需要发送引导事件
func (sds *SimpleDrainService) shouldSendBootstrap(drainID, connectionKey string, now int64) bool {
	if state, err := sds.stateManager.GetDrainState(drainID); err == nil && state != nil {
		if state.Status == DrainStatusCompleted || state.Status == DrainStatusFailed || state.Status == DrainStatusCanceled {
			return false
		}
	}
	return true
}

// CancelDrain 取消 Drain 操作
func (sds *SimpleDrainService) CancelDrain(drainID string) error {
	// 广播取消到控制通道，确保跨实例可见
	if sds.redisHandler != nil {
		_ = sds.redisHandler.Pub(fmt.Sprintf("drain:control:%s", drainID), "cancel")
	}
	// 本实例也尝试本地取消
	if cf := sds.popDrainCancel(drainID); cf != nil {
		cf()
	}
	if err := sds.stateManager.UpdateDrainStatus(drainID, DrainStatusCanceled); err != nil {
		sds.eventManager.SendError(drainID, "failed to update drain status to canceled", err)
	}
	sds.operationManager.CompleteDrain(drainID, false, "Drain操作已取消")
	sds.eventManager.CloseStreams(drainID, "canceled")
	sds.cleanup(drainID)
	return nil
}

// GetDrainMigrations 获取 Drain 的迁移信息
func (sds *SimpleDrainService) GetDrainMigrations(drainID string) ([]*DrainPodMigrationInfo, error) {
	return sds.resourceManager.GetAllMigrations(drainID), nil
}

// SimpleDrainRequest 定义简化 Drain 请求
type SimpleDrainRequest struct {
	NodeName    string `json:"nodeName" binding:"required"`
	ClusterName string `json:"clusterName" binding:"required"`
	DryRun      bool   `json:"dryRun"`
	Force       bool   `json:"force"`
}

// SimpleDrainResponse 定义简化 Drain 响应
type SimpleDrainResponse struct {
	DrainID string `json:"drainId"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// buildDetailedEvictionError 组装驱逐失败详情
func (sds *SimpleDrainService) buildDetailedEvictionError(pod *DrainPodInfo, groupKey string, err error, failedCount, totalInGroup int) string {
	var reason string
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "disruption"), strings.Contains(errStr, "pdb"):
		reason = "PDB 限制"
	case strings.Contains(errStr, "timeout"):
		reason = "驱逐超时"
	case strings.Contains(errStr, "forbidden"), strings.Contains(errStr, "denied"):
		reason = "权限不足 (RBAC)"
	case strings.Contains(errStr, "not found"):
		reason = "Pod 不存在"
	default:
		reason = err.Error()
	}

	msg := fmt.Sprintf("pod %s/%s (组: %s) 驱逐失败。原因: %s。统计: 当前组失败 %d/%d 个Pod。",
		pod.Namespace, pod.Name, groupKey, reason, failedCount, totalInGroup)

	if failedCount > 1 {
		msg += " 警告: 同一组内多个Pod驱逐失败，可能存在系统性问题。"
	}
	return msg
}

// generateDrainID 生成 Drain ID
func generateDrainID() string {
	return fmt.Sprintf("drain-%d-%d", time.Now().UnixNano(), rand.Int63())
}

// isPDBBlockError 判断错误是否为 PDB 阻塞
func (sds *SimpleDrainService) isPDBBlockError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "disruption budget") || strings.Contains(e, "pdb") || strings.Contains(e, "violate the pod's disruption budget")
}

// schedulePDBRetry 安排 PDB 受限后的重试
func (sds *SimpleDrainService) schedulePDBRetry(ctx context.Context, client kubernetes.Interface, clusterName, drainID, rsKey, migrationID, namespace, podName string) {
	sds.pdbMutex.Lock()
	state, exists := sds.pdbRetryStates[migrationID]
	if !exists {
		state = &pdbRetryState{migrationID: migrationID, rsKey: rsKey, namespace: namespace, podName: podName, clusterName: clusterName}
		sds.pdbRetryStates[migrationID] = state
	} else if clusterName != "" {
		state.clusterName = clusterName
	}
	base := 10 * time.Second
	maxDur := 30 * time.Second
	delay := base * time.Duration(1<<min(state.attempts, 4))
	if delay > maxDur {
		delay = maxDur
	}
	jitter := 0.7 + rand.Float64()*0.6
	delay = time.Duration(float64(delay) * jitter)
	state.nextRetryAt = time.Now().Add(delay)
	nextAt := state.nextRetryAt // 复制时间戳，避免解锁后读写竞态
	state.attempts++
	sds.pdbMutex.Unlock()

	sds.eventManager.SendProgressWithPod(drainID, "pdb_wait",
		fmt.Sprintf("PDB限制，稍后将自动重试: %s", nextAt.Format(time.RFC3339)), 0, podName, namespace)

	go func(expectedAttempt int) {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			sds.tryEvictAfterPDB(ctx, client, state.clusterName, drainID, migrationID, rsKey, namespace, podName, expectedAttempt)
		}
	}(state.attempts)
}

// tryEvictAfterPDB 在 PDB 约束放宽后重试驱逐
func (sds *SimpleDrainService) tryEvictAfterPDB(ctx context.Context, client kubernetes.Interface, clusterName, drainID, migrationID, rsKey, namespace, podName string, attempt int) {
	sds.pdbMutex.Lock()
	if sds.pdbInFlightPerRS[rsKey] {
		sds.pdbMutex.Unlock()
		sds.schedulePDBRetry(ctx, client, clusterName, drainID, rsKey, migrationID, namespace, podName)
		return
	}
	sds.pdbInFlightPerRS[rsKey] = true
	sds.pdbMutex.Unlock()

	defer func() {
		sds.pdbMutex.Lock()
		sds.pdbInFlightPerRS[rsKey] = false
		sds.pdbMutex.Unlock()
	}()

	mi := sds.resourceManager.GetPodMigrationInfo(migrationID)
	if mi == nil || mi.Status == DrainMigrationCompleted || mi.Status == DrainMigrationFailed || mi.Status == DrainMigrationTimeout {
		return
	}

	sds.eventManager.SendProgressWithPod(drainID, "pdb_retry", "因PDB限制触发的自动重试", 0, podName, namespace)

	sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationEvicting, "")
	if err := sds.evictPodWithRetry(ctx, client, clusterName, namespace, podName); err != nil {
		if sds.isPDBBlockError(err) {
			sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationPending, "PDB限制，等待可中断窗口")
			sds.schedulePDBRetry(ctx, client, clusterName, drainID, rsKey, migrationID, namespace, podName)
			return
		}
		detailedError := sds.buildDetailedEvictionError(&DrainPodInfo{Namespace: namespace, Name: podName}, rsKey, err, 1, 1)
		sds.resourceManager.UpdatePodMigrationStatus(migrationID, DrainMigrationFailed, detailedError)
		return
	}
	sds.handleSuccessfulEviction(drainID, migrationID, &DrainPodInfo{Namespace: namespace, Name: podName, Labels: map[string]string{"pod-template-hash": ""}})
}

// PDBCleanupContext 表示 PDB 清理上下文
type PDBCleanupContext struct {
	expirationDays int
	cutoffTime     time.Time
	cleanedCount   int
}

// NewPDBCleanupContext 创建 PDB 清理上下文
func NewPDBCleanupContext(expirationDays int) *PDBCleanupContext {
	return &PDBCleanupContext{
		expirationDays: expirationDays,
		cutoffTime:     time.Now().AddDate(0, 0, -expirationDays),
		cleanedCount:   0,
	}
}

// cleanupExpiredPDBs 清理超期 PDB
func (sds *SimpleDrainService) cleanupExpiredPDBs() {
	const expirationDays = 7
	ctx := NewPDBCleanupContext(expirationDays)

	keys, err := sds.scanPDBKeys()
	if err != nil {
		sds.eventManager.SendError("", "Failed to scan PDB keys for cleanup", err)
		return
	}

	for _, key := range keys {
		drainID := sds.extractDrainIDFromKey(key)
		if drainID == "" {
			continue
		}

		pdbInfos := sds.resourceManager.getPDBInfos(drainID)
		if len(pdbInfos) == 0 {
			continue
		}

		expiredPDBs := sds.filterExpiredPDBs(pdbInfos, ctx.cutoffTime)
		if len(expiredPDBs) == 0 {
			continue
		}

		clusterName := sds.getClusterNameForDrain(drainID)
		sds.cleanupPDBsForDrain(drainID, key, pdbInfos, expiredPDBs, clusterName, ctx)
	}

	if ctx.cleanedCount > 0 {
		sds.sendCleanupSummary(ctx)
	}
}

// scanPDBKeys 扫描 PDB 键名
func (sds *SimpleDrainService) scanPDBKeys() ([]string, error) {
	pattern := "drain:pdb:*"
	return sds.resourceManager.redisHandler.ScanKeys(pattern)
}

// extractDrainIDFromKey 从 Redis 键名中提取 drainID
func (sds *SimpleDrainService) extractDrainIDFromKey(key string) string {
	const prefix = "drain:pdb:"
	if len(key) <= len(prefix) {
		return ""
	}
	return key[len(prefix):]
}

// filterExpiredPDBs 过滤超期 PDB
func (sds *SimpleDrainService) filterExpiredPDBs(pdbInfos []PDBInfo, cutoffTime time.Time) []PDBInfo {
	var expiredPDBs []PDBInfo
	for _, pdbInfo := range pdbInfos {
		if pdbInfo.CreatedAt.Before(cutoffTime) {
			expiredPDBs = append(expiredPDBs, pdbInfo)
		}
	}
	return expiredPDBs
}

// getClusterNameForDrain 获取 Drain 对应的集群名
func (sds *SimpleDrainService) getClusterNameForDrain(drainID string) string {
	clusterName := "default"
	if state, err := sds.stateManager.GetDrainState(drainID); err == nil && state != nil {
		clusterName = state.ClusterName
	}
	return clusterName
}

// cleanupPDBsForDrain 清理指定 Drain 的超期 PDB
func (sds *SimpleDrainService) cleanupPDBsForDrain(drainID, key string, pdbInfos, expiredPDBs []PDBInfo, clusterName string, ctx *PDBCleanupContext) {
	for _, pdbInfo := range expiredPDBs {
		if sds.deletePDBFromCluster(drainID, pdbInfo, clusterName) {
			sds.updateRedisAfterPDBDeletion(key, pdbInfos, pdbInfo)
			ctx.cleanedCount++
			sds.sendPDBCleanupEvent(drainID, pdbInfo)
		}
	}
}

// deletePDBFromCluster 从集群中删除 PDB
func (sds *SimpleDrainService) deletePDBFromCluster(drainID string, pdbInfo PDBInfo, clusterName string) bool {
	ctx := context.Background()
	client, err := sds.clientFactory.GetClient(clusterName)
	if err != nil {
		return false
	}

	err = client.PolicyV1().PodDisruptionBudgets(pdbInfo.Namespace).Delete(ctx, pdbInfo.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		sds.eventManager.SendError(drainID, fmt.Sprintf("Failed to delete expired PDB %s/%s", pdbInfo.Namespace, pdbInfo.Name), err)
		return false
	}

	return true
}

// updateRedisAfterPDBDeletion 删除 PDB 后更新 Redis
func (sds *SimpleDrainService) updateRedisAfterPDBDeletion(key string, pdbInfos []PDBInfo, deletedPDB PDBInfo) {
	remainingPDBs := make([]PDBInfo, 0, len(pdbInfos)-1)
	for _, pdb := range pdbInfos {
		if pdb.Key != deletedPDB.Key {
			remainingPDBs = append(remainingPDBs, pdb)
		}
	}

	if len(remainingPDBs) > 0 {
		data, _ := json.Marshal(remainingPDBs)
		sds.resourceManager.redisHandler.Set(key, string(data))
	} else {
		sds.resourceManager.redisHandler.Delete(key)
	}
}

// sendPDBCleanupEvent 发送 PDB 清理事件
func (sds *SimpleDrainService) sendPDBCleanupEvent(drainID string, pdbInfo PDBInfo) {
	sds.eventManager.SendMessage(drainID, "pdb_cleanup",
		fmt.Sprintf("Cleaned up expired PDB: %s/%s (created: %s)",
			pdbInfo.Namespace, pdbInfo.Name, pdbInfo.CreatedAt.Format(time.RFC3339)),
		map[string]interface{}{
			"action":    "expired_cleanup",
			"pdbName":   pdbInfo.Name,
			"namespace": pdbInfo.Namespace,
			"pdbKey":    pdbInfo.Key,
			"createdAt": pdbInfo.CreatedAt,
			"daysOld":   int(time.Since(pdbInfo.CreatedAt).Hours() / 24),
		})
}

// sendCleanupSummary 发送 PDB 清理汇总
func (sds *SimpleDrainService) sendCleanupSummary(ctx *PDBCleanupContext) {
	sds.eventManager.SendMessage("", "pdb_cleanup_summary",
		fmt.Sprintf("PDB cleanup completed: cleaned %d expired PDB(s) older than %d days", ctx.cleanedCount, ctx.expirationDays),
		map[string]interface{}{
			"cleanedCount":   ctx.cleanedCount,
			"expirationDays": ctx.expirationDays,
			"cutoffTime":     ctx.cutoffTime,
		})
}

// checkNodeUnschedulable 校验节点是否为不可调度状态
// shouldIgnorePod 判断Pod是否应该被忽略（不进行驱逐）
func (sds *SimpleDrainService) shouldIgnorePod(pod *DrainPodInfo) bool {
	// 检查是否由DaemonSet管理
	if sds.isDaemonSetOwner(pod.OwnerReferences) {
		return true
	}

	// 检查是否由StatefulSet管理
	if sds.isStatefulSetOwner(pod.OwnerReferences) {
		return true
	}

	// 检查是否为静态Pod（没有OwnerReference）
	if len(pod.OwnerReferences) == 0 {
		return true
	}

	// 检查是否为系统命名空间的Pod
	systemNamespaces := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
	}
	if systemNamespaces[pod.Namespace] {
		return true
	}

	return false
}

// isDaemonSetOwner 判断是否由 DaemonSet 管理
func (sds *SimpleDrainService) isDaemonSetOwner(owners []metav1.OwnerReference) bool {
	for _, owner := range owners {
		if owner.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// isStatefulSetOwner 判断是否由 StatefulSet 管理
func (sds *SimpleDrainService) isStatefulSetOwner(owners []metav1.OwnerReference) bool {
	for _, owner := range owners {
		if owner.Kind == "StatefulSet" {
			return true
		}
	}
	return false
}
