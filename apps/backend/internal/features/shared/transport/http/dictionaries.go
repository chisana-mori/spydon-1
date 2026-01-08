package http

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// =====================================================
// 字典管理 API Handlers
// =====================================================

// ListDictionaries 获取字典列表
// GET /api/v1/shared/dictionaries?module=xxx&is_enabled=true&keyword=xxx
func (h *Handler) ListDictionaries(c *gin.Context) {
	var query services.DictionaryListQuery

	// 解析 module 和 keyword
	query.Module = c.Query("module")
	query.Keyword = c.Query("keyword")

	// 解析 is_enabled
	if isEnabledStr := c.Query("is_enabled"); isEnabledStr != "" {
		isEnabled := isEnabledStr == "true"
		query.IsEnabled = &isEnabled
	}

	dictionaries, err := h.dictionaryService.ListDictionaries(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dictionaries})
}

// CreateDictionary 创建字典
// POST /api/v1/shared/dictionaries
func (h *Handler) CreateDictionary(c *gin.Context) {
	var req services.CreateDictionaryRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	dict, err := h.dictionaryService.CreateDictionary(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": dict})
}

// GetDictionary 获取字典详情（含字典项）
// GET /api/v1/shared/dictionaries/:id
func (h *Handler) GetDictionary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	dict, err := h.dictionaryService.GetDictionary(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if dict == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "字典不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dict})
}

// UpdateDictionary 更新字典
// PUT /api/v1/shared/dictionaries/:id
func (h *Handler) UpdateDictionary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	var req services.UpdateDictionaryRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	dict, err := h.dictionaryService.UpdateDictionary(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dict})
}

// DeleteDictionary 删除字典
// DELETE /api/v1/shared/dictionaries/:id
func (h *Handler) DeleteDictionary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	if err := h.dictionaryService.DeleteDictionary(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// =====================================================
// 字典项管理 API Handlers
// =====================================================

// GetDictionaryItems 获取字典项列表
// GET /api/v1/shared/dictionaries/:id/items
func (h *Handler) GetDictionaryItems(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	dict, err := h.dictionaryService.GetDictionary(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if dict == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "字典不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dict.Items})
}

// CreateDictionaryItem 创建字典项
// POST /api/v1/shared/dictionaries/:id/items
func (h *Handler) CreateDictionaryItem(c *gin.Context) {
	dictID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	var req services.CreateDictionaryItemRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	item, err := h.dictionaryService.CreateDictionaryItem(dictID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

// UpdateDictionaryItem 更新字典项
// PUT /api/v1/shared/dictionaries/:id/items/:item_id
func (h *Handler) UpdateDictionaryItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典项ID"})
		return
	}

	var req services.UpdateDictionaryItemRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	item, err := h.dictionaryService.UpdateDictionaryItem(itemID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

// DeleteDictionaryItem 删除字典项
// DELETE /api/v1/shared/dictionaries/:id/items/:item_id
func (h *Handler) DeleteDictionaryItem(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("item_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典项ID"})
		return
	}

	if err := h.dictionaryService.DeleteDictionaryItem(itemID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// BatchUpdateDictionaryItems 批量更新字典项
// PUT /api/v1/shared/dictionaries/:id/items
func (h *Handler) BatchUpdateDictionaryItems(c *gin.Context) {
	dictID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字典ID"})
		return
	}

	var req services.BatchUpdateDictionaryItemsRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	// 日志：打印接收到的请求数据 (使用 zap logger 以确保持久化)
	logger.L().Info("接收到批量更新请求",
		zap.Uint64("dict_id", dictID),
		zap.Int("item_count", len(req.Items)),
	)

	// 同时也保留 fmt.Printf 用于调试
	logger.S().Debugf("[Handler] BatchUpdateDictionaryItems DictID=%d, ItemCount=%d", dictID, len(req.Items))

	if err := h.dictionaryService.BatchUpdateItems(dictID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "批量更新成功"})
}

// =====================================================
// 通用接口 - 前端模版使用
// =====================================================

// GetDictionaryItemsByCode 通过字典编码获取字典项
// GET /api/v1/shared/dict/:code/items
// 这是所有前端模版的通用接口
func (h *Handler) GetDictionaryItemsByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "字典编码不能为空"})
		return
	}

	items, err := h.dictionaryService.GetItemsByCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}
