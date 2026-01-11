package services

import (
	"encoding/json"
	"robusta-web/backend/internal/models"
)

// ParamType 参数类型
type ParamType string

const (
	ParamTypeInput    ParamType = "input"    // 文本输入
	ParamTypeSelect   ParamType = "select"   // 下拉选择（关联字典）
	ParamTypeDatetime ParamType = "datetime" // 日期时间选择
	ParamTypeResource ParamType = "resource" // K8s 资源选择
)

// ParamDefinition 模板参数定义
type ParamDefinition struct {
	Name         string    `json:"name"`                   // 参数键 (原 Key)
	Title        string    `json:"title"`                  // 显示标题 (原 Label)
	Type         ParamType `json:"type"`                   // 参数类型
	Value        string    `json:"value,omitempty"`        // 默认值/绑定值
	Required     bool      `json:"required,omitempty"`     // 是否必填
	DictCode     string    `json:"dictCode,omitempty"`     // type=select 时关联的字典编码
	ResourceType string    `json:"resourceType,omitempty"` // type=resource 时的资源类型 (nodes/pods/deployments)
	Placeholder  string    `json:"placeholder,omitempty"`  // 占位提示
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	Tables      []string          `json:"tables"`      // 关联的数据表 (对应 email_sender.go 中的常量)
	Definitions []ParamDefinition `json:"definitions"` // 用户输入参数定义
}

// ParseParams 解析参数定义 JSON
func ParseParams(paramsJSON string) (TemplateConfig, error) {
	var config TemplateConfig
	if paramsJSON == "" {
		return config, nil
	}
	
	if err := json.Unmarshal([]byte(paramsJSON), &config); err != nil {
		return config, err
	}
	
	return config, nil
}

// ========================================
// EmailTemplate DTOs
// ========================================

// EmailTemplateListQuery 模板列表查询参数
type EmailTemplateListQuery struct {
	Page      int    `form:"page"`
	Size      int    `form:"size"`
	Keyword   string `form:"keyword"`
	IsEnabled *bool  `form:"isEnabled"`
}

// EmailTemplateResponse 模板响应
type EmailTemplateResponse struct {
	ID        uint64         `json:"id"`
	Name      string         `json:"name"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Params    TemplateConfig `json:"params"`
	IsEnabled bool           `json:"is_enabled"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

// CreateEmailTemplateRequest 创建模板请求
type CreateEmailTemplateRequest struct {
	Name      string         `json:"name" binding:"required,max=100"`
	Title     string         `json:"title" binding:"required,max=255"`
	Body      string         `json:"body" binding:"required"`
	Params    TemplateConfig `json:"params"`
	IsEnabled *bool          `json:"is_enabled"`
}

// UpdateEmailTemplateRequest 更新模板请求
type UpdateEmailTemplateRequest struct {
	Name      *string         `json:"name"`
	Title     *string         `json:"title"`
	Body      *string         `json:"body"`
	Params    *TemplateConfig `json:"params"`
	IsEnabled *bool           `json:"is_enabled"`
}

// ========================================
// EmailContact DTOs
// ========================================

// EmailContactListQuery 联系人列表查询参数
type EmailContactListQuery struct {
	Page    int    `form:"page"`
	Size    int    `form:"size"`
	Keyword string `form:"keyword"`
}

// EmailContactResponse 联系人响应
type EmailContactResponse struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateEmailContactRequest 创建联系人请求
type CreateEmailContactRequest struct {
	Name    string `json:"name" binding:"required,max=100"`
	Address string `json:"address" binding:"required,max=500"`
}

// UpdateEmailContactRequest 更新联系人请求
type UpdateEmailContactRequest struct {
	Name    *string `json:"name"`
	Address *string `json:"address"`
}

// ========================================
// Email Sending DTOs
// ========================================

// PreviewEmailRequest 预览邮件请求
type PreviewEmailRequest struct {
	TemplateID  uint64                 `json:"template_id" binding:"required"`
	ClusterName string                 `json:"cluster_name"`
	Nodes       []string               `json:"nodes"`
	Params      map[string]interface{} `json:"params"`
}

// PreviewEmailResponse 预览邮件响应
type PreviewEmailResponse struct {
	Subject           string             `json:"subject"`
	HTMLBody          string             `json:"html_body"`
	AffectedResources []AffectedResource `json:"affected_resources"`
	AttachmentName    string             `json:"attachment_name,omitempty"` // Excel 附件文件名
}

// AffectedResource 受影响的资源
type AffectedResource struct {
	Type      string `json:"type"`                // node, pod, deployment
	Name      string `json:"name"`                // 资源名称
	Namespace string `json:"namespace,omitempty"` // 命名空间（pod/deployment）
	Status    string `json:"status,omitempty"`    // 资源状态
	IP        string `json:"ip,omitempty"`        // IP 地址
	App       string `json:"app,omitempty"`       // 应用名称
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	TemplateID  uint64                 `json:"template_id" binding:"required"`
	ClusterName string                 `json:"cluster_name"`
	Nodes       []string               `json:"nodes"`
	Params      map[string]interface{} `json:"params"`
	Recipients  []string               `json:"recipients" binding:"required,min=1"`
}

// GetAffectedResourcesRequest 获取受影响资源请求
type GetAffectedResourcesRequest struct {
	ClusterName  string   `form:"cluster" binding:"required"`
	Nodes        []string `form:"nodes"`
	ResourceType string   `form:"resourceType"` // nodes, pods, deployments
}

// PodInfo Pod 信息（用于模板渲染）
type PodInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	NodeName  string `json:"node_name"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
}

// DeploymentInfo Deployment 信息（用于模板渲染）
type DeploymentInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
	Ready     int32  `json:"ready"`
}

// NodeInfo Node 信息（用于模板渲染）
type NodeInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	IP     string `json:"ip"`
}

// TemplateResources 模板可用的资源数据
type TemplateResources struct {
	Nodes       []NodeInfo       `json:"nodes"`
	Pods        []PodInfo        `json:"pods"`
	Deployments []DeploymentInfo `json:"deployments"`
}

type SendEmailsReq struct {
	Emails []SendEmailReq
}

type CheckRst struct {
	Exist         bool   `json:"exist"`
	ResourceCount int    `json:"count"`
	ResourceType  string `json:"type"`
}

type ExComponentCheck struct {
	Ingress    CheckRst `json:"ingress"`
	Higress    CheckRst `json:"higress"`
	SolarAgent CheckRst `json:"solarAgent"`
	JMXDS      CheckRst `json:"jmxDS"`
	CatAgent   CheckRst `json:"catAgent"`
	Asta       CheckRst `json:"asta"`
	HIDS       CheckRst `json:"hids"`
	Istio      CheckRst `json:"istio"`
	Armada     CheckRst `json:"armada"`
}

type AppSlice []AppBoard

func (a AppSlice) Len() int {
	return len(a)
}

func (a AppSlice) Less(i, j int) bool {
	if a[i].AppLevel == "" {
		return false
	}
	if a[j].AppLevel == "" {
		return true
	}
	return a[i].AppLevel < a[j].AppLevel
}

func (a AppSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type AppBoard struct {
	AppId           string
	AppLevel        string
	NamespaceName   string
	NamespaceCNName string
	DeploymentName  string
	Team            string
	Develops        string
	Operations      string
}

type App struct {
	AppId     string
	Name      string
	Namespace string
}

type Cluster struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type MailGenReq struct {
	TemplateId int                    `json:"emailTemplateId"`
	AddressId  int                    `json:"addressId"`
	Additional map[string]interface{} `json:"additional"`
}

type Node struct {
	Name     string
	Function string
	IP       string
}

type DiffAppsBoard struct {
	HistoryDeployCount int
	InstantDeployCount int
	Apps               AppSlice
}

type Migration struct {
	Apps          AppSlice
	InitialCount  int
	MigratedCount int
	RemainCount   int
}

type MailContentProperty struct {
	Cluster        models.Cluster
	NodeCidr       []string
	Nodes          []Node
	Apps           AppSlice
	Migration      Migration
	ComponentCheck ExComponentCheck
	DiffAppsBoard  DiffAppsBoard
	AppIdCount     int
	Additional     map[string]interface{}
}

type AttachFile struct {
	Name    string
	Content string
}

type SendEmailReq struct {
	TemplateId  int          `json:"templateId"`
	Subject     string       `json:"subject"`
	Content     string       `json:"content"`
	AttachFiles []AttachFile `json:"attachFiles"`
	Addresses   []string     `json:"addresses"`
	AppId       []string     `json:"appid"`
}
