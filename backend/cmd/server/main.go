package main

import (
	"log"
	"os"

	"robusta-web/backend/internal/api"
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用系统环境变量")
	}

	// 加载配置
	cfg := config.Load()

	// 初始化数据库连接
	database, err := db.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 启用PostgreSQL扩展
	if err := database.EnableExtensions(); err != nil {
		log.Fatalf("启用数据库扩展失败: %v", err)
	}

	// 运行自动迁移
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库自动迁移失败: %v", err)
	}

	// 创建索引
	if err := database.CreateIndexes(); err != nil {
		log.Fatalf("创建数据库索引失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.Default()

	// 设置API路由
	if err := api.SetupRoutes(router, database, cfg); err != nil {
		log.Fatalf("初始化路由失败: %v", err)
	}

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("服务器启动在端口 %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
