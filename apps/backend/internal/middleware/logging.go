package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"robusta-web/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestResponseLogger 请求响应日志中间件
func RequestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("request_id", c.GetString("request_id")),
		}
		if errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
		}

		if status >= 500 {
			logger.L().Error("请求处理失败", fields...)
		} else if status >= 400 {
			logger.L().Warn("请求返回客户端错误", fields...)
		} else {
			logger.L().Info("请求处理成功", fields...)
		}
	}
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

const auditRequestBodyPreviewLimit = 1 << 20 // 1MB

// AuditLogMiddleware 审计日志中间件
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// GET 请求不记录审计日志，直接跳过
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// 记录请求开始时间
		start := time.Now()

		// 读取请求体（如果需要记录）
		var requestBody []byte
		truncated := false
		if c.Request.Body != nil {
			preview, restoredBody, wasTruncated, err := peekRequestBody(c.Request.Body, auditRequestBodyPreviewLimit)
			if err != nil {
				// 恢复原始请求体，确保后续流程不受影响
				c.Request.Body = restoredBody
			} else {
				requestBody = preview
				truncated = wasTruncated
				c.Request.Body = restoredBody
			}
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
			if truncated {
				auditLog["request_body_truncated"] = true
			}
		}

		// 如果响应状态码表示错误，记录响应体
		if c.Writer.Status() >= 400 {
			auditLog["response_body"] = blw.body.String()
		}

		// 输出审计日志（实际应用中应该写入专门的审计日志系统）
		emitAuditLog(auditLog, truncated, blw.body.String())
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

func emitAuditLog(payload map[string]interface{}, truncated bool, responseBody string) {
	s := logger.S()
	fields := make([]interface{}, 0, len(payload)*2+4)
	for key, value := range payload {
		fields = append(fields, key, value)
	}

	if truncated {
		fields = append(fields, "request_body_truncated", true)
	}

	if responseBody != "" {
		fields = append(fields, "response_body", responseBody)
	}

	status := 0
	switch v := payload["status_code"].(type) {
	case int:
		status = v
	case int64:
		status = int(v)
	case float64:
		status = int(v)
	}

	message := "审计事件"
	if status >= 500 {
		s.Errorw(message, fields...)
	} else if status >= 400 {
		s.Warnw(message, fields...)
	} else {
		s.Infow(message, fields...)
	}
}

// peekRequestBody 读取请求体预览并返回可重复读取的body
func peekRequestBody(body io.ReadCloser, limit int) ([]byte, io.ReadCloser, bool, error) {
	if limit <= 0 {
		limit = auditRequestBodyPreviewLimit
	}

	buf := make([]byte, limit+1)
	n, err := io.ReadFull(body, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			// 请求体短于缓冲区，n 为实际长度
		} else {
			return nil, body, false, err
		}
	}

	if n < 0 {
		n = 0
	}

	previewLen := n
	if previewLen > limit {
		previewLen = limit
	}

	preview := make([]byte, previewLen)
	copy(preview, buf[:previewLen])

	restored := &readMultiCloser{
		Reader: io.MultiReader(bytes.NewReader(buf[:n]), body),
		Closer: body,
	}

	truncated := n > limit
	return preview, restored, truncated, nil
}

type readMultiCloser struct {
	io.Reader
	Closer io.Closer
}

func (r *readMultiCloser) Close() error {
	if r.Closer != nil {
		return r.Closer.Close()
	}
	return nil
}
