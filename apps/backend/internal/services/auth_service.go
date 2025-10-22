package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	cas "gopkg.in/cas.v2"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db     *db.Database
	config *config.Config
	client *http.Client
}

// OIDCUserInfo OIDC用户信息
type OIDCUserInfo struct {
	Sub           string   `json:"sub"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Picture       string   `json:"picture"`
	Groups        []string `json:"groups"`
}

// TokenResponse OIDC token响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	IsAdmin  bool   `json:"is_admin"`
}

// NewAuthService 创建认证服务
func NewAuthService(database *db.Database, cfg *config.Config) *AuthService {
	return &AuthService{
		db:     database,
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAuthURL 获取OIDC认证URL
func (s *AuthService) GetAuthURL(state string) (string, error) {
	if s.config.OIDCIssuer == "" {
		return "", fmt.Errorf("OIDC未配置")
	}

	params := url.Values{
		"client_id":     {s.config.OIDCClientID},
		"response_type": {"code"},
		"scope":         {"openid profile email groups"},
		"redirect_uri":  {s.getRedirectURI()},
		"state":         {state},
	}

	authURL := fmt.Sprintf("%s/auth?%s", s.config.OIDCIssuer, params.Encode())
	return authURL, nil
}

// HandleCallback 处理OIDC回调
func (s *AuthService) HandleCallback(code, state string) (*LoginResponse, error) {
	// 验证state参数（防止CSRF攻击）
	if !s.validateState(state) {
		return nil, fmt.Errorf("无效的state参数")
	}

	// 交换授权码获取token
	tokenResp, err := s.exchangeCodeForToken(code)
	if err != nil {
		return nil, fmt.Errorf("交换token失败: %w", err)
	}

	// 获取用户信息
	userInfo, err := s.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 创建或更新用户
	user, err := s.createOrUpdateUser(userInfo)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 生成JWT token
	accessToken, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("生成JWT失败: %w", err)
	}

	// 生成refresh token
	refreshToken, err := s.generateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
			Picture:  user.Picture,
			IsAdmin:  user.IsAdmin,
		},
	}, nil
}

// LoginWithCAS 使用 CAS 登录并颁发本地令牌
func (s *AuthService) LoginWithCAS(username string, attributes cas.UserAttributes) (*LoginResponse, error) {
	if username == "" {
		return nil, fmt.Errorf("CAS 用户名为空")
	}

	user, err := s.createOrUpdateCASUser(username, attributes)
	if err != nil {
		return nil, fmt.Errorf("创建或更新 CAS 用户失败: %w", err)
	}

	accessToken, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("生成JWT失败: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
			Picture:  user.Picture,
			IsAdmin:  user.IsAdmin,
		},
	}, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(refreshToken string) (*LoginResponse, error) {
	// 验证refresh token
	userID, err := s.validateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("无效的refresh token: %w", err)
	}

	// 获取用户信息
	var user models.User
	if err := s.db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 生成新的JWT token
	accessToken, err := s.generateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("生成JWT失败: %w", err)
	}

	// 生成新的refresh token
	newRefreshToken, err := s.generateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Name:     user.Name,
			Picture:  user.Picture,
			IsAdmin:  user.IsAdmin,
		},
	}, nil
}

// Logout 登出
func (s *AuthService) Logout(refreshToken string) error {
	// 删除refresh token
	return s.revokeRefreshToken(refreshToken)
}

// GetUserByID 根据ID获取用户信息（用于获取最新的用户状态）
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	if err := s.db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// 私有方法

func (s *AuthService) getRedirectURI() string {
	// 这里应该根据环境配置返回正确的回调URL
	return "http://localhost:3000/auth/callback"
}

func (s *AuthService) validateState(state string) bool {
	// 这里应该验证state参数，防止CSRF攻击
	// 简化实现，实际应该存储和验证state
	return len(state) > 0
}

func (s *AuthService) exchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.getRedirectURI()},
		"client_id":     {s.config.OIDCClientID},
		"client_secret": {s.config.OIDCClientSecret},
	}

	tokenURL := fmt.Sprintf("%s/token", s.config.OIDCIssuer)
	resp, err := s.client.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token请求失败: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *AuthService) getUserInfo(accessToken string) (*OIDCUserInfo, error) {
	userInfoURL := fmt.Sprintf("%s/userinfo", s.config.OIDCIssuer)

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取用户信息失败: %d", resp.StatusCode)
	}

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (s *AuthService) createOrUpdateUser(oidcUser *OIDCUserInfo) (*models.User, error) {
	var user models.User

	// 尝试根据email查找现有用户
	err := s.db.DB.Where("email = ?", oidcUser.Email).First(&user).Error
	if err != nil {
		// 用户不存在，创建新用户
		// 检查是否是第一个用户，如果是则设置为管理员
		var userCount int64
		s.db.DB.Model(&models.User{}).Count(&userCount)
		isFirstUser := userCount == 0

		user = models.User{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			Username:      s.generateUsername(oidcUser.Email),
			Email:         oidcUser.Email,
			Name:          oidcUser.Name,
			Picture:       oidcUser.Picture,
			IsAdmin:       isFirstUser, // 第一个用户自动成为管理员
			EmailVerified: oidcUser.EmailVerified,
			Provider:      "oidc",
			ProviderID:    oidcUser.Sub,
		}

		if err := s.db.DB.Create(&user).Error; err != nil {
			return nil, err
		}
	} else {
		// 更新现有用户信息
		user.Name = oidcUser.Name
		user.Picture = oidcUser.Picture
		user.EmailVerified = oidcUser.EmailVerified
		user.LastLoginAt = &[]time.Time{time.Now()}[0]

		if err := s.db.DB.Save(&user).Error; err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (s *AuthService) createOrUpdateCASUser(username string, attributes cas.UserAttributes) (*models.User, error) {
	now := time.Now()
	emailAttr := firstCASAttribute(attributes, s.config.CAS.EmailAttribute, "mail", "email")
	email := s.deriveCASEmail(username, emailAttr)

	nameAttr := firstCASAttribute(attributes, s.config.CAS.NameAttribute, "displayName", "cn", "name")
	if nameAttr == "" {
		nameAttr = username
	}

	picture := firstCASAttribute(attributes, "picture", "avatar", "photo")
	// 检查CAS属性中是否包含管理员标识
	isAdmin := s.checkCASAdminRole(attributes)

	var user models.User
	err := s.db.DB.Where("username = ? OR email = ?", username, email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 检查是否是第一个用户，如果是则设置为管理员
		var userCount int64
		s.db.DB.Model(&models.User{}).Count(&userCount)
		isFirstUser := userCount == 0

		user = models.User{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			Username:      username,
			Email:         email,
			Name:          nameAttr,
			Picture:       picture,
			IsAdmin:       isFirstUser || isAdmin, // 第一个用户或CAS标识的管理员
			EmailVerified: true,
			Provider:      "cas",
			ProviderID:    username,
			LastLoginAt:   &now,
		}

		if err := s.db.DB.Create(&user).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		user.Email = email
		user.Name = nameAttr
		if picture != "" {
			user.Picture = picture
		}
		// 如果CAS标识为管理员，则更新管理员状态
		if isAdmin {
			user.IsAdmin = true
		}
		user.LastLoginAt = &now

		if err := s.db.DB.Save(&user).Error; err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (s *AuthService) generateUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return "user"
}

// checkCASAdminRole 检查CAS属性中是否包含管理员标识
func (s *AuthService) checkCASAdminRole(attributes cas.UserAttributes) bool {
	if attributes == nil {
		return false
	}

	// 检查配置的角色属性
	keys := []string{s.config.CAS.RolesAttribute, "roles", "groups"}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		values, ok := attributes[key]
		if !ok {
			continue
		}

		for _, value := range values {
			role := strings.ToLower(strings.TrimSpace(value))
			// 检查是否包含管理员相关的角色
			if role == "admin" || role == "administrator" || role == "administrators" {
				return true
			}
		}
	}

	return false
}

func firstCASAttribute(attributes cas.UserAttributes, keys ...string) string {
	if attributes == nil {
		return ""
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if values, ok := attributes[key]; ok {
			for _, value := range values {
				trimmed := strings.TrimSpace(value)
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}

	return ""
}

func (s *AuthService) deriveCASEmail(username, candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		return candidate
	}

	domain := strings.TrimSpace(s.config.CAS.DefaultEmailDomain)
	if domain == "" {
		domain = "cas.local"
	}

	return fmt.Sprintf("%s@%s", username, domain)
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":      user.ID.String(),
		"email":    user.Email,
		"name":     user.Name,
		"is_admin": user.IsAdmin,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iss":      "robusta-central-hub",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AuthService) generateRefreshToken(userID string) (string, error) {
	// 生成随机refresh token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	refreshToken := base64.URLEncoding.EncodeToString(bytes)

	// 存储refresh token到数据库
	token := models.RefreshToken{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		UserID:    uuid.MustParse(userID),
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30天过期
	}

	if err := s.db.DB.Create(&token).Error; err != nil {
		return "", err
	}

	return refreshToken, nil
}

func (s *AuthService) validateRefreshToken(refreshToken string) (string, error) {
	var token models.RefreshToken
	err := s.db.DB.Where("token = ? AND expires_at > ?", refreshToken, time.Now()).First(&token).Error
	if err != nil {
		return "", err
	}

	return token.UserID.String(), nil
}

func (s *AuthService) revokeRefreshToken(refreshToken string) error {
	return s.db.DB.Where("token = ?", refreshToken).Delete(&models.RefreshToken{}).Error
}
