package services

import (
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/errs"
	"robusta-web/backend/pkg/logger"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
)

var (
	// deployService 单例实例，用于向后兼容的函数调用
	deployService *DeployService
)

// InitDeployService 初始化 Deployment 服务单例
func InitDeployService(nodeManager *nodesync.Manager, database *db.Database) {
	deployService = NewDeployService(nodeManager, database)
}

// DeployService Deployment 服务
type DeployService struct {
	nodeManager *nodesync.Manager
	db          *db.Database
}

// NewDeployService 创建 Deployment 服务
func NewDeployService(nodeManager *nodesync.Manager, database *db.Database) *DeployService {
	return &DeployService{
		nodeManager: nodeManager,
		db:          database,
	}
}

// InstantDeploys 获取指定集群的实时 Deployments（服务方法）
func (s *DeployService) InstantDeploys(cluster string) []App {
	apps := make([]App, 0)

	deployList, err := s.nodeManager.ListDeployments(cluster, "")
	if err != nil {
		logger.L().Error("获取 Deployments 失败",
			zap.String("cluster", cluster),
			zap.Error(err))
		return apps
	}

	for _, deploy := range deployList {
		ele := App{}
		if deploy.Labels != nil {
			ele.AppId = deploy.Labels["appid"]
		}
		ele.Namespace = deploy.Namespace
		ele.Name = deploy.Name
		apps = append(apps, ele)
	}
	return apps
}

// InstantDeploysByNamespace 获取指定集群和命名空间的 Deployments
func (s *DeployService) InstantDeploysByNamespace(cluster, namespace string) []appsv1.Deployment {
	deployList, err := s.nodeManager.ListDeployments(cluster, namespace)
	if err != nil {
		logger.L().Error("获取 Deployments 失败",
			zap.String("cluster", cluster),
			zap.String("namespace", namespace),
			zap.Error(err))
		return make([]appsv1.Deployment, 0)
	}
	return deployList
}

// nextDay 计算下一天
func nextDay(begin string) string {
	if begin == "" {
		return ""
	}
	t, err := time.Parse(time.DateOnly, begin)
	if err != nil {
		return begin
	}
	return t.AddDate(0, 0, 1).Format(time.DateOnly)
}

// HistoryDeploys 获取历史 Deployments（服务方法）
func (s *DeployService) HistoryDeploys(clusterId uint, begin string) ([]App, errs.Error) {
	rst := make([]App, 0)
	end := nextDay(begin)
	sql := `
        SELECT
            appid, namespace, deployment as resourceName
        FROM
            navy.k8s_deployment_info
        WHERE created_at BETWEEN ? AND ? AND k8s_cluster_id = ? AND appid != ''
        ORDER BY id DESC
    `

	dbRst := make([]map[string]interface{}, 0)
	if qErr := s.db.Raw(sql, begin, end, clusterId).Scan(&dbRst).Error; qErr != nil {
		logger.L().Error("获取历史deploy失败", zap.Error(qErr))
		err := errs.NewErrorWithMsg(errs.InternalErr, qErr.Error())
		return rst, err
	}

	for _, record := range dbRst {
		appId, ok := record["appid"].(string)
		if !ok {
			continue
		}
		resourceName, ok := record["resourceName"].(string)
		if !ok {
			continue
		}
		namespace, ok := record["namespace"].(string)
		if !ok {
			continue
		}
		ele := App{
			AppId:     appId,
			Name:      resourceName,
			Namespace: namespace,
		}
		rst = append(rst, ele)
	}
	return rst, nil
}

// InstantDeploys 包级函数，用于向后兼容
func InstantDeploys(cluster string) []App {
	if deployService == nil {
		logger.L().Warn("DeployService 未初始化，返回空结果")
		return make([]App, 0)
	}
	return deployService.InstantDeploys(cluster)
}

// HistoryDeploys 包级函数，用于向后兼容
func HistoryDeploys(clusterId uint, begin string) ([]App, errs.Error) {
	if deployService == nil {
		logger.L().Warn("DeployService 未初始化，返回错误")
		return nil, errs.NewErrorWithMsg(errs.InternalErr, "DeployService 未初始化")
	}
	return deployService.HistoryDeploys(clusterId, begin)
}
