package http

import (
	holmesservice "robusta-web/backend/internal/features/holmes/services"
	ingestservice "robusta-web/backend/internal/features/ingest/services"
	"robusta-web/backend/internal/features/rca/services"

	"github.com/gin-gonic/gin"
)

// Handler handles RCA-related HTTP routes using feature-first structure.
type Handler struct {
	holmesService *holmesservice.HolmesService
	rcaService    *services.RCAService
	alertService  *ingestservice.AlertService
}

// New creates a new RCA HTTP handler.
func New(holmesService *holmesservice.HolmesService, rcaService *services.RCAService, alertService *ingestservice.AlertService) *Handler {
	return &Handler{
		holmesService: holmesService,
		rcaService:    rcaService,
		alertService:  alertService,
	}
}

// RegisterRoutes registers RCA-related routes under the given query group.
// queryGroup is /api/v1 group with admin middlewares already applied.
func (h *Handler) RegisterRoutes(queryGroup *gin.RouterGroup) {
	rcaGroup := queryGroup.Group("/rca")
	{
		rcaGroup.GET("/:alert_id", h.GetRCAByAlertID)
		rcaGroup.GET("/:alert_id/stream", h.StreamRCA)
		rcaGroup.GET("/:alert_id/cache", h.GetRCACacheByAlertID)
		rcaGroup.POST("/:alert_id/trigger", h.TriggerRCAByAlertID)
		rcaGroup.POST("/trigger", h.TriggerRCA)
		rcaGroup.GET("/runs", h.ListRCARuns)
		rcaGroup.GET("/runs/:run_id", h.GetRCARunStatus)
		rcaGroup.GET("/stats", h.GetRCAStats)
	}
}
