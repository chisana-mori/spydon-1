package api

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

// CASLogin 触发CAS登录
func (h *AuthHandler) CASLogin(c *gin.Context) {
	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		Error(c, http.StatusNotImplemented, "CAS_NOT_ENABLED", "CAS 未启用")
		return
	}

	var query struct {
		Service string `form:"service"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	redirectTarget := strings.TrimSpace(query.Service)
	if redirectTarget == "" {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}

	secure := isSecureRequest(c.Request)
	encodedTarget := base64.URLEncoding.EncodeToString([]byte(redirectTarget))
	c.SetCookie(casRedirectCookieName, encodedTarget, 600, "/", "", secure, true)

	if strings.TrimSpace(h.cfg.CAS.ServerURL) == "" {
		InternalError(c, "CAS_SERVER_NOT_CONFIGURED", "CAS 服务地址未配置")
		return
	}

	callbackURL, err := h.buildCallbackURL(c.Request)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "BUILD_CALLBACK_URL_FAILED", "构造回调地址失败", err.Error())
		return
	}

	loginURL, err := buildCASLoginURL(h.cfg.CAS.ServerURL, callbackURL.String())
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "BUILD_CAS_LOGIN_URL_FAILED", "构造CAS登录地址失败", err.Error())
		return
	}

	c.Redirect(http.StatusFound, loginURL)
}

// CASCallback 处理CAS回调
func (h *AuthHandler) CASCallback(c *gin.Context) {
	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		Error(c, http.StatusNotImplemented, "CAS_NOT_ENABLED", "CAS 未启用")
		return
	}

	var query struct {
		Ticket string `form:"ticket"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	ticket := strings.TrimSpace(query.Ticket)
	if ticket == "" {
		h.casClient.RedirectToLogin(c.Writer, c.Request)
		return
	}

	authResp, err := h.validateCASTicket(c.Request, ticket)
	if err != nil {
		ErrorWithDetails(c, http.StatusUnauthorized, "CAS_TICKET_VALIDATION_FAILED", "CAS ticket 验证失败", err.Error())
		return
	}

	username := authResp.User
	attributes := authResp.Attributes
	loginResp, err := h.authService.LoginWithCAS(username, attributes)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "CAS_LOGIN_FAILED", "CAS 登录失败", err.Error())
		return
	}

	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true)
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)
	h.setKiteCookie(c, loginResp.User, loginResp.RefreshToken, loginResp.ExpiresAt)

	redirectTarget := ""
	if value, err := c.Cookie(casRedirectCookieName); err == nil && value != "" {
		if decoded, decodeErr := base64.URLEncoding.DecodeString(value); decodeErr == nil {
			redirectTarget = string(decoded)
		}
	}
	if redirectTarget == "" {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}
	h.clearRedirectCookie(c)

	c.Redirect(http.StatusFound, redirectTarget)
}

// CASValidate 验证 CAS ticket (用于前端 callback 调用)
func (h *AuthHandler) CASValidate(c *gin.Context) {
	// 这个方法应该通过 CAS 中间件调用，而不是直接处理
	// 实际的验证逻辑在 CASCallback 中处理
	Error(c, http.StatusNotImplemented, "USE_CAS_CALLBACK", "请使用 /auth/cas/callback 接口")
}

// CASLogout 注销CAS并清理本地会话
func (h *AuthHandler) CASLogout(c *gin.Context) {
	var query struct {
		Redirect string `form:"redirect"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	redirectTarget := strings.TrimSpace(query.Redirect)
	if redirectTarget == "" && h.cfg != nil {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}

	h.performLocalLogout(c)

	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		c.Redirect(http.StatusFound, redirectTarget)
		return
	}

	logoutURL, err := h.casClient.LogoutUrlForRequest(c.Request)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "BUILD_CAS_LOGOUT_URL_FAILED", "构造CAS登出地址失败", err.Error())
		return
	}

	if redirectTarget != "" {
		if parsed, parseErr := url.Parse(logoutURL); parseErr == nil {
			query := parsed.Query()
			query.Set("service", redirectTarget)
			parsed.RawQuery = query.Encode()
			logoutURL = parsed.String()
		}
	}

	c.Redirect(http.StatusFound, logoutURL)
}

func (h *AuthHandler) validateCASTicket(r *http.Request, ticket string) (*cas.AuthenticationResponse, error) {
	serviceURL, err := h.buildCallbackURL(r)
	if err != nil {
		return nil, fmt.Errorf("构造回调地址失败: %w", err)
	}

	casBase := strings.TrimSpace(h.cfg.CAS.ServerURL)
	if casBase == "" {
		return nil, fmt.Errorf("CAS 服务地址未配置")
	}

	casURL, err := url.Parse(casBase)
	if err != nil {
		return nil, fmt.Errorf("CAS 服务地址无效: %w", err)
	}

	// 创建自定义HTTP客户端，根据环境决定是否跳过TLS验证
	tlsConfig := &tls.Config{}
	// 仅在开发环境跳过TLS验证
	if h.cfg.Environment == "development" {
		tlsConfig.InsecureSkipVerify = true
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

	validator := cas.NewServiceTicketValidator(client, casURL)
	resp, err := validator.ValidateTicket(serviceURL, ticket)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("CAS 未返回有效认证信息")
	}

	return resp, nil
}

func buildCASLoginURL(serverURL, serviceURL string) (string, error) {
	server := strings.TrimRight(serverURL, "/") + "/login"
	loginURL, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	query := loginURL.Query()
	query.Set("service", serviceURL)
	loginURL.RawQuery = query.Encode()
	return loginURL.String(), nil
}
