package httpx

import (
	"errors"
	"net/http"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BindJSON 绑定并校验 JSON 请求体
func BindJSON(c *gin.Context, target any) apperrors.DomainError {
	if err := c.ShouldBindJSON(target); err != nil {
		return translateBindingError(err)
	}
	return nil
}

// BindQuery 绑定并校验查询参数
func BindQuery(c *gin.Context, target any) apperrors.DomainError {
	if err := c.ShouldBindQuery(target); err != nil {
		return translateBindingError(err)
	}
	return nil
}

// BindURI 绑定并校验路径参数
func BindURI(c *gin.Context, target any) apperrors.DomainError {
	if err := c.ShouldBindUri(target); err != nil {
		return translateBindingError(err)
	}
	return nil
}

func translateBindingError(err error) apperrors.DomainError {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make(map[string]string, len(validationErrs))
		for _, fieldErr := range validationErrs {
			details[fieldErr.Field()] = fieldErr.Error()
		}
		return apperrors.Validation("请求参数校验失败", details, apperrors.WithCause(err))
	}

	return apperrors.New(
		http.StatusBadRequest,
		constants.ErrorCodeBadRequest,
		"请求参数解析失败",
		apperrors.WithDetails(err.Error()),
		apperrors.WithCause(err),
	)
}
