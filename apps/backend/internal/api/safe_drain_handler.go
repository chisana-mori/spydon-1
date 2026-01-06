package api

import (
	"net/http"
	"time"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type SafeDrainHandler struct {
	service *services.SafeDrainService
}

func NewSafeDrainHandler(service *services.SafeDrainService) *SafeDrainHandler {
	return &SafeDrainHandler{service: service}
}

func (h *SafeDrainHandler) StartDrain(c *gin.Context) {
	var req services.SafeDrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.StartDrain(c.Request.Context(), &req)
	if err != nil {
		// Differentiate errors if needed (e.g. 409 Conflict for existing lease)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *SafeDrainHandler) CancelDrain(c *gin.Context) {
	drainID := c.Param("drain_id")
	if drainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "drain_id is required"})
		return
	}

	if err := h.service.CancelDrain(drainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Drain canceled"})
}

func (h *SafeDrainHandler) DrainEvents(c *gin.Context) {
	drainID := c.Param("drain_id")
	if drainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "drain_id is required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	eventCh, unsubscribe := h.service.GetEventManager().Subscribe(drainID)
	defer unsubscribe()

	// Notify connection established
	c.SSEvent("status", "connected")
	c.Writer.Flush()

	// Bootstrap: Send current drain state to the newly connected client
	// This solves the race condition where events are sent before client subscribes
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay to ensure subscription is ready
		h.service.SendBootstrapEvents(drainID)
	}()

	timeout := time.After(60 * time.Minute) // Max connection duration

	// Heartbeat ticker to keep connections alive and help proxies
	hbTicker := time.NewTicker(15 * time.Second)
	defer hbTicker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-timeout:
			return
		case <-hbTicker.C:
			// SSE comment heartbeat (ignored by browsers, keeps connection warm)
			_, _ = c.Writer.Write([]byte(": ping\n\n"))
			c.Writer.Flush()
		case msg, ok := <-eventCh:
			if !ok {
				return // Channel closed
			}
			// Send raw JSON payload as data
			_, _ = c.Writer.Write([]byte("data: " + msg + "\n\n"))
			c.Writer.Flush()
		}
	}
}
