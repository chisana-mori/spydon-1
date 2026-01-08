package dragonfly

// http://dragonfly.qa.pab.com.cn/api/v2/

type FistCTI string

const (
	CloudPlat FistCTI = "云平台"
)

func (c FistCTI) GetName() string {
	return string(c)
}

type SecondCTI string

const (
	StandardChange SecondCTI = "标准变更（预授权）"
	GeneralChange  SecondCTI = "常规变更"
)

func (c SecondCTI) GetName() string {
	return string(c)
}

type ThirdCTI string

const (
	LabelNode          ThirdCTI = "K8S节点标签管理"
	CordonNode         ThirdCTI = "K8S节点cordon"
	UncordonNode       ThirdCTI = "K8S节点uncordon"
	TaintNode          ThirdCTI = "K8S节点污点管理"
	DrainNode          ThirdCTI = "K8S节点drain"
	RebootNode         ThirdCTI = "K8S节点重启"
	ShutdownNode       ThirdCTI = "K8S节点关机"
	AddNode            ThirdCTI = "K8s-增加node节点"
	WorkLoadChange     ThirdCTI = "k8s工作负载变更"
	K8sMaintainChange  ThirdCTI = "Kubernetes集群变更维护"
	K8sComponentChange ThirdCTI = "K8s核心组件变更"
	K8sModeChange      ThirdCTI = "K8s容器节点变更"
)

func (c ThirdCTI) GetName() string {
	return string(c)
}

type ChangeTitle string

const (
	LabelNodeTitle    ChangeTitle = "K8S节点标签管理"
	CordonNodeTitle   ChangeTitle = "K8S节点cordon"
	UncordonNodeTitle ChangeTitle = "K8S节点uncordon"
	TaintNodeTitle    ChangeTitle = "K8S节点污点管理"
	DrainNodeTitle    ChangeTitle = "K8S节点drain"
	RebootNodeTitle   ChangeTitle = "K8S节点重启"
	ShutdownNodeTitle ChangeTitle = "K8S节点关机"
	AddNodeTitle      ChangeTitle = "K8S节点入池"
	DelPodTitle       ChangeTitle = "K8S Pod删除"
)

func (c ChangeTitle) GetName() string {
	return string(c)
}

type ChangeENV string

const (
	Dr       ChangeENV = "zaibei"
	Sandbox  ChangeENV = "SXHJ"
	Prd      ChangeENV = "shengchan"
	Test     ChangeENV = "ceshi"
	HKEnv    ChangeENV = "hongk"
	Office   ChangeENV = "BGFH"
	DGDCDGBK ChangeENV = "DGDCDGBK"
	LCZPRD   ChangeENV = "licazisc"
)

func (c ChangeENV) GetName() string {
	tmp := map[ChangeENV]string{
		Dr:       "灾备",
		Sandbox:  "预发环境",
		Prd:      "生产",
		Test:     "测试",
		HKEnv:    "香港分行",
		Office:   "办公",
		DGDCDGBK: "东莞数据中心(备份)",
		LCZPRD:   "理财子生产",
	}
	return tmp[c]
}

func (c ChangeENV) GetCode() string {
	return string(c)
}

type ChangeOBJType string

const (
	K8SPod     ChangeOBJType = "k8s_pod"
	CICODE     ChangeOBJType = "cicode"
	K8SCluster ChangeOBJType = "k8s_cluster"
)

func (c ChangeOBJType) GetCode() string {
	return string(c)
}

type ChangeUrgentReason string

const (
	BusinessUrgent    ChangeUrgentReason = "businessurgent"
	CapacityExpansion ChangeUrgentReason = "capacityexpansion"
	ProduceException  ChangeUrgentReason = "produceexception"
	HardwareFault     ChangeUrgentReason = "hardwarefault"
	SecurityBreach    ChangeUrgentReason = "securitybreach"
	PHWBDanWei        ChangeUrgentReason = "phwbdanwei"
)

func (c ChangeUrgentReason) GetCode() string {
	return string(c)
}

func (c ChangeUrgentReason) GetName() string {
	tmp := map[ChangeUrgentReason]string{
		BusinessUrgent:    "业务紧急需求",
		CapacityExpansion: "容量紧急扩容",
		ProduceException:  "生产异常处置",
		HardwareFault:     "硬件故障维护",
		SecurityBreach:    "安全漏洞修复",
		PHWBDanWei:        "配合外部单位",
	}
	return tmp[c]
}

type AffectRange string

const (
	B3 AffectRange = "B3"
	B2 AffectRange = "B2"
	B1 AffectRange = "B1"
	B4 AffectRange = "B4"
	B5 AffectRange = "B5"
)

func (c AffectRange) GetName() string {
	tmp := map[AffectRange]string{
		B3: "对业务无影响",
		B2: "影响内部用户业务操作",
		B1: "导致非关键业务中断≤5分钟",
		B4: "导致非关键业务中断>5分钟",
		B5: "导致银行关键业务中断",
	}
	return tmp[c]
}

func (c AffectRange) GetCode() string {
	return string(c)
}

type RiskControl string

const (
	C1 RiskControl = "C1"
	C2 RiskControl = "C2"
	C3 RiskControl = "C3"
	C4 RiskControl = "C4"
	C5 RiskControl = "C5"
)

func (c RiskControl) GetName() string {
	tmp := map[RiskControl]string{
		C1: "重要设备维护或软件大版本升级(如核心网络设备升级、系统升级)",
		C2: "引起应用集群容量<50%(或服务性能下降50%)",
		C3: "需向监管单位（银保监、人行、网联、银联、外管局）报备",
		C4: "变更操作配置项数量>5(或实施复杂度高，多于3个职能组配合)",
		C5: "以上都不涉及",
	}
	return tmp[c]
}

func (c RiskControl) GetCode() string {
	return string(c)
}

type RollbackDifficulty string

const (
	Difficulty03 RollbackDifficulty = "retreatDifficulty03"
	Difficulty02 RollbackDifficulty = "retreatDifficulty02"
	Difficulty01 RollbackDifficulty = "retreatDifficulty01"
)

func (c RollbackDifficulty) GetName() string {
	tmp := map[RollbackDifficulty]string{
		Difficulty03: "易于回退(回退时间小于15分钟)",
		Difficulty02: "回退难度适中(回退时间处于[15,30]分钟)",
		Difficulty01: "不能回退或回退较难(回退时间大于30分钟)",
	}
	return tmp[c]
}

func (c RollbackDifficulty) GetCode() string {
	return string(c)
}

type PreferredCreateNonUrgentChangeReq struct {
	Title        string
	Applicant    string
	ExpectTime   string
	PlanEndTime  string
	Env          string
	SecondCTI    string
	ThirdCTI     string
	Detail       string
	SOP          string
	SOPAddress   string
	UMChecker    string
	UMOperator   string
	CHObjType    string
	CHObj        []string
	UrgentReason string
}

type CreateChangeReq struct {
	Title              string
	Applicant          string
	ExpectTime         string
	PlanEndTime        string
	Env                string
	SecondCTI          string
	ThirdCTI           string
	Detail             string
	SOP                string
	SOPAddress         string
	UMChecker          string
	UMOperator         string
	CHObjType          string
	CHObj              []string
	ChObjList          []ChangeObj `json:"ch_obj_list"`
	IsUrgent           bool
	UrgentReason       string
	RiskAffect         string
	RollbackDifficulty string
	RiskCtrlPoint      string
	RiskAssessment     string
	OperatePlan        string
	CheckMethod        string
	RollbackPlan       string
}

type ChangeObj struct {
	ObjType   string   `json:"obj_type"`
	ObjValues []string `json:"obj_values"`
}
