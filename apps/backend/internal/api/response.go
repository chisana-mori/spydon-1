package api

import (
	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
)

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

func (r PaginatedResponse) toMap() gin.H {
	result := gin.H{}
	if r.Code != "" {
		result["code"] = r.Code
	}
	if r.Message != "" {
		result["message"] = r.Message
	}
	if r.Error != "" {
		result["error"] = r.Error
	}
	if r.Data != nil {
		result["data"] = r.Data
	}
	if r.Pagination != nil {
		result["pagination"] = r.Pagination
	}
	for key, value := range r.Extras {
		result[key] = value
	}
	return result
}

// Success 成功响应
func Success(c *gin.Context, data interface{}, extras ...gin.H) {
	resp := Response{
		Data: data,
	}
	if len(extras) > 0 {
		resp.Extras = extras[0]
	}
	c.AbortWithStatusJSON(constants.StatusOK, resp.toMap())
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}, extras ...gin.H) {
	resp := Response{
		Message: message,
		Data:    data,
	}
	if len(extras) > 0 {
		resp.Extras = extras[0]
	}
	c.AbortWithStatusJSON(constants.StatusOK, resp.toMap())
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.AbortWithStatusJSON(constants.StatusCreated, Response{
		Data: data,
	}.toMap())
}

// SuccessPaginated 分页成功响应
func SuccessPaginated(c *gin.Context, data interface{}, pagination *Pagination, extras ...gin.H) {
	resp := PaginatedResponse{
		Data:       data,
		Pagination: pagination,
	}
	if len(extras) > 0 {
		resp.Extras = extras[0]
	}
	c.AbortWithStatusJSON(constants.StatusOK, resp.toMap())
}

// Error 错误响应
func Error(c *gin.Context, statusCode int, code string, message string) {
	if code == "" {
		code = constants.ErrorCodeInternal
	}
	domainErr := apperrors.New(
		statusCode,
		code,
		message,
	)
	AbortWithDomainError(c, domainErr)
}

// BadRequest 400错误
func BadRequest(c *gin.Context, code string, message string) {
	err := apperrors.BadRequest(
		message,
		nil,
		buildErrorOptions(code)...,
	)
	AbortWithDomainError(c, err)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, code string, message string) {
	err := apperrors.Unauthorized(
		message,
		buildErrorOptions(code)...,
	)
	AbortWithDomainError(c, err)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, code string, message string) {
	err := apperrors.Forbidden(
		message,
		buildErrorOptions(code)...,
	)
	AbortWithDomainError(c, err)
}

// NotFound 404错误
func NotFound(c *gin.Context, code string, message string) {
	opts := buildErrorOptions(code)
	if message != "" {
		opts = append(opts, apperrors.WithMessage(message))
	}
	err := apperrors.NotFound(
		"",
		opts...,
	)
	AbortWithDomainError(c, err)
}

// InternalError 500错误
func InternalError(c *gin.Context, code string, message string) {
	err := apperrors.New(
		constants.StatusInternalServerError,
		code,
		message,
	)
	AbortWithDomainError(c, err)
}

// ErrorWithDetails 带详细信息的错误响应
func ErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details interface{}) {
	if code == "" {
		code = constants.ErrorCodeInternal
	}
	domainErr := apperrors.New(
		statusCode,
		code,
		message,
		apperrors.WithDetails(details),
	)
	AbortWithDomainError(c, domainErr)
}

// AbortWithDomainError 将领域错误交由统一中间件处理
func AbortWithDomainError(c *gin.Context, err apperrors.DomainError) {
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
