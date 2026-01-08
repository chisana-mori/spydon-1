package dragonfly

import "fmt"

// 变更单相关接口
const (
	createPreAuthChangeUrl = "/api/v2/itsm/preauthorization/change_v3" // 以前是分开的，现在使用 createChangeUrl 即可
	createChangeUrl        = "/api/v2/original/change_v3"              // 创建常规变更单
	startChangeUrl         = "/api/v2/original/change/has_started_v2/" // 开始变更单
	closeChangeUrl         = "/api/v2/original/change/has_done/"
	cancelChangeUrl        = "/api/v2/original/change/cancel"
	verifyChangeUrl        = "/api/v2/original/change/verification"
	// closeVerifyChangeUrl = "/api/v2/original/change/has_done_with_verifty" // 以前使用，现在不使用了
)

// CreateCHOrder 创建普通 CH 单
func CreateCHOrder(req CreateCHOrderReq) (CreateCHOrderRst, error) {
	var err error
	rst := CreateCHOrderRst{}
	if rst.OriginData, err = doPost(createChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Create CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}

// CreatePreAuthCHOrder 创建预授权 CH 单
func CreatePreAuthCHOrder(req CreatePreAuthCHOrderReq) (CreateCHOrderRst, error) {
	var err error
	rst := CreateCHOrderRst{}
	if rst.OriginData, err = doPost(createPreAuthChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Create PreAuthCH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}

// StartCHOrder 开始 CH 单
func StartCHOrder(req StartCHOrderReq) (StartCHOrderRst, error) {
	var err error
	rst := StartCHOrderRst{}
	if rst.OriginData, err = doPost(startChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Start CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}

// CloseCHOrder 关闭 CH 单
func CloseCHOrder(req CloseCHOrderReq) (CloseCHOrderRst, error) {
	var err error
	rst := CloseCHOrderRst{}
	if rst.OriginData, err = doPost(closeChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Close CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}

// CancelCHOrder 取消 CH 单
func CancelCHOrder(req CancelCHOrderReq) (CancelCHOrderRst, error) {
	var err error
	rst := CancelCHOrderRst{}
	if rst.OriginData, err = doPost(cancelChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Cancel CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}

// VerifyCHOrder 验证 CH 单
func VerifyCHOrder(req VerifyCHOrderReq) (VerifyCHOrderRst, error) {
	var err error
	rst := VerifyCHOrderRst{}
	if rst.OriginData, err = doPost(verifyChangeUrl, req, &rst); nil != err {
		return rst, err
	}
	if 0 != rst.Code {
		return rst, fmt.Errorf("dragonfly: Verify CH error, rst code [%d], msg [%s]", rst.Code, rst.Message)
	}
	return rst, nil
}
