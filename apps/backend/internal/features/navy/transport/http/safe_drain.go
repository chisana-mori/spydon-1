package http

import (
	"net/http"
	"time"

	"robusta-web/backend/internal/features/navy/services"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// SafeDrainHandler - 安全驱逐处理器
// =============================================================================

// SafeDrainHandler 处理安全驱逐相关的 HTTP 请求
type SafeDrainHandler struct {
	svc           *services.SimpleDrainService
	changeManager *services.ChangeManager
	validator     *services.DeviceValidator
}

// NewSafeDrainHandler 创建 SafeDrainHandler
func NewSafeDrainHandler(
	svc *services.SimpleDrainService,
	changeMgr *services.ChangeManager,
	validator *services.DeviceValidator,
) *SafeDrainHandler {
	return &SafeDrainHandler{
		svc:           svc,
		changeManager: changeMgr,
		validator:     validator,
	}
}

// RegisterRoutes 注册安全驱逐路由
func (h *SafeDrainHandler) RegisterRoutes(navyGroup *gin.RouterGroup) {
	drain := navyGroup.Group(RouteGroupDrain)
	{
		drain.POST(RouteStart, h.StartDrain)
		drain.POST(RouteParamDrainIDCancel, h.CancelDrain)
		drain.GET(RouteParamDrainIDMigrations, h.GetDrainMigrations)
		drain.GET(RouteParamDrainIDEvents, h.DrainEvents)
	}
}

// StartDrain 启动安全驱逐
// @Summary Start node drain
// @Tags Drain
// @Accept json
// @Produce json
// @Param request body services.SimpleDrainRequest true "Drain request"
// @Success 200 {object} services.GenericResponse "Success"
// @Failure 400 {object} services.GenericResponse "Bad request"
// @Failure 500 {object} services.GenericResponse "Internal error"
// @Router /api/v1/navy/drain/start [post]
func (h *SafeDrainHandler) StartDrain(c *gin.Context) {
	var req services.SimpleDrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, services.GenericResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	var response *services.SimpleDrainResponse
	var opErr error

	ticketID, err := h.changeManager.WithChange(
		c.Request.Context(),
		services.ChangeOpDrain,
		[]string{req.NodeName},
		map[string]any{"cluster": req.ClusterName, "force": req.Force, "dry_run": req.DryRun},
		func(tid string) error {
			response, opErr = h.svc.StartDrain(c.Request.Context(), &req)
			return opErr
		},
		services.WithValidator(h.validator.ValidateDrainPrerequisite),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, services.GenericResponse{
			Success: false,
			Message: "Failed to start drain: " + err.Error(),
		})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, services.GenericResponse{
			Success: false,
			Message: "Failed to start drain: " + opErr.Error(),
		})
		return
	}

	// Attach drainID to the change ticket for async tracking
	if response != nil && ticketID != "" {
		_ = h.changeManager.AttachDrainIDs(ticketID, []string{response.DrainID})
	}

	c.JSON(http.StatusOK, services.GenericResponse{
		Success: true,
		Message: response.Message,
		Data: map[string]any{
			"drainId":   response.DrainID,
			"nodeName":  req.NodeName,
			"status":    response.Status,
			"startTime": time.Now(),
		},
	})
}

// CancelDrain 取消安全驱逐
// @Summary Cancel drain
// @Tags Drain
// @Param drainId path string true "Drain ID"
// @Success 200 {object} services.GenericResponse "Success"
// @Failure 400 {object} services.GenericResponse "Bad request"
// @Failure 500 {object} services.GenericResponse "Internal error"
// @Router /api/v1/navy/drain/{drainId}/cancel [post]
func (h *SafeDrainHandler) CancelDrain(c *gin.Context) {
	drainID := c.Param("drain_id")
	if drainID == "" {
		drainID = c.Param("drainId")
	}
	if drainID == "" {
		c.JSON(http.StatusBadRequest, services.GenericResponse{
			Success: false,
			Message: "drain_id is required",
		})
		return
	}

	if err := h.svc.CancelDrain(drainID); err != nil {
		c.JSON(http.StatusInternalServerError, services.GenericResponse{
			Success: false,
			Message: "Cancel failed: " + err.Error(),
		})
		return
	}

	// Cancel 也视为成功，关闭关联的变更单
	h.changeManager.OnDrainComplete(drainID, true)

	c.JSON(http.StatusOK, services.GenericResponse{
		Success: true,
		Message: "Drain canceled",
		Data: map[string]any{
			"drainId":    drainID,
			"cancelTime": time.Now(),
		},
	})
}

// GetDrainMigrations 获取驱逐迁移详情
// @Summary Get drain migrations
// @Tags Drain
// @Param drainId path string true "Drain ID"
// @Success 200 {object} services.GenericResponse "Success"
// @Failure 400 {object} services.GenericResponse "Bad request"
// @Failure 500 {object} services.GenericResponse "Internal error"
// @Router /api/v1/navy/drain/{drainId}/migrations [get]
func (h *SafeDrainHandler) GetDrainMigrations(c *gin.Context) {
	drainID := c.Param("drain_id")
	if drainID == "" {
		drainID = c.Param("drainId")
	}
	if drainID == "" {
		c.JSON(http.StatusBadRequest, services.GenericResponse{
			Success: false,
			Message: "drain_id is required",
		})
		return
	}

	migrations, err := h.svc.GetDrainMigrations(drainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, services.GenericResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if migrations == nil {
		migrations = []*services.DrainPodMigrationInfo{}
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.JSON(http.StatusOK, services.GenericResponse{
		Success: true,
		Message: "Migrations retrieved",
		Data: map[string]any{
			"drainId":    drainID,
			"migrations": migrations,
			"summary":    buildMigrationsSummary(migrations),
		},
	})
}

// DrainEvents 安全驱逐事件流（SSE）
// @Summary SSE event stream
// @Tags Drain
// @Param drainId path string true "Drain ID"
// @Success 200 {string} string "SSE stream"
// @Router /api/v1/navy/drain/{drainId}/stream [get]
func (h *SafeDrainHandler) DrainEvents(c *gin.Context) {
	h.svc.HandleSSE(c)
}

// buildMigrationsSummary 构建迁移统计摘要
func buildMigrationsSummary(migrations []*services.DrainPodMigrationInfo) map[string]int {
	summary := map[string]int{
		"total":     len(migrations),
		"pending":   0,
		"evicting":  0,
		"evicted":   0,
		"creating":  0,
		"completed": 0,
		"failed":    0,
		"migrating": 0,
		"ignored":   0,
	}

	for _, m := range migrations {
		switch m.Status {
		case services.DrainMigrationPending:
			summary["pending"]++
		case services.DrainMigrationEvicting:
			summary["evicting"]++
			summary["migrating"]++
		case services.DrainMigrationEvicted:
			summary["evicted"]++
		case services.DrainMigrationCreating:
			summary["creating"]++
			summary["migrating"]++
		case services.DrainMigrationCompleted:
			summary["completed"]++
		case services.DrainMigrationFailed, services.DrainMigrationTimeout:
			summary["failed"]++
		case services.DrainMigrationIgnored:
			summary["ignored"]++
		}
	}

	return summary
}
