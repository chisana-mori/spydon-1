package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParsePaginationParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  map[string]string
		expectErr    bool
		expectedPage int
		expectedSize int
		expectedSort string
	}{
		{
			name:         "默认参数",
			queryParams:  map[string]string{},
			expectErr:    false,
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name: "自定义参数",
			queryParams: map[string]string{
				"page":      "2",
				"page_size": "50",
			},
			expectErr:    false,
			expectedPage: 2,
			expectedSize: 50,
		},
		{
			name: "使用 limit 参数（向后兼容）",
			queryParams: map[string]string{
				"page":  "3",
				"limit": "30",
			},
			expectErr:    false,
			expectedPage: 3,
			expectedSize: 30,
		},
		{
			name: "page_size 优先于 limit",
			queryParams: map[string]string{
				"page":      "1",
				"page_size": "25",
				"limit":     "30",
			},
			expectErr:    false,
			expectedPage: 1,
			expectedSize: 25,
		},
		{
			name: "包含排序参数",
			queryParams: map[string]string{
				"sort": "-created_at",
			},
			expectErr:    false,
			expectedPage: 1,
			expectedSize: 20,
			expectedSort: "-created_at",
		},
		{
			name: "无效的 page（小于1）",
			queryParams: map[string]string{
				"page":      "0",
				"page_size": "20",
			},
			expectErr: true,
		},
		{
			name: "page_size 超过最大值",
			queryParams: map[string]string{
				"page":      "1",
				"page_size": "150",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// 构建查询参数
			req, _ := http.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			params, err := ParsePaginationParams(c)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPage, params.Page)
			assert.Equal(t, tt.expectedSize, params.PageSize)
			assert.Equal(t, tt.expectedSort, params.Sort)
		})
	}
}

func TestPaginationParams_GetOffset(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
	}{
		{
			name:           "第一页",
			page:           1,
			pageSize:       20,
			expectedOffset: 0,
		},
		{
			name:           "第二页",
			page:           2,
			pageSize:       20,
			expectedOffset: 20,
		},
		{
			name:           "第三页，每页50条",
			page:           3,
			pageSize:       50,
			expectedOffset: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := PaginationParams{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			offset := params.GetOffset()
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

func TestPaginationParams_GetLimit(t *testing.T) {
	params := PaginationParams{
		Page:     1,
		PageSize: 25,
	}

	limit := params.GetLimit()
	assert.Equal(t, 25, limit)
}

func TestNewPagination(t *testing.T) {
	tests := []struct {
		name               string
		page               int
		pageSize           int
		total              int64
		expectedTotalPages int
	}{
		{
			name:               "整除情况",
			page:               1,
			pageSize:           20,
			total:              100,
			expectedTotalPages: 5,
		},
		{
			name:               "有余数情况",
			page:               1,
			pageSize:           20,
			total:              95,
			expectedTotalPages: 5,
		},
		{
			name:               "总数为0",
			page:               1,
			pageSize:           20,
			total:              0,
			expectedTotalPages: 0,
		},
		{
			name:               "总数小于每页数量",
			page:               1,
			pageSize:           20,
			total:              10,
			expectedTotalPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pagination := NewPagination(tt.page, tt.pageSize, tt.total)

			assert.Equal(t, tt.page, pagination.Page)
			assert.Equal(t, tt.pageSize, pagination.PageSize)
			assert.Equal(t, tt.total, pagination.Total)
			assert.Equal(t, tt.expectedTotalPages, pagination.TotalPages)
		})
	}
}
