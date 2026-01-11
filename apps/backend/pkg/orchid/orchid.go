package orchid

import (
	"fmt"
)

func QueryUser(um string) (ListUserRst, error) {
	rst := ListUserRst{}
	if err := doGet(fmt.Sprintf("api/query/user_info_for_85400?id=%s", um), &rst); err != nil {
		return rst, err
	}
	return rst, nil
}

func QueryAppIdOwner(appid string) (ListAppIdRst, error) {
	rst := ListAppIdRst{}
	if err := doGet(fmt.Sprintf("api/query/app_info_for_85400?id=%s", appid), &rst); err != nil {
		return rst, err
	}
	return rst, nil
}
