package db

import (
	"context"
	"fmt"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Database 数据库连接包装器
type Database struct {
	*gorm.DB
}

// Initialize 初始化数据库连接
func Initialize(databaseURL string) (*Database, error) {
	// 配置GORM日志
	gormConfig := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(databaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	logger.S().Infow("数据库连接成功")

	return &Database{db}, nil
}

func (d *Database) AutoMigrate() error {
	err := d.DB.AutoMigrate(
		&models.Cluster{},
		&models.Alert{},
		&models.RCARun{},
		&models.AuditLog{},
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.KnowledgeArticle{},
		&models.KnowledgeArticleVersion{},
		&models.SystemSetting{},
	)
	if err != nil {
		return err
	}

	logger.S().Infow("数据库表结构迁移完成，正在移除外键约束")

	// 移除 alerts -> clusters 外键约束（如果存在）
	// 这样可以避免在数据库层面显示外键错误，由应用层保证数据完整性
	if d.Migrator().HasConstraint("alerts", "fk_alerts_cluster") {
		err = d.Migrator().DropConstraint(&models.Alert{}, "fk_alerts_cluster")
		if err != nil {
			logger.S().Warnw("移除alerts -> clusters外键失败", "error", err)
		} else {
			logger.S().Infow("已移除alerts -> clusters外键约束")
		}
	}

	// 移除 rca_runs -> alerts 外键约束（如果存在）
	if d.Migrator().HasConstraint("rca_runs", "fk_rca_runs_alert") {
		err = d.Migrator().DropConstraint(&models.RCARun{}, "fk_rca_runs_alert")
		if err != nil {
			logger.S().Warnw("移除rca_runs -> alerts外键失败", "error", err)
		} else {
			logger.S().Infow("已移除rca_runs -> alerts外键约束")
		}
	}

	// Manually create foreign key for: refresh_tokens -> users
	if !d.Migrator().HasConstraint("refresh_tokens", "fk_refresh_tokens_user") {
		err = d.Exec(`
			ALTER TABLE "refresh_tokens"
			ADD CONSTRAINT "fk_refresh_tokens_user"
			FOREIGN KEY ("user_id")
			REFERENCES "users"("id")
			ON UPDATE CASCADE
			ON DELETE CASCADE;
		`).Error
		if err != nil {
			return fmt.Errorf("手动创建refresh_tokens -> users外键失败: %w", err)
		}
	}

	// Manually create foreign key for: api_keys -> users
	if !d.Migrator().HasConstraint("api_keys", "fk_api_keys_user") {
		err = d.Exec(`
			ALTER TABLE "api_keys"
			ADD CONSTRAINT "fk_api_keys_user"
			FOREIGN KEY ("user_id")
			REFERENCES "users"("id")
			ON UPDATE CASCADE
			ON DELETE CASCADE;
		`).Error
		if err != nil {
			return fmt.Errorf("手动创建api_keys -> users外键失败: %w", err)
		}
	}

	logger.S().Infow("外键约束处理完成")
	return nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext 返回绑定指定上下文的 *gorm.DB 会话
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return d.DB
	}
	return d.DB.WithContext(ctx)
}

// CreateIndexes 创建数据库索引
func (d *Database) CreateIndexes() error {
	// 为alerts表创建复合唯一索引
	if err := d.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_alerts_fingerprint_cluster
		ON alerts(fingerprint, cluster_id)
	`).Error; err != nil {
		return fmt.Errorf("创建alerts复合索引失败: %w", err)
	}

	// 为alerts表创建查询索引
	if err := d.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alerts_cluster_severity
		ON alerts(cluster_id, severity)
	`).Error; err != nil {
		return fmt.Errorf("创建alerts查询索引失败: %w", err)
	}

	if err := d.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alerts_status_created
		ON alerts(status, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("创建alerts状态索引失败: %w", err)
	}

	// 为rca_runs表创建索引
	if err := d.Exec(`
		CREATE INDEX IF NOT EXISTS idx_rca_runs_alert_status
		ON rca_runs(alert_id, status)
	`).Error; err != nil {
		return fmt.Errorf("创建rca_runs索引失败: %w", err)
	}

	// 为audit_logs表创建索引
	if err := d.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action
		ON audit_logs(user_id, action, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("创建audit_logs索引失败: %w", err)
	}

	// 为api_keys表创建索引
	if err := d.Exec(`
		CREATE INDEX IF NOT EXISTS idx_api_keys_user_id
		ON api_keys(user_id)
	`).Error; err != nil {
		return fmt.Errorf("创建api_keys用户索引失败: %w", err)
	}

	if err := d.Exec(`
        CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix
        ON api_keys(key_prefix)
    `).Error; err != nil {
		return fmt.Errorf("创建api_keys前缀索引失败: %w", err)
	}

	// knowledge 索引
	if err := d.Exec(`
        CREATE INDEX IF NOT EXISTS idx_kb_rule_norm_status
        ON knowledge_articles(alert_rule_name_normalized, status)
    `).Error; err != nil {
		return fmt.Errorf("创建knowledge_articles索引失败: %w", err)
	}

	if err := d.Exec(`
        CREATE INDEX IF NOT EXISTS idx_kb_tags_gin
        ON knowledge_articles USING GIN (tags)
    `).Error; err != nil {
		return fmt.Errorf("创建knowledge_articles GIN索引失败: %w", err)
	}

	logger.S().Infow("数据库索引创建完成")
	return nil
}

// EnableExtensions 启用PostgreSQL扩展
func (d *Database) EnableExtensions() error {
	// 启用UUID扩展
	if err := d.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("启用uuid-ossp扩展失败: %w", err)
	}

	// 启用pgcrypto扩展（用于gen_random_uuid）
	if err := d.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Error; err != nil {
		return fmt.Errorf("启用pgcrypto扩展失败: %w", err)
	}

	logger.S().Infow("PostgreSQL扩展启用完成")
	return nil
}
