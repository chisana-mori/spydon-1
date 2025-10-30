package middleware

import (
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 跨域资源共享中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的源列表（生产环境应该配置具体的域名）
		allowedOrigins := []string{
			constants.CORSPort3000HTTP,
			constants.CORSPort5173HTTP,
			constants.CORSPort3000HTTPS,
			constants.CORSPort5173HTTPS,
		}

		// 检查是否为允许的源
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header(constants.HeaderAccessControlAllowOrigin, origin)
		}

		c.Header(constants.HeaderAccessControlAllowMethods, constants.AllowedHTTPMethods)
		c.Header(constants.HeaderAccessControlAllowHeaders, "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, "+constants.HeaderRobustaSignature)
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header(constants.HeaderAccessControlAllowCredentials, "true")
		c.Header(constants.HeaderAccessControlMaxAge, constants.CORSMaxAge)

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(constants.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware 安全头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header(constants.SecurityXFrameOptions, constants.SecurityXFrameOptionsDeny)

		// 防止MIME类型嗅探
		c.Header(constants.SecurityXContentTypeOptions, constants.SecurityXContentTypeOptionsNoSniff)

		// XSS保护
		c.Header(constants.SecurityXXSSProtection, constants.SecurityXXSSProtectionBlock)

		// 强制HTTPS（生产环境）
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
