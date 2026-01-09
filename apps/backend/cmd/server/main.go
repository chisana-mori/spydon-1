package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"robusta-web/backend/internal/api"
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/db"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/support"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// =============================================================================
// Application 应用程序结构体，管理所有运行时组件
// =============================================================================

type Application struct {
	cfg         *config.Config
	database    *db.Database
	router      *gin.Engine
	server      *http.Server
	nodeSyncMgr *nodesync.Manager
	bgServices  *api.BackgroundServices
	awxRuntime  *pipelineservice.AWXRuntime

	// 用于优雅关闭的 context
	nodeSyncCtx    context.Context
	nodeSyncCancel context.CancelFunc
	bgCtx          context.Context
	bgCancel       context.CancelFunc
}

// =============================================================================
// main 入口函数
// =============================================================================

func main() {
	app := &Application{}

	// 1. 初始化基础设施
	if err := app.initInfrastructure(); err != nil {
		logger.L().Fatal("初始化基础设施失败", zap.Error(err))
	}

	// 2. 初始化业务服务
	if err := app.initServices(); err != nil {
		logger.L().Fatal("初始化业务服务失败", zap.Error(err))
	}

	// 3. 启动服务器
	app.startServer()

	// 4. 等待终止信号
	app.waitForShutdown()
}

// =============================================================================
// 初始化方法
// =============================================================================

// initInfrastructure 初始化基础设施：配置、日志、数据库、外部依赖
func (app *Application) initInfrastructure() error {
	// 初始化临时日志（用于启动阶段）
	if err := logger.Init(nil); err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认日志失败: %v\n", err)
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	app.cfg = cfg

	// 初始化正式日志
	if err := app.initLogger(); err != nil {
		return err
	}

	// 初始化外部依赖
	app.initExternalDependencies()

	// 初始化数据库
	if err := app.initDatabase(); err != nil {
		return err
	}

	return nil
}

// initLogger 初始化日志组件
func (app *Application) initLogger() error {
	loggerCfg := &logger.Config{
		LogLevel:    app.cfg.LogLevel,
		LogFilePath: app.cfg.LogFilePath,
	}
	if err := logger.Init(loggerCfg); err != nil {
		return fmt.Errorf("初始化日志组件失败: %w", err)
	}
	return nil
}

// initExternalDependencies 初始化外部依赖（dlink, narwhal, dragonfly, mailer, redis）
func (app *Application) initExternalDependencies() {
	if v := config.GetViper(); v != nil {
		if _, err := support.Init(v); err != nil {
			logger.L().Warn("初始化外部依赖失败", zap.Error(err))
		}
	}
}

// initDatabase 初始化数据库连接
func (app *Application) initDatabase() error {
	database, err := db.Initialize(app.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("数据库初始化失败: %w", err)
	}
	app.database = database

	// 运行自动迁移
	if err := database.AutoMigrate(); err != nil {
		return fmt.Errorf("数据库自动迁移失败: %w", err)
	}

	// 创建索引
	if err := database.CreateIndexes(); err != nil {
		return fmt.Errorf("创建数据库索引失败: %w", err)
	}

	return nil
}

// initServices 初始化业务服务：路由、后台任务
func (app *Application) initServices() error {
	// 设置 Gin 模式
	if app.cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	app.router = gin.New()
	app.router.Use(gin.Recovery())

	// 启动节点同步管理器
	app.startNodeSyncManager()

	// 创建 AWX Runtime
	app.awxRuntime = pipelineservice.NewAWXRuntimeFromConfig(app.cfg)

	// 设置 API 路由
	bgServices, err := api.SetupRoutes(
		app.router,
		app.database,
		app.cfg,
		app.nodeSyncMgr,
		app.awxRuntime,
	)
	if err != nil {
		return fmt.Errorf("初始化路由失败: %w", err)
	}
	app.bgServices = bgServices

	// 启动后台服务
	app.startBackgroundServices()

	return nil
}

// startNodeSyncManager 启动节点同步管理器
func (app *Application) startNodeSyncManager() {
	app.nodeSyncMgr = nodesync.NewManager(app.database)
	app.nodeSyncCtx, app.nodeSyncCancel = context.WithCancel(context.Background())

	go func() {
		if err := app.nodeSyncMgr.Start(app.nodeSyncCtx); err != nil {
			logger.L().Error("节点同步管理器启动失败", zap.Error(err))
		}
	}()
}

// startBackgroundServices 启动后台服务
func (app *Application) startBackgroundServices() {
	app.bgCtx, app.bgCancel = context.WithCancel(context.Background())
	if app.bgServices != nil {
		app.bgServices.Start(app.bgCtx)
	}
}

// =============================================================================
// 服务器生命周期管理
// =============================================================================

// startServer 启动 HTTP 服务器
func (app *Application) startServer() {
	port := app.cfg.Port
	if port == "" {
		port = strconv.Itoa(constants.DefaultPort)
	}

	addr := "0.0.0.0:" + port
	logger.L().Info("服务器启动", zap.String("addr", addr))

	app.server = &http.Server{
		Addr:         addr,
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE 流式请求禁止写超时
		IdleTimeout:  0, // 交给上游/代理控制空闲连接
	}

	go func() {
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L().Fatal("服务器启动失败", zap.Error(err))
		}
	}()
}

// waitForShutdown 等待终止信号并优雅关闭
func (app *Application) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L().Info("收到终止信号，开始优雅关闭...")
	app.shutdown()
}

// shutdown 优雅关闭所有服务
func (app *Application) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 停止后台服务
	app.bgCancel()
	if app.bgServices != nil {
		app.bgServices.Stop()
	}

	// 2. 停止节点同步管理器
	app.nodeSyncCancel()
	app.nodeSyncMgr.Stop()

	// 3. 关闭外部依赖
	support.Close()

	// 4. 同步日志
	logger.Sync()

	// 5. 关闭 HTTP 服务器
	if err := app.server.Shutdown(ctx); err != nil {
		logger.L().Error("服务优雅关闭失败", zap.Error(err))
	}

	logger.L().Info("服务器已关闭")
}
