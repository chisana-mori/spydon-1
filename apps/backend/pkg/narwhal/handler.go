package narwhal

import (
	"net/http"
	"robusta-web/backend/pkg/logger"
	"time"

	"github.com/imroc/req"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func TodayOfficer() ([]string, error) {
	officers := make([]string, 2)
	nowDay := time.Now().Format(time.DateOnly)
	body := req.BodyJSON(map[string]interface{}{
		"day": []string{nowDay},
	})

	var resp []byte
	resp, err := doRequest("/modifybackend/api/v2/get_duty_info", http.MethodPost, body)
	if err != nil {
		logger.L().Error("获取当天应用值班人员失败", zap.String("day", nowDay), zap.Error(err))
	}

	morningOfficer := gjson.GetBytes(resp, "data.0.duty_morning").String()
	nightOfficer := gjson.GetBytes(resp, "data.0.duty_night").String()

	officers[0] = morningOfficer
	officers[1] = nightOfficer
	return officers, nil
}
