package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PrometheusRuntime Prometheus 指标运行时实现
// 实现 MetricsRuntime 接口
type PrometheusRuntime struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewPrometheusRuntime 创建 Prometheus 运行时
func NewPrometheusRuntime(logger *zap.Logger) *PrometheusRuntime {
	return &PrometheusRuntime{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// NewPrometheusRuntimeWithClient 使用自定义 HTTP Client 创建 Prometheus 运行时
func NewPrometheusRuntimeWithClient(client *http.Client, logger *zap.Logger) *PrometheusRuntime {
	return &PrometheusRuntime{
		httpClient: client,
		logger:     logger,
	}
}

// Name 返回运行时名称
func (r *PrometheusRuntime) Name() string {
	return "prometheus"
}

// Query 执行 PromQL 查询，返回第一个结果的数值
func (r *PrometheusRuntime) Query(ctx context.Context, endpoint string, query string) (float64, error) {
	rawResult, err := r.QueryRaw(ctx, endpoint, query)
	if err != nil {
		return 0, err
	}

	// 解析结果
	respData, ok := rawResult.(*PromQueryResponse)
	if !ok {
		return 0, fmt.Errorf("unexpected response type")
	}

	if len(respData.Data.Result) == 0 {
		return 0, fmt.Errorf("no data returned from query")
	}

	// 取第一个结果的值
	valInterface := respData.Data.Result[0].Value
	if len(valInterface) < 2 {
		return 0, fmt.Errorf("invalid data format")
	}

	valStr, ok := valInterface[1].(string)
	if !ok {
		return 0, fmt.Errorf("value is not a string")
	}

	return strconv.ParseFloat(valStr, 64)
}

// QueryRaw 执行 PromQL 查询，返回原始响应
func (r *PrometheusRuntime) QueryRaw(ctx context.Context, endpoint string, query string) (interface{}, error) {
	apiURL := fmt.Sprintf("%s/api/v1/query", strings.TrimRight(endpoint, "/"))
	params := url.Values{}
	params.Add("query", query)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?%s", apiURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Prometheus 查询错误: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus 返回错误(%d): %s", resp.StatusCode, string(body))
	}

	var promResp PromQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("解析 Prometheus 响应失败: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("Prometheus 查询失败: %s", promResp.ErrorType)
	}

	return &promResp, nil
}

// Ensure PrometheusRuntime implements MetricsRuntime
var _ MetricsRuntime = (*PrometheusRuntime)(nil)
