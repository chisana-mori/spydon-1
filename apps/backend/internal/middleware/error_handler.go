package middleware

import (
	"net/http"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		last := c.Errors.Last()
		domainErr := apperrors.EnsureDomainError(last.Err)
		status := domainErr.Status()
		if status == 0 {
			status = http.StatusInternalServerError
		}

		response := gin.H{
			"code":  domainErr.Code(),
			"error": domainErr.Message(),
		}

		if domainErr.Message() != "" {
			response["message"] = domainErr.Message()
		}

		if details := domainErr.Details(); details != nil {
			response["details"] = details
		} else if last.Meta != nil {
			response["details"] = last.Meta
		}

		if reqID, exists := c.Get("request_id"); exists {
			response["request_id"] = reqID
		}

		if status >= http.StatusInternalServerError {
			logger.L().Error("请求处理出现服务器错误", zap.Error(domainErr), zap.Int("status", status))
		} else {
			logger.L().Warn("请求处理出现客户端错误", zap.Error(domainErr), zap.Int("status", status))
		}

		c.AbortWithStatusJSON(status, response)
	}
}
