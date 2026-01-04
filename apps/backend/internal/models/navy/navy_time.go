package navy

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// NavyTime 自定义时间类型.
type NavyTime time.Time

const (
	timeFormat = time.DateOnly
)

// MarshalJSON 实现json序列化接口.
func (t NavyTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", time.Time(t).Format(timeFormat))
	return []byte(formatted), nil
}

// UnmarshalJSON 实现json反序列化接口.
func (t *NavyTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	// 去掉引号
	if len(data) < 2 {
		return fmt.Errorf("invalid time format")
	}
	str := string(data)[1 : len(data)-1]
	parsed, err := time.Parse(timeFormat, str)
	if err != nil {
		return err
	}
	*t = NavyTime(parsed)
	return nil
}

// Value 实现 driver.Valuer 接口.
func (t NavyTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// Scan 实现 sql.Scanner 接口.
func (t *NavyTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		*t = NavyTime(v)
	default:
		return fmt.Errorf("cannot scan type %T into NavyTime", value)
	}
	return nil
}

// String 实现 Stringer 接口.
func (t NavyTime) String() string {
	return time.Time(t).Format(timeFormat)
}

// UnmarshalParam 实现gin参数绑定接口.
func (t *NavyTime) UnmarshalParam(param string) error {
	if param == "" {
		return nil
	}
	parsed, err := time.Parse(timeFormat, param)
	if err != nil {
		return err
	}
	*t = NavyTime(parsed)
	return nil
}
