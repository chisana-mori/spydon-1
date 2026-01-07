package navy

type DeviceApp struct {
	BaseModel
	AppId       string
	Type        int    `gorm:"int:2"` // 0 设备 1 组件
	Name        string // 类型说明
	Owner       string `gorm:"varchar:255"` // 申请人
	Feature     string `gorm:"feature"`     // 用途 (Note: Tag value might be incomplete)
	Description string `gorm:"text:1000"`   // 描述信息
	Status      int    `gorm:"int:1"`       // 0 收集 1 停止收集
}

func (DeviceApp) TableName() string {
	return "device_app"
}
