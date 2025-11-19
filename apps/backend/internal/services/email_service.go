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
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"
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
	if s == nil || s.cfg == nil || !s.cfg.Email.Enabled {
		logger.S().Infow("跳过邮件发送", "module", "email", "reason", "disabled")
		return nil
	}
	if s.cfg.Email.SMTPHost == "" || s.cfg.Email.SMTPUser == "" || s.cfg.Email.SMTPPass == "" || s.cfg.Email.From == "" {
		logger.S().Warnw("邮件配置不完整，跳过发送", "module", "email")
		return nil
	}
	if to == "" {
		logger.S().Warnw("目标邮箱为空，跳过发送", "module", "email")
		return nil
	}

	subject := fmt.Sprintf("[RCA完成] %s | 集群: %s | 严重级别: %s", alert.Title, alert.ClusterID, alert.Severity)
	// 组装简要文本内容（避免外部模板依赖）
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
	body.WriteString(fmt.Sprintf("集群: %s\n", alert.ClusterID))
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

	// SMTP 发送
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.cfg.Email.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	msg.WriteString(body.String())

	addr := net.JoinHostPort(s.cfg.Email.SMTPHost, strconv.Itoa(s.cfg.Email.SMTPPort))
	auth := smtp.PlainAuth("", s.cfg.Email.SMTPUser, s.cfg.Email.SMTPPass, s.cfg.Email.SMTPHost)

	// 支持 STARTTLS/直连TLS 场景：先尝试 TLS，失败则退回非TLS
	// 1) 直连TLS
	tlsConfig := &tls.Config{
		ServerName:         s.cfg.Email.SMTPHost,
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
		c, cerr := smtp.NewClient(conn, s.cfg.Email.SMTPHost)
		if cerr == nil {
			defer func() {
				if quitErr := c.Quit(); quitErr != nil {
					// 记录退出错误但不影响邮件发送
					logger.S().Warnw("SMTP客户端退出时发生错误", "module", "email", "error", quitErr)
				}
			}()
			if err = c.Auth(auth); err == nil {
				if err = c.Mail(s.cfg.Email.From); err == nil {
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

	// 2) 普通SMTP（可能由服务器升级到STARTTLS）
	// 直接使用 smtp.SendMail 简化
	if err := smtp.SendMail(addr, auth, s.cfg.Email.From, []string{to}, []byte(msg.String())); err != nil {
		// 再做一次兜底：非认证直连（某些内网MTA）
		logger.S().Errorw("SendMail失败，尝试非认证兜底", "module", "email", "error", err)
		dialer := &net.Dialer{}
		c, dErr := dialer.DialContext(ctx, "tcp", addr)
		if dErr != nil {
			return fmt.Errorf("连接SMTP失败: %w", err)
		}
		_ = c.Close()
		// 返回原始错误以便定位
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	return nil
}
