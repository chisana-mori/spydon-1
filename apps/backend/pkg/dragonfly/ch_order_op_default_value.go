package dragonfly

import (
	"fmt"
	"robusta-web/backend/pkg/errs"
	"robusta-web/backend/pkg/redis"
	"strings"
)

// CreateCHOrderDefaultValue 创建变更单（预授权 + 普通），无论是否预授权，在创建紧急变更时都走紧急流程
func CreateCHOrderDefaultValue(req CreateChangeReq) (*CreateCHOrderRst, errs.Error) {
	chObj := make([]ChObj, 0)
	for _, obj := range req.ChObjList {
		element := ChObj{
			ObjType:   obj.ObjType,
			ObjValues: obj.ObjValues,
		}
		chObj = append(chObj, element)
	}

	redisCli := redis.GetFactory().MustGetClient()

	dragonflyReq := CreateCHOrderReq{
		Title:              req.Title,
		Env:                req.Env,             // 获取环境名称
		FirstCti:           CloudPlat.GetName(), // 变更分类
		SecondCti:          req.SecondCTI,
		ThirdCti:           req.ThirdCTI,
		RequestNumber:      "",
		ExternalOrder:      "", // 关联服务请求单号
		UmChecker:          req.UMChecker,
		UmOperator:         req.UMOperator,
		Applicant:          req.Applicant,
		Detail:             req.Detail,
		ExpectTime:         req.ExpectTime,
		PlanEndTime:        req.PlanEndTime,
		RiskOperate:        "A2", // 操作规范：有标准SOP
		RiskAffect:         req.RiskAffect,
		RollbackDifficulty: req.RollbackDifficulty,
		RiskCtrlPoint:      req.RiskCtrlPoint,
		RiskAssessment:     req.RiskAssessment,
		OperatePlan:        req.OperatePlan,
		CheckMethod:        req.CheckMethod,
		RollbackPlan:       req.RollbackPlan,
		IsUrgent:           false,
		UrgentReason:       "",
		InfluenceSystem:    redisCli.Get("85400.namespaceCNName"), // IT生产数据和公共服务子系统
		SopAddress:         req.SOPAddress,
		ChObj:              req.CHObj,
		ChObjType:          req.CHObjType,
		OprAccounts:        []OprAccount{}, // OPR相关账号权限
		ChObjList:          chObj,
	}

	rst, err := CreateCHOrder(dragonflyReq)
	if err != nil {
		return nil, errs.NewErrorWithMsg(
			errs.ThirdCallReqErr,
			fmt.Sprintf("request dragonfly code: %d, msg: %s, err: %s", rst.Code, rst.Message, err.Error()),
		)
	}
	return &rst, nil
}

// PreferredCreateNonUrgentChangeDefaultValue 尝试创建普通变更失败，则创建紧急变更单
func PreferredCreateNonUrgentChangeDefaultValue(req PreferredCreateNonUrgentChangeReq) (*CreateCHOrderRst, errs.Error) {
	dragonflyReq := CreateCHOrderReq{
		Title:              req.Title,
		Env:                req.Env,
		FirstCti:           CloudPlat.GetName(),
		SecondCti:          req.SecondCTI,
		ThirdCti:           req.ThirdCTI,
		RequestNumber:      "",
		ExternalOrder:      "",
		UmChecker:          req.UMChecker,
		UmOperator:         req.UMOperator,
		Applicant:          req.Applicant,
		Detail:             req.Detail,
		ExpectTime:         req.ExpectTime,
		PlanEndTime:        req.PlanEndTime,
		RiskOperate:        "A1", // 操作规范
		RiskAffect:         "B3", // 影响范围程度
		RollbackDifficulty: "C5", // 回滚难度
		RiskCtrlPoint:      "详见SOP",
		RiskAssessment:     "详见SOP",
		OperatePlan:        "详见SOP",
		CheckMethod:        "详见SOP",
		RollbackPlan:       "详见SOP",
		IsUrgent:           false,
		UrgentReason:       "",
		InfluenceSystem:    "IT生产数据和公共服务子系统",
		SopAddress:         req.SOPAddress,
		ChObj:              req.CHObj,
		ChObjType:          req.CHObjType,
		OprAccounts:        []OprAccount{},
	}

	// 先尝试创建普通变更
	rst, err := CreateCHOrder(dragonflyReq)
	if err != nil {
		return nil, errs.NewErrorWithMsg(
			errs.ThirdCallReqErr,
			fmt.Sprintf("request dragonfly code: %d, msg: %s", rst.Code, rst.Message),
		)
	}

	// 如果返回码为 20009（非操作窗口期），则创建紧急变更单
	if rst.Code == 20009 {
		dragonflyReq.IsUrgent = true
		urgentRst, urgentErr := CreateCHOrder(dragonflyReq)
		if urgentErr != nil {
			return nil, errs.NewErrorWithMsg(
				errs.ThirdCallReqErr,
				fmt.Sprintf("request dragonfly code: %d, msg: %s", urgentRst.Code, urgentRst.Message),
			)
		}
		return &urgentRst, nil
	}

	return &rst, nil
}

// GetEnv 根据集群名获取环境标识（如 gl、ft、qf、lc）
func GetEnv(clusterName string) (env string) {
	if strings.HasPrefix(clusterName, "gl") ||
		strings.HasPrefix(clusterName, "ft") ||
		strings.HasPrefix(clusterName, "qf") ||
		strings.HasPrefix(clusterName, "lc") {
		if strings.Contains(clusterName, "sandbox") {
			env = Sandbox.GetCode()
		} else {
			env = Prd.GetCode()
		}
	} else if strings.HasPrefix(clusterName, "gm") {
		env = Test.GetCode()
	} else if strings.HasPrefix(clusterName, "hk") {
		env = HKEnv.GetCode()
	} else if strings.HasPrefix(clusterName, "wg") {
		env = Dr.GetCode()
	}
	return
}
