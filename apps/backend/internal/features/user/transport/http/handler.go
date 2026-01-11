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

func New(userService *services.UserService) *Handler {
	return &Handler{
		userService: userService,
	}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	group := admin.Group("/users")
	{
		group.GET("", h.GetUsers)
		group.GET(":id", h.GetUser)
		group.PUT(":id/admin", h.SetUserAdmin)
		group.DELETE(":id", h.DeleteUser)
	}
}

// GetUsers 获取用户列表
// @Summary 分页查询系统用户
// @Description 管理员权限接口，用于分页检索平台所有已注册的用户信息。支持通过关键字对用户名、姓名或电子邮件进行模糊匹配过滤。该功能为用户管理界面提供了核心数据支撑，方便管理员掌握平台用户规模及详情。
// @Tags Admin,User
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "搜素关键字"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users [get]
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

// GetUser 获取单个用户详情
// @Summary 获取特定用户详细资料
// @Description 根据唯一ID获取特定用户的完整档案信息。为了保护隐私，返回的数据会经过掩码处理（如隐藏部分邮箱和手机号内容）。接口涵盖了用户的登录历史、所属权限制、注册时间以及最后一次活跃的时间戳。
// @Tags Admin,User
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 404 {object} httpx.ErrorResponse
// @Router /admin/users/{id} [get]
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

// SetUserAdmin 设置用户管理员权限
// @Summary 调整用户管理员角色
// @Description 授予或撤销指定用户的超级管理员权限。系统强制要求至少保留一个活跃的管理员，以防权限配置错误导致平台陷入不可管理状态。该操作会直接影响对应用户登录后的菜单展示内容和接口访问权限范围。
// @Tags Admin,User
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body object true "权限更新参数 (is_admin boolean)"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users/{id}/admin [put]
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

// DeleteUser 删除用户
// @Summary 物理删除用户账号
// @Description 从系统中彻底移除指定的用户记录。接口内置了安全保护机制，禁止删除当前正在操作的登录账号，且禁止删除系统最后一个管理员。删除操作不可逆，执行后该用户关联的所有私有配置和API Key将一并失效。
// @Tags Admin,User
// @Param id path int true "用户ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	currentUserID, exists := c.Get("user_id")
	if exists {
		if uid, err := toUint64(currentUserID); err == nil && uid == path.ID {
			httpx.BadRequest(c, "CANNOT_DELETE_SELF", "不能删除自己的账号")
			return
		}
	}

	user, err := h.userService.GetUserByID(path.ID)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在", err.Error())
		return
	}

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
