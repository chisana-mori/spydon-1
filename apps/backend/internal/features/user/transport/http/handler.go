package http

import (
	"fmt"
	"net/http"

	"robusta-web/backend/internal/features/user/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/transport/httpx"
	"robusta-web/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// Handler 用户管理处理器
type Handler struct {
	userService *services.UserService
}

// New 创建用户管理处理器
func New(userService *services.UserService) *Handler {
	return &Handler{
		userService: userService,
	}
}

// RegisterRoutes 注册用户管理相关路由（管理员功能）
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	group := admin.Group("/users")
	{
		group.GET("", h.GetUsers)
		group.GET(":id", h.GetUser)
		group.PUT(":id/admin", h.SetUserAdmin)
		group.DELETE(":id", h.DeleteUser)
	}
}

// GetUsers 获取用户列表（管理员功能）
func (h *Handler) GetUsers(c *gin.Context) {
	params, derr := httpx.ParsePaginationParams(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	keyword := c.Query("keyword")

	users, total, svcErr := h.userService.GetUsers(params.Page, params.PageSize, keyword)
	if svcErr != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_USERS_FAILED", "获取用户列表失败", svcErr.Error())
		return
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, users, pagination)
}

// GetUser 获取单个用户详情（管理员功能）
func (h *Handler) GetUser(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	user, err := h.userService.GetUserByID(path.ID)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在", err.Error())
		return
	}

	// 对敏感信息进行掩码处理
	maskedUsername, maskedEmail, maskedName := utils.MaskUserInfo(user.Username, user.Email, user.Name)

	httpx.Success(c, gin.H{
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
func (h *Handler) SetUserAdmin(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	var req struct {
		IsAdmin *bool `json:"is_admin" binding:"required"`
	}

	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	if req.IsAdmin == nil {
		httpx.BadRequest(c, "INVALID_REQUEST", "缺少管理员标记")
		return
	}

	// 检查是否至少保留一个管理员
	if !*req.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			httpx.ErrorWithDetails(c, http.StatusInternalServerError, "CHECK_ADMIN_FAILED", "检查管理员数量失败", err.Error())
			return
		}

		if adminCount <= 1 {
			httpx.BadRequest(c, "LAST_ADMIN", "系统至少需要保留一个管理员")
			return
		}
	}

	if err := h.userService.SetUserAdmin(path.ID, *req.IsAdmin); err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "SET_ADMIN_FAILED", "设置用户权限失败", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "用户权限设置成功", gin.H{
		"user_id":  models.FormatID(path.ID),
		"is_admin": *req.IsAdmin,
	})
}

// DeleteUser 删除用户（管理员功能）
func (h *Handler) DeleteUser(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 获取当前用户ID，防止删除自己
	currentUserID, exists := c.Get("user_id")
	if exists {
		if uid, err := toUint64(currentUserID); err == nil && uid == path.ID {
			httpx.BadRequest(c, "CANNOT_DELETE_SELF", "不能删除自己的账号")
			return
		}
	}

	// 检查要删除的用户是否是管理员
	user, err := h.userService.GetUserByID(path.ID)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在", err.Error())
		return
	}

	// 如果是管理员，检查是否至少保留一个管理员
	if user.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			httpx.ErrorWithDetails(c, http.StatusInternalServerError, "CHECK_ADMIN_FAILED", "检查管理员数量失败", err.Error())
			return
		}

		if adminCount <= 1 {
			httpx.BadRequest(c, "LAST_ADMIN", "系统至少需要保留一个管理员，无法删除")
			return
		}
	}

	if err := h.userService.DeleteUser(path.ID); err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "DELETE_USER_FAILED", "删除用户失败", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "用户删除成功", nil)
}

// toUint64 将多种类型转换为 uint64
func toUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint:
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d cannot be converted to uint64", v)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d cannot be converted to uint64", v)
		}
		return uint64(v), nil
	case string:
		id, err := models.ParseID(v)
		if err != nil {
			return 0, err
		}
		return id, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for uint64 conversion", value)
	}
}
