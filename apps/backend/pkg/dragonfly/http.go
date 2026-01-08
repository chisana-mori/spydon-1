package dragonfly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"robusta-web/backend/pkg/logger"
)

func debugLogf(format string, a ...interface{}) {
	if gCfg.IsDebug {
		logger.S().Debugf("Dragonfly-Debug: "+format, a...)
	}
}

// unicodeToZh 处理 python json.dumps 造成的中文 utf8 的 \u 问题
func unicodeToZh(byt []byte) string {
	var rst interface{}
	err := json.Unmarshal(byt, &rst)
	if nil != err {
		return string(byt)
	}
	dat, err := json.Marshal(rst)
	if nil != err {
		return string(byt)
	}
	return string(dat)
}

func doGet(reqUrl string, rst interface{}) error {
	reqUrl = gCfg.Host + reqUrl
	r, err := http.NewRequest("GET", reqUrl, nil)
	if nil != err {
		// log.Error("dragonfly http init error", url, err.Error())
		return err
	}
	r.Header.Set("Token", gCfg.Token)

	cli := http.Client{}
	cli.Timeout = time.Duration(gCfg.TimeOut) * time.Second

	// 如果需要使用代理
	if gCfg.UseProxy {
		debugLogf("request dragonflyUrl: %v, use proxy: %v", reqUrl, gCfg.Proxy)
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(gCfg.Proxy)
		}
		transPort := &http.Transport{
			Proxy: proxy,
		}
		cli.Transport = transPort
	}

	res, err := cli.Do(r)
	if nil != err {
		debugLogf("dragonfly http client init error, [%s], [%s]", reqUrl, err.Error())
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	dat, err := ioutil.ReadAll(res.Body)
	if err != nil {
		debugLogf("dragonfly http io read error, [%s], [%s]", reqUrl, err.Error())
		return err
	}

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("request dragonfly POST: %v, code not 200 :%v, body → %s", reqUrl, res.StatusCode, unicodeToZh(dat))
	}

	err = json.Unmarshal(dat, &rst)
	if err != nil {
		debugLogf("dragonfly http json unmarshal error, [%s], [%s]", reqUrl, err.Error())
		return err
	}

	return nil
}

func doPost(reqUrl string, req interface{}, rst interface{}) (string, error) {
	reqUrl = gCfg.Host + reqUrl
	body, err := json.Marshal(req)
	if nil != err {
		return "", fmt.Errorf("dragonfly json encode error, [%s], [%v], [%s]", reqUrl, req, err.Error())
	}
	r, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(body))
	if nil != err {
		return "", fmt.Errorf("dragonfly http init error, [%s], [%s]", reqUrl, err.Error())
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Token", gCfg.Token)

	cli := http.Client{}
	cli.Timeout = time.Duration(gCfg.TimeOut) * time.Second

	// 如果需要使用代理
	if gCfg.UseProxy {
		debugLogf("request dragonflyUrl: %v, use proxy: %v", reqUrl, gCfg.Proxy)
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(gCfg.Proxy)
		}
		transPort := &http.Transport{
			Proxy: proxy,
		}
		cli.Transport = transPort
	}

	res, err := cli.Do(r)
	if nil != err {
		return "", fmt.Errorf("dragonfly http io write error, [%s], [%s]", reqUrl, err.Error())
	}
	defer func() {
		_ = res.Body.Close()
	}()

	dat, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return unicodeToZh(dat), fmt.Errorf("dragonfly http io read error, [%s], [%s]", reqUrl, err.Error())
	}

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return unicodeToZh(dat), fmt.Errorf("request dragonfly POST: %v, code not 200 :%v, body → %s", reqUrl, res.StatusCode, unicodeToZh(dat))
	}

	debugLogf("%s | http body origin byte data → %s", reqUrl, unicodeToZh(dat))
	err = json.Unmarshal(dat, &rst)
	if err != nil {
		return unicodeToZh(dat), fmt.Errorf("dragonfly http json unmarshal error, [%s], [%s]", reqUrl, err.Error())
	}

	return unicodeToZh(dat), nil
}
