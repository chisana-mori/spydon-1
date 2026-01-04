package services

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/pkg/awx"
)

// NewAWXClientFromConfig 从应用配置创建AWX客户端
func NewAWXClientFromConfig(cfg *config.Config) *awx.Client {
	awxCfg := awx.Config{
		URL:      cfg.AWX.URL,
		Username: cfg.AWX.Username,
		Password: cfg.AWX.Password,
		Token:    cfg.AWX.Token,
		Timeout:  cfg.AWX.Timeout,
		Insecure: cfg.AWX.Insecure,
	}

	return awx.NewClient(awxCfg, logger.L())
}

// NewAWXRuntimeFromConfig 从应用配置创建AWX运行时
func NewAWXRuntimeFromConfig(cfg *config.Config) *AWXRuntime {
	client := NewAWXClientFromConfig(cfg)
	return NewAWXRuntime(client, logger.L())
}

// NewPrometheusRuntimeFromConfig 创建 Prometheus 运行时
func NewPrometheusRuntimeFromConfig() *PrometheusRuntime {
	return NewPrometheusRuntime(logger.L())
}
