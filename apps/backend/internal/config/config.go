package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// HolmesGPTConfig HolmesGPT 相关配置
type HolmesGPTConfig struct {
	URL            string `mapstructure:"url" json:"url" yaml:"url"`
	APIKey         string `mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds" json:"timeout_seconds" yaml:"timeout_seconds"`
	Enabled        bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	DefaultDepth   string `mapstructure:"default_depth" json:"default_depth" yaml:"default_depth"`
	Model          string `mapstructure:"model" json:"model" yaml:"model"`
	ProxyAuthToken string `mapstructure:"proxy_auth_token" json:"proxy_auth_token" yaml:"proxy_auth_token"`
}

// CASConfig CAS 单点登录配置
type CASConfig struct {
	Enabled            bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	ServerURL          string `mapstructure:"server_url" json:"server_url" yaml:"server_url"`
	CallbackPath       string `mapstructure:"callback_path" json:"callback_path" yaml:"callback_path"`
	RedirectURL        string `mapstructure:"redirect_url" json:"redirect_url" yaml:"redirect_url"`
	DefaultEmailDomain string `mapstructure:"default_email_domain" json:"default_email_domain" yaml:"default_email_domain"`
	EmailAttribute     string `mapstructure:"email_attribute" json:"email_attribute" yaml:"email_attribute"`
	NameAttribute      string `mapstructure:"name_attribute" json:"name_attribute" yaml:"name_attribute"`
	RolesAttribute     string `mapstructure:"roles_attribute" json:"roles_attribute" yaml:"roles_attribute"`
}

// EmailConfig SMTP 邮件配置
type EmailConfig struct {
	SMTPHost string `mapstructure:"smtp_host" json:"smtp_host" yaml:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port" json:"smtp_port" yaml:"smtp_port"`
	SMTPUser string `mapstructure:"smtp_user" json:"smtp_user" yaml:"smtp_user"`
	SMTPPass string `mapstructure:"smtp_pass" json:"smtp_pass" yaml:"smtp_pass"`
	From     string `mapstructure:"from" json:"from" yaml:"from"`
	RCATo    string `mapstructure:"rca_to" json:"rca_to" yaml:"rca_to"`
	Enabled  bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// MinIOConfig MinIO 存储配置
type MinIOConfig struct {
	Endpoint   string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	AccessKey  string `mapstructure:"access_key" json:"access_key" yaml:"access_key"`
	SecretKey  string `mapstructure:"secret_key" json:"secret_key" yaml:"secret_key"`
	BucketName string `mapstructure:"bucket_name" json:"bucket_name" yaml:"bucket_name"`
	UseSSL     bool   `mapstructure:"use_ssl" json:"use_ssl" yaml:"use_ssl"`
}

// Config 应用配置结构
type Config struct {
	Environment string `mapstructure:"environment" json:"environment" yaml:"environment"`
	Port        string `mapstructure:"port" json:"port" yaml:"port"`

	DatabaseURL string `mapstructure:"database_url" json:"database_url" yaml:"database_url"`

	JWTSecret    string `mapstructure:"jwt_secret" json:"jwt_secret" yaml:"jwt_secret"`
	HMACSecret   string `mapstructure:"hmac_secret" json:"hmac_secret" yaml:"hmac_secret"`
	IngestAPIKey string `mapstructure:"ingest_api_key" json:"ingest_api_key" yaml:"ingest_api_key"`

	OIDCIssuer       string `mapstructure:"oidc_issuer" json:"oidc_issuer" yaml:"oidc_issuer"`
	OIDCClientID     string `mapstructure:"oidc_client_id" json:"oidc_client_id" yaml:"oidc_client_id"`
	OIDCClientSecret string `mapstructure:"oidc_client_secret" json:"oidc_client_secret" yaml:"oidc_client_secret"`

	HolmesGPT HolmesGPTConfig `mapstructure:"holmes_gpt" json:"holmes_gpt" yaml:"holmes_gpt"`
	CAS       CASConfig       `mapstructure:"cas" json:"cas" yaml:"cas"`
	Email     EmailConfig     `mapstructure:"email" json:"email" yaml:"email"`
	MinIO     MinIOConfig     `mapstructure:"minio" json:"minio" yaml:"minio"`

	RateLimitRPS int    `mapstructure:"rate_limit_rps" json:"rate_limit_rps" yaml:"rate_limit_rps"`
	LogLevel     string `mapstructure:"log_level" json:"log_level" yaml:"log_level"`
}

// Load 从 YAML 文件及环境变量加载配置
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		if info, err := os.Stat(configFile); err == nil && info.IsDir() {
			v.AddConfigPath(configFile)
			v.SetConfigName("config")
			v.SetConfigType("yaml")
		} else {
			if err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("无法访问配置路径 %s: %w", configFile, err)
			}
			v.SetConfigFile(configFile)
		}
	} else {
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("environment", "development")
	v.SetDefault("port", "8080")

	v.SetDefault("database_url", "postgres://postgres:password@localhost:5432/robusta_hub?sslmode=disable")

	v.SetDefault("jwt_secret", "your-jwt-secret-key")
	v.SetDefault("hmac_secret", "your-hmac-secret-key")
	v.SetDefault("ingest_api_key", "")

	v.SetDefault("oidc_issuer", "")
	v.SetDefault("oidc_client_id", "")
	v.SetDefault("oidc_client_secret", "")

	v.SetDefault("holmes_gpt.url", "http://localhost:8081")
	v.SetDefault("holmes_gpt.api_key", "")
	v.SetDefault("holmes_gpt.timeout_seconds", 300)
	v.SetDefault("holmes_gpt.enabled", true)
	v.SetDefault("holmes_gpt.default_depth", "standard")
	v.SetDefault("holmes_gpt.model", "deepseek-reasoner")
	v.SetDefault("holmes_gpt.proxy_auth_token", "")

	v.SetDefault("cas.enabled", false)
	v.SetDefault("cas.server_url", "")
	v.SetDefault("cas.callback_path", "/auth/cas/callback")
	v.SetDefault("cas.redirect_url", "http://localhost:3000")
	v.SetDefault("cas.default_email_domain", "cas.local")
	v.SetDefault("cas.email_attribute", "mail")
	v.SetDefault("cas.name_attribute", "displayName")
	v.SetDefault("cas.roles_attribute", "roles")

	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.access_key", "minioadmin")
	v.SetDefault("minio.secret_key", "minioadmin")
	v.SetDefault("minio.bucket_name", "robusta-artifacts")
	v.SetDefault("minio.use_ssl", false)

	v.SetDefault("rate_limit_rps", 100)
	v.SetDefault("log_level", "info")

	v.SetDefault("email.smtp_host", "")
	v.SetDefault("email.smtp_port", 587)
	v.SetDefault("email.smtp_user", "")
	v.SetDefault("email.smtp_pass", "")
	v.SetDefault("email.from", "")
	v.SetDefault("email.rca_to", "")
	v.SetDefault("email.enabled", false)
}
