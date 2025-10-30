package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := map[string]string{"message": "test"}
	Success(c, testData)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Data)
}

func TestSuccessWithExtras(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"id": "1"}
	Success(c, data, gin.H{"cache_hit": true})

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["cache_hit"])
}

func TestSuccessWithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := map[string]string{"id": "123"}
	SuccessWithMessage(c, "操作成功", testData)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "操作成功", response.Message)
	assert.NotNil(t, response.Data)
}

func TestCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := map[string]string{"id": "new-123"}
	Created(c, testData)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Data)
}

func TestSuccessPaginated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := []map[string]string{
		{"id": "1"},
		{"id": "2"},
	}
	pagination := NewPagination(1, 20, 100)
	SuccessPaginated(c, testData, pagination)

	assert.Equal(t, http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Data)
	assert.NotNil(t, response.Pagination)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 20, response.Pagination.PageSize)
	assert.Equal(t, int64(100), response.Pagination.Total)
	assert.Equal(t, 5, response.Pagination.TotalPages)
}

func TestBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "INVALID_INPUT", "输入参数无效")
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_INPUT", response.Code)
	assert.Equal(t, "输入参数无效", response.Error)
}

func TestUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "UNAUTHORIZED", "未授权")
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", response.Code)
	assert.Equal(t, "未授权", response.Error)
}

func TestForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c, "FORBIDDEN", "禁止访问")
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "FORBIDDEN", response.Code)
	assert.Equal(t, "禁止访问", response.Error)
}

func TestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "NOT_FOUND", "资源不存在")
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "NOT_FOUND", response.Code)
	assert.Equal(t, "资源不存在", response.Error)
}

func TestInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalError(c, "INTERNAL_ERROR", "服务器内部错误")
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", response.Code)
	assert.Equal(t, "服务器内部错误", response.Error)
}

func TestErrorWithDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	details := "详细的错误信息"
	ErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "验证失败", details)
	middleware.ErrorHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", response["code"])
	assert.Equal(t, "验证失败", response["error"])
	assert.Equal(t, details, response["details"])
}
