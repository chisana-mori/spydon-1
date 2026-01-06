package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

// CASMiddleware 使用 gopkg.in/cas.v2 处理 CAS 会话与 ticket 验证，再交给 Gin 继续处理.
func CASMiddleware(client *cas.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.Next()
			return
		}

		handler := client.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CAS 验证成功后，更新 Gin 请求并继续后续处理中间件/处理器.
			c.Request = r
			c.Next()
		}))

		handler.ServeHTTP(c.Writer, c.Request)

		// 防止 Gin 在 ServeHTTP 返回后再次继续执行，确保响应只写入一次.
		c.Abort()
	}
}
