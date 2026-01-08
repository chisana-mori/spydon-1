package errs

var eMap map[int]string

const (
	Success                 = 200
	BadRequest              = -400
	Unauthorized            = -401
	Forbidden               = -403
	NotFound                = -404
	UnauthorizedDegradation = -499
	InternalErr             = -500

	SessCreateErr             = -1000
	SessFoundErr              = -1001
	SessLoginPasswordErr      = -1003
	DegradationLoginNotEnable = -1004
	DegradationLoginErr       = -1005

	MicroErr         = -2000
	QueryCMSErr      = 2100
	QueryCMSNotFound = 2404

	// 序列化错误
	MarshalError   = -3000
	UnMarshalError = -3001

	GeneralError = -3002

	// DB操作相关错误
	DBConnectErr    = -9000
	DBExecErr       = -9010
	DBReadErr       = -9001
	DBUpdateErr     = -9002
	DBInsertErr     = -9003
	DBRemoveErr     = -9004
	DBDuplicateItem = -9010
	DBMissingItem   = -9011

	// StrMaxErr 常用参数校验错误
	StrMaxErr  = -9100
	StrMinErr  = -9101
	PageMinErr = -9102
	IdBlankErr = -9103

	// third call
	ThirdCallReqErr    = -9200
	ThirdCallHandleErr = -9201
	ThirdCallRespErr   = -9202

	KafkaPushErr     = -9300
	ElasticSearchErr = -9400

	// k8s
	K8sClientCreateErr = -9500
	DeletePodErr       = -9501
	CordonUncordonErr  = -9502
	GetPodErr          = -9503
)

func init() {
	eMap = map[int]string{
		Success:      "成功",
		BadRequest:   "不合法的请求参数",
		Unauthorized: "身份未认证",
		Forbidden:    "请求被拒绝",
		NotFound:     "接口不存在",
		InternalErr:  "系统内部错误",
		DBConnectErr: "数据库连接错误",
		DBUpdateErr:  "数据库更新失败",
		DBInsertErr:  "数据库插入失败",
		DBReadErr:    "数据库读失败",
		DBExecErr:    "SQL exec error",
		StrMaxErr:    "字符过长",
		StrMinErr:    "字符过短",
		PageMinErr:   "分页 page 必须大于 0",
		IdBlankErr:   "Id 不能为空",
		MicroErr:     "micro error",
	}
}

func GetMsg(code int) string {
	if msg, ok := eMap[code]; ok {
		return msg
	}
	return eMap[InternalErr]
}
