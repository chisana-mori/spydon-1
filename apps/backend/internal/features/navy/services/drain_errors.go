package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ==================== 错误类型定义 ====================

// DrainErrorCode 错误代码
type DrainErrorCode string

const (
	// 集群连接错误
	ErrorCodeClusterConnection DrainErrorCode = "CLUSTER_CONNECTION"
	ErrorCodeClusterTimeout    DrainErrorCode = "CLUSTER_TIMEOUT"
	ErrorCodeClusterAuth       DrainErrorCode = "CLUSTER_AUTH"

	// 节点相关错误
	ErrorCodeNodeNotFound DrainErrorCode = "NODE_NOT_FOUND"
	ErrorCodeNodeNotReady DrainErrorCode = "NODE_NOT_READY"
	ErrorCodeNodeCordon   DrainErrorCode = "NODE_CORDON"

	// Pod相关错误
	ErrorCodePodEviction  DrainErrorCode = "POD_EVICTION"
	ErrorCodePodTimeout   DrainErrorCode = "POD_TIMEOUT"
	ErrorCodePodStuck     DrainErrorCode = "POD_STUCK"
	ErrorCodePodDaemonSet DrainErrorCode = "POD_DAEMONSET"

	// PDB相关错误
	ErrorCodePDBViolation DrainErrorCode = "PDB_VIOLATION"
	ErrorCodePDBCreation  DrainErrorCode = "PDB_CREATION"
	ErrorCodePDBCleanup   DrainErrorCode = "PDB_CLEANUP"

	// 资源相关错误
	ErrorCodeResourceQuota        DrainErrorCode = "RESOURCE_QUOTA"
	ErrorCodeResourceInsufficient DrainErrorCode = "RESOURCE_INSUFFICIENT"
	ErrorCodeResourceConflict     DrainErrorCode = "RESOURCE_CONFLICT"

	// 权限相关错误
	ErrorCodePermissionDenied DrainErrorCode = "PERMISSION_DENIED"
	ErrorCodeRBACViolation    DrainErrorCode = "RBAC_VIOLATION"

	// 配置相关错误
	ErrorCodeConfiguration DrainErrorCode = "CONFIGURATION"
	ErrorCodeValidation    DrainErrorCode = "VALIDATION"

	// 系统相关错误
	ErrorCodeInternal DrainErrorCode = "INTERNAL"
	ErrorCodeTimeout  DrainErrorCode = "TIMEOUT"
	ErrorCodeCanceled DrainErrorCode = "CANCELED"
	ErrorCodeUnknown  DrainErrorCode = "UNKNOWN"
)

// DrainErrorSeverity 错误严重程度
type DrainErrorSeverity string

const (
	SeverityLow      DrainErrorSeverity = "LOW"
	SeverityMedium   DrainErrorSeverity = "MEDIUM"
	SeverityHigh     DrainErrorSeverity = "HIGH"
	SeverityCritical DrainErrorSeverity = "CRITICAL"
)

// DrainError 结构化错误类型
type DrainError struct {
	Code        DrainErrorCode     `json:"code"`
	Message     string             `json:"message"`
	Cause       error              `json:"-"`
	Retryable   bool               `json:"retryable"`
	Severity    DrainErrorSeverity `json:"severity"`
	Timestamp   time.Time          `json:"timestamp"`
	Context     map[string]string  `json:"context,omitempty"`
	Suggestions []string           `json:"suggestions,omitempty"`
}

func (e *DrainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DrainError) Unwrap() error {
	return e.Cause
}

// WithContext 添加上下文信息
func (e *DrainError) WithContext(key, value string) *DrainError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithSuggestion 添加解决建议
func (e *DrainError) WithSuggestion(suggestion string) *DrainError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// NewDrainError 创建新的drain错误
func NewDrainError(code DrainErrorCode, message string, cause error) *DrainError {
	return &DrainError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
		Context:   make(map[string]string),
	}
}

// ==================== 错误分类器 ====================

// ErrorClassifier 错误分类器接口
type ErrorClassifier interface {
	Classify(err error) *DrainError
}

// DefaultErrorClassifier 默认错误分类器
type DefaultErrorClassifier struct{}

func NewDefaultErrorClassifier() *DefaultErrorClassifier {
	return &DefaultErrorClassifier{}
}

// Classify 分类错误
func (c *DefaultErrorClassifier) Classify(err error) *DrainError {
	if err == nil {
		return nil
	}

	// 如果已经是DrainError，直接返回
	var derr *DrainError
	if errors.As(err, &derr) {
		return derr
	}

	// 分类K8s API错误
	if k8sErr := c.classifyK8sError(err); k8sErr != nil {
		return k8sErr
	}

	// 分类网络错误
	if netErr := c.classifyNetworkError(err); netErr != nil {
		return netErr
	}

	// 分类超时错误
	if timeoutErr := c.classifyTimeoutError(err); timeoutErr != nil {
		return timeoutErr
	}

	// 分类权限错误
	if permErr := c.classifyPermissionError(err); permErr != nil {
		return permErr
	}

	// 分类PDB相关错误
	if pdbErr := c.classifyPDBError(err); pdbErr != nil {
		return pdbErr
	}

	// 默认为未知错误
	return NewDrainError(ErrorCodeUnknown, "未知错误", err).
		WithSeverity(SeverityMedium).
		WithRetryable(false)
}

// classifyK8sError 分类K8s API错误
func (c *DefaultErrorClassifier) classifyK8sError(err error) *DrainError {
	// 检查是否是K8s API错误
	if !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) &&
		!apierrors.IsUnauthorized(err) && !apierrors.IsTimeout(err) &&
		!apierrors.IsServerTimeout(err) && !apierrors.IsConflict(err) &&
		!apierrors.IsTooManyRequests(err) && !apierrors.IsInternalError(err) {
		return nil
	}

	var statusErr *apierrors.StatusError
	if !errors.As(err, &statusErr) {
		return nil
	}

	switch {
	case apierrors.IsNotFound(err):
		return NewDrainError(ErrorCodeNodeNotFound, "资源不存在", err).
			WithSeverity(SeverityMedium).
			WithRetryable(false).
			WithSuggestion("检查资源名称是否正确")

	case apierrors.IsForbidden(err):
		return NewDrainError(ErrorCodePermissionDenied, "权限不足", err).
			WithSeverity(SeverityHigh).
			WithRetryable(false).
			WithSuggestion("检查RBAC权限配置")

	case apierrors.IsUnauthorized(err):
		return NewDrainError(ErrorCodeClusterAuth, "认证失败", err).
			WithSeverity(SeverityHigh).
			WithRetryable(false).
			WithSuggestion("检查集群认证配置")

	case apierrors.IsTimeout(err):
		return NewDrainError(ErrorCodeClusterTimeout, "请求超时", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("检查网络连接或增加超时时间")

	case apierrors.IsServerTimeout(err):
		return NewDrainError(ErrorCodeClusterTimeout, "服务器超时", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("稍后重试或联系集群管理员")

	case apierrors.IsConflict(err):
		return NewDrainError(ErrorCodeResourceConflict, "资源冲突", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("稍后重试")

	case apierrors.IsTooManyRequests(err):
		return NewDrainError(ErrorCodeClusterTimeout, "请求过于频繁", err).
			WithSeverity(SeverityLow).
			WithRetryable(true).
			WithSuggestion("降低请求频率")

	case apierrors.IsInternalError(err):
		return NewDrainError(ErrorCodeInternal, "集群内部错误", err).
			WithSeverity(SeverityHigh).
			WithRetryable(true).
			WithSuggestion("联系集群管理员")

	default:
		return NewDrainError(ErrorCodeUnknown, fmt.Sprintf("K8s API错误: %s", statusErr.Status().Message), err).
			WithSeverity(SeverityMedium).
			WithRetryable(false)
	}
}

// classifyNetworkError 分类网络错误
func (c *DefaultErrorClassifier) classifyNetworkError(err error) *DrainError {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "connection refused"):
		return NewDrainError(ErrorCodeClusterConnection, "连接被拒绝", err).
			WithSeverity(SeverityHigh).
			WithRetryable(true).
			WithSuggestion("检查集群API服务器地址和端口")

	case strings.Contains(errMsg, "connection timeout"):
		return NewDrainError(ErrorCodeClusterTimeout, "连接超时", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("检查网络连接")

	case strings.Contains(errMsg, "no such host"):
		return NewDrainError(ErrorCodeClusterConnection, "主机不存在", err).
			WithSeverity(SeverityHigh).
			WithRetryable(false).
			WithSuggestion("检查集群API服务器地址")

	case strings.Contains(errMsg, "network unreachable"):
		return NewDrainError(ErrorCodeClusterConnection, "网络不可达", err).
			WithSeverity(SeverityHigh).
			WithRetryable(true).
			WithSuggestion("检查网络配置")

	default:
		return nil
	}
}

// classifyTimeoutError 分类超时错误
func (c *DefaultErrorClassifier) classifyTimeoutError(err error) *DrainError {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return NewDrainError(ErrorCodeTimeout, "操作超时", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("增加超时时间或稍后重试")
	}

	return nil
}

// classifyPermissionError 分类权限错误
func (c *DefaultErrorClassifier) classifyPermissionError(err error) *DrainError {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "forbidden"):
		return NewDrainError(ErrorCodePermissionDenied, "操作被禁止", err).
			WithSeverity(SeverityHigh).
			WithRetryable(false).
			WithSuggestion("检查RBAC权限配置")

	case strings.Contains(errMsg, "unauthorized"):
		return NewDrainError(ErrorCodeClusterAuth, "未授权", err).
			WithSeverity(SeverityHigh).
			WithRetryable(false).
			WithSuggestion("检查认证配置")

	default:
		return nil
	}
}

// classifyPDBError 分类PDB相关错误
func (c *DefaultErrorClassifier) classifyPDBError(err error) *DrainError {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "poddisruptionbudget"):
		if strings.Contains(errMsg, "violation") || strings.Contains(errMsg, "disruption") {
			return NewDrainError(ErrorCodePDBViolation, "PDB违规", err).
				WithSeverity(SeverityMedium).
				WithRetryable(true).
				WithSuggestion("等待Pod重新调度或调整PDB配置")
		}
		return NewDrainError(ErrorCodePDBCreation, "PDB操作失败", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true)

	case strings.Contains(errMsg, "eviction"):
		return NewDrainError(ErrorCodePodEviction, "Pod驱逐失败", err).
			WithSeverity(SeverityMedium).
			WithRetryable(true).
			WithSuggestion("检查PDB配置或Pod状态")

	default:
		return nil
	}
}

// WithSeverity 设置错误严重程度
func (e *DrainError) WithSeverity(severity DrainErrorSeverity) *DrainError {
	e.Severity = severity
	return e
}

// WithRetryable 设置是否可重试
func (e *DrainError) WithRetryable(retryable bool) *DrainError {
	e.Retryable = retryable
	return e
}
