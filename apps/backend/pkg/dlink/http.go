package dlink

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func unicodeToZh(text string) string {
	splits := strings.Split(text, "\\u")
	rst := ""
	for _, v := range splits {
		if len(v) < 1 {
			continue
		}
		rst += fmt.Sprintf("%v", v)
	}
	return rst
}

func doGet(reqUrl string, rst interface{}) error {
	reqUrl = gCfg.Host + reqUrl
	r, err := http.NewRequest("GET", reqUrl, nil)
	if nil != err {
		return err
	}
	r.Header.Set("Authorization", gCfg.Token)

	cli := http.Client{}
	cli.Timeout = time.Duration(10) * time.Second

	res, err := cli.Do(r)
	if nil != err {
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
		return fmt.Errorf("request saturn GET: %v, code not 200 :%v, body -> %s", reqUrl, res.StatusCode, unicodeToZh(string(dat)))
	}

	err = json.Unmarshal(dat, rst)
	if err != nil {
		return err
	}

	return nil
}
