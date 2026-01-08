package http

// =============================================================================
// Route Groups - 路由组路径
// =============================================================================

const (
	RouteGroupNavy      = "/navy"
	RouteGroupDevices   = "/devices"
	RouteGroupDrain     = "/drain"
	RouteGroupTemplates = "/templates"
	RouteGroupDeviceOps = "/device-ops"
	RouteGroupK8sNodes  = "/k8s-nodes"
	RouteGroupF5        = "/f5"
)

// =============================================================================
// Route Paths - 路由路径
// =============================================================================

const (
	// Device routes
	RouteQuery             = "/query"
	RouteFilterOptions     = "/filter-options"
	RouteLabelValues       = "/label-values"
	RouteTaintValues       = "/taint-values"
	RouteDeviceFieldValues = "/device-field-values"
	RouteFeatures          = "/features"
	RouteFeatureDetails    = "/feature-details"
	RouteExport            = "/export"

	// Path parameters
	RouteParamID            = "/:id"
	RouteParamIDRole        = "/:id/role"
	RouteParamIDGroup       = "/:id/group"
	RouteParamNode          = "/:node"
	RouteParamNodeDrainOps  = "/:node/drain/start"
	RouteParamNodeDrainStop = "/:node/drain/cancel"

	// Drain routes
	RouteStart                  = "/start"
	RouteParamDrainIDCancel     = "/:drain_id/cancel"
	RouteParamDrainIDMigrations = "/:drain_id/migrations"
	RouteParamDrainIDEvents     = "/:drain_id/events"

	// DeviceOps routes
	RouteCordon   = "/cordon"
	RouteUncordon = "/uncordon"
	RouteDrain    = "/drain"
	RouteTaint    = "/taint"
	RouteLabel    = "/label"
	RouteShutdown = "/shutdown"
	RouteReboot   = "/reboot"

	// K8s nodes routes
	RouteLabelsAndTaints = "/labels-taints"
)

// =============================================================================
// Response Messages - 响应消息
// =============================================================================

const (
	// Success messages
	MsgSuccess             = "操作成功"
	MsgDeviceRoleUpdated   = "设备角色已更新"
	MsgDeviceGroupUpdated  = "设备分组已更新"
	MsgDrainStarted        = "Drain 操作已启动"
	MsgDrainCanceled       = "Drain 操作已取消"
	MsgMigrationsRetrieved = "迁移信息获取成功"
	MsgF5Updated           = "F5 信息更新成功"
	MsgF5Deleted           = "F5 信息删除成功"

	// Error messages
	MsgInvalidID          = "无效的 ID"
	MsgInvalidParams      = "无效的请求参数"
	MsgInvalidQueryParams = "无效的查询参数"
	MsgMissingParams      = "缺少必要参数"
	MsgDrainIDRequired    = "drain_id 不能为空"
	MsgClusterRequired    = "cluster 参数不能为空"
	MsgF5NotFound         = "F5 信息未找到"
	MsgInternalError      = "服务内部错误"
)
