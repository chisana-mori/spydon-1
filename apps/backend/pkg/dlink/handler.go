package dlink

import "fmt"

func ListVMHostInfoByBMCICicode(cicode string) (ListVMHostInfoByBMCIRst, error) {
	rst := ListVMHostInfoByBMCIRst{}
	if err := doGet("/api/v1/query/dwd/list-vm-host-info-by-bm-ci?cicode="+cicode, &rst); nil != err {
		return rst, err
	}
	return rst, nil
}

func ListHostByCI(cicode string) (ListHostByCIRst, error) {
	rst := ListHostByCIRst{}
	if err := doGet("/api/v1/query/dwd/list-host-by-ci?cicode="+cicode, &rst); nil != err {
		return rst, err
	}
	return rst, nil
}

func ListHostByHostIP(hostIP string) (ListHostByCIRst, error) {
	rst := ListHostByCIRst{}
	if err := doGet("/api/v1/query/dwd/list-host-by-ci?host_ip="+hostIP, &rst); nil != err {
		return rst, err
	}
	return rst, nil
}

func ListDeviceByAppId(appId, environment string) (ListDeviceByAppIdRst, error) {
	rst := ListDeviceByAppIdRst{}
	if err := doGet(fmt.Sprintf("/api/v1/query/dws/list-host-info-by-appid?status=3&appid=%s&environment=%s", appId, environment), &rst); nil != err {
		return rst, err
	}
	return rst, nil
}
