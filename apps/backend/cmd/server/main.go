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
	"robusta-web/backend/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	if err := logger.Init(nil); err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认日志失败: %v\n", err)
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.L().Fatal("加载配置失败", zap.Error(err))
	}

	if initErr := logger.Init(cfg); initErr != nil {
		logger.L().Fatal("初始化日志组件失败", zap.Error(initErr))
	}
	defer logger.Sync()

	// 初始化数据库连接
	database, err := db.Initialize(cfg.DatabaseURL)
	if err != nil {
		logger.L().Fatal("数据库初始化失败", zap.Error(err))
	}

	// 运行自动迁移
	if err := database.AutoMigrate(); err != nil {
		logger.L().Fatal("数据库自动迁移失败", zap.Error(err))
	}

	// 创建索引
	if err := database.CreateIndexes(); err != nil {
		logger.L().Fatal("创建数据库索引失败", zap.Error(err))
	}

	// 设置Gin模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()
	router.Use(gin.Recovery())

	// 设置API路由
	if err := api.SetupRoutes(router, database, cfg); err != nil {
		logger.L().Fatal("初始化路由失败", zap.Error(err))
	}

	// 启动服务器
	port := cfg.Port
	if port == "" {
		port = strconv.Itoa(constants.DefaultPort)
	}

	host := "0.0.0.0"
	logger.L().Info("服务器启动", zap.String("host", host), zap.String("port", port))
	srv := &http.Server{
		Addr:         host + ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE 流式请求可能运行较久，禁止写超时避免被意外中断
		IdleTimeout:  0, // 交给上游/代理控制空闲连接，避免长连接被提前关闭
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L().Fatal("服务器启动失败", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Error("服务优雅关闭失败", zap.Error(err))
	}
}
