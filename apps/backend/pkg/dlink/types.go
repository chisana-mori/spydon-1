package dlink

type ListVMHostInfoByBMCIFilter struct {
	CICode            string `json:"cicode"`
	CICodeEntityField string `json:"cicode[entity.field]"`
	PageSIze          int    `json:"pageSize"`
}

type ListVMHostInfoByBMCIItem struct {
	Id          int    `json:"_id"`
	Appid       string `json:"appid"`
	CICode      string `json:"cicode"`
	Environment string `json:"environment"`
	HostIp      string `json:"host_ip"`
	Idc         string `json:"idc"`
	Rid         string `json:"rid"`
	Status      string `json:"status"`
	UsedBy      string `json:"used_by"`
	VSoftType   string `json:"vsoft_type"`
}

type ListVMHostInfoByBMCIRst struct {
	Code      int                        `json:"code"`
	CostTime  float64                    `json:"cost_time"`
	Filter    ListVMHostInfoByBMCIFilter `json:"filter"`
	List      []ListVMHostInfoByBMCIItem `json:"list"`
	ListNum   int                        `json:"list_num"`
	Msg       string                     `json:"msg"`
	Timestamp int                        `json:"timestamp"`
	TotalNum  int                        `json:"total_num"`
	TraceId   string                     `json:"trace_id"`
}

type ListHostByCIFilter struct {
	CICode            []string `json:"cicode"`
	CICodeEntityField string   `json:"cicode[entity.field]"`
	PageSIze          int      `json:"pageSize"`
}

type ListHostByCIItem struct {
	Id     int    `json:"_id"`
	CICode string `json:"cicode"`
	HostIp string `json:"host_ip"`
	Status string `json:"status"`
}

type ListHostByCIRst struct {
	Code      int                `json:"code"`
	CostTime  float64            `json:"cost_time"`
	Filter    ListHostByCIFilter `json:"filter"`
	List      []ListHostByCIItem `json:"list"`
	ListNum   int                `json:"list_num"`
	Msg       string             `json:"msg"`
	Timestamp int                `json:"timestamp"`
	TotalNum  int                `json:"total_num"`
	TraceId   string             `json:"trace_id"`
}

type DLinkDevice struct {
	CPUArch        string  `json:"cpu_arch"`
	CICode         string  `json:"cicode"`
	AppID          string  `json:"appid"`
	CPULogical     float64 `json:"cpu_logical"`
	CPUPhysical    float64 `json:"cpu_physical"`
	Environment    string  `json:"environment"`
	IP             string  `json:"host_ip"`
	IDC            string  `json:"idc"`
	IsLocalization bool    `json:"is_localization"`
	Memory         float64 `json:"mem"`
	OSName         string  `json:"os_name"`
	OSType         string  `json:"os_type"`
	OSVersion      string  `json:"os_version"`
	OsIssue        string  `json:"agent_osversion"`
	Status         string  `json:"status"`
	Kernel         string  `json:"agent_oskernel"`
	SubEnv         string  `json:"sub_env"`
	Model          string  `json:"model"`
	Company        string  `json:"manufacture"`
	NetZone        string  `json:"agent_netzone"`
	Cabinet        string  `json:"cabinet_id"`
	KVMIP          string  `json:"kvm_ip"`
	OSCreateTime   string  `json:"os_create_time"`
	Room           string  `json:"room_id"`
	InfraType      string  `json:"infra_type"`
	IPv6           string  `json:"host_ipv6"`
}

type ListDeviceByAppIdRst struct {
	Code      int           `json:"code"`
	CostTime  float64       `json:"cost_time"`
	List      []DLinkDevice `json:"list"`
	ListNum   int           `json:"list_num"`
	Msg       string        `json:"msg"`
	Timestamp int           `json:"timestamp"`
	TotalNum  int           `json:"total_num"`
	TraceId   string        `json:"trace_id"`
}
