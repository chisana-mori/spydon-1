package dragonfly

import "strings"

// CHOrderDetailReq 查询变更单的请求参数
type CHOrderDetailReq struct {
	ChangeNo  string `json:"changeNo"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// CHOrderDetail 变更单详情
type CHOrderDetail struct {
	RetreatDifficulty  string    `json:"retreatDifficulty"` // 回退难易程度
	ReturnDiffCulty    string    `json:"returnDiffCulty"`   // 回退难易程度（别名）
	DeleteFlag         int       `json:"deleteFlag"`
	CompanyId          string    `json:"companyId"`
	FallbackNum        int       `json:"fallback_num"`
	ChangeSource       string    `json:"changesource"`
	OrderKey           int       `json:"orderKey"`
	ChangePlan         string    `json:"changeplan"`
	ChangeId           int       `json:"changeid"`
	ChangeTitle        string    `json:"changetitle"` // 标题
	OperateAccount     string    `json:"operateAccount"`
	ChangeNumber       string    `json:"changnumber"`
	ChangeStatus       CHStatus  `json:"chagestatus"`
	ProcessResult      string    `json:"processresult"`
	FormKey            string    `json:"FORM_KEY"` // OperationalChanges-运营技术 Versionchange-版本发布
	InfluenceSystem    string    `json:"InfluenceSystem"`
	DocuAddress        string    `json:"docuaddress"`
	ImplMethod         string    `json:"implMethod"`
	Changedegree       string    `json:"chengedegree"`
	Offecscope         string    `json:"offecscoe"`
	CompanyKey         string    `json:"companykey"`
	Offescop           string    `json:"offecscoop"`
	UpdateDate         string    `json:"updateDate"`
	BiaoshiCount       string    `json:"biaoshicount"`
	SupplementOrder    string    `json:"supplementOrder"`
	TaskKey            string    `json:"taskKey"`
	IsUrgentChange     int       `json:"isUrgentChange"` // 紧急变更
	UrgentChange       string    `json:"urgentChange"`   // 紧急变更
	ChangeCtiType      string    `json:"chargetype"`
	RiskFormulaOperate string    `json:"riskformulaOperate"`
	RiskAssessName     string    `json:"riskassessName"`
	RiskAssessOperate  string    `json:"riskAssessOperate"`
	RiskAssess         string    `json:"riskassess"`
	ChangeSourceVar    string    `json:"changesourcevar"`
	IsDisaster         string    `json:"isDisaster"`
	ChangeRisk         string    `json:"changerisk"`  // 变更方案信息 风险评估
	CheckMethod        string    `json:"checkMethod"` // 变更方案信息 验证方法
	Checkmethod        string    `json:"checkmethod"` // 变更方案信息 验证方法
	Backspace          string    `json:"backspace"`   // 变更方案信息 回退方案
	ChangeUserName     string    `json:"changeUserName"`
	UpdateBy           string    `json:"updateBy"`
	ChangeDetails      string    `json:"changedetails"`
	ModuleId           string    `json:"moduleId"`
	CHNAGEMAJOR        string    `json:"CHNAGEMAJOR"`
	Recorder           string    `json:"recorder"`
	RiskFormulaaffect  string    `json:"riskformulaAffect"`
	ChangeType         int       `json:"changeType"`
	ChangeMajor        int       `json:"change_major"` // 是否重大变更 1-否 0-是
	RejectedNum        int       `json:"rejected_num"`
	HappenTime         string    `json:"happentime"`    // 发生时间
	ChangUserId        string    `json:"changuserid"`   // 申请人
	ChangeCheck        string    `json:"changecheck"`   // 验证人
	ChangeForceId      string    `json:"changeforceId"` // 变更实施人
	ChangeForce        string    `json:"changeforce"`   // 变更实施人
	RequestUser        string    `json:"requestUser"`   // 何斌斌(hebinbin330_15618977060)
	ForceTime          string    `json:"forcetime"`
	PlanEndTime        string    `json:"planEndTime"` // 预计结束时间
	StartTime          string    `json:"startTime"`   // 实际开始时间
	EndTime            string    `json:"endTime"`     // 实际结束时间
	CreateTime         string    `json:"creatime"`    // 提交时间
	AppId              string    `json:"appId"`       // APPID
	CabFlag            string    `json:"cab_flag"`
	Environment        string    `json:"environment"`
	PreLeader          string    `json:"preleader"`
	IsChange           string    `json:"isChange"`
	TaskId             string    `json:"taskId"`
	ExecType           string    `json:"exeType"`       // 实施完成类型
	ChangeObjType      CHObjType `json:"changeObjType"` // ITSM-老变更对象类型
	ChangeObjVal       string    `json:"changeObjVal"`  // ITSM-老变更对象
	ChObjType          CHObjType `json:"ch_obj_type"`   // dragonfly-新变更对象类型
	ChObj              []string  `json:"ch_obj"`        // dragonfly-新变更对象
}

// GetChObj 优先使用 ChObj，没有则使用 ChangeObjVal 切割成数组
func (c CHOrderDetail) GetChObj() []string {
	if len(c.ChObj) > 0 {
		return c.ChObj
	}
	if len(c.ChangeObjVal) > 0 {
		return strings.Split(c.ChangeObjVal, ",")
	}
	return []string{}
}

// GetChObjType 优先使用 ChObjType，没有则使用 ChangeObjType
func (c CHOrderDetail) GetChObjType() CHObjType {
	if len(c.ChObjType) > 0 {
		return c.ChObjType
	}
	return c.ChangeObjType
}

// EnumItemITSM ITSM 返回的枚举项结构
type EnumItemITSM struct {
	Code   string `json:"code"`
	NameCN string `json:"nameCn"`
}

// EnumItem 通用枚举项结构
type EnumItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// CHDataEnum 枚举值集合
type CHDataEnum struct {
	Env                []EnumItem        `json:"env"`                 // 环境 HJ
	UrgentReason       []EnumItem        `json:"urgent_reason"`       // 紧急变更原因 urgentchangereason
	RollbackDifficulty []EnumItem        `json:"rollback_difficulty"` // 回退难易程度 retreatDifficulty
	RiskOperate        []EnumItem        `json:"risk_operate"`        // 操作规范 operateNorm
	RiskControl        []EnumItem        `json:"risk_control"`        // 风险控制点 riskControl
	AffectRange        []EnumItem        `json:"affect_range"`        // 影响范围程度 affectRange
	CHObjType          []EnumItem        `json:"ch_obj_type"`         // 变更对象类型 changeObjType
	Desc               map[string]string `json:"_desc"`               // 描述字段
}
