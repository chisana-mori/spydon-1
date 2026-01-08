package http

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// Handler - Navy 功能聚合处理器
// =============================================================================

// Handler 聚合所有 Navy 相关 HTTP 处理器
// 包括：设备查询、设备操作、安全驱逐、K8s 节点管理、F5 管理
type Handler struct {
	cfg    *config.Config
	navyDB *db.NavyDatabase

	// 子处理器
	device *DeviceHandler
	ops    *DeviceOpsHandler
	drain  *SafeDrainHandler
	k8s    *K8sNodeHandler
	f5     *F5Handler
}

// New 创建 Navy 功能聚合处理器
func New(
	cfg *config.Config,
	navyDB *db.NavyDatabase,
	navyDeviceService *services.NavyDeviceService,
	deviceOpsService *services.DeviceOperationsService,
	safeDrainService *services.SimpleDrainService,
	k8sNodeManageService *services.K8sNodeManageService,
	changeManager *services.ChangeManager,
	f5Service *services.F5InfoService,
) *Handler {
	return &Handler{
		cfg:    cfg,
		navyDB: navyDB,

		device: NewDeviceHandler(navyDeviceService),
		ops:    NewDeviceOpsHandler(deviceOpsService, changeManager),
		drain:  NewSafeDrainHandler(safeDrainService, changeManager),
		k8s:    NewK8sNodeHandler(k8sNodeManageService),
		f5:     NewF5Handler(f5Service),
	}
}

// RegisterRoutes 挂载所有 Navy 相关路由
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	// /api/v1/navy
	navyGroup := v1.Group(RouteGroupNavy)
	navyGroup.Use(middleware.CookieAuthMiddleware(h.cfg))

	// 设备查询 + 模板管理
	h.device.RegisterRoutes(navyGroup)

	// 安全驱逐（无需管理员权限的只读/启动操作）
	h.drain.RegisterRoutes(navyGroup)

	// 设备批量操作（需要管理员权限 + 审计日志）
	opsGroup := navyGroup.Group("")
	opsGroup.Use(middleware.RequireAdmin())
	opsGroup.Use(middleware.AuditLogMiddleware())
	opsGroup.Use(middleware.ValidateClusterAssociation(h.navyDB))
	h.ops.RegisterRoutes(opsGroup)

	// K8s 节点管理（需要管理员权限 + 审计日志）
	k8sGroup := navyGroup.Group("")
	k8sGroup.Use(middleware.RequireAdmin())
	k8sGroup.Use(middleware.AuditLogMiddleware())
	h.k8s.RegisterRoutes(k8sGroup)

	// F5 管理
	h.f5.RegisterRoutes(navyGroup)
}
