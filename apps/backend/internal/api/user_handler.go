package api

import (
	"net/http"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"
	"robusta-web/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUsers 获取用户列表（管理员功能）
func (h *UserHandler) GetUsers(c *gin.Context) {
	// 解析查询参数
	params, err := ParsePaginationParams(c)
	if err != nil {
		AbortWithDomainError(c, err)
		return
	}
	keyword := c.Query("keyword")

	users, total, svcErr := h.userService.GetUsers(params.Page, params.PageSize, keyword)
	if svcErr != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_USERS_FAILED", "获取用户列表失败", svcErr.Error())
		return
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, users, pagination)
}

// GetUser 获取单个用户详情（管理员功能）
func (h *UserHandler) GetUser(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	user, err := h.userService.GetUserByID(path.ID)
	if err != nil {
		ErrorWithDetails(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在", err.Error())
		return
	}

	// 对敏感信息进行掩码处理
	maskedUsername, maskedEmail, maskedName := utils.MaskUserInfo(user.Username, user.Email, user.Name)

	Success(c, gin.H{
		"id":             models.FormatID(user.ID),
		"username":       maskedUsername,
		"email":          maskedEmail,
		"name":           maskedName,
		"picture":        user.Picture,
		"is_admin":       user.IsAdmin,
		"email_verified": user.EmailVerified,
		"provider":       user.Provider,
		"last_login_at":  user.LastLoginAt,
		"created_at":     user.CreatedAt,
		"updated_at":     user.UpdatedAt,
	})
}

// SetUserAdmin 设置用户管理员权限（管理员功能）
func (h *UserHandler) SetUserAdmin(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	var req struct {
		IsAdmin *bool `json:"is_admin" binding:"required"`
	}

	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	if req.IsAdmin == nil {
		BadRequest(c, "INVALID_REQUEST", "缺少管理员标记")
		return
	}

	// 检查是否至少保留一个管理员
	if !*req.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			ErrorWithDetails(c, http.StatusInternalServerError, "CHECK_ADMIN_FAILED", "检查管理员数量失败", err.Error())
			return
		}

		if adminCount <= 1 {
			BadRequest(c, "LAST_ADMIN", "系统至少需要保留一个管理员")
			return
		}
	}

	if err := h.userService.SetUserAdmin(path.ID, *req.IsAdmin); err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "SET_ADMIN_FAILED", "设置用户权限失败", err.Error())
		return
	}

	SuccessWithMessage(c, "用户权限设置成功", gin.H{
		"user_id":  models.FormatID(path.ID),
		"is_admin": *req.IsAdmin,
	})
}

// DeleteUser 删除用户（管理员功能）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 获取当前用户ID，防止删除自己
	currentUserID, exists := c.Get("user_id")
	if exists {
		if uid, err := toUint64(currentUserID); err == nil && uid == path.ID {
			BadRequest(c, "CANNOT_DELETE_SELF", "不能删除自己的账号")
			return
		}
	}

	// 检查要删除的用户是否是管理员
	user, err := h.userService.GetUserByID(path.ID)
	if err != nil {
		ErrorWithDetails(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在", err.Error())
		return
	}

	// 如果是管理员，检查是否至少保留一个管理员
	if user.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			ErrorWithDetails(c, http.StatusInternalServerError, "CHECK_ADMIN_FAILED", "检查管理员数量失败", err.Error())
			return
		}

		if adminCount <= 1 {
			BadRequest(c, "LAST_ADMIN", "系统至少需要保留一个管理员，无法删除")
			return
		}
	}

	if err := h.userService.DeleteUser(path.ID); err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "DELETE_USER_FAILED", "删除用户失败", err.Error())
		return
	}

	SuccessWithMessage(c, "用户删除成功", nil)
}
