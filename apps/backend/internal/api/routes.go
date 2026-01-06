package api

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/services/nodesync"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
func SetupRoutes(router *gin.Engine, database *db.Database, navyDatabase *db.NavyDatabase, cfg *config.Config, nodesyncManager *nodesync.Manager) error {
	handlers, err := buildHandlerSet(database, navyDatabase, cfg, nodesyncManager)
	if err != nil {
		return err
	}

	registrar := newRouteRegistrar(router, cfg, handlers, navyDatabase)
	registrar.register()
	return nil
}
