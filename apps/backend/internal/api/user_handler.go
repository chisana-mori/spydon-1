package api

import (
	"net/http"
	"strconv"

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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := h.userService.GetUsers(page, limit, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取用户列表失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetUser 获取单个用户详情（管理员功能）
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "用户不存在",
			"details": err.Error(),
		})
		return
	}

	// 对敏感信息进行掩码处理
	maskedUsername, maskedEmail, maskedName := utils.MaskUserInfo(user.Username, user.Email, user.Name)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":             user.ID.String(),
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
		},
	})
}

// SetUserAdmin 设置用户管理员权限（管理员功能）
func (h *UserHandler) SetUserAdmin(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		IsAdmin bool `json:"is_admin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 检查是否至少保留一个管理员
	if !req.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "检查管理员数量失败",
				"details": err.Error(),
			})
			return
		}

		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "系统至少需要保留一个管理员",
			})
			return
		}
	}

	if err := h.userService.SetUserAdmin(userID, req.IsAdmin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "设置用户权限失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户权限设置成功",
		"data": gin.H{
			"user_id":  userID,
			"is_admin": req.IsAdmin,
		},
	})
}

// DeleteUser 删除用户（管理员功能）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// 获取当前用户ID，防止删除自己
	currentUserID, exists := c.Get("user_id")
	if exists && currentUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不能删除自己的账号",
		})
		return
	}

	// 检查要删除的用户是否是管理员
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "用户不存在",
			"details": err.Error(),
		})
		return
	}

	// 如果是管理员，检查是否至少保留一个管理员
	if user.IsAdmin {
		adminCount, err := h.userService.GetAdminCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "检查管理员数量失败",
				"details": err.Error(),
			})
			return
		}

		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "系统至少需要保留一个管理员，无法删除",
			})
			return
		}
	}

	if err := h.userService.DeleteUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "删除用户失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户删除成功",
	})
}
