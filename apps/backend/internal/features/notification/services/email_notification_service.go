package services

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/mailer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"gorm.io/gorm"
)

// K8sResourceFetcher K8s 资源获取接口（用于动态获取 nodes/pods/deployments）
type K8sResourceFetcher interface {
	ListNodes(clusterName string) ([]corev1.Node, error)
	ListPodsOnNodes(clusterName string, nodeNames []string) ([]corev1.Pod, error)
	ListDeployments(clusterName string, namespace string) ([]appsv1.Deployment, error)
}

// EmailNotificationService 邮件通知发送服务
type EmailNotificationService struct {
	db              *db.Database
	cfg             *config.Config
	emailService    *services.EmailService
	templateService *EmailTemplateService
	resourceFetcher K8sResourceFetcher
}

// NewEmailNotificationService 创建邮件通知服务
func NewEmailNotificationService(
	database *db.Database,
	cfg *config.Config,
	emailService *services.EmailService,
	templateService *EmailTemplateService,
) *EmailNotificationService {
	return &EmailNotificationService{
		db:              database,
		cfg:             cfg,
		emailService:    emailService,
		templateService: templateService,
	}
}

// SetResourceFetcher 设置资源获取器（注入 NodeSyncManager）
func (s *EmailNotificationService) SetResourceFetcher(fetcher K8sResourceFetcher) {
	s.resourceFetcher = fetcher
}

// PreviewEmail 预览邮件
func (s *EmailNotificationService) PreviewEmail(req PreviewEmailRequest) (*PreviewEmailResponse, error) {
	// 获取模板
	var tpl models.EmailTemplate
	if err := s.db.DB.First(&tpl, req.TemplateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("模板不存在")
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	if !tpl.IsEnabled {
		return nil, fmt.Errorf("模板已禁用")
	}

	// 获取受影响资源
	affectedResources := s.buildAffectedResources(req.ClusterName, req.Nodes)

	// 构建模板数据（包含受影响资源）
	data := s.buildTemplateData(req.ClusterName, req.Nodes, req.Params, affectedResources)

	// 渲染标题
	subject, err := s.renderTemplate("subject", tpl.Title, data)
	if err != nil {
		return nil, fmt.Errorf("渲染标题失败: %w", err)
	}

	// 渲染正文（受影响资源已包含在模板数据中）
	body, err := s.renderTemplate("body", tpl.Body, data)
	if err != nil {
		return nil, fmt.Errorf("渲染正文失败: %w", err)
	}

	// 生成附件文件名
	attachmentName := fmt.Sprintf("affected_resources_%s.xlsx", time.Now().Format("20060102_150405"))

	return &PreviewEmailResponse{
		Subject:           subject,
		HTMLBody:          body,
		AffectedResources: affectedResources,
		AttachmentName:    attachmentName,
	}, nil
}

// SendEmail 发送邮件
func (s *EmailNotificationService) SendEmail(req SendEmailRequest) error {
	// 直接使用请求中的 标题/正文
	// 前端已经完成了预览和确认，这里直接发送
	subject := req.Subject
	htmlBody := req.Body

	if subject == "" || htmlBody == "" {
		// 虽然前端应该保证不为空，但为了健壮性，如果为空可以报错或者尝试回退（但不建议回退到模板计算，保持逻辑简单）
		return fmt.Errorf("邮件标题或正文不能为空")
	}

	// 合并收件人
	allRecipients := s.mergeRecipients(req.Recipients)
	if len(allRecipients) == 0 {
		return fmt.Errorf("收件人列表为空")
	}

	// 准备附件列表
	// 直接使用前端传递的附件列表（包含之前预览生成的 Excel）
	var attachments []AttachFile
	if len(req.AttachFiles) > 0 {
		attachments = append(attachments, req.AttachFiles...)
	}

	// 发送邮件
	// Log count of attachments?

	for _, recipient := range allRecipients {
		if err := s.sendSingleEmail(recipient, subject, htmlBody, attachments); err != nil {
			logger.S().Errorw("发送邮件失败",
				"recipient", recipient,
				"error", err,
			)
			// 继续发送给其他收件人
		}
	}

	logger.S().Infow("邮件发送完成",
		"template_id", req.TemplateID,
		"recipients_count", len(allRecipients),
		"attachment_count", len(attachments),
	)

	return nil
}

// GetAffectedResources 获取受影响的资源
func (s *EmailNotificationService) GetAffectedResources(req GetAffectedResourcesRequest) ([]AffectedResource, error) {
	return s.buildAffectedResources(req.ClusterName, req.Nodes), nil
}

// buildAffectedResources 构建受影响资源列表（用于 Excel 附件和 affected_resources_table）
func (s *EmailNotificationService) buildAffectedResources(clusterName string, nodes []string) []AffectedResource {
	resources := make([]AffectedResource, 0, len(nodes))

	// 1. 获取节点信息
	for _, node := range nodes {
		resources = append(resources, AffectedResource{
			Type: "node",
			Name: node,
		})
	}

	// 2. 如果有 resourceFetcher，获取更详细的 Pod 信息
	if s.resourceFetcher != nil && clusterName != "" {
		pods, err := s.resourceFetcher.ListPodsOnNodes(clusterName, nodes)
		if err != nil {
			logger.S().Warnw("获取 Pods 失败", "cluster", clusterName, "error", err)
		} else {
			for _, pod := range pods {
				resources = append(resources, AffectedResource{
					Type:      "pod",
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Status:    string(pod.Status.Phase),
					IP:        pod.Status.PodIP,
				})
			}
		}
	}

	return resources
}

// buildTemplateData 构建模板数据（包含动态资源：Nodes, Pods, Deployments）
func (s *EmailNotificationService) buildTemplateData(clusterName string, nodes []string, params map[string]interface{}, affectedResources []AffectedResource) map[string]interface{} {
	data := make(map[string]interface{})

	// 复制用户参数
	for k, v := range params {
		data[k] = v
	}

	// 添加系统参数
	data["cluster"] = clusterName
	data["nodes"] = nodes
	data["node_count"] = len(nodes)
	data["affected_resources"] = affectedResources
	data["resource_count"] = len(affectedResources)

	// 生成受影响资源的 HTML 表格
	data["affected_resources_table"] = s.generateResourcesHTMLTable(affectedResources)

	// 动态获取资源
	if s.resourceFetcher != nil && clusterName != "" {
		s.enrichWithK8sResources(data, clusterName, nodes)
	}

	return data
}

func (s *EmailNotificationService) enrichWithK8sResources(data map[string]interface{}, clusterName string, nodes []string) {
	// Nodes
	k8sNodes, err := s.resourceFetcher.ListNodes(clusterName)
	if err == nil {
		data["Nodes"] = s.mapNodes(k8sNodes, nodes)
	}

	// Pods
	pods, err := s.resourceFetcher.ListPodsOnNodes(clusterName, nodes)
	if err == nil {
		podInfos := s.mapPods(pods)
		data["Pods"] = podInfos
		data["pod_count"] = len(podInfos)
	}

	// Deployments
	deploys, err := s.resourceFetcher.ListDeployments(clusterName, "")
	if err == nil {
		deployInfos := s.mapDeployments(deploys)
		data["Deployments"] = deployInfos
		data["deployment_count"] = len(deployInfos)
	}
}

func (s *EmailNotificationService) mapNodes(k8sNodes []corev1.Node, filterNodes []string) []NodeInfo {
	nodeSet := make(map[string]bool)
	for _, n := range filterNodes {
		nodeSet[n] = true
	}

	nodeInfos := make([]NodeInfo, 0, len(k8sNodes))
	for _, n := range k8sNodes {
		if len(filterNodes) > 0 && !nodeSet[n.Name] {
			continue
		}

		nodeInfo := NodeInfo{Name: n.Name}

		// Status
		for _, cond := range n.Status.Conditions {
			if cond.Type == "Ready" {
				nodeInfo.Status = "NotReady"
				if cond.Status == "True" {
					nodeInfo.Status = "Ready"
				}
				break
			}
		}

		// IP
		for _, addr := range n.Status.Addresses {
			if addr.Type == "InternalIP" {
				nodeInfo.IP = addr.Address
				break
			}
		}
		nodeInfos = append(nodeInfos, nodeInfo)
	}
	return nodeInfos
}

func (s *EmailNotificationService) mapPods(pods []corev1.Pod) []PodInfo {
	podInfos := make([]PodInfo, 0, len(pods))
	for _, pod := range pods {
		podInfos = append(podInfos, PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			NodeName:  pod.Spec.NodeName,
			Status:    string(pod.Status.Phase),
			IP:        pod.Status.PodIP,
		})
	}
	return podInfos
}

func (s *EmailNotificationService) mapDeployments(deploys []appsv1.Deployment) []DeploymentInfo {
	deployInfos := make([]DeploymentInfo, 0, len(deploys))
	for _, d := range deploys {
		deployInfos = append(deployInfos, DeploymentInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Replicas:  *d.Spec.Replicas,
			Ready:     d.Status.ReadyReplicas,
		})
	}
	return deployInfos
}

// generateResourcesHTMLTable 生成受影响资源的 HTML 表格
func (s *EmailNotificationService) generateResourcesHTMLTable(resources []AffectedResource) template.HTML {
	if len(resources) == 0 {
		return template.HTML("<p>暂无受影响资源</p>")
	}

	var sb strings.Builder
	sb.WriteString(`<table border="1" cellpadding="8" cellspacing="0" style="border-collapse: collapse; width: 100%;">`)
	sb.WriteString(`<thead style="background-color: #f5f5f5;">`)
	sb.WriteString(`<tr><th>类型</th><th>名称</th><th>命名空间</th><th>状态</th><th>IP</th><th>应用</th></tr>`)
	sb.WriteString(`</thead><tbody>`)

	for _, r := range resources {
		sb.WriteString("<tr>")
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.Type))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.Name))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.Namespace))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.Status))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.IP))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", r.App))
		sb.WriteString("</tr>")
	}

	sb.WriteString(`</tbody></table>`)
	return template.HTML(sb.String())
}

// renderTemplate 渲染模板
func (s *EmailNotificationService) renderTemplate(name, templateStr string, data map[string]interface{}) (string, error) {
	tpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// mergeRecipients 合并并去重收件人
func (s *EmailNotificationService) mergeRecipients(recipients []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, r := range recipients {
		// 支持逗号分隔的地址
		addrs := ParseAddresses(r)
		for _, addr := range addrs {
			addr = strings.TrimSpace(strings.ToLower(addr))
			if addr != "" && !seen[addr] {
				seen[addr] = true
				result = append(result, addr)
			}
		}
	}

	return result
}

// sendSingleEmail 发送单封邮件（支持 HTML 正文和多个附件）
// attachments: 包含 Name 和 Content (Content 可能是 base64 或者是 raw string，取决于来源)
// 对于 Excel 生成的，我们暂时将它转为 string 存储在 AttachFile Content 中?
// 不，AttachFile Content 在 JSON 中通常是 Base64。但如果是内部生成的 Excel []byte，我们需要区分。
// 为了简化，我们重新定义一个内部使用的 Attachment struct 或者在 sendSingleEmail 中处理。
// 让我们修改 sendSingleEmail 接受 []AttachFile，但要注意 AttachFile.Content 的含义。
// 假设前端传来的 AttachFile.Content 是 Base64。
// 后端生成的 Excel 是 []byte。
// 我们可以在调用前统一处理：
//  - 前端来的: 解码 Base64 -> []byte
//  - 后端Excel: 直接是 []byte
// 为了避免混淆，SendEmail 函数里应该负责解码/转换，传给 sendSingleEmail 的是统一的结构，比如 mailer.Attachment。
// 但这里为了最小改动，我们让 sendSingleEmail 接收 []AttachFile，并在内部判断/转换。
// 可是 AttachFile 定义在 DTO 中，是 string content。
// 这样吧：我们先修改 sendSingleEmail 签名，接受 []AttachFile。
// 对于 Excel，我们在 SendEmail 中将 []byte 转为 Base64 string 放入 AttachFile，
// 并在 sendSingleEmail 中统一 Base64 解码。这样最一致。

func (s *EmailNotificationService) sendSingleEmail(to, subject, htmlBody string, attachments []AttachFile) error {
	if s.cfg == nil || !s.cfg.ExternalDependencies.Email.Enabled {
		logger.S().Infow("邮件功能未启用，跳过发送", "to", to)
		return nil
	}

	// 验证配置
	if s.cfg.ExternalDependencies.Email.Host == "" || s.cfg.ExternalDependencies.Email.From == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	// 创建 mailer
	m := mailer.New(mailer.Config{
		Host:     s.cfg.ExternalDependencies.Email.Host,
		Port:     s.cfg.ExternalDependencies.Email.Port,
		Username: s.cfg.ExternalDependencies.Email.User,
		Password: s.cfg.ExternalDependencies.Email.Secret,
		From:     s.cfg.ExternalDependencies.Email.From,
		FromName: s.cfg.ExternalDependencies.Email.FromName,
		UseTLS:   s.cfg.ExternalDependencies.Email.IsSSL,
	})

	// 构建消息
	msg := mailer.Message{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: htmlBody,
	}

	// 添加附件（如果有）
	if len(attachments) > 0 {
		var mailAttachments []mailer.Attachment
		for _, att := range attachments {
			// 1. 解码 Base64 内容
			contentBytes, err := base64.StdEncoding.DecodeString(att.Content)
			if err != nil {
				// Fallback: 如果解码失败，尝试作为普通字节处理（虽然预期是 Base64）
				contentBytes = []byte(att.Content)
			}

			// 2. 处理文件名后缀：如果没有 .xlsx/.xls 后缀，且这是一个 Excel 附件（这里假设所有附件都是 Excel，或者默认添加 .xlsx？）
			// 根据用户需求："前端传递的...只有文件名...需要后端处理文件名+.xlsx"
			// 我们检查后缀，如果没有常见 Excel 后缀，则追加 .xlsx
			fileName := att.Name
			if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") && !strings.HasSuffix(strings.ToLower(fileName), ".xls") {
				fileName += ".xlsx"
			}

			mailAttachments = append(mailAttachments, mailer.NewExcelAttachment(fileName, contentBytes))
		}
		msg.Attachments = mailAttachments
	}

	// 发送邮件
	if err := m.Send(msg); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	logger.S().Infow("邮件发送成功",
		"to", to,
		"subject", subject,
		"has_attachment", len(attachments) > 0,
	)

	return nil
}
