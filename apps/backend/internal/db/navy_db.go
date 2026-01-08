package db

import (
	"fmt"
	"robusta-web/backend/pkg/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NavyDatabase Navy 数据库连接包装器
type NavyDatabase struct {
	*gorm.DB
}

// InitializeNavy 初始化 Navy 数据库连接 (支持 SQLite 和 MySQL)
func InitializeNavy(databaseURL string) (*NavyDatabase, error) {
	// 配置GORM日志
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	}

	// 连接数据库
	db, err := gorm.Open(getDialector(databaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接 Navy 数据库失败: %w", err)
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 Navy 数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("Navy 数据库连接测试失败: %w", err)
	}

	logger.S().Infow("Navy 数据库连接成功", "path", databaseURL)

	return &NavyDatabase{db}, nil
}
