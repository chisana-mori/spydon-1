package orchid

type ListUserRst struct {
	Status  string       `json:"status"`
	Data    UserDateResp `json:"data"`
	Message string       `json:"message"`
}

type UserDateResp struct {
	User []User `json:"user"`
}

type User struct {
	Id          string `json:"id"`
	DisplayName string `json:"display-name"`
	Org         string `json:"org"`
	Team        string `json:"team"`
	Manager     string `json:"manager"`
	StaffType   string `json:"staff-type"`
	Mobile      string `json:"mobile"`
	Email       string `json:"email"`
}

type ListAppIdRst struct {
	Status  string        `json:"status"`
	Data    AppIdDateResp `json:"data"`
	Message string        `json:"message"`
}

type AppIdDateResp struct {
	App []AppIdDetail `json:"app"`
}

type AppIdDetail struct {
	Id         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Org        string `json:"org"`
	SubsysCode string `json:"subsys-code"`
	Dev        string `json:"dev"`
	Devs       string `json:"devs"`
	Qa         string `json:"qa"`
	Qas        string `json:"qas"`
	Pm         string `json:"pm"`
	Pms        string `json:"pms"`
	Ops        string `json:"ops"`
	OpsEdition string `json:"ops-edition"`
	Rm         string `json:"rm"`
	Level      string `json:"level"`
	Status     string `json:"status"`
}
