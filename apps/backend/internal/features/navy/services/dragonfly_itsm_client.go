package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/dragonfly"
)

// DragonflyITSMClient implements ITSMClient interface using Dragonfly
type DragonflyITSMClient struct {
	logger  *zap.Logger
	enabled bool // Feature flag to enable/disable
	db      *db.Database
}

// NewDragonflyITSMClient creates a new Dragonfly ITSM client
func NewDragonflyITSMClient(logger *zap.Logger, enabled bool, database *db.Database) *DragonflyITSMClient {
	return &DragonflyITSMClient{
		logger:  logger,
		enabled: enabled,
		db:      database,
	}
}

// User context for change orders
type ChangeOrderUserContext struct {
	UMChecker  string // Verifier (审批人)
	UMOperator string // Operator (操作人)
	Applicant  string // Applicant (申请人)
}

// CreateTicket creates a Dragonfly change order
func (c *DragonflyITSMClient) CreateTicket(
	ctx context.Context,
	ticket *ChangeTicket,
) (string, error) {
	if !c.enabled {
		// Fallback to local ticket if disabled
		return c.generateLocalTicketNumber(ticket.ID), nil
	}

	// Extract user context from ticket metadata
	userCtx := c.extractUserContext(ticket)

	// Try to create Dragonfly change order with retry
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		chNumber, isLocal, err := c.createDragonflyOrder(ticket, userCtx)
		if err == nil {
			if isLocal {
				c.logger.Warn("Using local ticket after Dragonfly retries",
					zap.String("ticketID", ticket.ID),
					zap.String("localTicket", chNumber))
			}
			return chNumber, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	// All retries failed, generate local ticket
	localTicket := c.generateLocalTicketNumber(ticket.ID)
	c.logger.Error("Dragonfly change order creation failed after retries, using local ticket",
		zap.String("ticketID", ticket.ID),
		zap.String("localTicket", localTicket),
		zap.Error(lastErr))
	return localTicket, nil
}

// CloseTicket closes a Dragonfly change order
func (c *DragonflyITSMClient) CloseTicket(
	ctx context.Context,
	itsmTicketID string,
	success bool,
	message string,
) error {
	if !c.enabled {
		c.logger.Info("Dragonfly disabled, skipping change order closure",
			zap.String("itsmTicketID", itsmTicketID))
		return nil
	}

	// Check if local ticket
	if strings.HasPrefix(itsmTicketID, "LOCAL-") {
		c.logger.Info("Local ticket completed",
			zap.String("localTicket", itsmTicketID),
			zap.Bool("success", success),
			zap.String("message", message))
		return nil
	}

	// Determine execution type
	var execType dragonfly.ExecType
	if success {
		execType = dragonfly.CHExecTypeSuccess // 1
	} else {
		execType = dragonfly.CHExecTypeFailureBack // 2
	}

	// Get start time from message metadata (if available)
	startTime := time.Now() // Default to now if not available

	// Close Dragonfly change order
	req := dragonfly.CloseCHOrderReq{
		ChangeNumber:  itsmTicketID,
		ExecType:      fmt.Sprintf("%d", execType),
		ChangeUser:    "system",
		StartTime:     startTime.Format("2006-01-02 15:04:05"),
		VerifyContext: message,
	}

	rst, err := dragonfly.CloseCHOrder(req)
	if err != nil {
		c.logger.Error("Failed to close Dragonfly change order",
			zap.String("chNumber", itsmTicketID),
			zap.Error(err),
			zap.Int("code", rst.Code),
			zap.String("msg", rst.Message))
		return err
	}

	c.logger.Info("Closed Dragonfly change order",
		zap.String("chNumber", itsmTicketID),
		zap.String("execType", execType.GetDesc()),
		zap.Bool("success", success))
	return nil
}

// createDragonflyOrder creates a Dragonfly change order
func (c *DragonflyITSMClient) createDragonflyOrder(
	ticket *ChangeTicket,
	userCtx *ChangeOrderUserContext,
) (string, bool, error) {
	// Map operation type to Dragonfly ThirdCTI
	thirdCTI := c.mapOperationToThirdCTI(ticket.OperationType)

	// Extract cluster from metadata
	clusterName := c.extractClusterFromMetadata(ticket.Metadata)

	// Get operation-specific details
	opDetails := c.getOperationDetails(ticket.OperationType, ticket.Targets, clusterName)

	// Build change object list (cluster + nodes)
	chObjList := []dragonfly.ChangeObj{
		{
			ObjType:   dragonfly.CHObjTypeClusterID.GetCode(),
			ObjValues: []string{clusterName},
		},
		{
			ObjType:   dragonfly.CHObjTypeCICode.GetCode(),
			ObjValues: ticket.Targets, // Node names as CI codes
		},
	}

	env := dragonfly.GetEnv(clusterName)
	now := time.Now()
	planEndTime := now.Add(2 * time.Hour)

	chReq := dragonfly.CreateChangeReq{
		Title:     c.buildChangeTitle(ticket),
		Env:       env,
		SecondCTI: dragonfly.StandardChange.GetName(),
		ThirdCTI:  thirdCTI,

		// User information
		UMChecker:  c.defaultValue(userCtx.UMChecker, "system"),
		UMOperator: c.defaultValue(userCtx.UMOperator, "system"),
		Applicant:  c.defaultValue(userCtx.Applicant, "system"),

		// Timing
		ExpectTime:  now.Format("2006-01-02 15:04:05"),
		PlanEndTime: planEndTime.Format("2006-01-02 15:04:05"),

		// Change objects
		ChObjList: chObjList,

		// Risk assessment (低风险 - 操作人员知悉风险并确认)
		RiskAffect:         dragonfly.B1.GetCode(),           // 低风险影响
		RollbackDifficulty: dragonfly.Difficulty03.GetName(), // 容易回滚 (<15min)
		RiskCtrlPoint:      dragonfly.C5.GetCode(),           // 标准风险控制
		RiskAssessment:     opDetails.RiskAssessment,
		OperatePlan:        opDetails.OperatePlan,
		CheckMethod:        opDetails.CheckMethod,
		RollbackPlan:       opDetails.RollbackPlan,

		// Detail description
		Detail: c.buildChangeDetail(ticket),

		// SOP address - 从系统字典获取
		SOPAddress: opDetails.SOPAddress,
	}

	rst, err := dragonfly.CreateCHOrderDefaultValue(chReq)
	if err != nil {
		return "", false, err
	}

	c.logger.Info("Created Dragonfly change order",
		zap.String("ticketID", ticket.ID),
		zap.String("chNumber", rst.CHInfo.ChangeNumber),
		zap.String("operation", string(ticket.OperationType)),
		zap.Strings("targets", ticket.Targets))

	return rst.CHInfo.ChangeNumber, false, nil
}

// OperationDetails holds operation-specific SOP details
type OperationDetails struct {
	RiskAssessment string
	OperatePlan    string
	CheckMethod    string
	RollbackPlan   string
	SOPAddress     string
}

// getOperationDetails returns operation-specific details for Dragonfly
func (c *DragonflyITSMClient) getOperationDetails(opType ChangeOperationType, targets []string, clusterName string) OperationDetails {
	// Generate operation description
	targetsStr := strings.Join(targets, ", ")
	if len(targetsStr) > 50 {
		targetsStr = targetsStr[:50] + "..."
	}

	switch opType {
	case ChangeOpCordon:
		return OperationDetails{
			RiskAssessment: "风险点：节点标记为不可调度后，新建Pod无法调度到该节点。需确保集群有足够资源接收新Pod。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 通过kubectl或API执行: kubectl cordon %s\n2. 验证节点状态: kubectl get nodes %s -o wide\n3. 确认节点状态显示为SchedulingDisabled", targetsStr, targetsStr),
			CheckMethod:    "检查节点状态: kubectl get nodes -l kubernetes.io/hostname=<节点名> -o jsonpath='{.items[0].spec.unschedulable}'\n预期输出: true",
			RollbackPlan:   fmt.Sprintf("通过kubectl或API执行: kubectl uncordon %s", targetsStr),
			SOPAddress:     c.getSOPAddressFromDict("cordon_sop_address"),
		}

	case ChangeOpUncordon:
		return OperationDetails{
			RiskAssessment: "风险点：恢复节点可调度后，新Pod可能调度到该节点。需确保节点资源充足且状态正常。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 通过kubectl或API执行: kubectl uncordon %s\n2. 验证节点状态: kubectl get nodes %s -o wide\n3. 确认节点状态不再显示SchedulingDisabled", targetsStr, targetsStr),
			CheckMethod:    "检查节点状态: kubectl get nodes -l kubernetes.io/hostname=<节点名> -o jsonpath='{.items[0].spec.unschedulable}'\n预期输出: (空或false)",
			RollbackPlan:   fmt.Sprintf("如需再次不可调度: kubectl cordon %s", targetsStr),
			SOPAddress:     c.getSOPAddressFromDict("uncordon_sop_address"),
		}

	case ChangeOpTaint:
		return OperationDetails{
			RiskAssessment: "风险点：添加污点后，Pod若不容忍该污点将无法调度到该节点。需确保Pod配置了相应的容忍度。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 通过kubectl或API添加污点: kubectl taint nodes %s key=value:NoSchedule\n2. 验证污点: kubectl describe node %s | grep Taints", targetsStr, targetsStr),
			CheckMethod:    "检查节点污点: kubectl get nodes -l kubernetes.io/hostname=<节点名> -o jsonpath='{.items[0].spec.taints}'",
			RollbackPlan:   fmt.Sprintf("删除污点: kubectl taint nodes %s key:NoSchedule- (注意末尾的减号)", targetsStr),
			SOPAddress:     c.getSOPAddressFromDict("taint_sop_address"),
		}

	case ChangeOpLabel:
		return OperationDetails{
			RiskAssessment: "风险点：修改标签可能影响Pod调度策略、网络策略等。需评估对现有应用的影响。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 通过kubectl或API添加/删除标签: kubectl label nodes %s env=production --overwrite\n2. 验证标签: kubectl get nodes %s -L <label-key>", targetsStr, targetsStr),
			CheckMethod:    "检查节点标签: kubectl get nodes -l kubernetes.io/hostname=<节点名> -o jsonpath='{.items[0].metadata.labels}'",
			RollbackPlan:   "删除标签: kubectl label nodes <节点名> <label-key>- (注意末尾的减号) 或恢复原值: kubectl label nodes <节点名> <label-key>=<原值>",
			SOPAddress:     c.getSOPAddressFromDict("label_sop_address"),
		}

	case ChangeOpDrain:
		return OperationDetails{
			RiskAssessment: "风险点：Drain会驱逐节点上所有Pod，可能导致服务中断。需确保: 1) 集群有足够资源 2) Pod有反亲和性 3) PDB配置合理 4) 已执行cordon。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 前置条件: 节点已cordon (kubectl cordon %s)\n2. 执行drain: kubectl drain %s --ignore-daemonsets --delete-emptydir-data --timeout=5m\n3. 监控驱逐进度: kubectl get pods -A -o wide | grep %s\n4. 确认所有Pod已安全迁移", targetsStr, targetsStr, targetsStr),
			CheckMethod:    "1. 检查节点Pod数: kubectl get pods -A --field-selector spec.nodeName=<节点名> --no-headers | wc -l (应仅剩daemonset)\n2. 检查Pod状态: kubectl get pods -A -o wide | grep <节点名>",
			RollbackPlan:   fmt.Sprintf("1. 恢复调度: kubectl uncordon %s\n2. 对于未成功驱逐的Pod，手动清理并重试", targetsStr),
			SOPAddress:     c.getSOPAddressFromDict("drain_sop_address"),
		}

	case ChangeOpShutdown:
		return OperationDetails{
			RiskAssessment: "严重风险点：关机操作将导致节点完全不可用。影响: 1) 所有Pod停止运行 2) 数据可能丢失 3) 服务中断。必须: 1) 已执行drain 2) 确认业务允许停机 3) 有维护窗口。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 前置条件: 节点已完成drain (kubectl drain %s)\n2. 通过云平台API或BMC执行关机\n3. 验证关机: kubectl get nodes %s (状态变为NotReady)\n4. 确认Pod已迁移至其他节点", targetsStr, targetsStr),
			CheckMethod:    "1. 检查节点状态: kubectl get nodes <节点名> (状态应为NotReady)\n2. 检查节点Pod: kubectl get pods -A --field-selector spec.nodeName=<节点名> (应无结果或仅有daemonset)\n3. 检查BMC或云平台确认节点已关机",
			RollbackPlan:   "1. 通过BMC或云平台API开机\n2. 等待节点Ready: kubectl wait --for=condition=Ready node/<节点名> --timeout=5m\n3. 如需恢复服务，重新调度Pod",
			SOPAddress:     c.getSOPAddressFromDict("shutdown_sop_address"),
		}

	case ChangeOpReboot:
		return OperationDetails{
			RiskAssessment: "严重风险点：重启操作将导致节点临时不可用(通常5-15分钟)。影响: 1) Pod驱逐 2) 服务短暂中断 3) 网络中断。必须: 1) 已执行drain 2) 确认业务可接受 3) 有维护窗口。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    fmt.Sprintf("1. 前置条件: 节点已完成drain (kubectl drain %s)\n2. 通过云平台API或BMC执行重启\n3. 监控重启进度: kubectl get nodes %s -w\n4. 等待节点Ready: kubectl wait --for=condition=Ready node/%s --timeout=10m", targetsStr, targetsStr, targetsStr),
			CheckMethod:    "1. 检查节点状态: kubectl get nodes <节点名> (状态应为Ready)\n2. 检查节点年龄: kubectl get nodes <节点名> -o jsonpath='{.items[0].metadata.creationTimestamp}' (应显示重启后时间)\n3. 检查kubelet运行: kubectl get --raw /api/v1/nodes/<节点名>/proxy/healthz",
			RollbackPlan:   "1. 如重启失败，通过BMC或云平台控制台强制关机再开机\n2. 如节点异常，标记为不可调度并排查: kubectl cordon <节点名>\n3. 必要时从集群移除节点: kubectl delete node <节点名>",
			SOPAddress:     c.getSOPAddressFromDict("reboot_sop_address"),
		}

	default:
		return OperationDetails{
			RiskAssessment: "标准节点操作，请参考K8s官方文档。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
			OperatePlan:    "使用kubectl或K8s API执行相应操作",
			CheckMethod:    "通过kubectl get nodes/pods验证操作结果",
			RollbackPlan:   "根据具体操作执行相应的回滚命令",
			SOPAddress:     "http://wiki.pab.com.cn/pages/viewpage.action?pageId=460363820",
		}
	}
}

// getSOPAddressFromDict retrieves SOP address from system dictionary
// Returns first value from the dictionary, or fallback URL if not found
func (c *DragonflyITSMClient) getSOPAddressFromDict(dictKey string) string {
	if c.db == nil {
		c.logger.Warn("Database not available for dictionary lookup", zap.String("key", dictKey))
		return c.getFallbackSOPAddress(dictKey)
	}

	// Query dictionary by code
	var dict models.Dictionary
	if err := c.db.Where("code = ? AND is_enabled = true", dictKey).First(&dict).Error; err != nil {
		c.logger.Debug("Dictionary not found",
			zap.String("key", dictKey),
			zap.Error(err))
		return c.getFallbackSOPAddress(dictKey)
	}

	// Query first enabled dictionary item
	var item models.DictionaryItem
	if err := c.db.Where("dictionary_id = ? AND is_enabled = true", dict.ID).
		Order("sort_order ASC, id ASC").
		First(&item).Error; err != nil {
		c.logger.Debug("Dictionary item not found",
			zap.String("key", dictKey),
			zap.Uint64("dictionary_id", dict.ID),
			zap.Error(err))
		return c.getFallbackSOPAddress(dictKey)
	}

	c.logger.Debug("Retrieved SOP address from dictionary",
		zap.String("key", dictKey),
		zap.String("value", item.Value))
	return item.Value
}

// getFallbackSOPAddress returns fallback SOP address when dictionary lookup fails
func (c *DragonflyITSMClient) getFallbackSOPAddress(dictKey string) string {
	fallbackAddresses := map[string]string{
		"cordon_sop_address":   "http://wiki.pab.com.cn/k8s/cordon",
		"uncordon_sop_address": "http://wiki.pab.com.cn/k8s/uncordon",
		"taint_sop_address":    "http://wiki.pab.com.cn/k8s/taint",
		"label_sop_address":    "http://wiki.pab.com.cn/k8s/label",
		"drain_sop_address":    "http://wiki.pab.com.cn/k8s/drain",
		"shutdown_sop_address": "http://wiki.pab.com.cn/k8s/shutdown",
		"reboot_sop_address":   "http://wiki.pab.com.cn/k8s/reboot",
	}

	if addr, ok := fallbackAddresses[dictKey]; ok {
		return addr
	}

	return "http://wiki.pab.com.cn/pages/viewpage.action?pageId=460363820"
}

// mapOperationToThirdCTI maps ChangeOperationType to Dragonfly ThirdCTI
func (c *DragonflyITSMClient) mapOperationToThirdCTI(opType ChangeOperationType) string {
	switch opType {
	case ChangeOpCordon:
		return dragonfly.CordonNode.GetName()
	case ChangeOpUncordon:
		return dragonfly.UncordonNode.GetName()
	case ChangeOpTaint:
		return dragonfly.TaintNode.GetName()
	case ChangeOpLabel:
		return dragonfly.LabelNode.GetName()
	case ChangeOpShutdown:
		return dragonfly.ShutdownNode.GetName()
	case ChangeOpReboot:
		return dragonfly.RebootNode.GetName()
	case ChangeOpDrain:
		return dragonfly.DrainNode.GetName()
	default:
		return "节点操作"
	}
}

// buildChangeTitle builds the change order title
func (c *DragonflyITSMClient) buildChangeTitle(ticket *ChangeTicket) string {
	opName := string(ticket.OperationType)
	targets := strings.Join(ticket.Targets, ",")
	if len(targets) > 20 {
		targets = targets[:20] + "..."
	}
	return fmt.Sprintf("K8S节点%s - %s", opName, targets)
}

// buildChangeDetail builds the change order detail
func (c *DragonflyITSMClient) buildChangeDetail(ticket *ChangeTicket) string {
	clusterName := c.extractClusterFromMetadata(ticket.Metadata)
	return fmt.Sprintf(
		"集群 %s %s操作，节点: %s。ChangeTicket ID: %s",
		clusterName,
		ticket.OperationType,
		strings.Join(ticket.Targets, ", "),
		ticket.ID,
	)
}

// extractClusterFromMetadata extracts cluster name from ticket metadata
func (c *DragonflyITSMClient) extractClusterFromMetadata(metadata map[string]any) string {
	if cluster, ok := metadata["cluster"].(string); ok {
		return cluster
	}
	return "unknown-cluster"
}

// extractUserContext extracts user context from ticket metadata
func (c *DragonflyITSMClient) extractUserContext(ticket *ChangeTicket) *ChangeOrderUserContext {
	userCtx := &ChangeOrderUserContext{}

	if metadata, ok := ticket.Metadata["user_context"].(map[string]any); ok {
		if umChecker, ok := metadata["um_checker"].(string); ok {
			userCtx.UMChecker = umChecker
		}
		if umOperator, ok := metadata["um_operator"].(string); ok {
			userCtx.UMOperator = umOperator
		}
		if applicant, ok := metadata["applicant"].(string); ok {
			userCtx.Applicant = applicant
		}
	}

	return userCtx
}

// generateLocalTicketNumber generates a local fallback ticket number
func (c *DragonflyITSMClient) generateLocalTicketNumber(ticketID string) string {
	timestamp := time.Now().Format("20060102150405")
	shortID := strings.TrimPrefix(ticketID, "")
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("LOCAL-%s-%s", timestamp, shortID)
}

// defaultValue returns val if non-empty, otherwise returns defaultVal
func (c *DragonflyITSMClient) defaultValue(val, defaultVal string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return defaultVal
}
