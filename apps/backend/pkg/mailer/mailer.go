// Package mailer 提供基于 gomail v2 的邮件发送工具
// 支持 HTML 邮件、附件、多收件人等功能
package mailer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/gomail.v2"
)

// Config 邮件服务器配置
type Config struct {
	Host     string // SMTP 服务器地址
	Port     int    // SMTP 端口 (25, 465, 587)
	Username string // 认证用户名
	Password string // 认证密码
	From     string // 发件人地址
	FromName string // 发件人名称（可选）
	UseTLS   bool   // 是否使用 TLS（端口 465 时推荐 true）
}

// Attachment 邮件附件
type Attachment struct {
	Filename    string    // 附件文件名
	ContentType string    // MIME 类型 (如 "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	Data        []byte    // 附件内容
	Reader      io.Reader // 附件读取器（与 Data 二选一）
}

// Message 邮件消息
type Message struct {
	To          []string          // 收件人列表
	Cc          []string          // 抄送列表
	Bcc         []string          // 密送列表
	Subject     string            // 邮件主题
	Body        string            // 纯文本正文
	HTMLBody    string            // HTML 正文（优先使用）
	Attachments []Attachment      // 附件列表
	Headers     map[string]string // 自定义邮件头
}

// Mailer 邮件发送器
type Mailer struct {
	config Config
	dialer *gomail.Dialer
}

// New 创建邮件发送器
func New(cfg Config) *Mailer {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	// 端口 465 通常使用 SSL/TLS
	if cfg.Port == 465 || cfg.UseTLS {
		d.SSL = true
	}

	return &Mailer{
		config: cfg,
		dialer: d,
	}
}

// Send 发送邮件
func (m *Mailer) Send(msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("收件人不能为空")
	}
	if msg.Subject == "" {
		return fmt.Errorf("邮件主题不能为空")
	}
	if msg.Body == "" && msg.HTMLBody == "" {
		return fmt.Errorf("邮件正文不能为空")
	}

	mail := gomail.NewMessage()

	// 设置发件人
	if m.config.FromName != "" {
		mail.SetAddressHeader("From", m.config.From, m.config.FromName)
	} else {
		mail.SetHeader("From", m.config.From)
	}

	// 设置收件人
	mail.SetHeader("To", msg.To...)

	// 设置抄送
	if len(msg.Cc) > 0 {
		mail.SetHeader("Cc", msg.Cc...)
	}

	// 设置密送
	if len(msg.Bcc) > 0 {
		mail.SetHeader("Bcc", msg.Bcc...)
	}

	// 设置主题
	mail.SetHeader("Subject", msg.Subject)

	// 设置自定义头
	for key, value := range msg.Headers {
		mail.SetHeader(key, value)
	}

	// 设置正文
	if msg.HTMLBody != "" {
		mail.SetBody("text/html", msg.HTMLBody)
		// 如果同时有纯文本，添加为备选
		if msg.Body != "" {
			mail.AddAlternative("text/plain", msg.Body)
		}
	} else {
		mail.SetBody("text/plain", msg.Body)
	}

	// 添加附件
	for _, att := range msg.Attachments {
		if err := m.attachFile(mail, att); err != nil {
			return fmt.Errorf("添加附件失败 [%s]: %w", att.Filename, err)
		}
	}

	// 发送邮件
	if err := m.dialer.DialAndSend(mail); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	return nil
}

// attachFile 添加附件到邮件
func (m *Mailer) attachFile(mail *gomail.Message, att Attachment) error {
	if att.Filename == "" {
		return fmt.Errorf("附件文件名不能为空")
	}

	var data []byte
	if att.Data != nil {
		data = att.Data
	} else if att.Reader != nil {
		var err error
		data, err = io.ReadAll(att.Reader)
		if err != nil {
			return fmt.Errorf("读取附件数据失败: %w", err)
		}
	} else {
		return fmt.Errorf("附件数据为空")
	}

	// 使用 gomail 的 AttachReader 方法
	mail.Attach(att.Filename, gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := io.Copy(w, bytes.NewReader(data))
		return err
	}))

	// 设置 Content-Type（如果指定）
	if att.ContentType != "" {
		mail.SetHeader("Content-Type", att.ContentType)
	}

	return nil
}

// SendSimple 简化的发送方法
func (m *Mailer) SendSimple(to []string, subject, body string) error {
	return m.Send(Message{
		To:      to,
		Subject: subject,
		Body:    body,
	})
}

// SendHTML 发送 HTML 邮件
func (m *Mailer) SendHTML(to []string, subject, htmlBody string) error {
	return m.Send(Message{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
	})
}

// SendWithAttachment 发送带附件的邮件
func (m *Mailer) SendWithAttachment(to []string, subject, htmlBody string, attachments []Attachment) error {
	return m.Send(Message{
		To:          to,
		Subject:     subject,
		HTMLBody:    htmlBody,
		Attachments: attachments,
	})
}

// ParseAddresses 解析逗号分隔的邮箱地址
func ParseAddresses(addressStr string) []string {
	if addressStr == "" {
		return nil
	}
	parts := strings.Split(addressStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr != "" {
			result = append(result, addr)
		}
	}
	return result
}

// NewAttachmentFromBytes 从字节数组创建附件
func NewAttachmentFromBytes(filename string, data []byte, contentType string) Attachment {
	return Attachment{
		Filename:    filename,
		Data:        data,
		ContentType: contentType,
	}
}

// NewExcelAttachment 创建 Excel 附件（.xlsx）
func NewExcelAttachment(filename string, data []byte) Attachment {
	return Attachment{
		Filename:    filename,
		Data:        data,
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
}

// NewPDFAttachment 创建 PDF 附件
func NewPDFAttachment(filename string, data []byte) Attachment {
	return Attachment{
		Filename:    filename,
		Data:        data,
		ContentType: "application/pdf",
	}
}
