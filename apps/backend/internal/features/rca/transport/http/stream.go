package http

import (
	"time"

	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// StreamRCA streams RCA analysis status and result via Server-Sent Events.
func (h *Handler) StreamRCA(c *gin.Context) {
	var uri struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(400, gin.H{"error": "MISSING_ALERT_ID", "message": "告警ID不能为空"})
		return
	}
	alertIDStr := models.FormatID(uri.AlertID)

	// Set SSE headers (must be set before any write)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// CORS: allow known origins and support credentials (cannot use *)
	origin := c.Request.Header.Get("Origin")
	allowedOrigins := map[string]struct{}{
		constants.CORSPort3000HTTP:  {},
		constants.CORSPort5173HTTP:  {},
		constants.CORSPort3000HTTPS: {},
		constants.CORSPort5173HTTPS: {},
	}
	if _, ok := allowedOrigins[origin]; ok {
		c.Header(constants.HeaderAccessControlAllowOrigin, origin)
		c.Header(constants.HeaderAccessControlAllowCredentials, "true")
	}

	// 1. Subscribe to broadcast first to avoid missing updates.
	ch, unsubscribe := h.rcaService.Subscribe(alertIDStr)
	defer unsubscribe()

	// 2. Check existing RCA status after subscribing.
	rcaRuns, err := h.rcaService.GetRCARunsByAlertID(uri.AlertID)
	if err != nil {
		c.SSEvent("error", gin.H{"error": "获取RCA状态失败", "details": err.Error()})
		return
	}

	var latestRun *models.RCARun
	if len(rcaRuns) > 0 {
		latestRun = &rcaRuns[0]
	}

	// 3. Send current status
	if latestRun != nil {
		c.SSEvent("status", gin.H{
			"status": latestRun.Status,
			"run_id": models.FormatID(latestRun.ID),
		})
		c.Writer.Flush()

		// If already completed or failed, wait briefly for possible delayed messages.
		if latestRun.Status == string(models.RCAStatusCompleted) || latestRun.Status == string(models.RCAStatusFailed) {
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()

			select {
			case msg, ok := <-ch:
				if ok {
					if _, writeErr := c.Writer.Write([]byte("data: " + msg + "\n\n")); writeErr != nil {
						return
					}
					c.Writer.Flush()
				}
			case <-timer.C:
				return
			case <-c.Request.Context().Done():
				return
			}
			return
		}
	} else {
		c.SSEvent("status", gin.H{"status": "none", "message": "等待 RCA 分析开始"})
		c.Writer.Flush()
	}

	// 4. Listen for messages
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, writeErr := c.Writer.Write([]byte("data: " + msg + "\n\n")); writeErr != nil {
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
