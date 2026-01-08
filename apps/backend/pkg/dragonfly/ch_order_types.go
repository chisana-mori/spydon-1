package dragonfly

// CHObjType 变更对象类型
type CHObjType string

// CHStatus 变更单状态
type CHStatus string

// ExecType 实施完成类型
type ExecType int

// 变更对象类型 枚举
const (
	CHObjTypeHostIP     CHObjType = "host_ip"
	CHObjTypeCICode     CHObjType = "cicode"
	CHObjTypeAppID      CHObjType = "appid"
	CHObjTypeClusterID  CHObjType = "cluster_id"
	CHObjTypeSubSysID   CHObjType = "sub_sys_id"
	CHObjTypeStaticID   CHObjType = "static_id"
	CHObjTypeAppKey     CHObjType = "app_key"
	CHObjTypeIDC        CHObjType = "idc"
	CHObjTypeIDCBarn    CHObjType = "idc_barn"
	CHObjTypeIDCCabinet CHObjType = "idc_cabinet"
)

func (c CHObjType) GetCode() string {
	return string(c)
}

func (c CHObjType) GetDesc() string {
	m := map[CHObjType]string{
		CHObjTypeHostIP:     "主机IP",
		CHObjTypeCICode:     "CICODE",
		CHObjTypeAppID:      "APPID",
		CHObjTypeClusterID:  "集群ID",
		CHObjTypeSubSysID:   "子系统ID",
		CHObjTypeStaticID:   "静态资源ID[待12月废弃]",
		CHObjTypeAppKey:     "前端应用",
		CHObjTypeIDC:        "IDC",
		CHObjTypeIDCBarn:    "机房",
		CHObjTypeIDCCabinet: "机柜",
	}
	if v, exist := m[c]; exist {
		return v
	}
	return string(c)
}

// 变更单状态 枚举
const (
	CHCancel     CHStatus = "cancel_change"
	CHEoaRefuse  CHStatus = "EOA_NO"
	CHDraft      CHStatus = "CG"
	CHFailedBack CHStatus = "failFallBack"
	CHReject     CHStatus = "reject"
	CHCreate     CHStatus = "change_create"
	CHSubmitEoa  CHStatus = "change_submit_eoa"
	CHClaim      CHStatus = "change_claim"
	CHDoingChang CHStatus = "changSh"
	CHStayClose  CHStatus = "change_stay_close"
	CHClosed     CHStatus = "change_close"
	CHCheckOut   CHStatus = "checkOut"
	CHChenckOut  CHStatus = "chenckOut"
	CHRm         CHStatus = "change_rm"
	CHChecking   CHStatus = "changeCheck"
)

func (stat CHStatus) GetCode() string {
	return string(stat)
}

func (stat CHStatus) GetDesc() string {
	m := map[CHStatus]string{
		CHCancel:     "取消作废",
		CHEoaRefuse:  "EOA被拒绝",
		CHDraft:      "草稿",
		CHFailedBack: "失败退回",
		CHReject:     "变更驳回",
		CHCreate:     "变更创建",
		CHSubmitEoa:  "提交至EOA",
		CHClaim:      "待认领",
		CHDoingChang: "实施中",
		CHStayClose:  "待关闭",
		CHClosed:     "关闭",
		CHCheckOut:   "验证中",
		CHChenckOut:  "验证中",
		CHRm:         "变更取消",
		CHChecking:   "审核中",
	}
	return m[stat]
}

func GetCHStatusList() []CHStatus {
	return []CHStatus{
		CHCancel,
		CHEoaRefuse,
		CHDraft,
		CHFailedBack,
		CHReject,
		CHCreate,
		CHSubmitEoa,
		CHClaim,
		CHDoingChang,
		CHStayClose,
		CHClosed,
		CHCheckOut,
		CHChenckOut,
		CHRm,
		CHChecking,
	}
}

// 实施完成类型 枚举
const (
	CHExecTypeSuccess             ExecType = 1
	CHExecTypeFailureBack         ExecType = 2
	CHExecTypeSuccessWithAbnormal ExecType = 3
	CHExecTypeCancel              ExecType = 4
	CHExecTypeBackSuccess         ExecType = 5
	CHExecTypePartialSuccess      ExecType = 6
)

func (et ExecType) GetCode() int {
	return int(et)
}

func (et ExecType) GetDesc() string {
	m := map[ExecType]string{
		CHExecTypeSuccess:             "变更成功",
		CHExecTypeFailureBack:         "失败回退",
		CHExecTypeSuccessWithAbnormal: "完成，有异常情况",
		CHExecTypeCancel:              "变更取消",
		CHExecTypeBackSuccess:         "完成回退",
		CHExecTypePartialSuccess:      "部分成功",
	}
	return m[et]
}

type CHRisk string

const (
	CHRiskSuperHigh CHRisk = "risklevel01"
	CHRiskHigh      CHRisk = "risklevel02"
	CHRiskMiddle    CHRisk = "risklevel03"
	CHRiskLow       CHRisk = "risklevel04"
	CHRiskSuperLow  CHRisk = "risklevel05"
)

func (risk CHRisk) GetCode() string {
	return string(risk)
}

func (risk CHRisk) GetDesc() string {
	m := map[CHRisk]string{
		CHRiskSuperHigh: "极高",
		CHRiskHigh:      "高",
		CHRiskMiddle:    "中",
		CHRiskLow:       "低",
		CHRiskSuperLow:  "极低",
	}
	return m[risk]
}

type CHEnv string

const (
	CHEnvPRD  CHEnv = "shengchan"
	CHEnvQA   CHEnv = "ceshi"
	CHEnvDR   CHEnv = "zaibei"
	CHEnvDG   CHEnv = "DGDCDGBK"
	CHEnvSX   CHEnv = "SXHJ"
	CHEnvBG   CHEnv = "BGFH"
	CHEnvHK   CHEnv = "hongk"
	CHEnvHKDR CHEnv = "HangHongDate"
)

func (env CHEnv) GetDesc() string {
	m := map[CHEnv]string{
		CHEnvPRD:  "生产",
		CHEnvQA:   "测试",
		CHEnvDR:   "灾备",
		CHEnvDG:   "东莞数据中心(备份)",
		CHEnvSX:   "模拟环境",
		CHEnvBG:   "办公",
		CHEnvHK:   "香港分行",
		CHEnvHKDR: "香港灾备",
	}
	return m[env]
}
