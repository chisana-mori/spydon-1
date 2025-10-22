package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestResponseLogger 请求响应日志中间件
func RequestResponseLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}

		if param.Latency > time.Minute {
			param.Latency = param.Latency.Truncate(time.Second)
		}

		return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// AuditLogMiddleware 审计日志中间件
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 读取请求体（如果需要记录）
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应写入器包装器
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		// 记录审计日志
		auditLog := map[string]interface{}{
			"request_id":  c.GetString("request_id"),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"status_code": c.Writer.Status(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"client_ip":   c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"user_id":     c.GetString("user_id"),
			"timestamp":   start.Format(time.RFC3339),
		}

		// 如果是敏感操作，记录请求体（排除密码等敏感信息）
		if shouldLogRequestBody(c.Request.Method, c.Request.URL.Path) {
			auditLog["request_body"] = sanitizeRequestBody(requestBody)
		}

		// 如果响应状态码表示错误，记录响应体
		if c.Writer.Status() >= 400 {
			auditLog["response_body"] = blw.body.String()
		}

		// 输出审计日志（实际应用中应该写入专门的审计日志系统）
		logJSON, _ := json.Marshal(auditLog)
		gin.DefaultWriter.Write(append(logJSON, '\n'))
	}
}

// bodyLogWriter 响应体日志写入器
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// shouldLogRequestBody 判断是否应该记录请求体
func shouldLogRequestBody(method, path string) bool {
	// 只记录POST、PUT、PATCH请求的请求体
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return false
	}

	// 排除某些路径
	excludePaths := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
	}

	for _, excludePath := range excludePaths {
		if path == excludePath {
			return false
		}
	}

	return true
}

// sanitizeRequestBody 清理请求体中的敏感信息
func sanitizeRequestBody(body []byte) interface{} {
	if len(body) == 0 {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body) // 如果不是JSON，直接返回字符串
	}

	// 移除敏感字段
	sensitiveFields := []string{"password", "token", "secret", "key"}
	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			data[field] = "[REDACTED]"
		}
	}

	return data
}
