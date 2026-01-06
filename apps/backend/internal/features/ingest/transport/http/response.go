package http

import (
	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
)

// Pagination 分页信息
type Pagination struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
	Sort       string `json:"sort,omitempty"`
}

// Response 统一响应结构
type Response struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Extras  gin.H       `json:"-"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Code       string      `json:"code,omitempty"`
	Message    string      `json:"message,omitempty"`
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination"`
	Error      string      `json:"error,omitempty"`
	Extras     gin.H       `json:"-"`
}

func (r Response) toMap() gin.H {
	result := gin.H{}
	if r.Code != "" {
		result["code"] = r.Code
	}
	if r.Message != "" {
		result["message"] = r.Message
	}
	if r.Data != nil {
		result["data"] = r.Data
	}
	if r.Error != "" {
		result["error"] = r.Error
	}
	for key, value := range r.Extras {
		result[key] = value
	}
	return result
}

// successWithMessage 带消息的成功响应
func successWithMessage(c *gin.Context, message string, data any, extras ...gin.H) {
	resp := Response{
		Message: message,
		Data:    data,
	}
	if len(extras) > 0 {
		resp.Extras = extras[0]
	}
	c.AbortWithStatusJSON(constants.StatusOK, resp.toMap())
}

// badRequest 400错误

// badRequest 400错误
func badRequest(c *gin.Context, code string, message string) {
	err := apperrors.BadRequest(
		message,
		nil,
		buildErrorOptions(code)...,
	)
	abortWithDomainError(c, err)
}

// internalError 500错误
func internalError(c *gin.Context, code string, message string) {
	err := apperrors.New(
		constants.StatusInternalServerError,
		code,
		message,
	)
	abortWithDomainError(c, err)
}

// errorWithDetails 带详细信息的错误响应
func errorWithDetails(c *gin.Context, statusCode int, code string, message string, details any) {
	if code == "" {
		code = constants.ErrorCodeInternal
	}
	domainErr := apperrors.New(
		statusCode,
		code,
		message,
		apperrors.WithDetails(details),
	)
	abortWithDomainError(c, domainErr)
}

// abortWithDomainError 将领域错误交由统一中间件处理
func abortWithDomainError(c *gin.Context, err apperrors.DomainError) {
	if err == nil {
		return
	}
	_ = c.Error(err)
	c.Abort()
}

func buildErrorOptions(code string) []apperrors.Option {
	if code == "" {
		return nil
	}
	return []apperrors.Option{apperrors.WithCode(code)}
}
