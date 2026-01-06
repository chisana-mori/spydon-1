package middleware

import (
	"net/http"
	"strings"

	"robusta-web/backend/internal/config"
	apikeyservice "robusta-web/backend/internal/features/apikey/services"

	"github.com/gin-gonic/gin"
)

// APIKeyMiddleware 校验入站请求的API Key（仅支持 Authorization: Bearer <token> 格式）
// 优先使用数据库验证，如果数据库中没有找到，则回退到配置文件中的静态Key
func APIKeyMiddleware(cfg *config.Config, apiKeyService *apikeyservice.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果配置的Key为空且没有数据库服务，跳过校验（开发模式）
		if cfg.IngestAPIKey == "" && apiKeyService == nil {
			c.Next()
			return
		}

		// 从请求头获取API Key（仅支持 Authorization: Bearer <token> 格式）
		var key string
		auth := c.GetHeader("Authorization")
		if auth != "" && strings.HasPrefix(auth, "Bearer ") {
			// 提取 Bearer 后面的 token，去除前后空格
			key = strings.TrimSpace(auth[7:])
		}

		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "缺少API Key",
				"code":  "MISSING_API_KEY",
			})
			return
		}

		// 优先尝试数据库验证
		if apiKeyService != nil {
			apiKey, err := apiKeyService.ValidateAPIKey(key)
			if err == nil && apiKey != nil {
				// 验证成功，将用户信息存入上下文
				c.Set("user_id", apiKey.UserID)
				c.Set("api_key_id", apiKey.ID)
				c.Set("api_key_permissions", apiKey.Permissions)
				c.Next()
				return
			}
		}

		// 回退到配置文件中的静态Key验证
		if cfg.IngestAPIKey != "" && key == cfg.IngestAPIKey {
			c.Next()
			return
		}

		// 验证失败
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "API Key无效",
			"code":  "INVALID_API_KEY",
		})
	}
}
