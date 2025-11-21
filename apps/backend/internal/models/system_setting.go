package models

import (
	"gorm.io/datatypes"
)

// SystemSetting 系统设置模型
type SystemSetting struct {
	BaseModel
	Key         string         `json:"key" gorm:"uniqueIndex;not null"`
	Value       datatypes.JSON `json:"value"`
	Description string         `json:"description"`
}

// TableName 指定表名
func (SystemSetting) TableName() string {
	return "system_settings"
}
