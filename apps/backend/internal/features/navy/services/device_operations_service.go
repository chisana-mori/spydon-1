package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"
	"robusta-web/backend/internal/models/navy"
	"robusta-web/backend/internal/pkg/nodesync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeviceOperationsService 设备批量操作服务
type DeviceOperationsService struct {
	navyDB           *db.Database
	mainDB           *db.Database
	nodeSyncManager  *nodesync.Manager
	awxRuntime       *pipelineservice.AWXRuntime
	cfg              *config.Config
	logger           *zap.Logger
	safeDrainService *SimpleDrainService
}

// NewDeviceOperationsService 创建设备操作服务
func NewDeviceOperationsService(
	navyDB *db.Database,
	mainDB *db.Database,
	nodeSyncManager *nodesync.Manager,
	awxRuntime *pipelineservice.AWXRuntime,
	cfg *config.Config,
	logger *zap.Logger,
	safeDrainService *SimpleDrainService,
) *DeviceOperationsService {
	return &DeviceOperationsService{
		navyDB:           navyDB,
		mainDB:           mainDB,
		nodeSyncManager:  nodeSyncManager,
		awxRuntime:       awxRuntime,
		cfg:              cfg,
		logger:           logger,
		safeDrainService: safeDrainService,
	}
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Results   []NodeOperationResult `json:"results"`
}

// NodeOperationResult 单个节点操作结果
type NodeOperationResult struct {
	CICode  string `json:"ci_code"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	DrainID string `json:"drain_id,omitempty"`
}

// DrainOptions Drain 操作选项
type DrainOptions struct {
	Force            bool `json:"force"`
	IgnoreDaemonsets bool `json:"ignore_daemonsets"`
	DeleteLocalData  bool `json:"delete_local_data"`
	Timeout          int  `json:"timeout"` // 秒，默认 300
}

// TaintOperation Taint 操作
type TaintOperation struct {
	Key    string `json:"key" binding:"required"`
	Value  string `json:"value"`
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"`
	Action string `json:"action" binding:"required,oneof=add remove"`
}

// LabelOperation Label 操作
type LabelOperation struct {
	Labels map[string]string `json:"labels" binding:"required"`
	Action string            `json:"action" binding:"required,oneof=add remove"`
}

// CordonNodes 设置节点为不可调度
func (s *DeviceOperationsService) CordonNodes(ctx context.Context, ciCodes []string) (*BatchOperationResult, error) {
	return s.executeNodeOperation(ctx, ciCodes, "cordon", func(k8sClient client.Client, nodeName string) error {
		var node corev1.Node
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
			return fmt.Errorf("获取节点失败: %w", err)
		}

		// Prepare the taint
		now := metav1.Now()
		unschedulableTaint := corev1.Taint{
			Key:       "node.kubernetes.io/unschedulable",
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: &now,
		}

		// Check if already cordoned or has taint
		alreadyUnschedulable := node.Spec.Unschedulable
		hasTaint := false
		for _, t := range node.Spec.Taints {
			if t.Key == unschedulableTaint.Key && t.Effect == unschedulableTaint.Effect {
				hasTaint = true
				break
			}
		}

		if alreadyUnschedulable && hasTaint {
			return nil // Already fully cordoned
		}

		node.Spec.Unschedulable = true
		if !hasTaint {
			node.Spec.Taints = append(node.Spec.Taints, unschedulableTaint)
		}

		if err := k8sClient.Update(ctx, &node); err != nil {
			return fmt.Errorf("更新节点失败: %w", err)
		}
		return nil
	})
}

// UncordonNodes 设置节点为可调度
func (s *DeviceOperationsService) UncordonNodes(ctx context.Context, ciCodes []string) (*BatchOperationResult, error) {
	return s.executeNodeOperation(ctx, ciCodes, "uncordon", func(k8sClient client.Client, nodeName string) error {
		var node corev1.Node
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
			return fmt.Errorf("获取节点失败: %w", err)
		}

		if !node.Spec.Unschedulable {
			return nil // 已经是 uncordon 状态
		}

		node.Spec.Unschedulable = false
		if err := k8sClient.Update(ctx, &node); err != nil {
			return fmt.Errorf("更新节点失败: %w", err)
		}
		return nil
	})
}

// DrainNodes 驱逐节点上的 Pod (使用 SafeDrainService)
func (s *DeviceOperationsService) DrainNodes(ctx context.Context, ciCodes []string, opts DrainOptions) (*BatchOperationResult, error) {
	// 获取设备信息（需要集群归属）
	devices, err := s.getDevicesByCICodes(ctx, ciCodes)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	result := &BatchOperationResult{
		Total:   len(ciCodes),
		Results: make([]NodeOperationResult, 0, len(ciCodes)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// 处理设备
	for _, d := range devices {
		if d.Cluster == "" {
			mu.Lock()
			result.Results = append(result.Results, NodeOperationResult{
				CICode:  d.CICode,
				Success: false,
				Error:   "设备未关联到任何集群",
			})
			result.Failed++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(device navy.Device) {
			defer wg.Done()

			// 调用 SimpleDrainService
			// K8s 集群中 nodename 都是小写的，需要将 CICode 转换为小写
			nodeName := strings.ToLower(device.CICode)
			resp, err := s.safeDrainService.StartDrain(ctx, &SimpleDrainRequest{
				ClusterName: device.Cluster,
				NodeName:    nodeName,
				Force:       opts.Force,
				DryRun:      false,
			})

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Results = append(result.Results, NodeOperationResult{
					CICode:  device.CICode,
					Success: false,
					Error:   err.Error(),
				})
				result.Failed++
			} else {
				result.Results = append(result.Results, NodeOperationResult{
					CICode:  device.CICode,
					Success: true,
					Message: fmt.Sprintf("Safe Drain 已启动 (ID: %s)", resp.DrainID),
					DrainID: resp.DrainID,
				})
				result.Succeeded++
			}
		}(d)
	}

	// 检查是否有未找到的 CI Code
	for _, ciCode := range ciCodes {
		found := false
		for _, d := range devices {
			if d.CICode == ciCode {
				found = true
				break
			}
		}
		if !found {
			mu.Lock()
			result.Results = append(result.Results, NodeOperationResult{
				CICode:  ciCode,
				Success: false,
				Error:   "设备不存在",
			})
			result.Failed++
			mu.Unlock()
		}
	}

	wg.Wait()

	s.logger.Info("批量 Safe Drain 操作请求完成",
		zap.Int("total", result.Total),
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed))

	return result, nil
}

// TaintNodes 添加或移除节点 Taint
func (s *DeviceOperationsService) TaintNodes(ctx context.Context, ciCodes []string, op TaintOperation) (*BatchOperationResult, error) {
	return s.executeNodeOperation(ctx, ciCodes, "taint-"+op.Action, func(k8sClient client.Client, nodeName string) error {
		var node corev1.Node
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
			return fmt.Errorf("获取节点失败: %w", err)
		}

		taintEffect := corev1.TaintEffect(op.Effect)
		newTaint := corev1.Taint{
			Key:    op.Key,
			Value:  op.Value,
			Effect: taintEffect,
		}

		if op.Action == "add" {
			// 检查是否已存在相同的 taint
			for i := range node.Spec.Taints {
				if node.Spec.Taints[i].Key == op.Key && node.Spec.Taints[i].Effect == taintEffect {
					// 更新 value
					node.Spec.Taints[i].Value = op.Value
					if err := k8sClient.Update(ctx, &node); err != nil {
						return fmt.Errorf("更新节点失败: %w", err)
					}
					return nil
				}
			}
			node.Spec.Taints = append(node.Spec.Taints, newTaint)
		} else {
			// 移除 taint
			newTaints := make([]corev1.Taint, 0, len(node.Spec.Taints))
			for _, t := range node.Spec.Taints {
				if t.Key == op.Key && t.Effect == taintEffect {
					continue
				}
				newTaints = append(newTaints, t)
			}
			node.Spec.Taints = newTaints
		}

		if err := k8sClient.Update(ctx, &node); err != nil {
			return fmt.Errorf("更新节点失败: %w", err)
		}
		return nil
	})
}

// LabelNodes 添加或移除节点 Label
func (s *DeviceOperationsService) LabelNodes(ctx context.Context, ciCodes []string, op LabelOperation) (*BatchOperationResult, error) {
	return s.executeNodeOperation(ctx, ciCodes, "label-"+op.Action, func(k8sClient client.Client, nodeName string) error {
		var node corev1.Node
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: nodeName}, &node); err != nil {
			return fmt.Errorf("获取节点失败: %w", err)
		}

		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}

		for key, value := range op.Labels {
			if op.Action == "add" {
				node.Labels[key] = value
			} else {
				delete(node.Labels, key)
			}
		}

		if err := k8sClient.Update(ctx, &node); err != nil {
			return fmt.Errorf("更新节点失败: %w", err)
		}
		return nil
	})
}

// ShutdownNodes 关机节点 (通过 AWX)
func (s *DeviceOperationsService) ShutdownNodes(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
	return s.executePowerOperation(ctx, ciCodes, "shutdown")
}

// RebootNodes 重启节点 (通过 AWX)
func (s *DeviceOperationsService) RebootNodes(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
	return s.executePowerOperation(ctx, ciCodes, "reboot")
}

// executePowerOperation 执行电源操作
func (s *DeviceOperationsService) executePowerOperation(ctx context.Context, ciCodes []string, operation string) (*pipelineservice.JobHandle, error) {
	if s.awxRuntime == nil {
		return nil, fmt.Errorf("AWX runtime 未配置")
	}

	// 获取设备信息（需要 IP 列表）
	devices, err := s.getDevicesByCICodes(ctx, ciCodes)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	// 构建 limit 字符串（IP 列表）
	ips := make([]string, 0, len(devices))
	for _, d := range devices {
		if d.IP != "" {
			ips = append(ips, d.IP)
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("没有有效的设备 IP")
	}

	// 查找对应的 AWX 模板
	templateID := s.cfg.AWX.ShutdownTemplateID
	if operation == "reboot" {
		templateID = s.cfg.AWX.RebootTemplateID
	}

	if templateID == 0 {
		return nil, fmt.Errorf("AWX %s 模板未配置", operation)
	}

	// 准备 ExtraVars
	extraVars := map[string]any{
		"target_hosts": ips,
		"operation":    operation,
	}

	// 启动 AWX Job
	jobHandle, err := s.awxRuntime.LaunchJob(ctx, pipelineservice.JobConfig{
		TemplateID: templateID,
		ExtraVars:  extraVars,
	})
	if err != nil {
		return nil, fmt.Errorf("启动 AWX Job 失败: %w", err)
	}

	s.logger.Info("电源操作 AWX Job 已启动",
		zap.String("operation", operation),
		zap.Int("job_id", jobHandle.JobID),
		zap.Strings("ci_codes", ciCodes))

	return jobHandle, nil
}

// executeNodeOperation 执行节点操作的通用方法
func (s *DeviceOperationsService) executeNodeOperation(
	ctx context.Context,
	ciCodes []string,
	operation string,
	fn func(client.Client, string) error,
) (*BatchOperationResult, error) {
	// 获取设备信息（需要集群归属）
	devices, err := s.getDevicesByCICodes(ctx, ciCodes)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	// 按集群分组
	clusterDevices := make(map[string][]navy.Device)
	for _, d := range devices {
		if d.Cluster != "" {
			clusterDevices[d.Cluster] = append(clusterDevices[d.Cluster], d)
		}
	}

	result := &BatchOperationResult{
		Total:   len(ciCodes),
		Results: make([]NodeOperationResult, 0, len(ciCodes)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发处理每个集群的节点
	for clusterName, clusterDeviceList := range clusterDevices {
		wg.Add(1)
		go func(cluster string, deviceList []navy.Device) {
			defer wg.Done()

			// 获取集群的 K8s 客户端
			k8sClient, err := s.getK8sClient(cluster)
			if err != nil {
				mu.Lock()
				for _, d := range deviceList {
					result.Results = append(result.Results, NodeOperationResult{
						CICode:  d.CICode,
						Success: false,
						Error:   fmt.Sprintf("获取集群客户端失败: %v", err),
					})
					result.Failed++
				}
				mu.Unlock()
				return
			}

			// 执行每个节点的操作
			for _, d := range deviceList {
				// K8s 集群中 nodename 都是小写的，需要将 CICode 转换为小写
				nodeName := strings.ToLower(d.CICode)
				opErr := fn(k8sClient, nodeName) // 使用小写的 ci_code 作为节点名
				mu.Lock()
				if opErr != nil {
					result.Results = append(result.Results, NodeOperationResult{
						CICode:  d.CICode,
						Success: false,
						Error:   opErr.Error(),
					})
					result.Failed++
				} else {
					result.Results = append(result.Results, NodeOperationResult{
						CICode:  d.CICode,
						Success: true,
						Message: fmt.Sprintf("%s 成功", operation),
					})
					result.Succeeded++
				}
				mu.Unlock()
			}
		}(clusterName, clusterDeviceList)
	}

	// 处理没有集群归属的设备
	for _, ciCode := range ciCodes {
		found := false
		for _, d := range devices {
			if d.CICode == ciCode && d.Cluster != "" {
				found = true
				break
			}
		}
		if !found {
			mu.Lock()
			result.Results = append(result.Results, NodeOperationResult{
				CICode:  ciCode,
				Success: false,
				Error:   "设备未关联到任何集群",
			})
			result.Failed++
			mu.Unlock()
		}
	}

	wg.Wait()

	s.logger.Info("批量节点操作完成",
		zap.String("operation", operation),
		zap.Int("total", result.Total),
		zap.Int("succeeded", result.Succeeded),
		zap.Int("failed", result.Failed))

	return result, nil
}

// getDevicesByCICodes 根据 CI Code 批量获取设备信息
func (s *DeviceOperationsService) getDevicesByCICodes(ctx context.Context, ciCodes []string) ([]navy.Device, error) {
	var devices []navy.Device
	if err := s.navyDB.Where("ci_code IN ?", ciCodes).Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

// getK8sClient 获取集群的 K8s 客户端 (使用 Nodesync Manager)
func (s *DeviceOperationsService) getK8sClient(clusterName string) (client.Client, error) {
	clientIface, err := s.nodeSyncManager.GetClient(clusterName)
	if err != nil {
		return nil, fmt.Errorf("集群 %s 不存在或不可用: %w", clusterName, err)
	}

	k8sClient, ok := clientIface.(client.Client)
	if !ok {
		return nil, fmt.Errorf("节点同步管理器返回的客户端类型错误")
	}

	return k8sClient, nil
}
