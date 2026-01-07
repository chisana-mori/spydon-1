package services

import (
	"time"
)

// FilterType 筛选类型
type FilterType string

const (
	FilterTypeDevice    FilterType = "device"
	FilterTypeNodeLabel FilterType = "nodeLabel"
	FilterTypeTaint     FilterType = "taint"
)

// ConditionType 条件类型
type ConditionType string

const (
	ConditionEqual      ConditionType = "equal"
	ConditionNotEqual   ConditionType = "notEqual"
	ConditionContains   ConditionType = "contains"
	ConditionNotContain ConditionType = "notContains"
	ConditionExists     ConditionType = "exists"
	ConditionNotExists  ConditionType = "notExists"
	ConditionIn         ConditionType = "in"
	ConditionNotIn      ConditionType = "notIn"
	ConditionGT         ConditionType = "greaterThan"
	ConditionLT         ConditionType = "lessThan"
	ConditionIsEmpty    ConditionType = "isEmpty"
	ConditionIsNotEmpty ConditionType = "isNotEmpty"
)

// LogicalOperator 逻辑运算符
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
)

// DeviceFieldDefinition 设备字段定义
type DeviceFieldDefinition struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Column   string `json:"-"`        // 数据库列名，不暴露给前端
	DataType string `json:"dataType"` // string, number, boolean, date
}

// GetAllDeviceFields 获取所有设备字段定义 (28个字段)
func GetAllDeviceFields() []DeviceFieldDefinition {
	return []DeviceFieldDefinition{
		// 基础信息
		{ID: "ciCode", Label: "设备编码", Column: "device.ci_code", DataType: "string"},
		{ID: "ip", Label: "IP地址", Column: "device.ip", DataType: "string"},
		{ID: "archType", Label: "CPU架构", Column: "device.arch_type", DataType: "string"},
		{ID: "model", Label: "设备型号", Column: "device.model", DataType: "string"},
		{ID: "company", Label: "厂商", Column: "device.company", DataType: "string"},

		// 位置信息
		{ID: "idc", Label: "IDC", Column: "device.idc", DataType: "string"},
		{ID: "room", Label: "机房", Column: "device.room", DataType: "string"},
		{ID: "cabinet", Label: "机柜", Column: "device.cabinet", DataType: "string"},
		{ID: "cabinetNo", Label: "机柜号", Column: "device.cabinet_no", DataType: "string"},
		{ID: "netZone", Label: "网络区域", Column: "device.net_zone", DataType: "string"},

		// 配置信息
		{ID: "cpu", Label: "CPU核数", Column: "device.cpu", DataType: "number"},
		{ID: "memory", Label: "内存(GB)", Column: "device.memory", DataType: "number"},
		{ID: "diskCount", Label: "磁盘数量", Column: "device.disk_count", DataType: "number"},
		{ID: "diskDetail", Label: "磁盘详情", Column: "device.disk_detail", DataType: "string"},
		{ID: "networkSpeed", Label: "网卡速率", Column: "device.network_speed", DataType: "string"},

		// 系统信息
		{ID: "os", Label: "操作系统", Column: "device.os", DataType: "string"},
		{ID: "osName", Label: "系统名称", Column: "device.os_name", DataType: "string"},
		{ID: "osIssue", Label: "系统版本", Column: "device.os_issue", DataType: "string"},
		{ID: "osKernel", Label: "内核版本", Column: "device.os_kernel", DataType: "string"},
		{ID: "osCreateTime", Label: "系统安装时间", Column: "device.os_create_time", DataType: "date"},

		// 业务信息
		{ID: "infraType", Label: "基础设施类型", Column: "device.infra_type", DataType: "string"},
		{ID: "isLocalization", Label: "国产化", Column: "device.is_localization", DataType: "boolean"},
		{ID: "group", Label: "机器分组/用途", Column: "device.`group`", DataType: "string"},
		{ID: "appId", Label: "应用ID", Column: "device.appid", DataType: "string"},

		// 集群信息
		{ID: "role", Label: "角色", Column: "device.role", DataType: "string"},
		{ID: "cluster", Label: "集群", Column: "device.cluster", DataType: "string"},

		// 状态信息
		{ID: "status", Label: "状态", Column: "device.status", DataType: "string"},
		{ID: "k8sStatus", Label: "K8s状态", Column: "device.k8s_status", DataType: "string"},
		{ID: "acceptanceTime", Label: "验收时间", Column: "device.acceptance_time", DataType: "date"},
	}
}

// GetDeviceFieldColumn 根据字段ID获取数据库列名
func GetDeviceFieldColumn(fieldID string) string {
	for _, f := range GetAllDeviceFields() {
		if f.ID == fieldID {
			return f.Column
		}
	}
	return ""
}

// DeviceQuery 基础设备查询参数
type DeviceQuery struct {
	Page        int    `form:"page"`        // 页码
	Size        int    `form:"size"`        // 每页数量
	Keyword     string `form:"keyword"`     // 搜索关键字
	OnlySpecial bool   `form:"onlySpecial"` // 仅显示特殊设备
}

// DeviceResponse 设备详情响应
type DeviceResponse struct {
	ID             int       `json:"id"`
	CICode         string    `json:"ci_code"`
	IP             string    `json:"ip"`
	ArchType       string    `json:"arch_type"`
	IDC            string    `json:"idc"`
	Room           string    `json:"room"`
	Cabinet        string    `json:"cabinet"`
	CabinetNO      string    `json:"cabinet_no"`
	InfraType      string    `json:"infra_type"`
	IsLocalization bool      `json:"is_localization"`
	NetZone        string    `json:"net_zone"`
	Group          string    `json:"group"`
	AppID          string    `json:"appid"`
	AppName        string    `json:"app_name"`
	OsCreateTime   string    `json:"os_create_time"`
	CPU            float64   `json:"cpu"`
	Memory         float64   `json:"memory"`
	Model          string    `json:"model"`
	KvmIP          string    `json:"kvm_ip"`
	OS             string    `json:"os"`
	Company        string    `json:"company"`
	OSName         string    `json:"os_name"`
	OSIssue        string    `json:"os_issue"`
	OSKernel       string    `json:"os_kernel"`
	Status         string    `json:"status"`
	Role           string    `json:"role"`
	Cluster        string    `json:"cluster"`
	ClusterID      int       `json:"cluster_id"`
	K8sStatus      string    `json:"k8s_status"`
	AcceptanceTime string    `json:"acceptance_time"`
	DiskCount      int       `json:"disk_count"`
	DiskDetail     string    `json:"disk_detail"`
	NetworkSpeed   string    `json:"network_speed"`
	IsSpecial      bool      `json:"is_special"`
	FeatureCount   int       `json:"feature_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DeviceListResponse 设备列表响应
type DeviceListResponse struct {
	List  []DeviceResponse `json:"list"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// DeviceRoleUpdateRequest 更新设备角色请求
type DeviceRoleUpdateRequest struct {
	Role string `json:"role"`
}

// DeviceGroupUpdateRequest 更新设备组请求
type DeviceGroupUpdateRequest struct {
	Group string `json:"group"`
}

// FilterBlock 复杂查询筛选块
type FilterBlock struct {
	ID            string      `json:"id"`
	Type          FilterType  `json:"type"`          // device, nodeLabel, taint
	ConditionType string      `json:"conditionType"` // equal, contains, exists, in, etc.
	Key           string      `json:"key"`
	Value         interface{} `json:"value"`
	Operator      string      `json:"operator"` // and, or
	IsActive      *bool       `json:"isActive"` // 是否激活
}

// FilterGroup 复杂查询筛选组
type FilterGroup struct {
	ID       string        `json:"id"`
	Blocks   []FilterBlock `json:"blocks"`
	Operator string        `json:"operator"`
}

// NavyDeviceQueryRequest 复杂查询请求
type NavyDeviceQueryRequest struct {
	Groups []FilterGroup `json:"groups"`
	Page   int           `json:"page"`
	Size   int           `json:"size"`
}

// FilterOption 筛选项
type FilterOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// DeviceFieldValues 字段可选值
type DeviceFieldValues struct {
	Field  string         `json:"field"`
	Values []FilterOption `json:"values"`
}

// FilterOptionsResponse 筛选选项响应
type FilterOptionsResponse struct {
	DeviceFields      []DeviceFieldDefinition `json:"deviceFields"`
	DeviceFieldValues []DeviceFieldValues     `json:"deviceFieldValues"`
	LabelKeys         []string                `json:"labelKeys"`
	TaintKeys         []string                `json:"taintKeys"`
}

// QueryTemplate 查询模板
type QueryTemplate struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Groups      []FilterGroup `json:"groups"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// QueryTemplateListResponse 模板列表响应
type QueryTemplateListResponse struct {
	List  []QueryTemplate `json:"list"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// LabelValue 标签键值
type LabelValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TaintValue 污点键值
type TaintValue struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}
