package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"
)

// EmailService 邮件发送服务
type EmailService struct {
	cfg *config.Config
}

// NewEmailService 创建EmailService
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendRCAResultEmail 发送RCA结果邮件
// 为安全与健壮性考虑：
// - 若未启用或关键配置缺失，将记录日志并跳过发送
// - 简单文本邮件，避免复杂HTML依赖
func (s *EmailService) SendRCAResultEmail(to string, alert *models.Alert, rca *models.RCARun) error {
	if s == nil || s.cfg == nil || !s.cfg.ExternalDependencies.Email.Enabled {
		logger.S().Infow("跳过邮件发送", "module", "email", "reason", "disabled")
		return nil
	}
	if s.cfg.ExternalDependencies.Email.Host == "" || s.cfg.ExternalDependencies.Email.User == "" || s.cfg.ExternalDependencies.Email.Secret == "" || s.cfg.ExternalDependencies.Email.From == "" {
		logger.S().Warnw("邮件配置不完整，跳过发送", "module", "email")
		return nil
	}
	if to == "" {
		logger.S().Warnw("目标邮箱为空，跳过发送", "module", "email")
		return nil
	}

	subject := fmt.Sprintf("[RCA完成] %s | 集群: %s | 严重级别: %s", alert.Title, alert.ClusterName, alert.Severity)
	var summary string
	if rca.Summary != nil {
		summary = *rca.Summary
	}
	started := rca.StartedAt.Format(time.RFC3339)
	completed := ""
	if rca.CompletedAt != nil {
		completed = rca.CompletedAt.Format(time.RFC3339)
	}
	body := strings.Builder{}
	body.WriteString("RCA分析已完成\n\n")
	body.WriteString(fmt.Sprintf("告警标题: %s\n", alert.Title))
	body.WriteString(fmt.Sprintf("集群: %s\n", alert.ClusterName))
	body.WriteString(fmt.Sprintf("严重级别: %s\n", alert.Severity))
	body.WriteString(fmt.Sprintf("开始时间: %s\n", started))
	if completed != "" {
		body.WriteString(fmt.Sprintf("完成时间: %s\n", completed))
	}
	body.WriteString(fmt.Sprintf("状态: %s\n\n", rca.Status))
	if summary != "" {
		body.WriteString("[概要]\n")
		body.WriteString(summary)
		body.WriteString("\n\n")
	}
	if rca.ErrorMessage != nil && *rca.ErrorMessage != "" {
		body.WriteString("[错误]\n")
		body.WriteString(*rca.ErrorMessage)
		body.WriteString("\n\n")
	}
	body.WriteString("— 本邮件由系统自动发送 —\n")

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.cfg.ExternalDependencies.Email.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	msg.WriteString(body.String())

	addr := net.JoinHostPort(s.cfg.ExternalDependencies.Email.Host, strconv.Itoa(s.cfg.ExternalDependencies.Email.Port))
	auth := smtp.PlainAuth("", s.cfg.ExternalDependencies.Email.User, s.cfg.ExternalDependencies.Email.Secret, s.cfg.ExternalDependencies.Email.Host)

	tlsConfig := &tls.Config{
		ServerName:         s.cfg.ExternalDependencies.Email.Host,
		InsecureSkipVerify: false,            // 明确设置为false以确保安全
		MinVersion:         tls.VersionTLS12, // 设置最低TLS版本为1.2
	}
	ctx := context.Background()
	tlsDialer := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    tlsConfig,
	}
	conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
	if err == nil {
		c, cerr := smtp.NewClient(conn, s.cfg.ExternalDependencies.Email.Host)
		if cerr == nil {
			defer func() {
				if quitErr := c.Quit(); quitErr != nil {
					logger.S().Warnw("SMTP客户端退出时发生错误", "module", "email", "error", quitErr)
				}
			}()
			if err = c.Auth(auth); err == nil {
				if err = c.Mail(s.cfg.ExternalDependencies.Email.From); err == nil {
					if err = c.Rcpt(to); err == nil {
						wc, werr := c.Data()
						if werr == nil {
							if _, werr = wc.Write([]byte(msg.String())); werr == nil {
								_ = wc.Close()
								return nil
							}
							_ = wc.Close()
						}
					}
				}
			}
		}
	}

	if err := smtp.SendMail(addr, auth, s.cfg.ExternalDependencies.Email.From, []string{to}, []byte(msg.String())); err != nil {
		logger.S().Errorw("SendMail失败，尝试非认证兜底", "module", "email", "error", err)
		dialer := &net.Dialer{}
		c, dErr := dialer.DialContext(ctx, "tcp", addr)
		if dErr != nil {
			return fmt.Errorf("连接SMTP失败: %w", err)
		}
		_ = c.Close()
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	return nil
}
