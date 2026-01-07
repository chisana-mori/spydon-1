package api

import (
	"context"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	navyservice "robusta-web/backend/internal/features/navy/services"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/pkg/nodesync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BackgroundServices 需要在 main 中启动的后台服务
type BackgroundServices struct {
	ChangeManager *navyservice.ChangeManager
	AWXJobPoller  *navyservice.AWXJobPoller
}

// Start 启动所有后台服务
func (s *BackgroundServices) Start(ctx context.Context) {
	if s.ChangeManager != nil {
		// 恢复未完成的变更单
		go func() {
			if err := s.ChangeManager.RecoverPendingTickets(ctx); err != nil {
				logger.L().Error("恢复待处理变更单失败", zap.Error(err))
			}
		}()
	}
	if s.AWXJobPoller != nil {
		// 启动 AWX Job 轮询器
		s.AWXJobPoller.Start(ctx)
	}
}

// Stop 停止所有后台服务
func (s *BackgroundServices) Stop() {
	if s.AWXJobPoller != nil {
		s.AWXJobPoller.Stop()
	}
}

// SetupRoutes 设置API路由并返回需要启动的后台服务
func SetupRoutes(router *gin.Engine, database *db.Database, navyDatabase *db.NavyDatabase, cfg *config.Config, nodesyncManager *nodesync.Manager, awxRuntime *pipelineservice.AWXRuntime) (*BackgroundServices, error) {
	handlers, bgServices, err := buildHandlerSet(database, navyDatabase, cfg, nodesyncManager, awxRuntime)
	if err != nil {
		return nil, err
	}

	registrar := newRouteRegistrar(router, cfg, handlers, navyDatabase)
	registrar.register()
	return bgServices, nil
}
