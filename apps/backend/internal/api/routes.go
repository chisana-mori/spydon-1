package api

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
func SetupRoutes(router *gin.Engine, database *db.Database, cfg *config.Config) error {
	handlers, err := buildHandlerSet(database, cfg)
	if err != nil {
		return err
	}

	registrar := newRouteRegistrar(router, cfg, handlers)
	registrar.register()
	return nil
}
