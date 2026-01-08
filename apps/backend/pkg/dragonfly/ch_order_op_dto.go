package dragonfly

import "strings"

// CreatePreAuthCHOrderReq 预授权变更单请求
type CreatePreAuthCHOrderReq struct {
	Title              string   `json:"title"`
	Env                string   `json:"env"`
	Type               string   `json:"type"`
	Event              int      `json:"event"`          // 预授权事件编码，找生产管理组申请
	RequestNumber      string   `json:"request_number"` // SR 单
	UmChecker          string   `json:"um_checker"`     // 验证人
	UmOperator         string   `json:"um_operator"`    // 操作人
	Applicant          string   `json:"applicant"`      // 申请人
	Detail             string   `json:"detail"`         // 详情描述
	ExpectTime         string   `json:"expect_time"`    // 预计开始时间
	PlanEndTime        string   `json:"plan_end_time"`  // 预计结束时间
	RiskOperate        string   `json:"risk_operate"`   // 操作规范
	Sop                string   `json:"sop"`
	RiskAffect         string   `json:"risk_affect"`         // 影响范围程度
	RollbackDifficulty string   `json:"rollback_difficulty"` // 回滚难度
	RiskCtrlPoint      string   `json:"risk_ctrl_point"`     // 风险控制点
	RiskAssessment     string   `json:"risk_assessment"`     // 风险评估
	OperatePlan        string   `json:"operate_plan"`        // 实施方案
	CheckMethod        string   `json:"check_method"`        // 检查方式
	RollbackPlan       string   `json:"rollback_plan"`       // 回滚方案
	IsUrgent           bool     `json:"is_urgent"`
	UrgentReason       string   `json:"urgent_reason"`
	InfluenceSystem    string   `json:"influence_system"`
	SopAddress         string   `json:"sop_address"`
	ChObj              []string `json:"ch_obj"`
	ChObjType          string   `json:"ch_obj_type"`
}

// ChObj 变更对象结构
type ChObj struct {
	ObjType   string   `json:"obj_type"`
	ObjValues []string `json:"obj_values"`
}

// CreateCHOrderReq 普通变更单请求
type CreateCHOrderReq struct {
	Title              string       `json:"title"`
	Env                string       `json:"env"`
	FirstCti           string       `json:"first_cti"`           // 第一分类
	SecondCti          string       `json:"second_cti"`          // 第二分类
	ThirdCti           string       `json:"third_cti"`           // 第三分类
	RequestNumber      string       `json:"request_number"`      // 关联SR单号
	ExternalOrder      string       `json:"external_order"`      // 告警或异常处置关联单号
	UmChecker          string       `json:"um_checker"`          // 验证人
	UmOperator         string       `json:"um_operator"`         // 操作人
	Applicant          string       `json:"applicant"`           // 申请人
	Detail             string       `json:"detail"`              // 详情描述
	ExpectTime         string       `json:"expect_time"`         // 预计开始时间
	PlanEndTime        string       `json:"plan_end_time"`       // 预计结束时间
	RiskOperate        string       `json:"risk_operate"`        // 操作规范
	RiskAffect         string       `json:"risk_affect"`         // 影响范围程度
	RollbackDifficulty string       `json:"rollback_difficulty"` // 回滚难度
	RiskCtrlPoint      string       `json:"risk_ctrl_point"`     // 风险控制点
	RiskAssessment     string       `json:"risk_assessment"`     // 风险评估
	OperatePlan        string       `json:"operate_plan"`        // 实施方案
	CheckMethod        string       `json:"check_method"`        // 检查方式
	RollbackPlan       string       `json:"rollback_plan"`       // 回滚方案
	IsUrgent           bool         `json:"is_urgent"`
	UrgentReason       string       `json:"urgent_reason"`
	InfluenceSystem    string       `json:"influence_system"`
	SopAddress         string       `json:"sop_address"`
	ChObj              []string     `json:"ch_obj"`
	ChObjType          string       `json:"ch_obj_type"`
	OprAccounts        []OprAccount `json:"operate_account"` // OPR相关账号权限
	ChObjList          []ChObj      `json:"ch_obj_list"`     // 变更对象列表
}

// OprAccountType 账号类型枚举
type OprAccountType string

const (
	OprAccountOS OprAccountType = "os"
	OprAccountDB OprAccountType = "db"
)

// OprAccount OPR账号信息
type OprAccount struct {
	Type    OprAccountType `json:"type"`
	Name    string         `json:"name"`
	Account []string       `json:"account"`
}

// CHInfo 变更单基本信息
type CHInfo struct {
	ChangeNumber  string `json:"change_number"`
	IsNeedApprove bool   `json:"is_need_approve"`
	EooNumber     string `json:"eoo_number"`
}

// CreateCHOrderRst 创建变更的返回结果
type CreateCHOrderRst struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	CHInfo     CHInfo `json:"data"`
	OriginData string `json:"-"` // 保留原始 Dragonfly 数据供应用层使用
}

// IsInnerControl 判断是否因变更管控失败（如非操作窗口）
func (rst *CreateCHOrderRst) IsInnerControl() bool {
	const ControlInnerCode = 20009
	const ControlKeyword = "CTRL_EVENT"
	if rst.Code == ControlInnerCode && strings.Contains(rst.OriginData, ControlKeyword) {
		return true
	}
	return false
}

// StartCHOrderReq 开始变更单请求
type StartCHOrderReq struct {
	ChangeNumber string `json:"change_number"`
	ChangeUser   string `json:"change_user"`
	IsMuteAlert  int    `json:"is_mute_alert"`
}

// StartCHOrderRst 开始变更单返回结果
type StartCHOrderRst struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	OriginData string      `json:"-"` // 保留原始数据
}

// CloseCHOrderReq 关闭变更单请求
type CloseCHOrderReq struct {
	ChangeNumber  string `json:"change_number"`
	ExecType      string `json:"exec_type"` // 1: '变更成功', 2: '失败回退', 3: '完成，有异常情况', 4: '变更取消', 5: '完成回退'
	VerifyContext string `json:"verify_context"`
	ChangeUser    string `json:"change_user"`
	StartTime     string `json:"start_time"`
}

// CloseCHOrderRst 关闭变更单返回结果
type CloseCHOrderRst struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	OriginData string `json:"-"` // 保留原始数据
}

// CancelCHOrderReq 取消变更单请求
type CancelCHOrderReq struct {
	ChangeNumber string `json:"change_number"`
	ChangeUser   string `json:"change_user"`
}

// CancelCHOrderRst 取消变更单返回结果
type CancelCHOrderRst struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Data       string `json:"data"`
	OriginData string `json:"-"` // 保留原始数据
}

// VerifyCHOrderReq 验证变更单请求
type VerifyCHOrderReq struct {
	ChangeNumber string `json:"change_number"`
	VerifyUser   string `json:"verify_user"`
	IsPassed     int    `json:"is_passed"`
}

// VerifyCHOrderRst 验证变更单返回结果
type VerifyCHOrderRst struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Data       string `json:"data"`
	OriginData string `json:"-"` // 保留原始数据
}
