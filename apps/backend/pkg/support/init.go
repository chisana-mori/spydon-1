package support

import (
	"fmt"

	"robusta-web/backend/pkg/dlink"
	"robusta-web/backend/pkg/dragonfly"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/mailer"
	"robusta-web/backend/pkg/narwhal"
	"robusta-web/backend/pkg/redis"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ExternalDependencies 外部依赖配置（从 config.yaml 读取）
type ExternalDependencies struct {
	Dlink     DlinkConfig     `mapstructure:"dlink" yaml:"dlink"`
	Narwhal   NarwhalConfig   `mapstructure:"narwhal" yaml:"narwhal"`
	Dragonfly DragonflyConfig `mapstructure:"dragonfly" yaml:"dragonfly"`
	Email     EmailConfig     `mapstructure:"email" yaml:"email"`
	Redis     RedisConfig     `mapstructure:"redis" yaml:"redis"`
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
	Host     string `mapstructure:"host" yaml:"host"`
	Port     int    `mapstructure:"port" yaml:"port"`
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
	From     string `mapstructure:"from" yaml:"from"`
	FromName string `mapstructure:"from_name" yaml:"from_name"`
	Cc       string `mapstructure:"cc" yaml:"cc"`
	UseTLS   bool   `mapstructure:"use_tls" yaml:"use_tls"`
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

// Init 从 viper 实例读取配置并初始化所有外部依赖
// 此方法应在 config.Load() 之后调用
func Init(v *viper.Viper) (*InitResult, error) {
	deps := &ExternalDependencies{}

	// 读取外部依赖配置
	if err := v.UnmarshalKey("external_dependencies", deps); err != nil {
		return nil, fmt.Errorf("解析外部依赖配置失败: %w", err)
	}

	externalDeps = deps
	result := &InitResult{}

	// 初始化 Dlink
	if deps.Dlink.Host != "" {
		dlink.Init(dlink.Config{
			Host:     deps.Dlink.Host,
			Token:    deps.Dlink.Token,
			TimeOut:  deps.Dlink.Timeout,
			UseProxy: deps.Dlink.UseProxy,
			Proxy:    deps.Dlink.Proxy,
			IsDebug:  deps.Dlink.IsDebug,
		})
		logger.L().Info("Dlink 初始化成功", zap.String("host", deps.Dlink.Host))
	}

	// 初始化 Narwhal
	if deps.Narwhal.Host != "" {
		narwhal.Init(narwhal.Config{
			Host:  deps.Narwhal.Host,
			Token: deps.Narwhal.Token,
		})
		logger.L().Info("Narwhal 初始化成功", zap.String("host", deps.Narwhal.Host))
	}

	// 初始化 Dragonfly
	if deps.Dragonfly.Host != "" {
		dragonfly.Init(dragonfly.Config{
			Host:     deps.Dragonfly.Host,
			Token:    deps.Dragonfly.Token,
			TimeOut:  deps.Dragonfly.Timeout,
			UseProxy: deps.Dragonfly.UseProxy,
			Proxy:    deps.Dragonfly.Proxy,
			IsDebug:  deps.Dragonfly.IsDebug,
		})
		logger.L().Info("Dragonfly 初始化成功", zap.String("host", deps.Dragonfly.Host))
	}

	// 初始化 Mailer
	if deps.Email.Host != "" {
		m := mailer.New(mailer.Config{
			Host:     deps.Email.Host,
			Port:     deps.Email.Port,
			Username: deps.Email.Username,
			Password: deps.Email.Password,
			From:     deps.Email.From,
			FromName: deps.Email.FromName,
			UseTLS:   deps.Email.UseTLS,
		})
		result.Mailer = m
		logger.L().Info("Mailer 初始化成功", zap.String("host", deps.Email.Host))
	}

	// 初始化 Redis
	if deps.Redis.Enabled && deps.Redis.URL != "" {
		factory := redis.GetFactory()
		if err := factory.Setup(deps.Redis.URL, deps.Redis.PoolSize); err != nil {
			logger.L().Warn("Redis 初始化失败", zap.Error(err))
		} else {
			client, _ := factory.GetClient()
			result.Redis = client
			logger.L().Info("Redis 初始化成功", zap.String("url", maskRedisURL(deps.Redis.URL)))
		}
	}

	initResult = result
	return result, nil
}

// InitFromConfig 从配置文件直接初始化（简化入口）
func InitFromConfig(configPath string) (*InitResult, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	return Init(v)
}

// GetExternalDeps 获取外部依赖配置
func GetExternalDeps() *ExternalDependencies {
	return externalDeps
}

// GetInitResult 获取初始化结果
func GetInitResult() *InitResult {
	return initResult
}

// Close 关闭所有外部依赖连接
func Close() {
	if redis.GetFactory().IsSetup() {
		if err := redis.GetFactory().Close(); err != nil {
			logger.L().Warn("关闭 Redis 连接失败", zap.Error(err))
		}
	}
	logger.L().Info("外部依赖资源已释放")
}

// maskRedisURL 隐藏 Redis URL 中的密码
func maskRedisURL(url string) string {
	// redis://:password@host:port -> redis://***@host:port
	if len(url) > 8 && url[8] == ':' {
		atIdx := -1
		for i := 8; i < len(url); i++ {
			if url[i] == '@' {
				atIdx = i
				break
			}
		}
		if atIdx > 8 {
			return url[:8] + "***" + url[atIdx:]
		}
	}
	return url
}
