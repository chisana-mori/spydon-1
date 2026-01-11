package services

import (
	"bytes"
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

	"github.com/xuri/excelize/v2"
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
	// 预览邮件（同时验证模板）
	preview, err := s.PreviewEmail(PreviewEmailRequest{
		TemplateID:  req.TemplateID,
		ClusterName: req.ClusterName,
		Nodes:       req.Nodes,
		Params:      req.Params,
	})
	if err != nil {
		return err
	}

	// 合并收件人
	allRecipients := s.mergeRecipients(req.Recipients)
	if len(allRecipients) == 0 {
		return fmt.Errorf("收件人列表为空")
	}

	// 生成 Excel 附件
	excelData, err := s.generateExcelAttachment(preview.AffectedResources)
	if err != nil {
		logger.S().Warnw("生成 Excel 附件失败", "error", err)
		// 即使附件生成失败也继续发送邮件
		excelData = nil
	}

	// 发送邮件
	for _, recipient := range allRecipients {
		if err := s.sendSingleEmail(recipient, preview.Subject, preview.HTMLBody, excelData, preview.AttachmentName); err != nil {
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
		"affected_resources_count", len(preview.AffectedResources),
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

// generateExcelAttachment 生成 Excel 附件
func (s *EmailNotificationService) generateExcelAttachment(resources []AffectedResource) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logger.S().Warnw("关闭 Excel 文件失败", "error", err)
		}
	}()

	sheetName := "受影响资源"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	// 设置表头
	headers := []string{"类型", "名称", "命名空间", "状态", "IP", "应用"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	_ = f.SetRowStyle(sheetName, 1, 1, headerStyle)

	// 填充数据
	for i, r := range resources {
		row := i + 2
		vals := []interface{}{r.Type, r.Name, r.Namespace, r.Status, r.IP, r.App}
		for j, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	// 设置列宽
	colWidths := map[string]float64{
		"A": 10, "B": 30, "C": 20, "D": 15, "E": 15, "F": 20,
	}
	for col, width := range colWidths {
		_ = f.SetColWidth(sheetName, col, col, width)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("写入 Excel 失败: %w", err)
	}

	return buf.Bytes(), nil
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

// sendSingleEmail 发送单封邮件（支持 HTML 正文和 Excel 附件）
func (s *EmailNotificationService) sendSingleEmail(to, subject, htmlBody string, attachment []byte, attachmentName string) error {
	if s.cfg == nil || !s.cfg.Email.Enabled {
		logger.S().Infow("邮件功能未启用，跳过发送", "to", to)
		return nil
	}

	// 验证配置
	if s.cfg.Email.SMTPHost == "" || s.cfg.Email.From == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	// 创建 mailer
	m := mailer.New(mailer.Config{
		Host:     s.cfg.Email.SMTPHost,
		Port:     s.cfg.Email.SMTPPort,
		Username: s.cfg.Email.SMTPUser,
		Password: s.cfg.Email.SMTPPass,
		From:     s.cfg.Email.From,
		FromName: s.cfg.Email.FromName,
		UseTLS:   s.cfg.Email.UseTLS,
	})

	// 构建消息
	msg := mailer.Message{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: htmlBody,
	}

	// 添加附件（如果有）
	if attachment != nil && attachmentName != "" {
		msg.Attachments = []mailer.Attachment{
			mailer.NewExcelAttachment(attachmentName, attachment),
		}
	}

	// 发送邮件
	if err := m.Send(msg); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	logger.S().Infow("邮件发送成功",
		"to", to,
		"subject", subject,
		"has_attachment", attachment != nil,
	)

	return nil
}
