package navy

import (
	"robusta-web/backend/internal/models"
	"time"
)

// F5Info F5负载均衡信息.
type F5Info struct {
	BaseModel
	Name          string          `gorm:"column:name;type:varchar(255);not null"`    // 名称
	VIP           string          `gorm:"column:vip;type:varchar(15);not null"`      // 虚拟IP
	Port          string          `gorm:"column:port;type:varchar(10);not null"`     // 端口
	AppID         string          `gorm:"column:appid;type:varchar(50)"`             // 应用ID
	InstanceGroup string          `gorm:"column:instance_group;type:varchar(50)"`    // 实例组
	Status        string          `gorm:"column:status;type:varchar(50)"`            // 状态
	PoolName      string          `gorm:"column:pool_name;type:varchar(50)"`         // Pool名称
	PoolStatus    string          `gorm:"column:pool_status;type:varchar(50)"`       // Pool状态
	PoolMembers   string          `gorm:"column:pool_members;type:text"`             // Pool成员
	ClusterID     *uint           `gorm:"column:k8s_cluster_id;type:int unsigned"`   // 集群ID (使用现有k8s_cluster_id列，映射到robusta_hub.clusters)
	Cluster       *models.Cluster `gorm:"foreignKey:ClusterID;references:ID;constraint:OnDelete:SET NULL"` // 集群信息 (关联robusta_hub.clusters表)
	Domains       string          `gorm:"column:domains;type:text"`                  // 域名
	GrafanaParams string          `gorm:"column:grafana_params;type:text"`           // Grafana参数
	Ignored       bool            `gorm:"column:ignored;type:boolean;default:false"` // 是否忽略
	CreatedAt     time.Time       `gorm:"column:created_at;type:datetime"`           // 创建时间
	UpdatedAt     time.Time       `gorm:"column:updated_at;type:datetime"`           // 更新时间
	Deleted       string          `gorm:"column:deleted;type:varchar(255)"`          // 软删除标记
}

// TableName 指定表名.
func (F5Info) TableName() string {
	return "k8s_f5_cluster"
}
