package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置结构
type Config struct {
	// 基础配置
	Environment string
	Port        string

	// 数据库配置
	DatabaseURL string

	// 安全配置
	JWTSecret  string
	HMACSecret string

	// 入站Webhook API Key（用于webhook_sink）
	IngestAPIKey string

	// OIDC配置
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string

	// HolmesGPT配置
	HolmesGPT struct {
		URL            string `json:"url"`
		APIKey         string `json:"api_key"`
		TimeoutSeconds int    `json:"timeout_seconds"`
		Enabled        bool   `json:"enabled"`
		DefaultDepth   string `json:"default_depth"`
		Model          string `json:"model"`
		ProxyAuthToken string `json:"proxy_auth_token"`
	}

	// CAS配置
	CAS struct {
		Enabled            bool   `json:"enabled"`
		ServerURL          string `json:"server_url"`
		CallbackPath       string `json:"callback_path"`
		RedirectURL        string `json:"redirect_url"`
		DefaultEmailDomain string `json:"default_email_domain"`
		EmailAttribute     string `json:"email_attribute"`
		NameAttribute      string `json:"name_attribute"`
		RolesAttribute     string `json:"roles_attribute"`
	}

	// MinIO配置
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucketName string
	MinIOUseSSL     bool

	// 限流配置
	RateLimitRPS int

	// 日志配置
	LogLevel string

	// 邮件配置（用于发送RCA结果）
	Email struct {
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
		SMTPUser string `json:"smtp_user"`
		SMTPPass string `json:"smtp_pass"`
		From     string `json:"from"`
		RCATo    string `json:"rca_to"`
		Enabled  bool   `json:"enabled"`
	}
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Port:        getEnv("PORT", "8080"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/robusta_hub?sslmode=disable"),

		JWTSecret:    getEnv("JWT_SECRET", "your-jwt-secret-key"),
		HMACSecret:   getEnv("HMAC_SECRET", "your-hmac-secret-key"),
		IngestAPIKey: getEnv("INGEST_API_KEY", ""),

		OIDCIssuer:       getEnv("OIDC_ISSUER", ""),
		OIDCClientID:     getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),

		HolmesGPT: struct {
			URL            string `json:"url"`
			APIKey         string `json:"api_key"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
			DefaultDepth   string `json:"default_depth"`
			Model          string `json:"model"`
			ProxyAuthToken string `json:"proxy_auth_token"`
		}{
			URL:            getEnv("HOLMES_GPT_URL", "http://localhost:8081"),
			APIKey:         getEnv("HOLMES_GPT_API_KEY", ""),
			TimeoutSeconds: getIntEnv("HOLMES_GPT_TIMEOUT", 300),
			Enabled:        getBoolEnv("HOLMES_GPT_ENABLED", true),
			DefaultDepth:   getEnv("HOLMES_GPT_DEFAULT_DEPTH", "standard"),
			Model:          getEnv("HOLMES_GPT_MODEL", "deepseek-reasoner"),
			ProxyAuthToken: getEnv("HOLMES_GPT_PROXY_TOKEN", ""),
		},

		CAS: struct {
			Enabled            bool   `json:"enabled"`
			ServerURL          string `json:"server_url"`
			CallbackPath       string `json:"callback_path"`
			RedirectURL        string `json:"redirect_url"`
			DefaultEmailDomain string `json:"default_email_domain"`
			EmailAttribute     string `json:"email_attribute"`
			NameAttribute      string `json:"name_attribute"`
			RolesAttribute     string `json:"roles_attribute"`
		}{
			Enabled:            getBoolEnv("CAS_ENABLED", false),
			ServerURL:          getEnv("CAS_SERVER_URL", ""),
			CallbackPath:       getEnv("CAS_CALLBACK_PATH", "/auth/cas/callback"),
			RedirectURL:        getEnv("CAS_REDIRECT_URL", "http://localhost:3000"),
			DefaultEmailDomain: getEnv("CAS_DEFAULT_EMAIL_DOMAIN", "cas.local"),
			EmailAttribute:     getEnv("CAS_EMAIL_ATTRIBUTE", "mail"),
			NameAttribute:      getEnv("CAS_NAME_ATTRIBUTE", "displayName"),
			RolesAttribute:     getEnv("CAS_ROLES_ATTRIBUTE", "roles"),
		},

		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucketName: getEnv("MINIO_BUCKET_NAME", "robusta-artifacts"),
		MinIOUseSSL:     getBoolEnv("MINIO_USE_SSL", false),

		RateLimitRPS: getIntEnv("RATE_LIMIT_RPS", 100),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		Email: struct {
			SMTPHost string `json:"smtp_host"`
			SMTPPort int    `json:"smtp_port"`
			SMTPUser string `json:"smtp_user"`
			SMTPPass string `json:"smtp_pass"`
			From     string `json:"from"`
			RCATo    string `json:"rca_to"`
			Enabled  bool   `json:"enabled"`
		}{
			SMTPHost: getEnv("EMAIL_SMTP_HOST", ""),
			SMTPPort: getIntEnv("EMAIL_SMTP_PORT", 587),
			SMTPUser: getEnv("EMAIL_SMTP_USER", ""),
			SMTPPass: getEnv("EMAIL_SMTP_PASS", ""),
			From:     getEnv("EMAIL_FROM", ""),
			RCATo:    getEnv("RCA_EMAIL_TO", ""),
			Enabled:  getBoolEnv("EMAIL_ENABLED", false),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv 获取整数类型环境变量
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getBoolEnv 获取布尔类型环境变量
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getDurationEnv 获取时间间隔类型环境变量
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
