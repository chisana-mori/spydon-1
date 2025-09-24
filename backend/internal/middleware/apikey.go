package middleware

import (
    "net/http"
    "strings"

    "robusta-web/backend/internal/config"

    "github.com/gin-gonic/gin"
)

// APIKeyMiddleware 校验入站请求的API Key（X-API-Key 或 Authorization: Bearer）
// 当 cfg.IngestAPIKey 为空时跳过校验，用于本地/开发模式；生产请务必设置。
func APIKeyMiddleware(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        if cfg.IngestAPIKey == "" {
            c.Next()
            return
        }

        key := c.GetHeader("X-API-Key")
        if key == "" {
            auth := c.GetHeader("Authorization")
            if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
                key = strings.TrimSpace(auth[7:])
            }
        }

        if key == "" || key != cfg.IngestAPIKey {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "API Key无效",
                "code":  "INVALID_API_KEY",
            })
            return
        }

        c.Next()
    }
}

