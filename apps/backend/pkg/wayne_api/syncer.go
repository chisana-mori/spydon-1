package wayne_api

import (
	"fmt"
	"net/http"
	"robusta-web/backend/pkg/logger"

	"github.com/imroc/req"
	"go.uber.org/zap"
)

// Config Wayne API 配置
type Config struct {
	Host  string
	Token string
}

var defaultClient *WayneSyncer

// Init 初始化全局 Wayne 客户端
func Init(cfg Config) {
	defaultClient = &WayneSyncer{
		host:  cfg.Host,
		token: cfg.Token,
	}
}

// GetClient 获取全局 Wayne 客户端
func GetClient() *WayneSyncer {
	return defaultClient
}

// IsConfigured 检查 Wayne 是否已配置
func IsConfigured() bool {
	return defaultClient != nil && defaultClient.host != ""
}

// WayneSyncer Wayne API 客户端
type WayneSyncer struct {
	host  string
	token string
}

// NewWayneSyncer 创建新的 WayneSyncer 实例（兼容旧代码）
func NewWayneSyncer() *WayneSyncer {
	if defaultClient != nil {
		return defaultClient
	}
	return &WayneSyncer{}
}

// SyncIPPool 同步 IPPool 到 Wayne
func (agent *WayneSyncer) SyncIPPool(body []byte) (resp []byte, err error) {
	if agent.host == "" {
		return nil, fmt.Errorf("Wayne Host 未配置")
	}

	url := fmt.Sprintf("%s/api/ippools/sync", agent.host)
	resp, err = agent.Request(url, http.MethodPost, body)
	if err != nil {
		logger.L().Error("同步wayne ippool失败", zap.Error(err))
	}
	return
}

// DefaultHeader 返回默认请求头
func (agent *WayneSyncer) DefaultHeader() req.Header {
	headers := req.Header{
		"access-token": agent.token,
		"Content-type": "application/json",
	}
	return headers
}

// Request 发送 HTTP 请求
func (agent *WayneSyncer) Request(url, method string, body interface{}) (response []byte, err error) {
	request := req.New()
	var resp *req.Resp
	resp, err = request.Do(method, url, body, agent.DefaultHeader())
	if err != nil {
		logger.L().Error("请求Wayne失败", zap.Error(err))
		return
	}
	defer resp.Response().Body.Close()
	response = resp.Bytes()
	return
}
