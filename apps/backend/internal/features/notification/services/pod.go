package services

import (
	"fmt"
	"sort"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/errs"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/redis"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// PodService Pod 服务
type PodService struct {
	nodeManager *nodesync.Manager
	db          *db.Database
	redis       redis.Client
}

// NewPodService 创建 Pod 服务
func NewPodService(nodeManager *nodesync.Manager, database *db.Database, redisClient redis.Client) *PodService {
	return &PodService{
		nodeManager: nodeManager,
		db:          database,
		redis:       redisClient,
	}
}

var (
	// podService 单例实例，用于向后兼容的函数调用
	podService *PodService
)

// InitPodService 初始化 Pod 服务单例
func InitPodService(nodeManager *nodesync.Manager, database *db.Database, redisClient redis.Client) {
	podService = NewPodService(nodeManager, database, redisClient)
}

func buildAppBoards(deploys []App) (AppSlice, map[string]struct{}) {
	boards := AppSlice{}
	appIdMap := make(map[string]struct{})

	for _, ele := range deploys {
		board := AppBoard{}

		board.AppId = ele.AppId
		appIdMap[ele.AppId] = struct{}{}
		board.DeploymentName = ele.Name
		board.NamespaceName = ele.Namespace

		if len(board.AppId) == 0 {
			boards = append(boards, board)
			continue
		}

		// 从 Redis 获取应用元数据
		level, dev, op, namespaceCNName, team := getAppMetadata(board.AppId)

		board.Operations = op
		board.NamespaceCNName = namespaceCNName
		board.Team = team
		board.AppLevel = level
		board.Develops = dev
		boards = append(boards, board)
	}

	sort.Sort(boards)
	return boards, appIdMap
}

func getAppMetadata(appId string) (string, string, string, string, string) {
	if podService == nil || podService.redis == nil {
		return "", "", "", "", ""
	}

	level := podService.redis.Get(fmt.Sprintf("%s.level", appId))
	dev := podService.redis.Get(fmt.Sprintf("%s.dev", appId))
	op := podService.redis.Get(fmt.Sprintf("%s.op", appId))
	namespaceCNName := podService.redis.Get(fmt.Sprintf("%s.namespaceCNName", appId))
	team := podService.redis.Get(fmt.Sprintf("%s.opsTeam", appId))

	return level, dev, op, namespaceCNName, team
}

// InstantPods 获取实时 Pods（包级函数，向后兼容）
func InstantPods(cluster string, nodes []string) ([]App, errs.Error) {
	if podService == nil {
		logger.L().Warn("PodService 未初始化，返回错误")
		return nil, errs.NewErrorWithMsg(errs.InternalErr, "PodService 未初始化")
	}
	return podService.InstantPods(cluster, nodes)
}

// InstantPods 获取实时 Pods
func (s *PodService) InstantPods(cluster string, nodes []string) ([]App, errs.Error) {
	aggregatedPods := make([]v1.Pod, 0)

	podList, err := s.nodeManager.ListPodsOnNodes(cluster, nodes)
	if err != nil {
		return nil, errs.NewErrorWithMsg(errs.InternalErr, err.Error())
	}

	for _, pod := range podList {
		if len(pod.OwnerReferences) > 0 {
			for _, owner := range pod.OwnerReferences {
				if owner.Kind == "DaemonSet" {
					aggregatedPods = append(aggregatedPods, pod)
				}
			}
		}
	}

	rst := make([]App, 0)
	for _, pod := range aggregatedPods {
		var appId string
		var ok bool
		if appId, ok = pod.Labels["appid"]; !ok {
			appId = ""
		}

		element := App{
			AppId:     appId,
			Namespace: pod.ObjectMeta.Namespace,
			Name:      pod.Name,
		}
		rst = append(rst, element)
	}

	return rst, nil
}

// HistoryPods 获取历史 Pods（包级函数，向后兼容）
func HistoryPods(clusterId uint, begin string, nodes []string) ([]App, errs.Error) {
	if podService == nil {
		logger.L().Warn("PodService 未初始化，返回错误")
		return nil, errs.NewErrorWithMsg(errs.InternalErr, "PodService 未初始化")
	}
	return podService.HistoryPods(clusterId, begin, nodes)
}

// HistoryPods 获取历史 Pods
func (s *PodService) HistoryPods(clusterId uint, begin string, nodes []string) ([]App, errs.Error) {
	rst := make([]App, 0)
	end := nextDay(begin)

	sql := `
SELECT DISTINCT
    (pod) as resourceName,
    appid,
    namespace
FROM
    k8s_pod_container_resource_day_p kpcrd
LEFT JOIN k8s_node kn ON kpcrd.node_id = kn.id
WHERE
    kpcrd.created_at between ? and ?
AND kpcrd.k8s_cluster_id = ?
AND appid != '85004'
AND kpcrd.owner = 'ReplicaSet'
AND kn.nodename in ?
`

	dbRst := make([]map[string]interface{}, 0)
	if qErr := s.db.Raw(sql, begin, end, clusterId, nodes).Scan(&dbRst).Error; qErr != nil {
		logger.L().Error("获取历史pod失败", zap.Error(qErr))
		err := errs.NewErrorWithMsg(errs.InternalErr, qErr.Error())
		return rst, err
	}

	for _, record := range dbRst {
		ele := App{
			AppId:     record["appid"].(string),
			Name:      record["resourceName"].(string),
			Namespace: record["namespace"].(string),
		}
		rst = append(rst, ele)
	}

	return rst, nil
}
