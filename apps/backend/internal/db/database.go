package db

import (
	"context"
	"fmt"
	"strings"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Database 数据库连接包装器
type Database struct {
	*gorm.DB
}

// getDialector 根据 URL 自动检测并返回 GORM Dialector
func getDialector(databaseURL string) gorm.Dialector {
	// 简单的启发式检测：MySQL DSN 通常包含 @tcp( 或 @unix( 或 Protocol=tcp
	if strings.Contains(databaseURL, "@tcp(") || strings.Contains(databaseURL, "@unix(") || strings.Contains(databaseURL, "Protocol=tcp") {
		return mysql.Open(databaseURL)
	}
	// 默认为 SQLite (File)
	return sqlite.Open(databaseURL)
}

// Initialize 初始化数据库连接
func Initialize(databaseURL string) (*Database, error) {
	// 配置GORM日志
	gormConfig := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// 连接数据库
	db, err := gorm.Open(getDialector(databaseURL), gormConfig)
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
	// 预清理：移除可能存在的PostgreSQL旧索引名称
	// 必须在AutoMigrate之前执行，以避免GORM误将索引识别为外键
	if err := d.cleanupLegacyIndexes(); err != nil {
		logger.S().Warnw("清理旧索引时出现警告", "error", err)
		// 不返回错误，继续迁移
	}

	// 逐表迁移，确保即使某个表的索引操作失败也不会阻止其他表创建
	tables := []interface{}{
		&models.PipelineTemplate{},
		&models.PipelineExecution{},
		&models.StageRun{},
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
		&models.Dictionary{},
		&models.DictionaryItem{},
		&models.EmailTemplate{},
		&models.EmailContact{},
	}

	for _, table := range tables {
		if err := d.DB.AutoMigrate(table); err != nil {
			errStr := err.Error()
			// MySQL Error 1091: Can't DROP ... check that column/key exists
			// 这在重复迁移或从 PostgreSQL 迁移后常见，索引/外键不存在时发生
			// 这类错误是警告性的，不应阻止迁移继续
			if strings.Contains(errStr, "Error 1091") || strings.Contains(errStr, "Can't DROP") {
				logger.S().Warnw("忽略迁移时的索引/外键清理错误（不影响表创建）",
					"table", fmt.Sprintf("%T", table),
					"error", err)
				// 继续执行下一个表
			} else {
				return fmt.Errorf("迁移表 %T 失败: %w", table, err)
			}
		}
	}

	if err := d.ensureForeignKeys(); err != nil {
		return err
	}

	logger.S().Infow("数据库表结构迁移完成")
	return nil
}

// cleanupLegacyIndexes 清理从PostgreSQL迁移可能遗留的旧索引
func (d *Database) cleanupLegacyIndexes() error {
	// 这些索引名称是GORM在PostgreSQL中自动创建的
	// 在MySQL中，GORM可能会误将它们识别为外键约束
	legacyIndexes := []struct {
		table string
		index string
	}{
		{"users", "uni_users_username"},
		{"users", "uni_users_email"},
		{"refresh_tokens", "uni_refresh_tokens_token"},
		{"system_settings", "uni_system_settings_key"},
	}

	for _, idx := range legacyIndexes {
		// 首先检查表是否存在
		if !d.Migrator().HasTable(idx.table) {
			continue
		}

		// 检查索引是否存在
		var count int64
		query := fmt.Sprintf(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '%s' AND index_name = '%s'",
			idx.table, idx.index,
		)
		if err := d.Raw(query).Count(&count).Error; err != nil {
			logger.S().Warnw("检查索引失败", "table", idx.table, "index", idx.index, "error", err)
			continue
		}

		if count > 0 {
			// 索引存在，删除它
			sql := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", idx.table, idx.index)
			if err := d.Exec(sql).Error; err != nil {
				logger.S().Warnw("删除旧索引失败", "table", idx.table, "index", idx.index, "error", err)
			} else {
				logger.S().Infow("已删除旧索引", "table", idx.table, "index", idx.index)
			}
		}
	}

	logger.S().Infow("旧索引清理完成")
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

func (d *Database) ensureForeignKeys() error {
	// 仅为需要的表创建外键，保持其余关系由应用层控制
	if !d.Migrator().HasConstraint(&models.RefreshToken{}, "User") {
		if err := d.Migrator().CreateConstraint(&models.RefreshToken{}, "User"); err != nil {
			return fmt.Errorf("创建refresh_tokens -> users外键失败: %w", err)
		}
	}

	if !d.Migrator().HasConstraint(&models.APIKey{}, "User") {
		if err := d.Migrator().CreateConstraint(&models.APIKey{}, "User"); err != nil {
			return fmt.Errorf("创建api_keys -> users外键失败: %w", err)
		}
	}

	logger.S().Infow("外键约束处理完成（按需）")
	return nil
}

// CreateIndexes 创建数据库索引
func (d *Database) CreateIndexes() error {
	indexes := []struct {
		name   string
		model  interface{}
		create string
	}{
		{
			name:   "idx_alerts_fingerprint_cluster",
			model:  &models.Alert{},
			create: "CREATE UNIQUE INDEX idx_alerts_fingerprint_cluster ON spydon_alerts(fingerprint, cluster_name)",
		},
		{
			name:   "idx_alerts_cluster_severity",
			model:  &models.Alert{},
			create: "CREATE INDEX idx_alerts_cluster_severity ON spydon_alerts(cluster_name, severity)",
		},
		{
			name:   "idx_alerts_status_created",
			model:  &models.Alert{},
			create: "CREATE INDEX idx_alerts_status_created ON spydon_alerts(status, created_at DESC)",
		},
		{
			name:   "idx_rca_runs_alert_status",
			model:  &models.RCARun{},
			create: "CREATE INDEX idx_rca_runs_alert_status ON spydon_rca_runs(alert_id, status)",
		},
		{
			name:   "idx_audit_logs_user_action",
			model:  &models.AuditLog{},
			create: "CREATE INDEX idx_audit_logs_user_action ON spydon_audit_logs(user_id, action, created_at DESC)",
		},
		{
			name:   "idx_api_keys_user_id",
			model:  &models.APIKey{},
			create: "CREATE INDEX idx_api_keys_user_id ON spydon_api_keys(user_id)",
		},
		{
			name:   "idx_api_keys_key_prefix",
			model:  &models.APIKey{},
			create: "CREATE INDEX idx_api_keys_key_prefix ON spydon_api_keys(key_prefix)",
		},
		{
			name:   "idx_kb_rule_norm_status",
			model:  &models.KnowledgeArticle{},
			create: "CREATE INDEX idx_kb_rule_norm_status ON spydon_knowledge_articles(alert_rule_name_normalized, status)",
		},
		{
			name:   "idx_pipeline_executions_template_status",
			model:  &models.PipelineExecution{},
			create: "CREATE INDEX idx_pipeline_executions_template_status ON spydon_pipeline_executions(pipeline_template_id, status, created_at DESC)",
		},
		{
			name:   "idx_pipeline_executions_cluster_status",
			model:  &models.PipelineExecution{},
			create: "CREATE INDEX idx_pipeline_executions_cluster_status ON spydon_pipeline_executions(cluster_id, status, created_at DESC)",
		},
		{
			name:   "idx_stage_runs_execution_id",
			model:  &models.StageRun{},
			create: "CREATE INDEX idx_stage_runs_execution_id ON spydon_stage_runs(execution_id)",
		},
	}

	for _, idx := range indexes {
		if d.Migrator().HasIndex(idx.model, idx.name) {
			continue
		}
		if err := d.Exec(idx.create).Error; err != nil {
			return fmt.Errorf("创建索引 %s 失败: %w", idx.name, err)
		}
	}

	logger.S().Infow("数据库索引创建完成")
	return nil
}
