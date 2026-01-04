package api

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
func SetupRoutes(router *gin.Engine, database *db.Database, navyDatabase *db.NavyDatabase, cfg *config.Config) error {
	handlers, err := buildHandlerSet(database, navyDatabase, cfg)
	if err != nil {
		return err
	}

	registrar := newRouteRegistrar(router, cfg, handlers)
	registrar.register()
	return nil
}
