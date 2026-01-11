package orchid

var gCfg Config

func Init(cfg Config) {
	gCfg = cfg
}

type Config struct {
	Host  string
	Token string
}
