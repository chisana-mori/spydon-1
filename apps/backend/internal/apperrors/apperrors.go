package apperrors

import (
	"errors"
	"fmt"
	"net/http"

	"robusta-web/backend/internal/constants"
)

// DomainError 统一领域错误接口
type DomainError interface {
	error
	Status() int
	Code() string
	Message() string
	Details() any
	Unwrap() error
}

type domainError struct {
	status  int
	code    string
	message string
	details any
	cause   error
}

// New 创建自定义领域错误
func New(status int, code, message string, opts ...Option) DomainError {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if code == "" {
		code = constants.ErrorCodeInternal
	}

	err := &domainError{
		status:  status,
		code:    code,
		message: message,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Wrap 包装底层错误
func Wrap(cause error, status int, code, message string, opts ...Option) DomainError {
	return New(status, code, message, append(opts, WithCause(cause))...)
}

// From 尝试将任意错误转换为领域错误
func From(err error) (DomainError, bool) {
	if err == nil {
		return nil, false
	}
	var domainErr DomainError
	if errors.As(err, &domainErr) {
		return domainErr, true
	}
	return nil, false
}

// EnsureDomainError 将普通错误转换为领域错误
func EnsureDomainError(err error) DomainError {
	if err == nil {
		return nil
	}
	if domainErr, ok := From(err); ok {
		return domainErr
	}
	return Wrap(err, http.StatusInternalServerError, constants.ErrorCodeInternal, "服务内部错误")
}

// 预定义错误构造器

func Validation(message string, details any, opts ...Option) DomainError {
	opts = append(opts, WithDetails(details))
	return New(http.StatusBadRequest, constants.ErrorCodeValidationFailed, message, opts...)
}

func BadRequest(message string, details any, opts ...Option) DomainError {
	opts = append(opts, WithDetails(details))
	return New(http.StatusBadRequest, constants.ErrorCodeBadRequest, message, opts...)
}

func Unauthorized(message string, opts ...Option) DomainError {
	return New(http.StatusUnauthorized, constants.ErrorCodeUnauthorized, message, opts...)
}

func Forbidden(message string, opts ...Option) DomainError {
	return New(http.StatusForbidden, constants.ErrorCodeForbidden, message, opts...)
}

func NotFound(resource string, opts ...Option) DomainError {
	msg := resource
	if msg == "" {
		msg = "资源不存在"
	} else {
		msg = fmt.Sprintf("%s不存在", resource)
	}
	return New(http.StatusNotFound, constants.ErrorCodeNotFound, msg, opts...)
}

func Conflict(message string, opts ...Option) DomainError {
	return New(http.StatusConflict, constants.ErrorCodeConflict, message, opts...)
}

// Option 自定义错误参数
type Option func(*domainError)

// WithCode 设置错误码
func WithCode(code string) Option {
	return func(err *domainError) {
		err.code = code
	}
}

// WithDetails 设置错误详情
func WithDetails(details any) Option {
	return func(err *domainError) {
		err.details = details
	}
}

// WithCause 设置底层错误
func WithCause(cause error) Option {
	return func(err *domainError) {
		err.cause = cause
	}
}

// WithMessage 覆盖默认消息
func WithMessage(message string) Option {
	return func(err *domainError) {
		err.message = message
	}
}

// WithStatus 覆盖 HTTP 状态码
func WithStatus(status int) Option {
	return func(err *domainError) {
		if status > 0 {
			err.status = status
		}
	}
}

func (e *domainError) Error() string {
	if e.cause != nil && e.message == "" {
		return e.cause.Error()
	}
	return e.message
}

func (e *domainError) Status() int {
	return e.status
}

func (e *domainError) Code() string {
	return e.code
}

func (e *domainError) Message() string {
	return e.message
}

func (e *domainError) Details() any {
	return e.details
}

func (e *domainError) Unwrap() error {
	return e.cause
}
