package dragonfly

import (
	"errors"
	"fmt"
	"sync"
)

// 1、创建常规变更单        /api/v2/original/change_v3    POST
// 2、创建预授权变更单      /api/v2/itsm/preauthorization/change_v3 POST
// 3、上传附件（按需调用）   /api/v2/file    POST
// 4、枚举值获取接口         /api/v2/itsm/data_enum_ch?code=xxx  GET
// 5、查询风险等级接口（按需调用） /api/v2/itsm/ch_risk_level  PUT

// 变更单相关接口
const (
	queryChangeDetailUrl = "/api/v2/itsm/change/ticket/query_range"
	// 稽核改造 2022115 后的版本
	queryCHDataEnum = "/api/v2/itsm/data_enum_ch" // 枚举值获取接口
)

// GetCHOrderDetail 获取 CH 详情
func GetCHOrderDetail(changeNumber string) (CHOrderDetail, error) {
	req := CHOrderDetailReq{
		ChangeNo:  changeNumber,
		StartTime: "",
		EndTime:   "",
	}
	var dat CHOrderDetail
	list, err := ListCHOrderDetailByReq(req)
	if nil != err {
		return dat, err
	}
	if 0 == len(list) {
		return dat, fmt.Errorf("未找到 CH 详情 [%s]", changeNumber)
	}
	if len(list) > 1 {
		return dat, fmt.Errorf("ITSM 返回 [%d] 个 [%s] 详情信息", len(list), changeNumber)
	}
	return list[0], nil
}

// ListCHOrderDetail 获取 CH 详情列表
func ListCHOrderDetail(startTime, endTime string) ([]CHOrderDetail, error) {
	req := CHOrderDetailReq{
		StartTime: startTime,
		EndTime:   endTime,
	}
	return ListCHOrderDetailByReq(req)
}

// ListCHOrderDetailByReq 获取 CH 详情列表
func ListCHOrderDetailByReq(req CHOrderDetailReq) ([]CHOrderDetail, error) {
	rst := struct {
		Code    int             `json:"code"`
		Data    []CHOrderDetail `json:"data"`
		Message string          `json:"message"`
	}{}
	queryPath := queryChangeDetailUrl
	if "" != req.StartTime {
		queryPath = fmt.Sprintf("%s?mintime=%s&maxtime=%s", queryChangeDetailUrl, req.StartTime, req.EndTime)
	}
	if "" != req.ChangeNo {
		queryPath = fmt.Sprintf("%s?changnumber=%s", queryChangeDetailUrl, req.ChangeNo)
	}
	if err := doGet(queryPath, &rst); nil != err {
		return rst.Data, err
	}
	if 0 != rst.Code {
		return rst.Data, fmt.Errorf("create CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst.Data, nil
}

// GetCHDataEnum 获取全部枚举值
func GetCHDataEnum() (CHDataEnum, error) {
	dat := CHDataEnum{
		Desc: map[string]string{
			"env":                 "环境，HJ",
			"urgent_reason":       "紧急变更原因，urgentchangereason",
			"rollback_difficulty": "回退难易程度，retreatDifficulty",
			"risk_operate":        "操作规范，operateNorm",
			"risk_control":        "风险控制点，riskControl",
			"affect_range":        "影响范围程度，affectRange",
			"ch_obj_type":         "变更对象类型 changeObjType",
		},
	}
	var eList []error
	wg := sync.WaitGroup{}
	wg.Add(7)

	// 紧急变更原因 urgentchangereason
	go func() {
		defer wg.Done()
		var err error
		dat.UrgentReason, err = GetCHDataEnumByCode("urgentchangereason")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 回退难易程度 retreatDifficulty
	go func() {
		defer wg.Done()
		var err error
		dat.RollbackDifficulty, err = GetCHDataEnumByCode("retreatDifficulty")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 操作规范性 operateNorm
	go func() {
		defer wg.Done()
		var err error
		dat.RiskOperate, err = GetCHDataEnumByCode("operateNorm")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 影响范围程度 affectRange
	go func() {
		defer wg.Done()
		var err error
		dat.AffectRange, err = GetCHDataEnumByCode("affectRange")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 风险控制点 riskControl
	go func() {
		defer wg.Done()
		var err error
		dat.RiskControl, err = GetCHDataEnumByCode("riskControl")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 环境 HJ
	go func() {
		defer wg.Done()
		var err error
		dat.Env, err = GetCHDataEnumByCode("HJ")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	// 变更对象类型 changeObjType
	go func() {
		defer wg.Done()
		var err error
		dat.CHObjType, err = GetCHDataEnumByCode("changeObjType")
		if nil != err {
			eList = append(eList, err)
		}
	}()

	wg.Wait()
	if len(eList) > 0 {
		msg := ""
		for _, item := range eList {
			msg = msg + item.Error() + "; "
		}
		return dat, errors.New(msg)
	}
	return dat, nil
}

// GetCHDataEnumByCode 通过 CODE 查询字段枚举
// ITSM 支持的枚举值
// 紧急变更原因 urgentchangereason
// 回退难易程度 retreatDifficulty
// 操作规范性 operateNorm
// 影响范围程度 affectRange
// 风险控制点 riskControl
// 环境 HJ
// 变更对象类型 changeObjType
func GetCHDataEnumByCode(code string) ([]EnumItem, error) {
	rst := struct {
		Code    int            `json:"code"`
		Data    []EnumItemITSM `json:"data"`
		Message string         `json:"message"`
	}{}
	queryPath := fmt.Sprintf("%s?code=%s", queryCHDataEnum, code)
	list := make([]EnumItem, 0)
	if err := doGet(queryPath, &rst); nil != err {
		return list, err
	}
	if 0 != rst.Code {
		return list, fmt.Errorf("变更单枚举[%s]查询失败，rst code [%d], msg [%s]", code, rst.Code, rst.Message)
	}
	for _, item := range rst.Data {
		list = append(list, EnumItem{
			Code: item.Code,
			Name: item.NameCN,
		})
	}
	return list, nil
}
