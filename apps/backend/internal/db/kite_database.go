package db

import (
	"fmt"
	"strings"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/logger"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// KiteDatabase Kite 数据库连接包装器
type KiteDatabase struct {
	*gorm.DB
}

// KiteCluster Kite 集群模型 (与 Kite 的 Cluster 表结构兼容)
type KiteCluster struct {
	ID            uint   `gorm:"primarykey" json:"id"`
	Name          string `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Description   string `json:"description" gorm:"type:text"`
	Config        string `json:"config" gorm:"type:text"` // KubeConfig
	PrometheusURL string `json:"prometheus_url" gorm:"type:varchar(255)"`
	InCluster     bool   `json:"in_cluster" gorm:"type:boolean;default:false"`
	IsDefault     bool   `json:"is_default" gorm:"type:boolean;default:false"`
	Enable        bool   `json:"enable" gorm:"type:boolean;default:true"`
}

// TableName 指定表名为 clusters
func (KiteCluster) TableName() string {
	return "clusters"
}

// InitializeKiteDB 初始化 Kite 数据库连接
func InitializeKiteDB(cfg config.KiteConfig) (*KiteDatabase, error) {
	if !cfg.Enabled || cfg.DatabaseURL == "" {
		return nil, nil
	}

	gormConfig := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	var db *gorm.DB
	var err error

	switch cfg.DBType {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DatabaseURL), gormConfig)
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	case "mysql":
		fallthrough
	default:
		dsn := strings.TrimPrefix(cfg.DatabaseURL, "mysql://")
		if !strings.Contains(dsn, "parseTime=") {
			separator := "?"
			if strings.Contains(dsn, "?") {
				separator = "&"
			}
			dsn = dsn + separator + "parseTime=true"
		}
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("连接 Kite 数据库失败: %w", err)
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 Kite 数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("Kite 数据库连接测试失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)

	logger.S().Infow("Kite 数据库连接成功")

	return &KiteDatabase{db}, nil
}

// Close 关闭 Kite 数据库连接
func (d *KiteDatabase) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateCluster 在 Kite 数据库中创建集群
func (d *KiteDatabase) CreateCluster(cluster *KiteCluster) error {
	return d.DB.Create(cluster).Error
}

// UpdateClusterByName 在 Kite 数据库中按名称更新集群
func (d *KiteDatabase) UpdateClusterByName(name string, updates map[string]interface{}) error {
	return d.DB.Model(&KiteCluster{}).Where("name = ?", name).Updates(updates).Error
}

// DeleteClusterByName 在 Kite 数据库中按名称删除集群
func (d *KiteDatabase) DeleteClusterByName(name string) error {
	return d.DB.Where("name = ?", name).Delete(&KiteCluster{}).Error
}

// GetClusterByName 在 Kite 数据库中按名称获取集群
func (d *KiteDatabase) GetClusterByName(name string) (*KiteCluster, error) {
	var cluster KiteCluster
	if err := d.DB.Where("name = ?", name).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}
