package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	cas "gopkg.in/cas.v2"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db     *db.Database
	config *config.Config
	client *resty.Client
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
	c := resty.New().SetTimeout(30 * time.Second)
	if cfg.OIDCIssuer != "" {
		c.SetBaseURL(strings.TrimRight(cfg.OIDCIssuer, "/"))
	}
	return &AuthService{db: database, config: cfg, client: c}
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
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       models.FormatID(user.ID),
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

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       models.FormatID(user.ID),
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
	if queryErr := s.db.DB.Where("id = ?", userID).First(&user).Error; queryErr != nil {
		return nil, fmt.Errorf("用户不存在: %w", queryErr)
	}

	// 生成新的JWT token
	accessToken, err := s.generateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("生成JWT失败: %w", err)
	}

	// 生成新的refresh token
	newRefreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       models.FormatID(user.ID),
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
func (s *AuthService) GetUserByID(userID uint64) (*models.User, error) {
	var user models.User
	if err := s.db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// 私有方法

func (s *AuthService) getRedirectURI() string {
	redirect := strings.TrimSpace(s.config.OIDCRedirectURL)
	if redirect != "" {
		return redirect
	}
	if fallback := strings.TrimSpace(s.config.CAS.RedirectURL); fallback != "" {
		return fallback
	}
	// 作为兜底，保持历史默认值
	return "http://localhost:3000/auth/callback"
}

func (s *AuthService) validateState(state string) bool {
	// TODO: Implement proper state validation to prevent CSRF
	// Ideally, we should store the state in Redis/Cache with a short expiration
	// and verify it here.
	return len(state) > 0
}

func (s *AuthService) exchangeCodeForToken(code string) (*TokenResponse, error) {
	resp, err := s.client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"redirect_uri":  s.getRedirectURI(),
			"client_id":     s.config.OIDCClientID,
			"client_secret": s.config.OIDCClientSecret,
		}).
		SetResult(&TokenResponse{}).
		Post("/token")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("token请求失败: %d", resp.StatusCode())
	}
	result, ok := resp.Result().(*TokenResponse)
	if !ok || result == nil {
		return nil, fmt.Errorf("token响应解析失败")
	}
	return result, nil
}

func (s *AuthService) getUserInfo(accessToken string) (*OIDCUserInfo, error) {
	resp, err := s.client.R().SetAuthToken(accessToken).SetResult(&OIDCUserInfo{}).Get("/userinfo")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("获取用户信息失败: %d", resp.StatusCode())
	}
	info, ok := resp.Result().(*OIDCUserInfo)
	if !ok || info == nil {
		return nil, fmt.Errorf("用户信息解析失败")
	}
	return info, nil
}

func (s *AuthService) createOrUpdateUser(oidcUser *OIDCUserInfo) (*models.User, error) {
	var user models.User

	// 尝试根据email查找现有用户
	err := s.db.DB.Where("email = ?", oidcUser.Email).First(&user).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		// 用户不存在，创建新用户
		isFirst, firstErr := s.isFirstUser()
		if firstErr != nil {
			return nil, firstErr
		}

		user = models.User{
			Username:      s.generateUsername(oidcUser.Email),
			Email:         oidcUser.Email,
			Name:          oidcUser.Name,
			Picture:       oidcUser.Picture,
			IsAdmin:       isFirst, // 第一个用户自动成为管理员
			EmailVerified: oidcUser.EmailVerified,
			Provider:      constants.AuthProviderOIDC,
			ProviderID:    oidcUser.Sub,
		}

		if createErr := s.db.DB.Create(&user).Error; createErr != nil {
			return nil, createErr
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
	queryErr := s.db.DB.Where("username = ? OR email = ?", username, email).First(&user).Error
	if errors.Is(queryErr, gorm.ErrRecordNotFound) {
		// 检查是否是第一个用户，如果是则设置为管理员
		isFirst, firstErr := s.isFirstUser()
		if firstErr != nil {
			return nil, firstErr
		}

		user = models.User{
			Username:      username,
			Email:         email,
			Name:          nameAttr,
			Picture:       picture,
			IsAdmin:       isFirst || isAdmin, // 第一个用户或CAS标识的管理员
			EmailVerified: true,
			Provider:      constants.AuthProviderCAS,
			ProviderID:    username,
			LastLoginAt:   &now,
		}

		if createErr := s.db.DB.Create(&user).Error; createErr != nil {
			return nil, createErr
		}
	} else if queryErr != nil {
		return nil, queryErr
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

func (s *AuthService) isFirstUser() (bool, error) {
	var userCount int64
	if err := s.db.DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return false, err
	}
	return userCount == 0, nil
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
		"sub":      models.FormatID(user.ID),
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

func (s *AuthService) generateRefreshToken(userID uint64) (string, error) {
	// 生成随机refresh token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	refreshToken := base64.URLEncoding.EncodeToString(bytes)
	hashed := hashRefreshToken(refreshToken)

	// 存储refresh token到数据库
	token := models.RefreshToken{
		UserID:    userID,
		TokenHash: hashed,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30天过期
	}

	if err := s.db.DB.Create(&token).Error; err != nil {
		return "", err
	}

	return refreshToken, nil
}

func (s *AuthService) validateRefreshToken(refreshToken string) (uint64, error) {
	var token models.RefreshToken
	hashed := hashRefreshToken(refreshToken)
	now := time.Now()

	err := s.db.DB.Where("token = ? AND expires_at > ?", hashed, now).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 兼容旧数据：尝试使用明文查询并升级为哈希存储
			legacyErr := s.db.DB.Where("token = ? AND expires_at > ?", refreshToken, now).First(&token).Error
			if legacyErr != nil {
				return 0, legacyErr
			}
			if updateErr := s.db.DB.Model(&models.RefreshToken{}).Where("id = ?", token.ID).Update("token", hashed).Error; updateErr != nil {
				return 0, updateErr
			}
			token.TokenHash = hashed
		} else {
			return 0, err
		}
	}

	if subtle.ConstantTimeCompare([]byte(token.TokenHash), []byte(hashed)) != 1 {
		return 0, gorm.ErrRecordNotFound
	}

	return token.UserID, nil
}

func (s *AuthService) revokeRefreshToken(refreshToken string) error {
	hashed := hashRefreshToken(refreshToken)
	result := s.db.DB.Where("token = ?", hashed).Delete(&models.RefreshToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 兼容旧数据
		legacy := s.db.DB.Where("token = ?", refreshToken).Delete(&models.RefreshToken{})
		return legacy.Error
	}
	return nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
