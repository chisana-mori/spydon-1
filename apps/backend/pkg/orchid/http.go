package orchid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func doGet(reqUrl string, rst interface{}) (err error) {
	reqUrl = gCfg.Host + reqUrl
	r, err := http.NewRequest("GET", reqUrl, nil)
	if nil != err {
		//log.Error("saturn http init error", url, err.Error())
		return err
	}
	r.Header.Set("x-access-token", gCfg.Token)
	cli := http.Client{}
	cli.Timeout = time.Duration(10) * time.Second

	res, err := cli.Do(r)
	if nil != err {
		//log.Error("saturn http init error", url, err.Error())
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("request saturn GET: %v, code not 200 :%v, body", reqUrl, res.StatusCode)
	}
	err = json.Unmarshal(dat, &rst)
	if err != nil {
		return err
	}
	return nil
}
