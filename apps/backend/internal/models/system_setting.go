package models

import (
	"gorm.io/datatypes"
)

// SystemSetting 系统设置模型
type SystemSetting struct {
	BaseModel
	SettingKey  string         `json:"key" gorm:"column:setting_key;type:varchar(191);uniqueIndex;not null"`
	Value       datatypes.JSON `json:"value" gorm:"type:json"`
	Description string         `json:"description"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}
