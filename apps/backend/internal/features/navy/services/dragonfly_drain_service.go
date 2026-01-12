package services

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"robusta-web/backend/pkg/dragonfly"
)

// DragonflyDrainService handles Dragonfly integration for drain operations
type DragonflyDrainService struct {
	logger *zap.Logger
}

func NewDragonflyDrainService(logger *zap.Logger) *DragonflyDrainService {
	return &DragonflyDrainService{logger: logger}
}

// DragonflyUserContext Dragonfly user context extracted from JWT
type DragonflyUserContext struct {
	UMChecker  string // Verifier (审批人)
	UMOperator string // Operator (操作人)
	Applicant  string // Applicant (申请人)
}

// CreateDrainChangeOrder creates or validates a Dragonfly change order
// Returns: changeNumber, isLocalTicket, error
func (dds *DragonflyDrainService) CreateDrainChangeOrder(
	req *SimpleDrainRequest,
	clusterName string,
	userCtx *DragonflyUserContext,
	drainID string,
) (string, bool, error) {
	// If pre-existing CH number provided, validate it
	if req.CHNumber != "" {
		if err := dds.ValidateChangeOrder(req.CHNumber); err != nil {
			return "", false, fmt.Errorf("invalid pre-existing change number: %w", err)
		}
		return req.CHNumber, false, nil
	}

	// Try to create Dragonfly change order with retry
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		chNumber, err := dds.createDragonflyOrder(req, clusterName, userCtx, drainID)
		if err == nil {
			return chNumber, false, nil
		}

		lastErr = err

		dds.logger.Warn("Dragonfly change order creation failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.String("drainID", drainID),
			zap.Error(lastErr),
		)
		time.Sleep(time.Duration(attempt+1) * time.Second) // 1s, 2s, 3s backoff
	}

	// All retries failed, generate local ticket number
	localTicket := dds.generateLocalTicketNumber(drainID)
	dds.logger.Error("Dragonfly change order creation failed after retries, using local fallback",
		zap.String("drainID", drainID),
		zap.String("localTicket", localTicket),
		zap.Error(lastErr),
	)
	return localTicket, true, nil
}

// createDragonflyOrder attempts to create a Dragonfly change order
func (dds *DragonflyDrainService) createDragonflyOrder(
	req *SimpleDrainRequest,
	clusterName string,
	userCtx *DragonflyUserContext,
	drainID string,
) (string, error) {
	// Build Dragonfly request
	env := dragonfly.GetEnv(clusterName)
	now := time.Now()
	planEndTime := now.Add(2 * time.Hour)

	// Build change object list (cluster + node CI codes)
	chObjList := []dragonfly.ChangeObj{
		{
			ObjType:   dragonfly.CHObjTypeClusterID.GetCode(),
			ObjValues: []string{clusterName},
		},
		{
			ObjType:   dragonfly.CHObjTypeCICode.GetCode(),
			ObjValues: []string{req.NodeName}, // User requirement: use nodeName as CI Code
		},
	}

	chReq := dragonfly.CreateChangeReq{
		Title:     fmt.Sprintf("K8S节点Drain - %s", req.NodeName),
		Env:       env,
		SecondCTI: dragonfly.StandardChange.GetName(),
		ThirdCTI:  dragonfly.DrainNode.GetName(),

		// User information from authentication context
		UMChecker:  dds.defaultValue(userCtx.UMChecker, "system"),
		UMOperator: dds.defaultValue(userCtx.UMOperator, "system"),
		Applicant:  dds.defaultValue(userCtx.Applicant, "system"),

		// Timing
		ExpectTime:  now.Format("2006-01-02 15:04:05"),
		PlanEndTime: planEndTime.Format("2006-01-02 15:04:05"),

		// Change objects
		ChObjList: chObjList,

		// Risk assessment (低风险 - 操作人员知悉风险并确认)
		RiskAffect:         dragonfly.B1.GetCode(),           // 低风险影响
		RollbackDifficulty: dragonfly.Difficulty03.GetName(), // 容易回滚 (<15min)
		RiskCtrlPoint:      dragonfly.C5.GetCode(),           // 标准风险控制
		RiskAssessment:     "风险点：Drain会驱逐节点上所有Pod，可能导致服务中断。需确保: 1) 集群有足够资源 2) Pod有反亲和性 3) PDB配置合理 4) 已执行cordon。\n风险评级：低风险 - 操作人员知悉相关风险，确认操作正常。",
		OperatePlan:        fmt.Sprintf("1. 前置条件: 节点已cordon (kubectl cordon %s)\n2. 执行drain: kubectl drain %s --ignore-daemonsets --delete-emptydir-data --timeout=5m\n3. 监控驱逐进度: kubectl get pods -A -o wide | grep %s\n4. 确认所有Pod已安全迁移", req.NodeName, req.NodeName, req.NodeName),
		CheckMethod:        "1. 检查节点Pod数: kubectl get pods -A --field-selector spec.nodeName=<节点名> --no-headers | wc -l (应仅剩daemonset)\n2. 检查Pod状态: kubectl get pods -A -o wide | grep <节点名>",
		RollbackPlan:       fmt.Sprintf("1. 恢复调度: kubectl uncordon %s\n2. 对于未成功驱逐的Pod，手动清理并重试", req.NodeName),

		// Detail description
		Detail: fmt.Sprintf(
			"集群 %s Drain节点 %s (Drain ID: %s, DryRun: %v)",
			clusterName, req.NodeName, drainID, req.DryRun,
		),

		// SOP address - 从系统字典获取
		SOPAddress: dds.getSOPAddressFromDict("drain_sop_address"),
	}

	// Call Dragonfly
	rst, err := dragonfly.CreateCHOrderDefaultValue(chReq)
	if err != nil {
		return "", fmt.Errorf("Dragonfly creation failed: %w", err)
	}

	dds.logger.Info("Created Dragonfly change order",
		zap.String("drainID", drainID),
		zap.String("chNumber", rst.CHInfo.ChangeNumber),
		zap.Bool("needApprove", rst.CHInfo.IsNeedApprove),
	)

	return rst.CHInfo.ChangeNumber, nil
}

// getSOPAddressFromDict retrieves SOP address from system dictionary
func (dds *DragonflyDrainService) getSOPAddressFromDict(dictKey string) string {
	// TODO: 从Redis或配置中心获取字典值
	// dictValues := redis.Get(fmt.Sprintf("dict:%s", dictKey))
	// if dictValues != nil && len(dictValues) > 0 {
	//     return dictValues[0]
	// }

	// 临时fallback
	fallbackAddresses := map[string]string{
		"drain_sop_address": "http://wiki.pab.com.cn/k8s/drain",
	}

	if addr, ok := fallbackAddresses[dictKey]; ok {
		return addr
	}

	return "http://wiki.pab.com.cn/pages/viewpage.action?pageId=460363820"
}

// ValidateChangeOrder validates an existing change order
func (dds *DragonflyDrainService) ValidateChangeOrder(chNumber string) error {
	detail, err := dragonfly.GetCHOrderDetail(chNumber)
	if err != nil {
		return fmt.Errorf("failed to query change order: %w", err)
	}

	// Check if status is "doing change" (in progress)
	if detail.ChangeStatus != dragonfly.CHDoingChang {
		return fmt.Errorf("change order status is %s, expected %s",
			detail.ChangeStatus.GetCode(),
			dragonfly.CHDoingChang.GetCode())
	}

	return nil
}

// CloseDrainChangeOrder closes a Dragonfly change order or records local ticket completion
func (dds *DragonflyDrainService) CloseDrainChangeOrder(
	chNumber string,
	drainID string,
	execType dragonfly.ExecType,
	isLocalTicket bool,
	success bool,
	errorMsg string,
	startTime time.Time,
) error {
	if chNumber == "" {
		return nil // No change order to close
	}

	// If local ticket, just log the completion
	if isLocalTicket {
		dds.logger.Info("Local drain ticket completed",
			zap.String("localTicket", chNumber),
			zap.String("drainID", drainID),
			zap.String("execType", execType.GetDesc()),
			zap.Bool("success", success),
			zap.String("duration", time.Since(startTime).String()),
			zap.String("errorMsg", errorMsg),
		)
		return nil
	}

	// Close Dragonfly change order
	req := dragonfly.CloseCHOrderReq{
		ChangeNumber:  chNumber,
		ExecType:      fmt.Sprintf("%d", execType),
		ChangeUser:    "system", // TODO: Get from context
		StartTime:     startTime.Format("2006-01-02 15:04:05"),
		VerifyContext: errorMsg,
	}

	rst, err := dragonfly.CloseCHOrder(req)
	if err != nil {
		dds.logger.Error("Failed to close Dragonfly change order",
			zap.String("chNumber", chNumber),
			zap.String("drainID", drainID),
			zap.String("execType", execType.GetDesc()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to close change order: %w (code: %d, msg: %s)",
			err, rst.Code, rst.Message)
	}

	dds.logger.Info("Closed Dragonfly change order",
		zap.String("chNumber", chNumber),
		zap.String("drainID", drainID),
		zap.String("execType", execType.GetDesc()),
		zap.Bool("success", success),
	)

	return nil
}

// GetExecType determines the appropriate execution type based on drain outcome
func (dds *DragonflyDrainService) GetExecType(
	drainSuccess bool,
	isCanceled bool,
	failedPods int,
) dragonfly.ExecType {
	switch {
	case isCanceled:
		return dragonfly.CHExecTypeCancel // 4
	case drainSuccess:
		return dragonfly.CHExecTypeSuccess // 1
	case failedPods > 0:
		return dragonfly.CHExecTypePartialSuccess // 6
	default:
		return dragonfly.CHExecTypeFailureBack // 2
	}
}

// generateLocalTicketNumber generates a local fallback ticket number
func (dds *DragonflyDrainService) generateLocalTicketNumber(drainID string) string {
	timestamp := time.Now().Format("20060102150405")
	shortID := strings.TrimPrefix(drainID, "drain-")
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("LOCAL-%s-%s", timestamp, shortID)
}

// defaultValue returns val if non-empty, otherwise returns defaultVal
func (dds *DragonflyDrainService) defaultValue(val, defaultVal string) string {
	if strings.TrimSpace(val) != "" {
		return val
	}
	return defaultVal
}
