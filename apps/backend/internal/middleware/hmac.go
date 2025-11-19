package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/gin-gonic/gin"
)

// HMACMiddleware HMAC签名验证中间件（用于Robusta webhook）
func HMACMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 若未配置HMAC密钥，则跳过校验（用于本地/开发环境便捷对接）
		if cfg.HMACSecret == "" {
			c.Next()
			return
		}
		// 获取签名头
		signature := c.GetHeader("X-Robusta-Signature")
		if signature == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少HMAC签名",
				"code":  "MISSING_SIGNATURE",
			})
			c.Abort()
			return
		}

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "读取请求体失败",
				"code":  "READ_BODY_ERROR",
			})
			c.Abort()
			return
		}

		// 重新设置请求体，以便后续处理器可以读取
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 验证HMAC签名
		if !verifyHMACSignature(body, signature, cfg.HMACSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "HMAC签名验证失败",
				"code":  "INVALID_SIGNATURE",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// verifyHMACSignature 验证HMAC签名
func verifyHMACSignature(body []byte, signature, secret string) bool {
	// 移除sha256=前缀（如果存在）
	signature = strings.TrimPrefix(signature, "sha256=")

	// 计算期望的HMAC签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// 使用恒定时间比较防止时序攻击
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// EnhancedHMACMiddleware 增强版HMAC签名验证中间件（支持时间戳验证）
func EnhancedHMACMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 若未配置HMAC密钥，则跳过校验（用于本地/开发环境便捷对接）
		if cfg.HMACSecret == "" {
			c.Next()
			return
		}
		// 获取签名头
		signature := c.GetHeader("X-Robusta-Signature")
		timestamp := c.GetHeader("X-Robusta-Timestamp")
		clusterID := c.GetHeader("X-Robusta-Cluster-ID")

		if signature == "" || timestamp == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":            "缺少必要的签名头",
				"code":             "MISSING_SIGNATURE_HEADERS",
				"required_headers": []string{"X-Robusta-Signature", "X-Robusta-Timestamp"},
			})
			c.Abort()
			return
		}

		// 验证时间戳（防止重放攻击）
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的时间戳格式",
				"code":  "INVALID_TIMESTAMP",
			})
			c.Abort()
			return
		}

		// 检查时间戳是否在允许范围内（5分钟）
		now := time.Now().Unix()
		timeDiff := abs(now - ts)
		if timeDiff > 300 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "请求时间戳过期",
				"code":    "TIMESTAMP_EXPIRED",
				"details": fmt.Sprintf("时间差: %d秒, 允许范围: 300秒", timeDiff),
			})
			c.Abort()
			return
		}

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "读取请求体失败",
				"code":  "BODY_READ_ERROR",
			})
			c.Abort()
			return
		}

		// 重新设置请求体
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 验证增强版HMAC签名（包含时间戳）
		if !verifyEnhancedHMAC(signature, timestamp, string(body), cfg.HMACSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "HMAC签名验证失败",
				"code":  "HMAC_VERIFICATION_FAILED",
				"hint":  "请检查HMAC密钥和签名算法",
			})
			c.Abort()
			return
		}

		// 将集群ID和时间戳存储到上下文中
		if clusterID != "" {
			c.Set("cluster_id", clusterID)
		}
		c.Set("request_timestamp", ts)

		c.Next()
	}
}

// verifyEnhancedHMAC 验证增强版HMAC签名（包含时间戳）
func verifyEnhancedHMAC(signature, timestamp, body, secret string) bool {
	// 移除sha256=前缀
	signature = strings.TrimPrefix(signature, "sha256=")

	// 构造签名数据（时间戳.请求体）
	data := fmt.Sprintf("%s.%s", timestamp, body)

	// 计算HMAC
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// 使用恒定时间比较防止时序攻击
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GenerateEnhancedHMAC 生成增强版HMAC签名（包含时间戳）
func GenerateEnhancedHMAC(timestamp, body, secret string) string {
	data := fmt.Sprintf("%s.%s", timestamp, body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// abs 返回整数的绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
