package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"sort"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	drainPDBKeyFmt           = "drain:pdb:%s"
	drainSnapshotKeyFmt      = "drain:snapshot:%s"
	nodeUnreachablePodReason = "NodeLost"
)

// RetryConfig 表示重试配置
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 5,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   10 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// RetryableError 表示可重试错误接口
type RetryableError interface {
	error
	IsRetryable() bool
	ErrorType() string
}

// KubernetesError 表示 K8s 相关错误
type KubernetesError struct {
	Err       error
	ErrType   string
	Retryable bool
}

// Error 实现 error 接口
func (e *KubernetesError) Error() string {
	return e.Err.Error()
}

// IsRetryable 返回是否可重试
func (e *KubernetesError) IsRetryable() bool {
	return e.Retryable
}

// ErrorType 返回错误类型
func (e *KubernetesError) ErrorType() string {
	return e.ErrType
}

// NewKubernetesError 创建 KubernetesError 并判断可重试性
func NewKubernetesError(err error) RetryableError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())
	errorType := "unknown"
	isRetryable := false

	switch {
	case strings.Contains(errStr, "rate limit"), strings.Contains(errStr, "too many requests"):
		isRetryable, errorType = true, "rate_limit"
	case strings.Contains(errStr, "context canceled"), strings.Contains(errStr, "context deadline exceeded"):
		isRetryable, errorType = true, "context_timeout"
	case strings.Contains(errStr, "temporary failure"), strings.Contains(errStr, "service unavailable"):
		isRetryable, errorType = true, "temporary_failure"
	case strings.Contains(errStr, "connection refused"), strings.Contains(errStr, "connection reset"):
		isRetryable, errorType = true, "connection_error"
	case strings.Contains(errStr, "timeout"):
		isRetryable, errorType = true, "timeout"
	}

	return &KubernetesError{
		Err:       err,
		ErrType:   errorType,
		Retryable: isRetryable,
	}
}

// RetryExecutor 封装带重试的执行器
type RetryExecutor struct {
	config RetryConfig
}

// NewRetryExecutor 创建 RetryExecutor
func NewRetryExecutor(config RetryConfig) *RetryExecutor {
	return &RetryExecutor{config: config}
}

// Execute 按配置进行重试执行
func (re *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= re.config.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		k8sErr := NewKubernetesError(err)
		if k8sErr == nil || !k8sErr.IsRetryable() {
			return err
		}

		lastErr = err
		if attempt < re.config.MaxRetries {
			delay := re.calculateDelay(attempt)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return fmt.Errorf("context canceled: %w", ctx.Err())
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries, last error: %w", re.config.MaxRetries, lastErr)
}

// calculateDelay 计算下一次重试延迟
func (re *RetryExecutor) calculateDelay(attempt int) time.Duration {
	delay := re.config.BaseDelay * time.Duration(re.config.Multiplier*float64(attempt+1))

	if delay > re.config.MaxDelay {
		delay = re.config.MaxDelay
	}

	if re.config.Jitter {
		delay = time.Duration(float64(delay) * (0.8 + rand.Float64()*0.4))
	}

	return delay
}

// DrainResourceManager 负责 Drain 相关资源管理
type DrainResourceManager struct {
	clientFactory    sharedservices.K8sClientFactory
	redisHandler     sharedservices.RedisClient
	eventManager     *SimpleDrainEventManager
	retryExecutor    *RetryExecutor
	migrationTracker map[string]*DrainPodMigrationInfo
	trackerMutex     sync.RWMutex
	podWatchers      map[string]context.CancelFunc
	watcherMutex     sync.RWMutex
	versionCache     *sharedservices.ClusterVersionCache
	// stats throttle per drain
	statsMutex    sync.Mutex
	lastStatsSent map[string]time.Time
}

// NewDrainResourceManager 创建 DrainResourceManager
func NewDrainResourceManager(clientFactory sharedservices.K8sClientFactory, redisHandler sharedservices.RedisClient, eventManager *SimpleDrainEventManager, versionCache *sharedservices.ClusterVersionCache) *DrainResourceManager {
	if versionCache == nil {
		versionCache = sharedservices.NewClusterVersionCache(5 * time.Minute)
	}
	return &DrainResourceManager{
		clientFactory:    clientFactory,
		redisHandler:     redisHandler,
		eventManager:     eventManager,
		retryExecutor:    NewRetryExecutor(DefaultRetryConfig()),
		migrationTracker: make(map[string]*DrainPodMigrationInfo),
		podWatchers:      make(map[string]context.CancelFunc),
		versionCache:     versionCache,
		lastStatsSent:    make(map[string]time.Time),
	}
}

// Redis keys for migration persistence
func (drm *DrainResourceManager) migIndexKey(drainID string) string {
	return fmt.Sprintf("drain:migrations:%s", drainID)
}
func (drm *DrainResourceManager) migInfoKey(migrationID string) string {
	return fmt.Sprintf("drain:migration:%s", migrationID)
}

var migrationTTL = 24 * time.Hour

func (drm *DrainResourceManager) saveMigrationInfo(m *DrainPodMigrationInfo) {
	if drm.redisHandler == nil || m == nil {
		return
	}

	// 防止跨实例旧状态覆盖：读取已存在的迁移信息并比较状态优先级
	if existing := drm.loadMigrationInfo(m.MigrationID); existing != nil {
		// 合并：选择更“新”的状态；同级别时保留包含 targetPod 的版本
		if statusPriority[m.Status] < statusPriority[existing.Status] {
			// 新数据状态不如旧数据，保持旧值，但可合并更多字段
			merged := *existing
			if merged.TargetPod == nil && m.TargetPod != nil {
				merged.TargetPod = m.TargetPod
			}
			if merged.ErrorMessage == "" && m.ErrorMessage != "" {
				merged.ErrorMessage = m.ErrorMessage
			}
			if b, err := json.Marshal(&merged); err == nil {
				drm.redisHandler.SetWithExpireTime(drm.migInfoKey(m.MigrationID), string(b), migrationTTL)
			}
			return
		}
		if statusPriority[m.Status] == statusPriority[existing.Status] {
			// 同优先级，尽量保留更完整的信息
			merged := *existing
			// 以 m 为主，但补齐缺失字段
			merged.Status = m.Status
			merged.Progress = m.Progress
			merged.EvictionTime = m.EvictionTime
			merged.CompletionTime = m.CompletionTime
			if m.TargetPod != nil {
				merged.TargetPod = m.TargetPod
			}
			if m.ErrorMessage != "" {
				merged.ErrorMessage = m.ErrorMessage
			}
			if b, err := json.Marshal(&merged); err == nil {
				drm.redisHandler.SetWithExpireTime(drm.migInfoKey(m.MigrationID), string(b), migrationTTL)
			}
			return
		}
		// m 的状态更先进，直接覆盖
	}

	if b, err := json.Marshal(m); err == nil {
		drm.redisHandler.SetWithExpireTime(drm.migInfoKey(m.MigrationID), string(b), migrationTTL)
	}
}
func (drm *DrainResourceManager) loadMigrationInfo(migrationID string) *DrainPodMigrationInfo {
	if drm.redisHandler == nil {
		return nil
	}
	if s := drm.redisHandler.Get(drm.migInfoKey(migrationID)); s != "" {
		var m DrainPodMigrationInfo
		if json.Unmarshal([]byte(s), &m) == nil {
			return &m
		}
	}
	return nil
}
func (drm *DrainResourceManager) appendMigrationID(drainID, migrationID string) {
	if drm.redisHandler == nil {
		return
	}
	key := drm.migIndexKey(drainID)
	var list []string
	if s := drm.redisHandler.Get(key); s != "" {
		_ = json.Unmarshal([]byte(s), &list)
	}
	// ensure unique
	found := false
	for _, id := range list {
		if id == migrationID {
			found = true
			break
		}
	}
	if !found {
		list = append(list, migrationID)
		if b, err := json.Marshal(list); err == nil {
			drm.redisHandler.SetWithExpireTime(key, string(b), migrationTTL)
		}
	}
}
func (drm *DrainResourceManager) listMigrationIDs(drainID string) []string {
	if drm.redisHandler == nil {
		return nil
	}
	if s := drm.redisHandler.Get(drm.migIndexKey(drainID)); s != "" {
		var list []string
		if json.Unmarshal([]byte(s), &list) == nil {
			return list
		}
	}
	return nil
}

// deleteMigrationsForDrain 保留为空实现：通过 TTL 自动清理
func (drm *DrainResourceManager) deleteMigrationsForDrain(drainID string) {
	// Intentionally no-op: we preserve migrations in Redis for a short time
	// to allow post-completion inspection. Keys expire via TTL (migrationTTL).
}

// getReader 获取 controller-runtime 的 client.Reader
func (drm *DrainResourceManager) getReader(clusterName string) (crclient.Client, error) {
	mgr := sharedservices.GlobalClusterConnectionManager
	if mgr == nil {
		return nil, fmt.Errorf("cluster connection manager not initialized")
	}

	m, err := mgr.GetManager(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager for cluster %s: %w", clusterName, err)
	}

	// 检查缓存是否已同步，设置超时避免长时间阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !m.GetCache().WaitForCacheSync(ctx) {
		log.Printf("Warning: manager cache not fully synced for cluster %s, queries may use fallback API calls", clusterName)
		// 不返回错误，让调用方使用带有fallback的查询机制
	}

	return m.GetClient(), nil
}

// CreatePDB 为一组 Pod 创建 PDB
func (drm *DrainResourceManager) CreatePDB(ctx context.Context, drainID, clusterName, namespace, groupName string, minAvailable int, pods []DrainPodInfo, ownerLease *coordinationv1.Lease) (string, error) {
	reader, err := drm.getReader(clusterName)
	if err != nil {
		return "", fmt.Errorf("failed to get manager reader: %w", err)
	}

	rsName, rsUID := drm.getReplicaSetInfoFromPods(ctx, reader, pods)
	pdbName := drm.generatePDBName(groupName, rsName)
	pdbKey := fmt.Sprintf("%s/%s", namespace, pdbName)

	if drm.pdbExists(drainID, pdbKey) {
		return pdbKey, nil
	}

	matchLabels := drm.generatePDBSelectorFromPods(pods)
	pdb := drm.buildPDB(drainID, namespace, pdbName, rsName, rsUID, matchLabels, ownerLease)

	if err := drm.createPDBResource(ctx, reader, clusterName, pdb); err != nil {
		return "", err
	}

	drm.savePDBInfo(drainID, pdbKey, pdbName, namespace)
	drm.eventManager.SendMessage(drainID, PdbEvent, "pdb created", map[string]interface{}{
		"action":         "created",
		"pdbKey":         pdbKey,
		"pdbName":        pdbName,
		"namespace":      namespace,
		"groupName":      groupName,
		"selector":       matchLabels,
		"maxUnavailable": 1,
	})
	if infos := drm.getPDBInfos(drainID); len(infos) >= 0 {
		drm.eventManager.SendMessage(drainID, DrainStatsEvt, "stats", map[string]interface{}{
			"pdbCreated": len(infos),
		})
	}

	return pdbKey, nil
}

// CleanupPDB 清理指定 Drain 创建的所有 PDB (改为清理 Lease)
func (drm *DrainResourceManager) CleanupPDB(ctx context.Context, drainID, clusterName string) error {
	infos := drm.getPDBInfos(drainID)
	if len(infos) == 0 {
		return nil
	}

	// 从快照获取节点名称
	snapshot := drm.GetSnapshot(drainID)
	var nodeName string
	if snapshot != nil {
		nodeName = snapshot.NodeName
	}
	if nodeName == "" {
		// 尝试从迁移记录获取
		migs := drm.GetAllMigrations(drainID)
		if len(migs) > 0 {
			nodeName = migs[0].SourcePod.NodeName
		}
	}

	if nodeName != "" {
		// 收集涉及的 namespace
		namespaces := make(map[string]struct{})
		for _, info := range infos {
			namespaces[info.Namespace] = struct{}{}
		}

		for ns := range namespaces {
			// 删除 Lease，PDB 会被 GC
			if err := drm.DeleteNodeLease(ctx, clusterName, ns, nodeName); err != nil {
				log.Printf("Warning: Failed to delete lease in %s: %v", ns, err)
			}
		}
	}

	drm.deletePDBInfo(drainID)
	drm.eventManager.SendMessage(drainID, DrainStatsEvt, "stats", map[string]interface{}{
		"pdbCreated": 0,
	})
	return nil
}

// EnsureNodeLease creates or updates a Lease for the drain operation in the specified namespace
func (drm *DrainResourceManager) EnsureNodeLease(ctx context.Context, clusterName, namespace, nodeName, drainID string) (*coordinationv1.Lease, error) {
	client, err := drm.clientFactory.GetClient(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	leaseName := fmt.Sprintf("drain-lease-%s", nodeName)
	holderIdentity := drainID
	leaseDuration := int32(600) // 10 minutes

	// Try to get existing lease
	existing, err := client.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
	if err == nil {
		// Update existing
		existing.Spec.HolderIdentity = &holderIdentity
		existing.Spec.LeaseDurationSeconds = &leaseDuration
		existing.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
		return client.CoordinationV1().Leases(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}

	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get lease: %w", err)
	}

	// Create new
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
			Labels:    map[string]string{"drain-id": drainID, "created-by": "safe-drain"},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &holderIdentity,
			LeaseDurationSeconds: &leaseDuration,
			AcquireTime:          &metav1.MicroTime{Time: time.Now()},
			RenewTime:            &metav1.MicroTime{Time: time.Now()},
		},
	}

	return client.CoordinationV1().Leases(namespace).Create(ctx, lease, metav1.CreateOptions{})
}

// DeleteNodeLease deletes the node lease in the specified namespace
func (drm *DrainResourceManager) DeleteNodeLease(ctx context.Context, clusterName, namespace, nodeName string) error {
	client, err := drm.clientFactory.GetClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	leaseName := fmt.Sprintf("drain-lease-%s", nodeName)
	err = client.CoordinationV1().Leases(namespace).Delete(ctx, leaseName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// StartPodMigrationTracking 初始化所有 Pod 的迁移跟踪
func (drm *DrainResourceManager) StartPodMigrationTracking(ctx context.Context, drainID, clusterName string, pods []DrainPodInfo) error {
	drm.trackerMutex.Lock()
	for _, pod := range pods {
		migrationID := fmt.Sprintf("%s-%s-%s", drainID, pod.Namespace, pod.Name)
		drm.migrationTracker[migrationID] = &DrainPodMigrationInfo{
			DrainID:        drainID,
			SourcePod:      pod,
			Status:         DrainMigrationPending,
			StartTime:      time.Now(),
			MigrationID:    migrationID,
			OwnerReference: drm.extractOwnerReference(&pod),
		}
		if drm.isDaemonSetOwner(pod.OwnerReferences) || drm.isStatefulSetOwner(pod.OwnerReferences) || len(pod.OwnerReferences) == 0 {
			drm.updatePodMigrationStatusInternal(migrationID, DrainMigrationIgnored, "daemonset/statefulset/static pod ignored")
		}
		// Persist to Redis index and detail
		drm.appendMigrationID(drainID, migrationID)
		drm.saveMigrationInfo(drm.migrationTracker[migrationID])
	}
	drm.trackerMutex.Unlock()

	drm.sendMigrationStats(drainID)

	if err := drm.startPodWatcher(ctx, drainID, clusterName); err != nil {
		return err
	}

	// Start the drain-level health monitor to report on blocking failures.
	go drm.monitorDrainHealth(ctx, drainID)

	return nil
}

// monitorDrainHealth periodically checks the overall status of a drain operation.
// If it finds failed pods that are blocking completion, it sends a periodic notification.
func (drm *DrainResourceManager) monitorDrainHealth(ctx context.Context, drainID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			drm.trackerMutex.RLock()

			var failedMigrationDetails []string
			var pendingMigrations int
			var totalMigrations int

			for _, m := range drm.migrationTracker {
				if m.DrainID == drainID {
					totalMigrations++
					if m.Status == DrainMigrationFailed || m.Status == DrainMigrationTimeout {
						reason := m.ErrorMessage
						if reason == "" {
							reason = "未知错误"
						}
						detail := fmt.Sprintf("%s (状态: %s, 原因: %s)", m.SourcePod.Name, m.Status, reason)
						failedMigrationDetails = append(failedMigrationDetails, detail)
					}
					if m.Status != DrainMigrationCompleted && m.Status != DrainMigrationIgnored && m.Status != DrainMigrationFailed && m.Status != DrainMigrationTimeout {
						pendingMigrations++
					}
				}
			}
			drm.trackerMutex.RUnlock()

			// If there are failed pods, send a blocking notification.
			// This check now happens *before* the exit condition.
			if len(failedMigrationDetails) > 0 {
				logMessage := fmt.Sprintf(
					"驱逐流程被阻塞。下列 %d 个 Pod 迁移失败: %s。",
					len(failedMigrationDetails),
					strings.Join(failedMigrationDetails, "; "),
				)
				drm.eventManager.SendMessage(
					drainID,
					"log", // Set type to 'log' so frontend LogViewer can process it
					logMessage,
					map[string]interface{}{
						"level":      "ERROR", // This is a blocking error
						"message":    logMessage,
						"reason":     "failed_pods_blocking",
						"failedPods": failedMigrationDetails,
					},
				)
			}
			// If all migrations are in a terminal state, stop monitoring.
			if pendingMigrations == 0 && totalMigrations > 0 {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// UpdatePodMigrationStatus 更新指定迁移的状态
func (drm *DrainResourceManager) UpdatePodMigrationStatus(migrationID string, status DrainPodMigrationStatus, errorMsg string) {
	var migrationCopy *DrainPodMigrationInfo
	var drainID string

	drm.trackerMutex.Lock()
	drm.updatePodMigrationStatusInternal(migrationID, status, errorMsg)
	if migration, exists := drm.migrationTracker[migrationID]; exists {
		migrationCopy = new(DrainPodMigrationInfo)
		*migrationCopy = *migration
		drainID = migration.DrainID
	}
	drm.trackerMutex.Unlock()

	if migrationCopy != nil {
		drm.saveMigrationInfo(migrationCopy)
	}
	if drainID != "" {
		drm.sendMigrationStats(drainID)
	}
}

// GetPodMigrationInfo 获取单个迁移信息
func (drm *DrainResourceManager) GetPodMigrationInfo(migrationID string) *DrainPodMigrationInfo {
	// Prefer Redis
	if m := drm.loadMigrationInfo(migrationID); m != nil {
		return m
	}
	drm.trackerMutex.RLock()
	defer drm.trackerMutex.RUnlock()
	if migration, exists := drm.migrationTracker[migrationID]; exists {
		migrationCopy := *migration
		return &migrationCopy
	}
	return nil
}

// GetAllMigrations 获取指定 Drain 的全部迁移信息
func (drm *DrainResourceManager) GetAllMigrations(drainID string) []*DrainPodMigrationInfo {
	ids := drm.listMigrationIDs(drainID)
	if len(ids) == 0 {
		return nil
	}

	migrations := make([]*DrainPodMigrationInfo, 0, len(ids))
	for _, id := range ids {
		if m := drm.loadMigrationInfo(id); m != nil {
			migrations = append(migrations, m)
		}
	}

	if len(migrations) == 0 {
		return nil
	}

	migrations = drm.mergeWithLocalTracker(drainID, migrations)

	sortMigrations(migrations)
	return migrations
}

var statusPriority = map[DrainPodMigrationStatus]int{
	DrainMigrationPending:   1,
	DrainMigrationEvicting:  2,
	DrainMigrationEvicted:   3,
	DrainMigrationCreating:  4,
	DrainMigrationCompleted: 5,
	DrainMigrationFailed:    6,
	DrainMigrationTimeout:   6,
	DrainMigrationIgnored:   7,
}

func (drm *DrainResourceManager) mergeWithLocalTracker(drainID string, remote []*DrainPodMigrationInfo) []*DrainPodMigrationInfo {
	if remote == nil {
		remote = []*DrainPodMigrationInfo{}
	}

	remoteMap := make(map[string]*DrainPodMigrationInfo, len(remote))
	for _, r := range remote {
		if r == nil {
			continue
		}
		remoteMap[r.MigrationID] = r
	}

	drm.trackerMutex.RLock()
	defer drm.trackerMutex.RUnlock()

	for id, local := range drm.migrationTracker {
		if local == nil || local.DrainID != drainID {
			continue
		}

		copyLocal := *local
		if remoteMig, ok := remoteMap[id]; ok {
			if statusPriority[copyLocal.Status] > statusPriority[remoteMig.Status] {
				remoteMap[id] = &copyLocal
			} else if statusPriority[copyLocal.Status] == statusPriority[remoteMig.Status] {
				if copyLocal.TargetPod != nil && remoteMig.TargetPod == nil {
					remoteMap[id] = &copyLocal
				}
			}
			continue
		}
		remoteMap[id] = &copyLocal
	}

	merged := make([]*DrainPodMigrationInfo, 0, len(remoteMap))
	for _, mig := range remoteMap {
		merged = append(merged, mig)
	}
	return merged
}

func sortMigrations(migrations []*DrainPodMigrationInfo) {
	sort.Slice(migrations, func(i, j int) bool {
		mi, mj := migrations[i], migrations[j]
		if mi == nil || mj == nil {
			return i < j
		}
		if mi.SourcePod.Namespace == mj.SourcePod.Namespace {
			return mi.SourcePod.Name < mj.SourcePod.Name
		}
		return mi.SourcePod.Namespace < mj.SourcePod.Namespace
	})
}

// CreateSnapshot 创建节点上 Pod 的快照
func (drm *DrainResourceManager) CreateSnapshot(ctx context.Context, drainID, clusterName, nodeName string) (*DrainSnapshotInfo, error) {
	var podList corev1.PodList

	// First try direct API client (works without controller-runtime manager, easier for tests)
	if cs, cerr := drm.clientFactory.GetClient(clusterName); cerr == nil {
		if podsFromAPI, aerr := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + nodeName}); aerr == nil {
			podList = *podsFromAPI
		}
	}

	// If API client path didn't populate, try manager reader with field index
	if len(podList.Items) == 0 {
		if reader, err := drm.getReader(clusterName); err == nil {
			_ = reader.List(ctx, &podList, crclient.MatchingFields{"spec.nodeName": nodeName})
		}
	}

	// If still empty, proceed with an empty snapshot (valid case: node has no pods)

	snapshot := &DrainSnapshotInfo{
		DrainID: drainID, NodeName: nodeName, ClusterName: clusterName, CreatedAt: time.Now(),
		Pods: make([]DrainPodInfo, 0, len(podList.Items)),
	}

	for _, pod := range podList.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		// 包含所有Pod（包括系统Pod），但标记哪些应该被忽略
		info := drm.podToDrainPodInfo(&pod, true)
		snapshot.Pods = append(snapshot.Pods, info)
	}
	snapshot.TotalPods = len(snapshot.Pods)

	drm.saveSnapshot(drainID, snapshot)
	drm.eventManager.SendMessage(drainID, DrainProgress,
		fmt.Sprintf("创建快照成功，共%d个Pod", len(snapshot.Pods)),
		map[string]interface{}{"podCount": len(snapshot.Pods)})

	return snapshot, nil
}

// GetSnapshot 获取先前创建的快照
func (drm *DrainResourceManager) GetSnapshot(drainID string) *DrainSnapshotInfo {
	return drm.getSnapshot(drainID)
}

// StopPodWatcher 停止指定 Drain 的 Pod 监听
func (drm *DrainResourceManager) StopPodWatcher(drainID string) {
	drm.watcherMutex.Lock()
	defer drm.watcherMutex.Unlock()

	if cancel, exists := drm.podWatchers[drainID]; exists {
		cancel()
		delete(drm.podWatchers, drainID)
	}
}

// CleanupMigrationTracking 清理指定 Drain 的迁移跟踪
func (drm *DrainResourceManager) CleanupMigrationTracking(drainID string) {
	drm.trackerMutex.Lock()
	defer drm.trackerMutex.Unlock()

	newTracker := make(map[string]*DrainPodMigrationInfo)
	for migrationID, migration := range drm.migrationTracker {
		if migration.DrainID != drainID {
			newTracker[migrationID] = migration
		}
	}
	drm.migrationTracker = newTracker

	drm.StopPodWatcher(drainID)
	// Remove persisted state in Redis
	drm.deleteMigrationsForDrain(drainID)
}

// generatePDBName 生成 PDB 名称
func (drm *DrainResourceManager) generatePDBName(groupName, rsName string) string {
	if rsName != "" {
		return rsName
	}
	if strings.Contains(groupName, "/") {
		if parts := strings.Split(groupName, "/"); len(parts) == 2 {
			return fmt.Sprintf("pdb-rs-%s", parts[1])
		}
	}
	return fmt.Sprintf("pdb-%s", groupName)
}

// buildPDB 构造 PDB 对象
func (drm *DrainResourceManager) buildPDB(drainID, namespace, pdbName, rsName, rsUID string, matchLabels map[string]string, ownerLease *coordinationv1.Lease) *policyv1.PodDisruptionBudget {
	objectMeta := metav1.ObjectMeta{
		Name:      pdbName,
		Namespace: namespace,
		Labels:    map[string]string{"drain-id": drainID, "created-by": "safe-drain"},
	}

	var ownerRefs []metav1.OwnerReference
	if rsName != "" && rsUID != "" {
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rsName, UID: types.UID(rsUID),
		})
	}
	if ownerLease != nil {
		blockOwnerDeletion := true
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion:         "coordination.k8s.io/v1",
			Kind:               "Lease",
			Name:               ownerLease.Name,
			UID:                ownerLease.UID,
			BlockOwnerDeletion: &blockOwnerDeletion,
		})
	}
	objectMeta.OwnerReferences = ownerRefs

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: objectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
			Selector:       &metav1.LabelSelector{MatchLabels: matchLabels},
		},
	}
}

// createPDBResource 创建 PDB 资源，包含回退机制
func (drm *DrainResourceManager) createPDBResource(ctx context.Context, reader crclient.Client, clusterName string, pdb *policyv1.PodDisruptionBudget) error {
	// 先根据集群版本决定使用 v1 还是 v1beta1
	cs, cerr := drm.clientFactory.GetClient(clusterName)
	if cerr != nil {
		return fmt.Errorf("failed to get K8s client for pdb creation: %w", cerr)
	}

	useBeta := drm.versionCache.ShouldUsePolicyV1beta1(clusterName, cs)

	if useBeta {
		beta := &policyv1beta1.PodDisruptionBudget{
			ObjectMeta: pdb.ObjectMeta,
			Spec: policyv1beta1.PodDisruptionBudgetSpec{
				MaxUnavailable: pdb.Spec.MaxUnavailable,
				Selector:       pdb.Spec.Selector,
			},
		}
		// 优先使用 controller-runtime 创建 v1beta1 对象
		if err := reader.Create(ctx, beta); err != nil && !apierrors.IsAlreadyExists(err) {
			if _, cerr := drm.createPDBWithRetryBeta(ctx, cs, beta.Namespace, beta); cerr != nil && !apierrors.IsAlreadyExists(cerr) {
				return fmt.Errorf("failed to create PDB(v1beta1): %w", cerr)
			}
		}
		return nil
	}

	// 默认使用 v1
	if err := reader.Create(ctx, pdb); err != nil && !apierrors.IsAlreadyExists(err) {
		if _, cerr := drm.createPDBWithRetry(ctx, cs, pdb.Namespace, pdb); cerr != nil && !apierrors.IsAlreadyExists(cerr) {
			return fmt.Errorf("failed to create PDB(v1): %w", cerr)
		}
	}
	return nil
}

// createPDBWithRetry 使用 client-go 重试创建 PDB
func (drm *DrainResourceManager) createPDBWithRetry(ctx context.Context, cs kubernetes.Interface, namespace string, pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
	var result *policyv1.PodDisruptionBudget
	err := drm.retryExecutor.Execute(ctx, func() error {
		pdbResult, err := cs.PolicyV1().PodDisruptionBudgets(namespace).Create(ctx, pdb, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		if err == nil {
			result = pdbResult
		}
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create PDB %s: %w", pdb.Name, err)
	}
	return result, nil
}

// createPDBWithRetryBeta 使用 client-go 重试创建 v1beta1 PDB
func (drm *DrainResourceManager) createPDBWithRetryBeta(ctx context.Context, cs kubernetes.Interface, namespace string, pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
	var result *policyv1beta1.PodDisruptionBudget
	err := drm.retryExecutor.Execute(ctx, func() error {
		pdbResult, err := cs.PolicyV1beta1().PodDisruptionBudgets(namespace).Create(ctx, pdb, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		if err == nil {
			result = pdbResult
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PDB %s: %w", pdb.Name, err)
	}
	return result, nil
}

// updatePodMigrationStatusInternal 无锁版本状态更新
func (drm *DrainResourceManager) updatePodMigrationStatusInternal(migrationID string, status DrainPodMigrationStatus, errorMsg string) {
	migration, exists := drm.migrationTracker[migrationID]
	if !exists {
		return
	}

	// 若迁移已进入成功/忽略等终态，则拒绝被失败或中间态覆盖，避免完成后被回写成失败
	if migration.Status == DrainMigrationCompleted || migration.Status == DrainMigrationIgnored {
		if status != migration.Status {
			return
		}
	}

	migration.Status = status
	migration.Duration = time.Since(migration.StartTime)
	if errorMsg != "" {
		migration.ErrorMessage = errorMsg
	}

	switch status {
	case DrainMigrationEvicting:
		migration.Progress = 25
	case DrainMigrationEvicted:
		migration.Progress = 50
		now := time.Now()
		migration.EvictionTime = &now
	case DrainMigrationCreating:
		migration.Progress = 75
	case DrainMigrationCompleted:
		migration.Progress = 100
		now := time.Now()
		migration.CompletionTime = &now
	case DrainMigrationFailed, DrainMigrationTimeout:
		migration.Progress = 0
	case DrainMigrationIgnored:
		migration.Progress = 0
	case DrainMigrationPending:
		migration.Progress = 0
	}

	drm.sendMigrationEvent(migration)
}

// sendMigrationEvent 按状态发送对应迁移事件
func (drm *DrainResourceManager) sendMigrationEvent(migration *DrainPodMigrationInfo) {
	switch migration.Status {
	case DrainMigrationEvicting:
		drm.eventManager.SendPodEvictionStarted(migration.DrainID, migration)
	case DrainMigrationEvicted:
		drm.eventManager.SendPodEvictionSucceeded(migration.DrainID, migration)
	case DrainMigrationCreating:
		drm.eventManager.SendNewPodDetected(migration.DrainID, migration)
	case DrainMigrationCompleted:
		drm.eventManager.SendPodMigrationCompleted(migration.DrainID, migration)
	case DrainMigrationFailed, DrainMigrationTimeout:
		drm.eventManager.SendPodMigrationFailed(migration.DrainID, migration)
	case DrainMigrationIgnored:
		drm.eventManager.SendPodMigrationIgnored(migration.DrainID, migration)
	case DrainMigrationPending:
		drm.eventManager.SendPodPending(migration.DrainID, migration)
	}
}

// sendMigrationStats 统计迁移数据并发送
func (drm *DrainResourceManager) sendMigrationStats(drainID string) {
	// throttle: at most once per 200ms per drainID
	drm.statsMutex.Lock()
	last := drm.lastStatsSent[drainID]
	if time.Since(last) < 500*time.Millisecond {
		drm.statsMutex.Unlock()
		return
	}
	drm.lastStatsSent[drainID] = time.Now()
	drm.statsMutex.Unlock()

	drm.trackerMutex.RLock()
	defer drm.trackerMutex.RUnlock()

	var total, migrated, failed, pending, evicting, evicted, creating, ignored int
	for _, m := range drm.migrationTracker {
		if m.DrainID != drainID {
			continue
		}
		total++
		switch m.Status {
		case DrainMigrationCompleted:
			migrated++
		case DrainMigrationFailed:
			failed++
		case DrainMigrationPending:
			pending++
		case DrainMigrationEvicting:
			evicting++
		case DrainMigrationEvicted:
			evicted++
		case DrainMigrationCreating:
			creating++
		case DrainMigrationIgnored:
			ignored++
		case DrainMigrationTimeout:
			failed++ // count timeout as failed in stats
		}
	}

	data := map[string]interface{}{
		"totalPods": total,
		"migrated":  migrated,
		"failed":    failed,
		"pending":   pending,
		"evicting":  evicting,
		"evicted":   evicted,
		"creating":  creating,
		"ignored":   ignored,
	}
	drm.eventManager.SendMessage(drainID, DrainStatsEvt, "stats", data)
}

// startPodWatcher 启动 Pod 监听
func (drm *DrainResourceManager) startPodWatcher(ctx context.Context, drainID, clusterName string) error {
	watchCtx, cancel := context.WithCancel(ctx)
	drm.watcherMutex.Lock()
	drm.podWatchers[drainID] = cancel
	drm.watcherMutex.Unlock()

	go drm.watchPods(watchCtx, drainID, clusterName)
	return nil
}

// getNamespacesToWatch 计算需要监听的命名空间及标签哈希
func (drm *DrainResourceManager) getNamespacesToWatch(drainID string) (map[string]bool, map[string]map[string]bool) {
	drm.trackerMutex.RLock()
	defer drm.trackerMutex.RUnlock()

	namespacesToWatch := make(map[string]bool)
	namespaceHashes := make(map[string]map[string]bool)

	for _, m := range drm.migrationTracker {
		if m.DrainID == drainID && len(m.SourcePod.OwnerReferences) > 0 {
			ns := m.SourcePod.Namespace
			namespacesToWatch[ns] = true
			if hash, ok := m.SourcePod.Labels["pod-template-hash"]; ok && hash != "" {
				if _, exists := namespaceHashes[ns]; !exists {
					namespaceHashes[ns] = make(map[string]bool)
				}
				namespaceHashes[ns][hash] = true
			}
		}
	}
	return namespacesToWatch, namespaceHashes
}

// watchPods 协调多命名空间的 Pod 监听
func (drm *DrainResourceManager) watchPods(ctx context.Context, drainID, clusterName string) {
	cs, err := drm.clientFactory.GetClient(clusterName)
	if err != nil {
		drm.eventManager.SendError(drainID, "Failed to get client for watcher", err)
		return
	}

	_, namespaceHashes := drm.getNamespacesToWatch(drainID)

	var wg sync.WaitGroup
	for ns, hashes := range namespaceHashes {
		wg.Add(1)
		go func(namespace string, podHashes map[string]bool) {
			defer wg.Done()
			drm.watchPodsInNamespace(ctx, cs, drainID, clusterName, namespace, podHashes)
		}(ns, hashes)
	}
	wg.Wait()
}

// watchPodsInNamespace 监听单命名空间中的 Pod 变更
func (drm *DrainResourceManager) watchPodsInNamespace(ctx context.Context, cs kubernetes.Interface, drainID, clusterName, namespace string, hashes map[string]bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			vals := make([]string, 0, len(hashes))
			for h := range hashes {
				vals = append(vals, h)
			}
			selector := fmt.Sprintf("pod-template-hash in (%s)", strings.Join(vals, ","))
			listOpts := metav1.ListOptions{LabelSelector: selector}

			watcher, err := cs.CoreV1().Pods(namespace).Watch(ctx, listOpts)
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}

			for event := range watcher.ResultChan() {
				if event.Type == watch.Added || event.Type == watch.Modified {
					if pod, ok := event.Object.(*corev1.Pod); ok {
						drm.checkForReplacement(ctx, drainID, clusterName, pod)
					}
				}
			}
			watcher.Stop()
		}
	}
}

// checkForReplacement 检查新 Pod 是否为替换者

func (drm *DrainResourceManager) checkForReplacement(ctx context.Context, drainID string, clusterName string, newPod *corev1.Pod) {
	// 收集候选迁移ID，避免在锁内进行K8s API调用
	candidateMigrations := make([]struct {
		migrationID string
		sourcePod   DrainPodInfo
		startTime   time.Time
		status      DrainPodMigrationStatus
	}, 0, len(drm.migrationTracker))

	drm.trackerMutex.Lock()
	for _, migration := range drm.migrationTracker {
		if migration.DrainID != drainID || migration.TargetPod != nil || migration.SourcePod.Namespace != newPod.Namespace {
			continue
		}

		ownerMatched := drm.isOwnerMatch(migration.SourcePod.OwnerReferences, newPod.OwnerReferences)
		labelRSMatch := drm.isSameReplicaSetByLabel(migration.SourcePod.Labels, newPod.Labels)
		if !(ownerMatched || labelRSMatch) {
			continue
		}

		candidateMigrations = append(candidateMigrations, struct {
			migrationID string
			sourcePod   DrainPodInfo
			startTime   time.Time
			status      DrainPodMigrationStatus
		}{
			migrationID: migration.MigrationID,
			sourcePod:   migration.SourcePod,
			startTime:   migration.StartTime,
			status:      migration.Status,
		})
	}
	drm.trackerMutex.Unlock()

	// 在锁外进行K8s API调用
	var validMigrations []string
	for _, candidate := range candidateMigrations {
		allowReplacement := candidate.status == DrainMigrationEvicted
		if !allowReplacement {
			if cs, err := drm.clientFactory.GetClient(clusterName); err == nil {
				if srcPod, gerr := cs.CoreV1().Pods(candidate.sourcePod.Namespace).Get(context.Background(), candidate.sourcePod.Name, metav1.GetOptions{}); gerr == nil {
					if srcPod.DeletionTimestamp != nil {
						allowReplacement = true
					}
				}
			}
		}
		if allowReplacement {
			validMigrations = append(validMigrations, candidate.migrationID)
		}
	}

	// 检查时间条件和新Pod分配状态
	createdAt := newPod.CreationTimestamp.Time
	finalMigrationIDs := make([]string, 0, len(validMigrations))

	drm.trackerMutex.Lock()
	// 检查新Pod是否已被分配
	isNewPodAlreadyAssigned := func(uid string) bool {
		for _, m := range drm.migrationTracker {
			if m.DrainID != drainID || m.TargetPod == nil {
				continue
			}
			if m.TargetPod.UID == uid {
				return true
			}
		}
		return false
	}

	for _, migrationID := range validMigrations {
		migration, exists := drm.migrationTracker[migrationID]
		if !exists {
			continue
		}

		if string(newPod.UID) == migration.SourcePod.UID {
			continue
		}

		lowerBound := migration.StartTime.Add(-2 * time.Second)
		if !createdAt.After(lowerBound) {
			continue
		}

		if isNewPodAlreadyAssigned(string(newPod.UID)) {
			continue
		}

		finalMigrationIDs = append(finalMigrationIDs, migrationID)
	}
	drm.trackerMutex.Unlock()

	// 处理最终的有效迁移
	for _, migrationID := range finalMigrationIDs {
		var migrationCopy *DrainPodMigrationInfo

		drm.trackerMutex.Lock()
		if migration, exists := drm.migrationTracker[migrationID]; exists {
			phaseReason := drm.deriveKubectlLikeStatus(newPod)
			migration.TargetPod = &DrainPodInfo{
				Name:            newPod.Name,
				Namespace:       newPod.Namespace,
				NodeName:        newPod.Spec.NodeName,
				Labels:          newPod.Labels,
				Annotations:     newPod.Annotations,
				UID:             string(newPod.UID),
				Phase:           string(newPod.Status.Phase),
				PhaseReason:     phaseReason,
				CreatedAt:       newPod.CreationTimestamp.Time,
				OwnerReferences: newPod.OwnerReferences,
			}
			drm.updatePodMigrationStatusInternal(migrationID, DrainMigrationCreating, "")

			// 创建拷贝用于后续处理
			migrationCopy = new(DrainPodMigrationInfo)
			*migrationCopy = *migration
		}
		drm.trackerMutex.Unlock()

		if migrationCopy != nil {
			// 异步保存信息
			go drm.saveMigrationInfo(migrationCopy)

			// 启动监控goroutine
			go func(parentCtx context.Context, mig *DrainPodMigrationInfo) {
				drm.monitorNewPod(parentCtx, clusterName, mig)
			}(ctx, migrationCopy)
		}
	}
}

// deriveKubectlLikeStatus 生成接近 kubectl 的 Pod 状态文案
// 重构后的函数，将复杂的逻辑拆分为多个专门的处理函数
// 优先级：删除状态 > Init容器状态 > 常规容器状态
//
// 参数:
//
//	pod: Kubernetes Pod对象
//
// 返回:
//
//	string: Pod状态的字符串表示
func (drm *DrainResourceManager) deriveKubectlLikeStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "Unknown"
	}

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	// Check pod conditions for a more specific reason, especially for pending pods.
	for _, c := range pod.Status.Conditions {
		// Example: PodScheduled=False, Reason=Unschedulable
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Reason != "" {
			reason = c.Reason
			// This is a definitive pending state, no need to check container statuses yet.
			if c.Reason == "Unschedulable" {
				return reason
			}
		}
	}

	initContainers := make(map[string]*corev1.Container, len(pod.Spec.InitContainers))
	for i := range pod.Spec.InitContainers {
		c := &pod.Spec.InitContainers[i]
		initContainers[c.Name] = c
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		status := pod.Status.InitContainerStatuses[i]
		containerSpec := initContainers[status.Name]

		switch {
		case status.State.Terminated != nil && status.State.Terminated.ExitCode == 0:
			continue
		case isRestartableInitContainer(containerSpec) && status.Started != nil && *status.Started:
			if status.Ready {
				continue
			}
			continue
		case status.State.Terminated != nil:
			if status.State.Terminated.Reason == "" {
				if status.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", status.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", status.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + status.State.Terminated.Reason
			}
			initializing = true
		case status.State.Waiting != nil && status.State.Waiting.Reason != "" && status.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + status.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		if initializing {
			break
		}
	}

	if !initializing || isPodInitializedConditionTrue(&pod.Status) {
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			status := pod.Status.ContainerStatuses[i]

			if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
				reason = status.State.Waiting.Reason
			} else if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
				reason = status.State.Terminated.Reason
			} else if status.State.Terminated != nil && status.State.Terminated.Reason == "" {
				if status.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", status.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", status.State.Terminated.ExitCode)
				}
			} else if status.Ready && status.State.Running != nil {
				hasRunning = true
			}
		}

		if reason == "Completed" && hasRunning {
			if hasPodReadyCondition(pod.Status.Conditions) {
				reason = "Running"
			} else {
				reason = "NotReady"
			}
		}
	}

	if pod.DeletionTimestamp != nil {
		if pod.Status.Reason == nodeUnreachablePodReason {
			reason = "Unknown"
		} else {
			reason = "Terminating"
		}
	}

	return reason
}

func isRestartableInitContainer(initContainer *corev1.Container) bool {
	if initContainer == nil || initContainer.RestartPolicy == nil {
		return false
	}
	return *initContainer.RestartPolicy == corev1.ContainerRestartPolicyAlways
}

func isPodInitializedConditionTrue(status *corev1.PodStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == corev1.PodInitialized {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// monitorNewPod 监控新 Pod 就绪状态
func (drm *DrainResourceManager) monitorNewPod(ctx context.Context, clusterName string, migration *DrainPodMigrationInfo) {
	if migration == nil || migration.TargetPod == nil {
		return
	}

	// Start a heartbeat goroutine to periodically send updates for stuck pods
	go func() {
		var lastLoggedStatus string

		// Give a grace period before starting to send heartbeats
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				drm.trackerMutex.RLock()
				m, exists := drm.migrationTracker[migration.MigrationID]
				if !exists || m.Status != DrainMigrationCreating {
					drm.trackerMutex.RUnlock()
					return
				}
				migrationCopy := *m
				drm.trackerMutex.RUnlock()

				// Send a periodic update with the current state for structured UI elements
				drm.eventManager.SendPodPhaseUpdated(migrationCopy.DrainID, &migrationCopy)

				// Also, send a human-readable log message to the event stream, but ONLY if the pod is in a problem state
				// and the state has changed since the last notification.
				if migrationCopy.TargetPod != nil {
					status := migrationCopy.TargetPod.Status
					isProblemState := strings.Contains(status, "BackOff") ||
						strings.Contains(status, "Failed") ||
						strings.Contains(status, "Error") ||
						strings.Contains(status, "Err") ||
						status == "Unschedulable"

					if isProblemState && status != lastLoggedStatus {
						logMessage := fmt.Sprintf(
							"等待位于 %s 命名空间下的新 Pod %s 进入就绪状态。当前状态: %s。",
							migrationCopy.TargetPod.Namespace,
							migrationCopy.TargetPod.Name,
							status,
						)
						drm.eventManager.SendMessage(
							migrationCopy.DrainID,
							"log",
							logMessage,
							map[string]interface{}{
								"level":   "WARN", // It's a problem state, so level should be WARN.
								"message": logMessage,
								"reason":  "heartbeat_problem_state",
							},
						)
						lastLoggedStatus = status // Remember the last logged status
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	if handled := drm.monitorNewPodWithWatch(ctx, clusterName, migration); handled {
		return
	}
	if ctx.Err() != nil {
		drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
		return
	}
	drm.monitorNewPodWithPolling(ctx, clusterName, migration)
}

func (drm *DrainResourceManager) monitorNewPodWithWatch(ctx context.Context, clusterName string, migration *DrainPodMigrationInfo) bool {
	cs, err := drm.clientFactory.GetClient(clusterName)
	if err != nil {
		return false
	}
	namespace := migration.TargetPod.Namespace
	podName := migration.TargetPod.Name
	if namespace == "" || podName == "" {
		drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationFailed, "target pod metadata incomplete")
		return true
	}

	selector := fields.OneTermEqualSelector("metadata.name", podName).String()
	resourceVersion := ""
	backoff := 500 * time.Millisecond

	for {
		if ctx.Err() != nil {
			drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
			return true
		}

		watcher, err := cs.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:   selector,
			ResourceVersion: resourceVersion,
		})
		if err != nil {
			return false
		}

		done, latestRV := drm.consumePodWatch(ctx, watcher, migration)
		watcher.Stop()
		if latestRV != "" {
			resourceVersion = latestRV
		}

		if done {
			if ctx.Err() != nil {
				drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
			}
			return true
		}

		select {
		case <-ctx.Done():
			drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
			return true
		case <-time.After(backoff):
		}
		if backoff < 5*time.Second {
			backoff *= 2
		}
	}
}

func (drm *DrainResourceManager) consumePodWatch(ctx context.Context, watcher watch.Interface, migration *DrainPodMigrationInfo) (bool, string) {
	latestRV := ""
	for {
		select {
		case <-ctx.Done():
			return true, latestRV
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return false, latestRV
			}
			pod, ok := event.Object.(*corev1.Pod)
			if ok && pod != nil && pod.ResourceVersion != "" {
				latestRV = pod.ResourceVersion
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				if pod == nil {
					continue
				}
				done, err := drm.processNewPodUpdate(migration, pod)
				if err != nil {
					drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationFailed, err.Error())
					return true, latestRV
				}
				if done {
					return true, latestRV
				}
			case watch.Deleted:
				drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationFailed, "new pod disappeared")
				return true, latestRV
			case watch.Error:
				return false, latestRV
			case watch.Bookmark:
				// ignore bookmark events
			}
		}
	}
}

func (drm *DrainResourceManager) monitorNewPodWithPolling(ctx context.Context, clusterName string, migration *DrainPodMigrationInfo) {
	reader, err := drm.getReader(clusterName)
	if err != nil {
		drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationFailed, "failed to get k8s reader")
		return
	}

	interval := 3 * time.Second
	for {
		if ctx.Err() != nil {
			drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
			return
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationTimeout, ctx.Err().Error())
			return
		case <-timer.C:
		}
		timer.Stop()

		isDone, err := drm.checkNewPodStatus(ctx, reader, migration)
		if err != nil {
			drm.UpdatePodMigrationStatus(migration.MigrationID, DrainMigrationFailed, err.Error())
			return
		}
		if isDone {
			return
		}

		if interval < 10*time.Second {
			interval = time.Duration(float64(interval) * 1.5)
			if interval > 10*time.Second {
				interval = 10 * time.Second
			}
		}
	}
}

func (drm *DrainResourceManager) processNewPodUpdate(migration *DrainPodMigrationInfo, pod *corev1.Pod) (bool, error) {
	if pod == nil {
		return false, nil
	}

	info := drm.podToDrainPodInfo(pod, false)
	phaseChanged := false
	var phaseCopy *DrainPodMigrationInfo
	var shouldSave bool
	var drainID string

	drm.trackerMutex.Lock()
	if mm, exists := drm.migrationTracker[migration.MigrationID]; exists {
		previous := mm.TargetPod
		mm.TargetPod = &info
		if previous == nil || previous.Phase != info.Phase || previous.PhaseReason != info.PhaseReason {
			phaseChanged = true
			migCopy := *mm
			phaseCopy = &migCopy
		}

		// Handle pod status changes directly to avoid reentrant lock
		if pod.Status.Phase == corev1.PodFailed {
			drm.updatePodMigrationStatusInternal(migration.MigrationID, DrainMigrationFailed, "new pod failed to start")
			shouldSave = true
		} else if isPodStablyReady(pod, 8*time.Second) { // 需要稳定就绪一段时间后再完成
			drm.updatePodMigrationStatusInternal(migration.MigrationID, DrainMigrationCompleted, "")
			shouldSave = true
		} else {
			drm.updatePodMigrationStatusInternal(migration.MigrationID, DrainMigrationCreating, "")
		}

		// Create copy for saving outside lock
		if shouldSave {
			migrationCopy := new(DrainPodMigrationInfo)
			*migrationCopy = *mm
			phaseCopy = migrationCopy
			drainID = mm.DrainID
		}
	}
	drm.trackerMutex.Unlock()

	migration.TargetPod = &info
	if phaseChanged && phaseCopy != nil {
		drm.eventManager.SendPodPhaseUpdated(migration.DrainID, phaseCopy)
	}

	// Save migration info outside lock
	if shouldSave && phaseCopy != nil {
		drm.saveMigrationInfo(phaseCopy)
		if drainID != "" {
			go drm.sendMigrationStats(drainID)
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		return true, fmt.Errorf("new pod failed to start")
	}

	if isPodStablyReady(pod, 8*time.Second) {
		return true, nil
	}

	return false, nil
}

// checkNewPodStatus 检查单个新 Pod 状态
func (drm *DrainResourceManager) checkNewPodStatus(ctx context.Context, reader crclient.Client, migration *DrainPodMigrationInfo) (bool, error) {
	var pod corev1.Pod
	if err := reader.Get(ctx, crclient.ObjectKey{Namespace: migration.TargetPod.Namespace, Name: migration.TargetPod.Name}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			return true, fmt.Errorf("new pod disappeared")
		}
		return false, nil
	}

	return drm.processNewPodUpdate(migration, &pod)
}

// pdbExists 判断 PDB 是否已记录
func (drm *DrainResourceManager) pdbExists(drainID, pdbKey string) bool {
	infos := drm.getPDBInfos(drainID)
	for _, info := range infos {
		if info.Key == pdbKey {
			return true
		}
	}
	return false
}

// savePDBInfo 保存 PDB 信息到 Redis
func (drm *DrainResourceManager) savePDBInfo(drainID, pdbKey, pdbName, namespace string) {
	key := fmt.Sprintf(drainPDBKeyFmt, drainID)
	list := drm.getPDBInfos(drainID)

	exists := false
	for _, it := range list {
		if it.Key == pdbKey {
			exists = true
			break
		}
	}
	if !exists {
		list = append(list, PDBInfo{
			Key:       pdbKey,
			Name:      pdbName,
			Namespace: namespace,
			CreatedAt: time.Now(),
		})
	}

	data, _ := json.Marshal(list)
	drm.redisHandler.Set(key, string(data))
}

// getPDBInfos 获取指定 Drain 的所有 PDB 信息
func (drm *DrainResourceManager) getPDBInfos(drainID string) []PDBInfo {
	val := drm.redisHandler.Get(fmt.Sprintf(drainPDBKeyFmt, drainID))
	if val == "" {
		return nil
	}

	var list []PDBInfo
	if json.Unmarshal([]byte(val), &list) == nil {
		return list
	}

	var single PDBInfo
	if json.Unmarshal([]byte(val), &single) == nil && single.Key != "" {
		return []PDBInfo{single}
	}
	return nil
}

// deletePDBInfo 删除指定 Drain 的 PDB 信息
func (drm *DrainResourceManager) deletePDBInfo(drainID string) {
	drm.redisHandler.Delete(fmt.Sprintf(drainPDBKeyFmt, drainID))
}

// saveSnapshot 保存快照到 Redis
func (drm *DrainResourceManager) saveSnapshot(drainID string, snapshot *DrainSnapshotInfo) {
	data, _ := json.Marshal(snapshot)
	drm.redisHandler.SetWithExpireTime(fmt.Sprintf(drainSnapshotKeyFmt, drainID), string(data), 24*time.Hour)
}

// getSnapshot 从 Redis 获取快照
func (drm *DrainResourceManager) getSnapshot(drainID string) *DrainSnapshotInfo {
	val := drm.redisHandler.Get(fmt.Sprintf(drainSnapshotKeyFmt, drainID))
	if val == "" {
		return nil
	}

	var snapshot DrainSnapshotInfo
	if json.Unmarshal([]byte(val), &snapshot) != nil {
		return nil
	}
	return &snapshot
}

// isDaemonSetOwner 判断是否由 DaemonSet 管理
func (drm *DrainResourceManager) isDaemonSetOwner(owners []metav1.OwnerReference) bool {
	for _, owner := range owners {
		if owner.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// isStatefulSetOwner 判断是否由 StatefulSet 管理
func (drm *DrainResourceManager) isStatefulSetOwner(owners []metav1.OwnerReference) bool {
	for _, owner := range owners {
		if owner.Kind == "StatefulSet" {
			return true
		}
	}
	return false
}

// isStaticPod 判断是否为静态Pod（由kubelet本地清单启动）
// 依据：
//   - Pod 没有 ownerReferences（无控制器管理的Pod）
//

// isPodReady 判断 Pod 是否就绪
func isPodReady(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	// 1) Phase 必须为 Running
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	// 2) 所有容器 Ready
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	// 3) Pod Ready 条件为 True
	readyCondOK := false
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			readyCondOK = true
			break
		}
	}
	if !readyCondOK {
		return false
	}
	// 4) 若存在 ReadinessGates，则必须全部满足（对应的 PodCondition 为 True）
	if len(pod.Spec.ReadinessGates) > 0 {
		// 构建条件查找表
		condMap := make(map[corev1.PodConditionType]corev1.ConditionStatus, len(pod.Status.Conditions))
		for _, cond := range pod.Status.Conditions {
			condMap[cond.Type] = cond.Status
		}
		for _, gate := range pod.Spec.ReadinessGates {
			if status, ok := condMap[gate.ConditionType]; !ok || status != corev1.ConditionTrue {
				return false
			}
		}
	}
	return true
}

// isPodStablyReady 判断 Pod 是否“稳定就绪”：
// 在满足 isPodReady 的前提下，要求 Ready 条件的最近一次转换已经超过 stableAfter 时长，
// 以规避短暂 Ready 随即 CrashLoop 等抖动带来的误判。
func isPodStablyReady(pod *corev1.Pod, stableAfter time.Duration) bool {
	if !isPodReady(pod) {
		return false
	}
	if stableAfter <= 0 {
		// 未指定稳定窗口则按即时就绪处理
		return true
	}
	var readyAt *time.Time
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			t := cond.LastTransitionTime.Time
			readyAt = &t
			break
		}
	}
	if readyAt == nil || readyAt.IsZero() {
		return false
	}
	return time.Since(*readyAt) >= stableAfter
}

// podToDrainPodInfo 转换 Pod 为 DrainPodInfo
func (drm *DrainResourceManager) podToDrainPodInfo(pod *corev1.Pod, withDetails bool) DrainPodInfo {
	if pod == nil {
		return DrainPodInfo{}
	}

	status := drm.deriveKubectlLikeStatus(pod)
	readyCount, totalCount := summarizeReadyContainers(pod.Status.ContainerStatuses)
	readySummary := fmt.Sprintf("%d/%d", readyCount, totalCount)
	restarts := calculateContainerRestarts(pod.Status.ContainerStatuses)
	age := formatPodAge(pod.CreationTimestamp.Time)

	info := DrainPodInfo{
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		NodeName:        pod.Spec.NodeName,
		UID:             string(pod.UID),
		Phase:           string(pod.Status.Phase),
		Status:          status,
		Ready:           readySummary,
		RestartCount:    restarts,
		Age:             age,
		CreatedAt:       pod.CreationTimestamp.Time,
		OwnerReferences: pod.OwnerReferences,
	}

	if withDetails {
		info.Labels = pod.Labels
		info.Annotations = pod.Annotations
	} else {
		// For SSE events, only include essential labels and strip large annotations.
		essentialLabels := make(map[string]string)
		if hash, ok := pod.Labels["pod-template-hash"]; ok {
			essentialLabels["pod-template-hash"] = hash
		}
		info.Labels = essentialLabels
		info.Annotations = nil
	}

	// PhaseReason 兼容旧字段，维持与 Status 一致
	info.PhaseReason = status
	return info
}

func summarizeReadyContainers(statuses []corev1.ContainerStatus) (ready, total int) {
	total = len(statuses)
	for _, cs := range statuses {
		if cs.Ready {
			ready++
		}
	}
	return
}

func calculateContainerRestarts(statuses []corev1.ContainerStatus) int32 {
	var restarts int32
	for _, cs := range statuses {
		restarts += cs.RestartCount
	}
	return restarts
}

func formatPodAge(createdAt time.Time) string {
	if createdAt.IsZero() {
		return "-"
	}
	delta := time.Since(createdAt)
	if delta < time.Minute {
		return fmt.Sprintf("%ds", int(delta.Seconds()))
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh", int(delta.Hours()))
	}
	return fmt.Sprintf("%dd", int(delta.Hours()/24))
}

// extractOwnerReference 基于标签推断拥有者标识
func (drm *DrainResourceManager) extractOwnerReference(pod *DrainPodInfo) string {
	if app, exists := pod.Labels["app"]; exists {
		return fmt.Sprintf("app:%s", app)
	}
	if deployment, exists := pod.Labels["app.kubernetes.io/name"]; exists {
		return fmt.Sprintf("deployment:%s", deployment)
	}
	return "unknown"
}

// generatePDBSelectorFromPods 基于 Pod 列表生成 PDB 选择器
func (drm *DrainResourceManager) generatePDBSelectorFromPods(pods []DrainPodInfo) map[string]string {
	if len(pods) > 0 {
		if hash, exists := pods[0].Labels["pod-template-hash"]; exists {
			return map[string]string{"pod-template-hash": hash}
		}
	}
	return map[string]string{}
}

// getReplicaSetInfoFromPods 从 Pod 列表提取 RS 信息
func (drm *DrainResourceManager) getReplicaSetInfoFromPods(ctx context.Context, reader crclient.Client, pods []DrainPodInfo) (rsName, rsUID string) {
	if len(pods) == 0 {
		return "", ""
	}

	pod := pods[0]
	if hash, exists := pod.Labels["pod-template-hash"]; exists {
		var rsList appsv1.ReplicaSetList
		if err := reader.List(ctx, &rsList, crclient.InNamespace(pod.Namespace), crclient.MatchingLabels{"pod-template-hash": hash}); err == nil && len(rsList.Items) > 0 {
			rs := rsList.Items[0]
			return rs.Name, string(rs.UID)
		}
	}
	return "", ""
}

// isOwnerMatch 判断两个 Pod 是否拥有相同 owner
func (drm *DrainResourceManager) isOwnerMatch(source, target []metav1.OwnerReference) bool {
	if len(source) == 0 || len(target) == 0 {
		return false
	}
	return source[0].UID == target[0].UID
}

// isSameReplicaSetByLabel 基于标签判断是否属于同一 RS
func (drm *DrainResourceManager) isSameReplicaSetByLabel(sourceLabels map[string]string, targetLabels map[string]string) bool {
	if sourceLabels == nil || targetLabels == nil {
		return false
	}
	src, ok1 := sourceLabels["pod-template-hash"]
	tgt, ok2 := targetLabels["pod-template-hash"]
	return ok1 && ok2 && src == tgt
}

// PDBInfo 表示已创建的 PDB 信息
type PDBInfo struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	CreatedAt time.Time `json:"created_at"`
}

// getDrainIDByMigration 根据迁移ID获取 DrainID
