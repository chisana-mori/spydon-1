package dragonfly

var gCfg Config

func Init(cfg Config) {
	gCfg = cfg
}

type Config struct {
	Host     string
	Token    string
	TimeOut  int64
	UseProxy bool
	Proxy    string
	IsDebug  bool
}
