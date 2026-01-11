package support

import (
	"fmt"
	"strings"

	"robusta-web/backend/pkg/dlink"
	"robusta-web/backend/pkg/dragonfly"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/mailer"
	"robusta-web/backend/pkg/narwhal"
	"robusta-web/backend/pkg/orchid"
	"robusta-web/backend/pkg/redis"
	"robusta-web/backend/pkg/wayne_api"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Configuration keys
const (
	configKeyExternalDeps = "external_dependencies"
	redisSchemePrefix     = "redis://"
	passwordMask          = "***"
	logFieldHost          = "host"
	logFieldURL           = "url"
)

// ExternalDependencies 外部依赖配置（从 config.yaml 读取）
type ExternalDependencies struct {
	Dlink     DlinkConfig     `mapstructure:"dlink" yaml:"dlink"`
	Narwhal   NarwhalConfig   `mapstructure:"narwhal" yaml:"narwhal"`
	Dragonfly DragonflyConfig `mapstructure:"dragonfly" yaml:"dragonfly"`
	Email     EmailConfig     `mapstructure:"email" yaml:"email"`
	Redis     RedisConfig     `mapstructure:"redis" yaml:"redis"`
	Wayne     WayneConfig     `mapstructure:"wayne" yaml:"wayne"`
	Orchid    OrchidConfig    `mapstructure:"orchid" yaml:"orchid"`
}

// WayneConfig Wayne 配置
type WayneConfig struct {
	Host  string `mapstructure:"host" yaml:"host"`
	Token string `mapstructure:"token" yaml:"token"`
}

// OrchidConfig Orchid 配置
type OrchidConfig struct {
	Host  string `mapstructure:"host" yaml:"host"`
	Token string `mapstructure:"token" yaml:"token"`
}

// DlinkConfig Dlink 配置
type DlinkConfig struct {
	Host     string `mapstructure:"host" yaml:"host"`
	Token    string `mapstructure:"token" yaml:"token"`
	Timeout  int64  `mapstructure:"timeout" yaml:"timeout"`
	UseProxy bool   `mapstructure:"use_proxy" yaml:"use_proxy"`
	Proxy    string `mapstructure:"proxy" yaml:"proxy"`
	IsDebug  bool   `mapstructure:"is_debug" yaml:"is_debug"`
}

// NarwhalConfig Narwhal 配置
type NarwhalConfig struct {
	Host  string `mapstructure:"host" yaml:"host"`
	Token string `mapstructure:"token" yaml:"token"`
}



// DragonflyConfig Dragonfly 变更管理配置
type DragonflyConfig struct {
	Host     string `mapstructure:"host" yaml:"host"`
	Token    string `mapstructure:"token" yaml:"token"`
	Timeout  int64  `mapstructure:"timeout" yaml:"timeout"`
	UseProxy bool   `mapstructure:"use_proxy" yaml:"use_proxy"`
	Proxy    string `mapstructure:"proxy" yaml:"proxy"`
	IsDebug  bool   `mapstructure:"is_debug" yaml:"is_debug"`
}

// EmailConfig 邮件服务配置
type EmailConfig struct {
	Host   string `mapstructure:"host" yaml:"host"`
	Port   int    `mapstructure:"port" yaml:"port"`
	From   string `mapstructure:"from" yaml:"from"`
	Cc     string `mapstructure:"cc" yaml:"cc"`
	IsSSL  bool   `mapstructure:"is-ssl" yaml:"is-ssl"`
	Secret string `mapstructure:"secret" yaml:"secret"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	URL      string `mapstructure:"url" yaml:"url"`
	PoolSize int    `mapstructure:"pool_size" yaml:"pool_size"`
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
}

// InitResult 初始化结果
type InitResult struct {
	Mailer *mailer.Mailer
	Redis  redis.Client
}

var (
	externalDeps *ExternalDependencies
	initResult   *InitResult
)

// Init 从 viper 实例读取配置并初始化所有外部依赖。
// 此方法应在 config.Load() 之后调用。
func Init(v *viper.Viper) (*InitResult, error) {
	deps, err := loadExternalDeps(v)
	if err != nil {
		return nil, err
	}

	externalDeps = deps
	result := &InitResult{}

	// 按依赖顺序初始化各个外部依赖
	initDlinkService(deps.Dlink)
	initNarwhalService(deps.Narwhal)
	initDragonflyService(deps.Dragonfly)
	initMailerService(deps.Email, result)
	initRedisService(deps.Redis, result)
	initWayneService(deps.Wayne)
	initOrchidService(deps.Orchid)

	initResult = result
	return result, nil
}

// loadExternalDeps 从 viper 加载外部依赖配置。
func loadExternalDeps(v *viper.Viper) (*ExternalDependencies, error) {
	deps := &ExternalDependencies{}
	if err := v.UnmarshalKey(configKeyExternalDeps, deps); err != nil {
		return nil, fmt.Errorf("解析外部依赖配置失败: %w", err)
	}
	return deps, nil
}

// initDlinkService 初始化 Dlink 服务。
func initDlinkService(cfg DlinkConfig) {
	if cfg.Host == "" {
		return
	}

	dlink.Init(dlink.Config{
		Host:     cfg.Host,
		Token:    cfg.Token,
		TimeOut:  cfg.Timeout,
		UseProxy: cfg.UseProxy,
		Proxy:    cfg.Proxy,
		IsDebug:  cfg.IsDebug,
	})
	logServiceInitialized("Dlink", cfg.Host)
}

// initNarwhalService 初始化 Narwhal 服务。
func initNarwhalService(cfg NarwhalConfig) {
	if cfg.Host == "" {
		return
	}

	narwhal.Init(narwhal.Config{
		Host:  cfg.Host,
		Token: cfg.Token,
	})
	logServiceInitialized("Narwhal", cfg.Host)
}

// initDragonflyService 初始化 Dragonfly 服务。
func initDragonflyService(cfg DragonflyConfig) {
	if cfg.Host == "" {
		return
	}

	dragonfly.Init(dragonfly.Config{
		Host:     cfg.Host,
		Token:    cfg.Token,
		TimeOut:  cfg.Timeout,
		UseProxy: cfg.UseProxy,
		Proxy:    cfg.Proxy,
		IsDebug:  cfg.IsDebug,
	})
	logServiceInitialized("Dragonfly", cfg.Host)
}

// initWayneService 初始化 Wayne 服务。
func initWayneService(cfg WayneConfig) {
	if cfg.Host == "" {
		return
	}

	wayne_api.Init(wayne_api.Config{
		Host:  cfg.Host,
		Token: cfg.Token,
	})
	logServiceInitialized("Wayne", cfg.Host)
}

// initOrchidService 初始化 Orchid 服务。
func initOrchidService(cfg OrchidConfig) {
	if cfg.Host == "" {
		return
	}

	orchid.Init(orchid.Config{
		Host:  cfg.Host,
		Token: cfg.Token,
	})
	logServiceInitialized("Orchid", cfg.Host)
}

// initMailerService 初始化邮件服务。
func initMailerService(cfg EmailConfig, result *InitResult) {
	if cfg.Host == "" {
		return
	}

	m := mailer.New(mailer.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.From,
		Password: cfg.Secret,
		From:     cfg.From,
		FromName: "",
		UseTLS:   cfg.IsSSL,
	})
	result.Mailer = m
	logServiceInitialized("Mailer", cfg.Host)
}

// initRedisService 初始化 Redis 服务。
func initRedisService(cfg RedisConfig, result *InitResult) {
	if !cfg.Enabled || cfg.URL == "" {
		return
	}

	factory := redis.GetFactory()
	if err := factory.Setup(cfg.URL, cfg.PoolSize); err != nil {
		logger.L().Warn("Redis 初始化失败", zap.Error(err))
		return
	}

	client, _ := factory.GetClient()
	result.Redis = client
	logger.L().Info("Redis 初始化成功",
		zap.String(logFieldURL, maskRedisURL(cfg.URL)))
}

// logServiceInitialized 记录服务初始化成功日志。
func logServiceInitialized(serviceName, host string) {
	logger.L().Info(serviceName+" 初始化成功", zap.String(logFieldHost, host))
}

// InitFromConfig 从配置文件直接初始化（简化入口）。
func InitFromConfig(configPath string) (*InitResult, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	return Init(v)
}

// GetExternalDeps 获取外部依赖配置。
func GetExternalDeps() *ExternalDependencies {
	return externalDeps
}

// GetInitResult 获取初始化结果。
func GetInitResult() *InitResult {
	return initResult
}

// Close 关闭所有外部依赖连接。
func Close() {
	factory := redis.GetFactory()
	if factory.IsSetup() {
		if err := factory.Close(); err != nil {
			logger.L().Warn("关闭 Redis 连接失败", zap.Error(err))
		}
	}
	logger.L().Info("外部依赖资源已释放")
}

// maskRedisURL 隐藏 Redis URL 中的密码。
// 示例：redis://:password@host:port -> redis://***@host:port
func maskRedisURL(url string) string {
	if !strings.HasPrefix(url, redisSchemePrefix) {
		return url
	}

	// 查找 @ 符号位置（密码结束标记）
	atIdx := strings.Index(url, "@")
	if atIdx <= len(redisSchemePrefix) {
		return url
	}

	// 替换密码部分
	return redisSchemePrefix + passwordMask + url[atIdx:]
}
