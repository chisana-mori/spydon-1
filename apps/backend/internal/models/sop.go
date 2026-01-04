package models

import "gorm.io/datatypes"

// SOPFlow AI-SOP 流程定义
type SOPFlow struct {
	ExecutionID  uint64         `json:"execution_id"`
	PipelineName string         `json:"pipeline_name"`
	ClusterName  string         `json:"cluster_name"`
	GlobalParams datatypes.JSON `json:"global_params"`
	Stages       []SOPStage     `json:"stages"`
}

// SOPStage SOP 阶段定义
type SOPStage struct {
	StageID     string               `json:"stage_id"`
	StageName   string               `json:"stage_name"`
	Type        StageType            `json:"type"`
	Description string               `json:"description,omitempty"`
	AWXJob      *SOPAWXJobDetail     `json:"awx_job,omitempty"`
	ManualGate  *SOPManualGateDetail `json:"manual_gate,omitempty"`
}

// SOPAWXJobDetail AWX 任务详情
type SOPAWXJobDetail struct {
	TemplateID    int                    `json:"template_id"`
	TemplateName  string                 `json:"template_name"`
	Playbook      string                 `json:"playbook"`       // Playbook 文件路径/名称
	Inventory     string                 `json:"inventory"`      // Inventory 名称
	InventoryVars string                 `json:"inventory_vars"` // Inventory 变量 (YAML string)
	Limit         string                 `json:"limit"`
	ExtraVars     map[string]interface{} `json:"extra_vars"`
}

// SOPManualGateDetail 人工审批详情
type SOPManualGateDetail struct {
	ApproverRoles []string `json:"approver_roles"`
	Timeout       int      `json:"timeout_minutes"`
}
