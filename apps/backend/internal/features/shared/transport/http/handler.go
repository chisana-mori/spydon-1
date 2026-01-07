package http

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler aggregates all Shared HTTP handlers
type Handler struct {
	cfg               *config.Config
	dictionaryService *services.DictionaryService
}

// New creates a new Shared feature handler.
func New(
	cfg *config.Config,
	dictionaryService *services.DictionaryService,
) *Handler {
	return &Handler{
		cfg:               cfg,
		dictionaryService: dictionaryService,
	}
}

// RegisterRoutes mounts all Shared routes under /api/v1.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	// /api/v1/shared
	sharedGroup := v1.Group("/shared")
	sharedGroup.Use(middleware.CookieAuthMiddleware(h.cfg))

	// /api/v1/shared/dictionaries - 字典管理 API（需要管理员权限）
	dictGroup := sharedGroup.Group("/dictionaries")
	dictGroup.Use(middleware.RequireAdmin())
	{
		dictGroup.GET("", h.ListDictionaries)
		dictGroup.POST("", h.CreateDictionary)
		dictGroup.GET("/:id", h.GetDictionary)
		dictGroup.PUT("/:id", h.UpdateDictionary)
		dictGroup.DELETE("/:id", h.DeleteDictionary)

		// 字典项管理
		dictGroup.GET("/:id/items", h.GetDictionaryItems)
		dictGroup.POST("/:id/items", h.CreateDictionaryItem)
		dictGroup.PUT("/:id/items/:item_id", h.UpdateDictionaryItem)
		dictGroup.DELETE("/:id/items/:item_id", h.DeleteDictionaryItem)
		dictGroup.PUT("/:id/items", h.BatchUpdateDictionaryItems)
	}

	// /api/v1/shared/dict/:code/items - 通用获取接口（无需管理员权限）
	// 这是所有前端模版的通用接口
	sharedGroup.GET("/dict/:code/items", h.GetDictionaryItemsByCode)
}
