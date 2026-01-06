package navy

// BaseModel 基础模型
type BaseModel struct {
	ID        int      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt NavyTime `gorm:"column:created_at;type:datetime" json:"created_at"`
	UpdatedAt NavyTime `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// Device 设备信息
type Device struct {
	BaseModel
	CICode         string  `gorm:"column:ci_code;type:varchar(255);index" json:"ci_code"`
	IP             string  `gorm:"column:ip;type:varchar(50);index" json:"ip"`
	ArchType       string  `gorm:"column:arch_type;type:varchar(50)" json:"arch_type"`
	IDC            string  `gorm:"column:idc;type:varchar(100)" json:"idc"`
	Room           string  `gorm:"column:room;type:varchar(100)" json:"room"`
	Cabinet        string  `gorm:"column:cabinet;type:varchar(100)" json:"cabinet"`
	CabinetNO      string  `gorm:"column:cabinet_no;type:varchar(100)" json:"cabinet_no"`
	InfraType      string  `gorm:"column:infra_type;type:varchar(100)" json:"infra_type"`
	IsLocalization bool    `gorm:"column:is_localization;type:boolean" json:"is_localization"`
	NetZone        string  `gorm:"column:net_zone;type:varchar(100)" json:"net_zone"`
	Group          string  `gorm:"column:group;type:varchar(100)" json:"group"`
	AppID          string  `gorm:"column:appid;type:varchar(100)" json:"appid"`
	OsCreateTime   string  `gorm:"column:os_create_time;type:varchar(100)" json:"os_create_time"`
	CPU            float64 `gorm:"column:cpu;type:float" json:"cpu"`
	Memory         float64 `gorm:"column:memory;type:float" json:"memory"`
	Model          string  `gorm:"column:model;type:varchar(100)" json:"model"`
	KvmIP          string  `gorm:"column:kvm_ip;type:varchar(50)" json:"kvm_ip"`
	OS             string  `gorm:"column:os;type:varchar(100)" json:"os"`
	Company        string  `gorm:"column:company;type:varchar(100)" json:"company"`
	OSName         string  `gorm:"column:os_name;type:varchar(100)" json:"os_name"`
	OSIssue        string  `gorm:"column:os_issue;type:varchar(100)" json:"os_issue"`
	OSKernel       string  `gorm:"column:os_kernel;type:varchar(100)" json:"os_kernel"`
	Status         string  `gorm:"column:status;type:varchar(50)" json:"status"`
	Role           string  `gorm:"column:role;type:varchar(100)" json:"role"`
	Cluster        string  `gorm:"column:cluster;type:varchar(255)" json:"cluster"`
	ClusterID      int     `gorm:"column:cluster_id;type:int" json:"cluster_id"`
	K8sStatus      string  `gorm:"column:k8s_status;type:varchar(50)" json:"k8s_status"` // K8s 节点状态: Ready/NotReady/Unschedulable
	AcceptanceTime string  `gorm:"column:acceptance_time;type:varchar(100)" json:"acceptance_time"`
	DiskCount      int     `gorm:"column:disk_count" json:"disk_count"`
	DiskDetail     string  `gorm:"column:disk_detail" json:"disk_detail"`
	NetworkSpeed   string  `gorm:"column:network_speed" json:"network_speed"`

	// 只读字段
	IsSpecial    bool   `gorm:"column:is_special;->" json:"is_special"`
	FeatureCount int    `gorm:"column:feature_count;->" json:"feature_count"`
	AppName      string `gorm:"column:app_name;->" json:"app_name"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "device"
}

// K8sNode 表示 Kubernetes 节点信息
type K8sNode struct {
	BaseModel
	NodeName                string `gorm:"column:nodename;type:varchar(191);not null" json:"nodename"`
	HostIP                  string `gorm:"column:hostip;type:varchar(191)" json:"hostip"`
	Role                    string `gorm:"column:role;type:varchar(191)" json:"role"`
	OSImage                 string `gorm:"column:osimage;type:varchar(128)" json:"osimage"`
	KernelVersion           string `gorm:"column:kernelversion;type:varchar(64)" json:"kernelversion"`
	KubeletVersion          string `gorm:"column:kubeletversion;type:varchar(64)" json:"kubeletversion"`
	ContainerRuntimeVersion string `gorm:"column:containerruntimeversion;type:varchar(64)" json:"containerruntimeversion"`
	KubeProxyVersion        string `gorm:"column:kubeproxyversion;type:varchar(64)" json:"kubeproxyversion"`
	CPULogic                string `gorm:"column:cpulogic;type:varchar(191)" json:"cpulogic"`
	MemLogic                string `gorm:"column:memlogic;type:varchar(191)" json:"memlogic"`
	CPUCapacity             string `gorm:"column:cpucapacity;type:varchar(191)" json:"cpucapacity"`
	MemCapacity             string `gorm:"column:memcapacity;type:varchar(191)" json:"memcapacity"`
	CPUAllocatable          string `gorm:"column:cpuallocatable;type:varchar(191)" json:"cpuallocatable"`
	MemAllocatable          string `gorm:"column:memallocatable;type:varchar(191)" json:"memallocatable"`
	FSTypeRoot              string `gorm:"column:fstyperoot;type:varchar(191)" json:"fstyperoot"`
	DiskRoot                string `gorm:"column:diskroot;type:varchar(191)" json:"diskroot"`
	DiskDocker              string `gorm:"column:diskdocker;type:varchar(191)" json:"diskdocker"`
	DiskKubelet             string `gorm:"column:diskkubelet;type:varchar(191)" json:"diskkubelet"`
	NodeCreated             string `gorm:"column:nodecreated;type:varchar(191)" json:"nodecreated"`
	Status                  string `gorm:"column:status;type:varchar(191)" json:"status"`
	K8sClusterID            int    `gorm:"column:k8s_cluster_id;type:bigint unsigned" json:"k8s_cluster_id"`
}

// TableName 指定表名
func (K8sNode) TableName() string {
	return "k8s_node"
}

// QueryTemplate 查询模板
type QueryTemplate struct {
	BaseModel
	Name        string `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description string `gorm:"column:description;type:text" json:"description"`
	Groups      string `gorm:"column:groups;type:text;not null" json:"groups"` // JSON string
	CreatedBy   string `gorm:"column:created_by;type:varchar(255)" json:"created_by"`
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(255)" json:"updated_by"`
}

func (QueryTemplate) TableName() string {
	return "query_template"
}

// K8sCluster 表示 Kubernetes 集群信息 (仅保留 device 查询所需的最小字段)
type K8sCluster struct {
	BaseModel
	Name string `gorm:"column:name;type:varchar(191);not null" json:"name"`
}

func (K8sCluster) TableName() string {
	return "k8s_cluster"
}
