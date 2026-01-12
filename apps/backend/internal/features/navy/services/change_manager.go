package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"robusta-web/backend/internal/db"
	sharedservices "robusta-web/backend/internal/features/shared/services"
)

// ChangeOperationType 变更操作类型
type ChangeOperationType string

const (
	ChangeOpCordon   ChangeOperationType = "cordon"
	ChangeOpUncordon ChangeOperationType = "uncordon"
	ChangeOpDrain    ChangeOperationType = "drain"
	ChangeOpTaint    ChangeOperationType = "taint"
	ChangeOpLabel    ChangeOperationType = "label"
	ChangeOpShutdown ChangeOperationType = "shutdown"
	ChangeOpReboot   ChangeOperationType = "reboot"
)

// ChangeTicketStatus 变更单状态
type ChangeTicketStatus string

const (
	TicketStatusPending    ChangeTicketStatus = "pending"
	TicketStatusInProgress ChangeTicketStatus = "in_progress"
	TicketStatusCompleted  ChangeTicketStatus = "completed"
)

// OperationValidator 定义验证函数签名
// 验证函数在创建变更单之前执行，用于验证参数的合法性
type OperationValidator func(ctx context.Context, targets []string, metadata map[string]any) error

// WithChangeOption 定义可选项
type WithChangeOption func(*changeOptions)

// changeOptions 存储可选配置
type changeOptions struct {
	validator OperationValidator
}

// WithValidator 提供验证函数的可选项
func WithValidator(v OperationValidator) WithChangeOption {
	return func(opts *changeOptions) {
		opts.validator = v
	}
}

// ChangeTicket 变更单
type ChangeTicket struct {
	ID            string              `json:"id"`             // 内部 UUID
	ITSMTicketID  string              `json:"itsm_ticket_id"` // ITSM 系统工单号
	OperationType ChangeOperationType `json:"operation_type"` // 操作类型
	Targets       []string            `json:"targets"`        // CI Codes
	Status        ChangeTicketStatus  `json:"status"`         // 状态
	AWXJobID      int                 `json:"awx_job_id"`     // 关联 AWX Job (可选)
	DrainIDs      []string            `json:"drain_ids"`      // 关联 Drain ID (可选)
	Metadata      map[string]any      `json:"metadata"`       // 额外元数据
	CreatedAt     time.Time           `json:"created_at"`
	CompletedAt   *time.Time          `json:"completed_at"`
	Error         string              `json:"error"`
}

// Redis key patterns
const (
	changeTicketKeyFmt    = "change:ticket:%s"
	changePendingIndexKey = "change:pending"
	changeTicketTTL       = 7 * 24 * time.Hour // 7 天过期
)

// ITSMClient ITSM 系统客户端接口 (预留)
type ITSMClient interface {
	CreateTicket(ctx context.Context, ticket *ChangeTicket) (itsmTicketID string, err error)
	CloseTicket(ctx context.Context, itsmTicketID string, success bool, message string) error
}

// NoopITSMClient 空实现，用于禁用 ITSM 集成时
type NoopITSMClient struct{}

func (c *NoopITSMClient) CreateTicket(ctx context.Context, ticket *ChangeTicket) (string, error) {
	return fmt.Sprintf("NOOP-%s", ticket.ID[:8]), nil
}

func (c *NoopITSMClient) CloseTicket(ctx context.Context, itsmTicketID string, success bool, message string) error {
	return nil
}

// ChangeManagerConfig 配置
type ChangeManagerConfig struct {
	Enabled          bool          // 是否启用变更管理
	DragonflyEnabled bool          // 是否启用Dragonfly集成
	Timeout          time.Duration // 超时时间 (默认 30 分钟)
}

// DefaultChangeManagerConfig 默认配置
func DefaultChangeManagerConfig() ChangeManagerConfig {
	return ChangeManagerConfig{
		Enabled:          false,
		DragonflyEnabled: false,
		Timeout:          30 * time.Minute,
	}
}

// ChangeManager 变更管理器
type ChangeManager struct {
	config     ChangeManagerConfig
	redis      sharedservices.RedisClient
	itsmClient ITSMClient
	logger     *zap.Logger
	database   *db.Database
}

// NewChangeManager 创建变更管理器
func NewChangeManager(
	config ChangeManagerConfig,
	redis sharedservices.RedisClient,
	itsmClient ITSMClient,
	logger *zap.Logger,
	database *db.Database,
) *ChangeManager {
	// 如果提供了外部ITSMClient，使用它
	if itsmClient != nil {
		return &ChangeManager{
			config:     config,
			redis:      redis,
			itsmClient: itsmClient,
			logger:     logger,
			database:   database,
		}
	}

	// 根据配置决定使用哪个ITSMClient
	if config.DragonflyEnabled {
		itsmClient = NewDragonflyITSMClient(logger, true, database)
	} else {
		itsmClient = &NoopITSMClient{}
	}

	return &ChangeManager{
		config:     config,
		redis:      redis,
		itsmClient: itsmClient,
		logger:     logger,
		database:   database,
	}
}

// IsEnabled 是否启用变更管理
func (m *ChangeManager) IsEnabled() bool {
	return m.config.Enabled
}

// WithChange 包装操作，自动管理变更单生命周期
// 对于同步操作，完成后自动关闭变更单
// 对于异步操作 (AWX/Drain)，需要通过 AttachAWXJob 或 AttachDrainIDs 关联，由 Poller 或回调关闭
func (m *ChangeManager) WithChange(
	ctx context.Context,
	opType ChangeOperationType,
	targets []string,
	metadata map[string]any,
	operation func(ticketID string) error,
	opts ...WithChangeOption,
) (string, error) {
	if !m.config.Enabled {
		// 禁用时直接执行操作
		return "", operation("")
	}

	// 【NEW】For drain operations, skip ChangeManager ITSM integration
	// since Dragonfly handles change orders directly in SimpleDrainService
	if opType == ChangeOpDrain {
		m.logger.Info("Drain operation: skipping ChangeManager ITSM, using Dragonfly integration",
			zap.String("op", string(opType)),
			zap.Strings("targets", targets))
		return "", operation("")
	}

	// 解析可选项
	options := &changeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 【关键改动】在创建变更单前执行验证
	if options.validator != nil {
		if err := options.validator(ctx, targets, metadata); err != nil {
			m.logger.Warn("变更验证失败",
				zap.String("op", string(opType)),
				zap.Strings("targets", targets),
				zap.Error(err))
			return "", fmt.Errorf("变更验证失败: %w", err)
		}
	}

	// 1. 创建变更单
	ticket := &ChangeTicket{
		ID:            uuid.New().String(),
		OperationType: opType,
		Targets:       targets,
		Status:        TicketStatusPending,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}

	// 2. 调用 ITSM 创建工单
	itsmTicketID, err := m.itsmClient.CreateTicket(ctx, ticket)
	if err != nil {
		m.logger.Error("创建 ITSM 工单失败", zap.Error(err), zap.String("op", string(opType)))
		// 继续执行操作，不因 ITSM 失败阻塞业务
	}
	ticket.ITSMTicketID = itsmTicketID
	ticket.Status = TicketStatusInProgress

	// 3. 持久化变更单
	if err := m.saveTicket(ticket); err != nil {
		m.logger.Error("保存变更单失败", zap.Error(err))
	}
	m.addToPendingIndex(ticket.ID)

	m.logger.Info("变更单已创建",
		zap.String("ticket_id", ticket.ID),
		zap.String("itsm_ticket_id", itsmTicketID),
		zap.String("op", string(opType)),
		zap.Strings("targets", targets))

	// 4. 执行实际操作
	opErr := operation(ticket.ID)

	// 5. 判断是否为异步操作
	isAsync := m.isAsyncOperation(opType)
	if !isAsync {
		// 同步操作：立即关闭变更单 (失败也视为成功关闭)
		if err := m.CompleteTicket(ticket.ID, true, ""); err != nil {
			m.logger.Error("完成变更单失败", zap.Error(err), zap.String("ticket_id", ticket.ID))
		}
	}

	return ticket.ID, opErr
}

// AttachAWXJob 关联 AWX Job 到变更单
func (m *ChangeManager) AttachAWXJob(ticketID string, awxJobID int) error {
	if !m.config.Enabled {
		return nil
	}

	ticket, err := m.getTicket(ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return fmt.Errorf("ticket not found: %s", ticketID)
	}

	ticket.AWXJobID = awxJobID
	return m.saveTicket(ticket)
}

// AttachDrainIDs 关联 Drain IDs 到变更单
func (m *ChangeManager) AttachDrainIDs(ticketID string, drainIDs []string) error {
	if !m.config.Enabled {
		return nil
	}

	ticket, err := m.getTicket(ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		return fmt.Errorf("ticket not found: %s", ticketID)
	}

	ticket.DrainIDs = append(ticket.DrainIDs, drainIDs...)
	return m.saveTicket(ticket)
}

// CompleteTicket 完成变更单
func (m *ChangeManager) CompleteTicket(ticketID string, success bool, errorMsg string) error {
	if !m.config.Enabled {
		return nil
	}

	ticket, err := m.getTicket(ticketID)
	if err != nil {
		return err
	}
	if ticket == nil {
		m.logger.Warn("变更单不存在", zap.String("ticket_id", ticketID))
		return nil
	}

	if ticket.Status == TicketStatusCompleted {
		return nil // 已完成，幂等
	}

	// 更新状态
	now := time.Now()
	ticket.Status = TicketStatusCompleted
	ticket.CompletedAt = &now
	if errorMsg != "" {
		ticket.Error = errorMsg
	}

	// 调用 ITSM 关闭工单 (失败也视为成功)
	if err := m.itsmClient.CloseTicket(context.Background(), ticket.ITSMTicketID, true, "操作完成"); err != nil {
		m.logger.Error("关闭 ITSM 工单失败", zap.Error(err), zap.String("itsm_ticket_id", ticket.ITSMTicketID))
	}

	// 保存并从 pending 移除
	if err := m.saveTicket(ticket); err != nil {
		m.logger.Error("保存变更单失败", zap.Error(err))
	}
	m.removeFromPendingIndex(ticketID)

	m.logger.Info("变更单已完成",
		zap.String("ticket_id", ticketID),
		zap.String("itsm_ticket_id", ticket.ITSMTicketID),
		zap.Bool("success", success))

	return nil
}

// OnDrainComplete Drain 完成回调
func (m *ChangeManager) OnDrainComplete(drainID string, success bool) {
	if !m.config.Enabled {
		return
	}

	// 查找关联此 drainID 的变更单
	ticket := m.findTicketByDrainID(drainID)
	if ticket == nil {
		return
	}

	// 检查是否所有 Drain 都完成了
	// 简化实现：只要有一个完成就关闭（实际可扩展为全部完成）
	if err := m.CompleteTicket(ticket.ID, success, ""); err != nil {
		m.logger.Error("完成变更单失败", zap.Error(err), zap.String("ticket_id", ticket.ID))
	}
}

// GetPendingTickets 获取所有待处理的变更单
func (m *ChangeManager) GetPendingTickets() []*ChangeTicket {
	if m.redis == nil {
		return nil
	}

	ids := m.getPendingTicketIDs()
	tickets := make([]*ChangeTicket, 0, len(ids))
	for _, id := range ids {
		if ticket, _ := m.getTicket(id); ticket != nil {
			tickets = append(tickets, ticket)
		}
	}
	return tickets
}

// GetPendingAWXTickets 获取所有关联 AWX Job 的待处理变更单
func (m *ChangeManager) GetPendingAWXTickets() []*ChangeTicket {
	pending := m.GetPendingTickets()
	result := make([]*ChangeTicket, 0)
	for _, t := range pending {
		if t.AWXJobID > 0 {
			result = append(result, t)
		}
	}
	return result
}

// RecoverPendingTickets 恢复未完成的变更单 (进程启动时调用)
func (m *ChangeManager) RecoverPendingTickets(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	m.logger.Info("开始恢复待处理的变更单...")

	tickets := m.GetPendingTickets()
	recovered := 0

	for _, ticket := range tickets {
		// 检查是否超时
		if time.Since(ticket.CreatedAt) > m.config.Timeout {
			m.logger.Info("变更单已超时，自动关闭",
				zap.String("ticket_id", ticket.ID),
				zap.Duration("age", time.Since(ticket.CreatedAt)))
			if err := m.CompleteTicket(ticket.ID, true, "操作超时，自动关闭"); err != nil {
				m.logger.Error("完成变更单失败", zap.Error(err), zap.String("ticket_id", ticket.ID))
			}
			recovered++
			continue
		}

		// AWX Job 由 Poller 处理，跳过
		if ticket.AWXJobID > 0 {
			continue
		}

		// Drain 任务需要检查状态 (由 SimpleDrainService 回调处理)
		if len(ticket.DrainIDs) > 0 {
			continue
		}

		// 其他情况：可能是进程崩溃导致的孤儿变更单
		m.logger.Warn("发现孤儿变更单，等待超时后自动关闭",
			zap.String("ticket_id", ticket.ID),
			zap.Duration("remaining", m.config.Timeout-time.Since(ticket.CreatedAt)))
	}

	m.logger.Info("变更单恢复完成", zap.Int("recovered", recovered), zap.Int("total_pending", len(tickets)))
	return nil
}

// ----- Internal Methods -----

func (m *ChangeManager) isAsyncOperation(opType ChangeOperationType) bool {
	switch opType {
	case ChangeOpDrain, ChangeOpShutdown, ChangeOpReboot:
		return true
	case ChangeOpCordon, ChangeOpUncordon, ChangeOpTaint, ChangeOpLabel:
		return false
	default:
		return false
	}
}

func (m *ChangeManager) saveTicket(ticket *ChangeTicket) error {
	if m.redis == nil {
		return nil
	}
	data, err := json.Marshal(ticket)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(changeTicketKeyFmt, ticket.ID)
	m.redis.SetWithExpireTime(key, string(data), changeTicketTTL)
	return nil
}

func (m *ChangeManager) getTicket(ticketID string) (*ChangeTicket, error) {
	if m.redis == nil {
		return nil, nil
	}
	key := fmt.Sprintf(changeTicketKeyFmt, ticketID)
	data := m.redis.Get(key)
	if data == "" {
		return nil, nil
	}
	var ticket ChangeTicket
	if err := json.Unmarshal([]byte(data), &ticket); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (m *ChangeManager) addToPendingIndex(ticketID string) {
	if m.redis == nil {
		return
	}
	// 使用 Redis Set
	if err := m.redis.SAdd(changePendingIndexKey, ticketID); err != nil {
		m.logger.Error("添加待处理变更单索引失败", zap.Error(err), zap.String("ticket_id", ticketID))
	}
}

func (m *ChangeManager) removeFromPendingIndex(ticketID string) {
	if m.redis == nil {
		return
	}
	if err := m.redis.SRem(changePendingIndexKey, ticketID); err != nil {
		m.logger.Error("从待处理变更单索引移除失败", zap.Error(err), zap.String("ticket_id", ticketID))
	}
}

func (m *ChangeManager) getPendingTicketIDs() []string {
	if m.redis == nil {
		return nil
	}
	ids, err := m.redis.SMembers(changePendingIndexKey)
	if err != nil {
		m.logger.Error("获取待处理变更单 ID 失败", zap.Error(err))
		return nil
	}
	return ids
}

func (m *ChangeManager) findTicketByDrainID(drainID string) *ChangeTicket {
	tickets := m.GetPendingTickets()
	for _, t := range tickets {
		for _, id := range t.DrainIDs {
			if id == drainID {
				return t
			}
		}
	}
	return nil
}
