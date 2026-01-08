package narwhal

import (
	"github.com/imroc/req"
)

func doRequest(reqUrl string, method string, body interface{}) ([]byte, error) {
	reqUrl = gCfg.Host + reqUrl
	request := req.New()

	var resp *req.Resp
	resp, err := request.Do(method, reqUrl, body, req.Header{
		"AUTHORIZATION": gCfg.Token,
		"Content-type":  "application/json",
	})
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Response().Body.Close() }()
	response := resp.Bytes()
	return response, nil
}
