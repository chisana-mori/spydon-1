package services

// F5InfoQuery defines query parameters for listing F5Info.
type F5InfoQuery struct {
	Page          int    `json:"page" form:"page" binding:"required,min=1"`
	Size          int    `json:"size" form:"size" binding:"required,min=1,max=100"`
	Name          string `json:"name" form:"name"`
	VIP           string `json:"vip" form:"vip"`
	Port          string `json:"port" form:"port"`
	AppID         string `json:"appid" form:"appid"`
	InstanceGroup string `json:"instance_group" form:"instance_group"`
	Status        string `json:"status" form:"status"`
	PoolName      string `json:"pool_name" form:"pool_name"`
	ClusterName   string `json:"cluster_name" form:"cluster_name"`
	Keyword       string `json:"keyword" form:"keyword"`
	SortBy        string `json:"sort_by" form:"sort_by"`
	SortOrder     string `json:"sort_order" form:"sort_order"`
}

// F5InfoUpdateDTO defines the data transfer object for updating F5Info.
type F5InfoUpdateDTO struct {
	Name          string `json:"name" binding:"required"`
	VIP           string `json:"vip" binding:"required"`
	Port          string `json:"port" binding:"required"`
	AppID         string `json:"appid" binding:"required"`
	InstanceGroup string `json:"instance_group"`
	Status        string `json:"status"`
	PoolName      string `json:"pool_name"`
	PoolStatus    string `json:"pool_status"`
	PoolMembers   string `json:"pool_members"`
	ClusterID     *uint  `json:"k8s_cluster_id"` // 使用现有字段名
	Domains       string `json:"domains"`
	GrafanaParams string `json:"grafana_params"`
	Ignored       bool   `json:"ignored"`
}

// F5InfoResponse defines the response structure for a single F5Info.
type F5InfoResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	VIP           string `json:"vip"`
	Port          string `json:"port"`
	AppID         string `json:"appid"`
	InstanceGroup string `json:"instance_group"`
	Status        string `json:"status"`
	PoolName      string `json:"pool_name"`
	PoolStatus    string `json:"pool_status"`
	PoolMembers   string `json:"pool_members"`
	ClusterID     *uint  `json:"k8s_cluster_id"` // 保持与现有表一致
	ClusterName   string `json:"cluster_name"`   // 从关联的Cluster表获取
	Domains       string `json:"domains"`
	GrafanaParams string `json:"grafana_params"`
	Ignored       bool   `json:"ignored"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// F5InfoListResponse defines the response structure for a list of F5Info.
type F5InfoListResponse struct {
	List  []*F5InfoResponse `json:"list"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
	Total int64             `json:"total"`
}
