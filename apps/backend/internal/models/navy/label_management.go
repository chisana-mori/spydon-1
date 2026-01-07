package navy

type LabelManagement struct {
	BaseModel
	Name        string       `gorm:"column:name"`
	Key         string       `gorm:"column:key"`
	Source      int          `gorm:"int:2;column:source"` // 0内部 1外部 2其他
	Status      int          `gorm:"column:status"`       // 0正常 1停用
	LabelValues []LabelValue `gorm:"foreignkey:LabelID"`
	Nodes       []K8sNode    `gorm:"many2many:k8s_node_label_features"`
}

func (l LabelManagement) TableName() string {
	return "label_feature"
}

type LabelValue struct {
	BaseModel
	LabelID int    `gorm:"column:label_id"`
	Value   string `gorm:"column:value"`
}

func (l LabelValue) TableName() string {
	return "label_feature_value"
}
