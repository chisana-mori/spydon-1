package http

import (
	"fmt"
	"net/http"
	"robusta-web/backend/internal/features/navy/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetF5Info handles GET /api/v1/navy/f5/:id
func (h *Handler) GetF5Info(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	f5Info, err := h.f5Service.GetF5Info(c.Request.Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf("F5 info with id %d not found", id) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, f5Info)
}

// ListF5Infos handles GET /api/v1/navy/f5
func (h *Handler) ListF5Infos(c *gin.Context) {
	var query services.F5InfoQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	response, err := h.f5Service.ListF5Infos(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list F5 infos: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateF5Info handles PUT /api/v1/navy/f5/:id
func (h *Handler) UpdateF5Info(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var dto services.F5InfoUpdateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := h.f5Service.UpdateF5Info(c.Request.Context(), id, &dto); err != nil {
		if err.Error() == fmt.Sprintf("F5 info with id %d not found", id) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update F5 info: %s", err.Error())})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "F5 info updated successfully"})
}

// DeleteF5Info handles DELETE /api/v1/navy/f5/:id
func (h *Handler) DeleteF5Info(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.f5Service.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf("F5 info with id %d not found", id) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete F5 info: %s", err.Error())})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "F5 info deleted successfully"})
}
