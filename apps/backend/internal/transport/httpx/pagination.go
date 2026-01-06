package httpx

import (
	"fmt"
	"net/http"
	"strings"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
)

// Pagination 分页信息（与 internal/api 的 Pagination 结构保持一致）
type Pagination struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
	Sort       string `json:"sort,omitempty"`
}

// PaginationParams 解析后的分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	Sort     string
}

// PaginationQuery 查询参数绑定体
type PaginationQuery struct {
	Page     int    `form:"page" binding:"omitempty,gte=1"`
	PageSize *int   `form:"page_size" binding:"omitempty,gte=1"`
	Limit    *int   `form:"limit" binding:"omitempty,gte=1"`
	Sort     string `form:"sort"`
}

// NewPagination 创建分页信息
func NewPagination(page, pageSize int, total int64) *Pagination {
	if page <= 0 {
		page = constants.PaginationDefaultPage
	}
	if pageSize <= 0 {
		pageSize = constants.PaginationDefaultPageSize
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int(total) / pageSize
		if int(total)%pageSize > 0 {
			totalPages++
		}
	}

	return &Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// ParsePaginationParams 从请求中解析分页参数
func ParsePaginationParams(c *gin.Context) (PaginationParams, apperrors.DomainError) {
	var query PaginationQuery
	if err := BindQuery(c, &query); err != nil {
		return PaginationParams{}, err
	}
	if err := query.Validate(); err != nil {
		return PaginationParams{}, err
	}

	return query.ToParams(), nil
}

// GetOffset 计算数据库查询的偏移量
func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取查询限制数量
func (p PaginationParams) GetLimit() int {
	return p.PageSize
}

func (q PaginationQuery) Validate() apperrors.DomainError {
	if q.PageSize != nil && *q.PageSize > constants.PaginationMaxPageSize {
		return newPaginationError("page_size",
			fmt.Sprintf("不能超过%d", constants.PaginationMaxPageSize))
	}
	if q.Limit != nil && *q.Limit > constants.PaginationMaxPageSize {
		return newPaginationError("limit",
			fmt.Sprintf("不能超过%d", constants.PaginationMaxPageSize))
	}
	return nil
}

// ToParams 转换为内部使用的 PaginationParams
func (q PaginationQuery) ToParams() PaginationParams {
	pageSize := constants.PaginationDefaultPageSize
	switch {
	case q.PageSize != nil:
		pageSize = *q.PageSize
	case q.Limit != nil:
		pageSize = *q.Limit
	}

	page := q.Page
	if page == 0 {
		page = constants.PaginationDefaultPage
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     strings.TrimSpace(q.Sort),
	}
}

func newPaginationError(field, message string) apperrors.DomainError {
	details := map[string]string{
		field: message,
	}
	return apperrors.New(
		http.StatusBadRequest,
		constants.ErrorCodeValidationFailed,
		"分页参数不合法",
		apperrors.WithDetails(details),
	)
}
