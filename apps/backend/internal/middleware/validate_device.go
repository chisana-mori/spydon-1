package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"robusta-web/backend/internal/db"

	"github.com/gin-gonic/gin"
)

// ciCodesRequest 用于解析请求体中的 ci_codes
type ciCodesRequest struct {
	CICodes []string `json:"ci_codes"`
}

// ValidateClusterAssociation 验证设备是否关联了集群
func ValidateClusterAssociation(navyDB *db.NavyDatabase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 读取 Body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
			return
		}
		// 恢复 Body 以供后续 Handler 使用
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 2. 解析 CI Codes
		var req ciCodesRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			// 如果解析失败，可能是请求格式不匹配此 middleware 预期的结构
			// 对于目标路由，ci_codes 是必须的，所以这里可以宽容处理或直接忽略
			// 但为了安全起见，如果 JSON 解析都错了，最好由后续 Handler 的 ShouldBindJSON 去报错
			c.Next()
			return
		}

		// 如果没有 CI Codes，跳过检查（让 Handler 层的验证去处理 required 校验）
		if len(req.CICodes) == 0 {
			c.Next()
			return
		}

		// 3. 数据库检查
		var devices []struct {
			CICode  string `gorm:"column:ci_code"`
			Cluster string `gorm:"column:cluster"`
		}

		if err := navyDB.Table("device").
			Select("ci_code, cluster").
			Where("ci_code IN ?", req.CICodes).
			Scan(&devices).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "验证设备信息时发生数据库错误"})
			return
		}

		// 4. 验证逻辑
		foundMap := make(map[string]string) // ci_code -> cluster
		for _, d := range devices {
			foundMap[d.CICode] = d.Cluster
		}

		var notFound []string
		var notInCluster []string

		for _, code := range req.CICodes {
			cluster, exists := foundMap[code]
			if !exists {
				notFound = append(notFound, code)
				continue
			}
			if cluster == "" {
				notInCluster = append(notInCluster, code)
			}
		}

		if len(notFound) > 0 || len(notInCluster) > 0 {
			var errMsgs []string
			if len(notFound) > 0 {
				errMsgs = append(errMsgs, fmt.Sprintf("节点不存在: %s", strings.Join(notFound, ", ")))
			}
			if len(notInCluster) > 0 {
				errMsgs = append(errMsgs, fmt.Sprintf("节点未关联集群: %s", strings.Join(notInCluster, ", ")))
			}

			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": strings.Join(errMsgs, "; "),
			})
			return
		}

		c.Next()
	}
}
