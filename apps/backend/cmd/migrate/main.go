package main

import (
	"fmt"
	"os"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/logger"

	"go.uber.org/zap"
)

func main() {
	if err := logger.Init(nil); err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认日志失败: %v\n", err)
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.L().Fatal("加载配置失败", zap.Error(err))
	}

	// 初始化数据库连接
	database, err := db.Initialize(cfg.DatabaseURL)
	if err != nil {
		logger.L().Fatal("数据库初始化失败", zap.Error(err))
	}

	logger.S().Info("开始执行数据库迁移...")

	// 1. 先执行自动迁移，添加新字段
	if err := database.AutoMigrate(); err != nil {
		logger.L().Fatal("数据库自动迁移失败", zap.Error(err))
	}

	// 2. 执行手动 SQL 迁移
	sqls := []string{
		// 确保 clusters 表的 name 字段有值（从 cluster_name 同步）
		"UPDATE clusters SET name = cluster_name WHERE name IS NULL OR name = ''",

		// 为 alerts 表添加 cluster_id 字段 (AutoMigrate 应该已经加了，但为了保险)
		"ALTER TABLE alerts ADD COLUMN IF NOT EXISTS cluster_id VARCHAR(255)",

		// 清理 clusters 表的旧字段和旧唯一索引 (如果存在)
		"ALTER TABLE clusters DROP INDEX IF EXISTS cluster_name",
		"ALTER TABLE clusters DROP INDEX IF EXISTS uk_cluster_name",
		"ALTER TABLE clusters DROP INDEX IF EXISTS status",

		// 此时如果尝试 DROP COLUMN cluster_name，如果有关联可能失败
		// 但由于我们在 AutoMigrate 中禁用了外键检查，应该没问题
		"ALTER TABLE clusters DROP COLUMN IF EXISTS cluster_name",
	}

	for _, sql := range sqls {
		logger.S().Infof("正在执行 SQL: %s", sql)
		if err := database.Exec(sql).Error; err != nil {
			// 有些错误（比如字段不存在）可以忽略，具体取决于数据库版本
			logger.S().Warnf("执行 SQL 失败 (可能是字段/索引不存在): %v", err)
		}
	}

	// 3. 重新建立索引
	if err := database.CreateIndexes(); err != nil {
		logger.L().Fatal("创建数据库索引失败", zap.Error(err))
	}

	logger.S().Info("数据库迁移成功完成！")
}
