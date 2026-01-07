package services

// =====================================================
// 字典管理 DTOs
// =====================================================

// DictionaryResponse 字典响应
type DictionaryResponse struct {
	ID             uint64 `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Module         string `json:"module"`
	Description    string `json:"description"`
	IsEnabled      bool   `json:"is_enabled"`
	KeySameAsValue bool   `json:"key_same_as_value"`
	SortOrder      int    `json:"sort_order"`
	ItemCount      int    `json:"item_count"` // 字典项数量
}

// DictionaryDetailResponse 字典详情响应（含字典项）
type DictionaryDetailResponse struct {
	DictionaryResponse
	Items []DictionaryItemResponse `json:"items"`
}

// DictionaryItemResponse 字典项响应
type DictionaryItemResponse struct {
	ID          uint64 `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	IsEnabled   bool   `json:"is_enabled"`
	SortOrder   int    `json:"sort_order"`
	Extra       string `json:"extra,omitempty"`
}

// CreateDictionaryRequest 创建字典请求
type CreateDictionaryRequest struct {
	Code           string `json:"code" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Module         string `json:"module"`
	Description    string `json:"description"`
	KeySameAsValue *bool  `json:"key_same_as_value"` // 可选，默认 true
	SortOrder      int    `json:"sort_order"`
}

// UpdateDictionaryRequest 更新字典请求
type UpdateDictionaryRequest struct {
	Name           string `json:"name"`
	Module         string `json:"module"`
	Description    string `json:"description"`
	IsEnabled      *bool  `json:"is_enabled"`
	KeySameAsValue *bool  `json:"key_same_as_value"`
	SortOrder      *int   `json:"sort_order"`
}

// CreateDictionaryItemRequest 创建字典项请求
type CreateDictionaryItemRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	SortOrder   int    `json:"sort_order"`
	Extra       string `json:"extra"`
}

// UpdateDictionaryItemRequest 更新字典项请求
type UpdateDictionaryItemRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsDefault   *bool  `json:"is_default"`
	IsEnabled   *bool  `json:"is_enabled"`
	SortOrder   *int   `json:"sort_order"`
	Extra       string `json:"extra"`
}

// BatchUpdateDictionaryItemsRequest 批量更新字典项请求
type BatchUpdateDictionaryItemsRequest struct {
	Items []DictionaryItemBatchItem `json:"items" binding:"required"`
}

// DictionaryItemBatchItem 批量更新项
type DictionaryItemBatchItem struct {
	ID          *uint64 `json:"id"` // 有 ID 则更新，无 ID 则创建
	Key         string  `json:"key" binding:"required"`
	Value       string  `json:"value" binding:"required"`
	Description string  `json:"description"`
	IsDefault   bool    `json:"is_default"`
	IsEnabled   bool    `json:"is_enabled"`
	SortOrder   int     `json:"sort_order"`
	Extra       string  `json:"extra"`
	Delete      bool    `json:"delete"` // 标记删除
}

// DictionaryListQuery 字典列表查询参数
type DictionaryListQuery struct {
	Module    string `form:"module"`     // 按模块筛选
	IsEnabled *bool  `form:"is_enabled"` // 按启用状态筛选
	Keyword   string `form:"keyword"`    // 搜索关键字
}
