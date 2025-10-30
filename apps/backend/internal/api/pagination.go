package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
)

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	Sort       string `json:"sort,omitempty"`
}

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	Sort     string
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
	pageStr := c.DefaultQuery("page", strconv.Itoa(constants.PaginationDefaultPage))
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return PaginationParams{}, newPaginationError("page", "必须为整数", err)
	}

	pageSizeStr := c.DefaultQuery("page_size", strconv.Itoa(constants.PaginationDefaultPageSize))
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		return PaginationParams{}, newPaginationError("page_size", "必须为整数", err)
	}

	// 兼容 limit
	if limit := c.Query("limit"); limit != "" {
		if parsedLimit, convErr := strconv.Atoi(limit); convErr == nil {
			pageSize = parsedLimit
		} else {
			return PaginationParams{}, newPaginationError("limit", "必须为整数", convErr)
		}
	}

	sortField := strings.TrimSpace(c.DefaultQuery("sort", ""))

	if page < 1 {
		return PaginationParams{}, newPaginationError("page", "必须大于等于1", nil)
	}
	if pageSize < 1 {
		return PaginationParams{}, newPaginationError("page_size", "必须大于等于1", nil)
	}
	if pageSize > constants.PaginationMaxPageSize {
		return PaginationParams{}, newPaginationError("page_size",
			fmt.Sprintf("不能超过%d", constants.PaginationMaxPageSize), nil)
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     sortField,
	}, nil
}

// GetOffset 计算数据库查询的偏移量
func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取查询限制数量
func (p PaginationParams) GetLimit() int {
	return p.PageSize
}

func newPaginationError(field, message string, cause error) apperrors.DomainError {
	details := map[string]string{
		field: message,
	}
	options := []apperrors.Option{apperrors.WithDetails(details)}
	if cause != nil {
		options = append(options, apperrors.WithCause(cause))
	}
	return apperrors.New(
		http.StatusBadRequest,
		constants.ErrorCodeValidationFailed,
		"分页参数不合法",
		options...,
	)
}
