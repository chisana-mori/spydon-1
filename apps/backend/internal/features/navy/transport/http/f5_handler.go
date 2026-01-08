package http

import (
	"fmt"
	"net/http"
	"robusta-web/backend/internal/features/navy/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// F5Handler - F5 负载均衡处理器
// =============================================================================

// F5Handler 处理 F5 负载均衡相关的 HTTP 请求
type F5Handler struct {
	svc *services.F5InfoService
}

// NewF5Handler 创建 F5Handler
func NewF5Handler(svc *services.F5InfoService) *F5Handler {
	return &F5Handler{svc: svc}
}

// RegisterRoutes 注册 F5 路由
func (h *F5Handler) RegisterRoutes(navyGroup *gin.RouterGroup) {
	f5 := navyGroup.Group(RouteGroupF5)
	{
		f5.GET("", h.ListF5Infos)
		f5.GET(RouteParamID, h.GetF5Info)
		f5.PUT(RouteParamID, h.UpdateF5Info)
		f5.DELETE(RouteParamID, h.DeleteF5Info)
	}
}

// GetF5Info handles GET /api/v1/navy/f5/:id
// @Summary 获取F5负载均衡详情
// @Description 根据指定的ID查询特定F5负载均衡器（Load Balancer）的详细配置信息。返回数据包含管理地址、所在的VLAN、分区以及当前的状态快照。该接口对于排查网络流量分发问题或进行F5资源清单配置审计非常有用。
// @Tags Navy,F5
// @Produce json
// @Param id path int true "F5配置ID"
// @Success 200 {object} services.F5Info
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/f5/{id} [get]
func (h *F5Handler) GetF5Info(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	f5Info, err := h.svc.GetF5Info(c.Request.Context(), id)
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
// @Summary 分页获取F5列表
// @Description 检索所有已登记的F5负载均衡设备列表。支持通过IP地址、厂商型号或所属机房进行过滤。接口返回结果集支撑了F5资产管理页面的核心展示，让管理员能全局掌握网络基础架构中所有软硬负载均衡实例的分布概况。
// @Tags Navy,F5
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Param ip query string false "按IP地址搜索"
// @Success 200 {object} services.F5InfoListResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/f5 [get]
func (h *F5Handler) ListF5Infos(c *gin.Context) {
	var query services.F5InfoQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	response, err := h.svc.ListF5Infos(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list F5 infos: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateF5Info handles PUT /api/v1/navy/f5/:id
// @Summary 修改F5配置信息
// @Description 更新已有F5实例的元数据描述，包括但不限于管理账号信息、所在的网络分区或业务用途描述。此接口仅更新系统内的配置记录，不涉及对真实F5设备的即时API改写，确保存量资产数据的实时准确性。
// @Tags Navy,F5
// @Accept json
// @Produce json
// @Param id path int true "F5配置ID"
// @Param request body services.F5InfoUpdateDTO true "F5更新参数"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/f5/{id} [put]
func (h *F5Handler) UpdateF5Info(c *gin.Context) {
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

	if err := h.svc.UpdateF5Info(c.Request.Context(), id, &dto); err != nil {
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
// @Summary 物理删除F5记录
// @Description 彻底移除数据库中指定的F5负载均衡器记录。执行此操作前请确认该设备已正式从基础设施中下线，或不再需要通过本系统进行统一管理。删除后，通过该ID将无法再找回任何关联的历史配置快照。
// @Tags Navy,F5
// @Param id path int true "F5配置ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/f5/{id} [delete]
func (h *F5Handler) DeleteF5Info(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.svc.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf("F5 info with id %d not found", id) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete F5 info: %s", err.Error())})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "F5 info deleted successfully"})
}
