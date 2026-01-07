package http

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler aggregates all Navy-related HTTP handlers
// (device inventory, device operations, safe drain, k8s node management).
type Handler struct {
	cfg                  *config.Config
	navyDB               *db.NavyDatabase
	navyDeviceService    *services.NavyDeviceService
	deviceOpsService     *services.DeviceOperationsService
	safeDrainService     *services.SimpleDrainService
	k8sNodeManageService *services.K8sNodeManageService
	changeManager        *services.ChangeManager
}

// New creates a new Navy feature handler.
func New(
	cfg *config.Config,
	navyDB *db.NavyDatabase,
	navyDeviceService *services.NavyDeviceService,
	deviceOpsService *services.DeviceOperationsService,
	safeDrainService *services.SimpleDrainService,
	k8sNodeManageService *services.K8sNodeManageService,
	changeManager *services.ChangeManager,
) *Handler {
	return &Handler{
		cfg:                  cfg,
		navyDB:               navyDB,
		navyDeviceService:    navyDeviceService,
		deviceOpsService:     deviceOpsService,
		safeDrainService:     safeDrainService,
		k8sNodeManageService: k8sNodeManageService,
		changeManager:        changeManager,
	}
}

// RegisterRoutes mounts all Navy-related routes under /api/v1.
// The URL paths and middleware stack are kept identical to the
// legacy internal/api router setup.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	// /api/v1/navy
	navyGroup := v1.Group("/navy")
	navyGroup.Use(middleware.CookieAuthMiddleware(h.cfg))

	// /api/v1/navy/devices
	deviceGroup := navyGroup.Group("/devices")
	{
		deviceGroup.GET("", h.ListDevices)
		deviceGroup.POST("/query", h.QueryDevices)
		deviceGroup.GET("/filter-options", h.GetFilterOptions)
		deviceGroup.GET("/label-values", h.GetLabelValues)
		deviceGroup.GET("/taint-values", h.GetTaintValues)
		deviceGroup.GET("/device-field-values", h.GetDeviceFieldValues)
		deviceGroup.POST("/features", h.GetDeviceFeatures)
		deviceGroup.GET("/feature-details", h.GetFeatureDetails)
		deviceGroup.GET("/export", h.ExportDevices)
		deviceGroup.GET("/:id", h.GetDevice)
		deviceGroup.PATCH("/:id/role", h.UpdateDeviceRole)
		deviceGroup.PATCH("/:id/group", h.UpdateDeviceGroup)

		// Safe drain shortcuts under device path (kept for backward compatibility)
		deviceGroup.POST("/:node/drain/start", h.StartDrain)
		deviceGroup.POST("/:node/drain/cancel", h.CancelDrain)
	}

	// /api/v1/navy/drain - primary Safe Drain endpoints
	drainGroup := navyGroup.Group("/drain")
	{
		drainGroup.POST("/start", h.StartDrain)
		drainGroup.POST("/:drain_id/cancel", h.CancelDrain)
		drainGroup.GET("/:drain_id/migrations", h.GetDrainMigrations)
		drainGroup.GET("/:drain_id/events", h.DrainEvents)
	}

	// /api/v1/navy/templates - query template management
	templateGroup := navyGroup.Group("/templates")
	{
		templateGroup.GET("", h.GetTemplates)
		templateGroup.POST("", h.SaveTemplate)
		templateGroup.GET("/:id", h.GetTemplate)
		templateGroup.DELETE("/:id", h.DeleteTemplate)
	}

	// /api/v1/navy/device-ops - bulk device operations
	opsGroup := navyGroup.Group("/device-ops")
	opsGroup.Use(middleware.RequireAdmin())
	opsGroup.Use(middleware.AuditLogMiddleware())
	{
		// K8s node operations with pre-check: node must exist & be associated with a cluster
		k8sOps := opsGroup.Group("")
		k8sOps.Use(middleware.ValidateClusterAssociation(h.navyDB))
		{
			k8sOps.POST("/cordon", h.CordonNodes)
			k8sOps.POST("/uncordon", h.UncordonNodes)
			// k8sOps.POST("/drain", h.DrainNodes)
			k8sOps.POST("/taint", h.TaintNodes)
			k8sOps.POST("/label", h.LabelNodes)
		}

		// Power operations (AWX) - only need IP, no K8s cluster association
		opsGroup.POST("/shutdown", h.ShutdownNodes)
		opsGroup.POST("/reboot", h.RebootNodes)
	}

	// /api/v1/navy/k8s-nodes - real-time K8s node label/taint management
	k8sNodeGroup := navyGroup.Group("/k8s-nodes")
	k8sNodeGroup.Use(middleware.RequireAdmin())
	k8sNodeGroup.Use(middleware.AuditLogMiddleware())
	{
		// Queries
		k8sNodeGroup.GET("", h.ListClusterNodes)
		k8sNodeGroup.GET("/labels-taints", h.GetNodeLabelsAndTaints)
	}
}
