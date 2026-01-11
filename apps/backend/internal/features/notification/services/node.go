package services

import (
	"context"
	"net"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"strings"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

// NodeService 节点服务
type NodeService struct {
	nodeManager *nodesync.Manager
	db          *db.Database
}

// NewNodeService 创建节点服务
func NewNodeService(nodeManager *nodesync.Manager, database *db.Database) *NodeService {
	return &NodeService{
		nodeManager: nodeManager,
		db:          database,
	}
}

var (
	// nodeService 单例实例，用于向后兼容的函数调用
	nodeService *NodeService
)

// InitNodeService 初始化节点服务单例
func InitNodeService(nodeManager *nodesync.Manager, database *db.Database) {
	nodeService = NewNodeService(nodeManager, database)
}

// isIPv4Address 检查给定的地址是否为IPv4地址
func isIPv4Address(ipAddress string) bool {
	ip := net.ParseIP(ipAddress)
	return ip.To4() != nil
}

func getIPv4Address(node v1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" && isIPv4Address(address.Address) {
			return address.Address
		}
	}
	return ""
}

// NodeFeatures 获取节点特征信息（包级函数，向后兼容）
func NodeFeatures(nodes []string, cluster string) []Node {
	if nodeService == nil {
		logger.L().Warn("NodeService 未初始化，返回空结果")
		return make([]Node, 0)
	}
	return nodeService.NodeFeatures(nodes, cluster)
}

// NodeFeatures 获取节点特征信息
func (s *NodeService) NodeFeatures(nodes []string, cluster string) []Node {
	nodesRst := make([]Node, 0)

	// 获取特殊标签管理信息
	specificLabels, err := s.getLabelManagementList(context.Background())
	if err != nil {
		logger.L().Error("获取特殊标签管理信息失败", zap.Error(err))
		return nodesRst
	}

	// 并行获取节点信息
	resultChan := make(chan Node, len(nodes))
	for _, node := range nodes {
		go func(n string) {
			nodeInfo, err := s.getNodeInfo(cluster, n, specificLabels)
			if err != nil {
				logger.L().Error("获取节点信息失败", zap.String("node", n), zap.Error(err))
				resultChan <- Node{Name: n, Function: ""}
				return
			}
			resultChan <- nodeInfo
		}(node)
	}

	for i := 0; i < len(nodes); i++ {
		nodeInfo := <-resultChan
		nodesRst = append(nodesRst, nodeInfo)
	}

	close(resultChan)
	return nodesRst
}

// getLabelManagementList 获取标签管理列表
func (s *NodeService) getLabelManagementList(ctx context.Context) ([]navy.LabelManagement, error) {
	var labels []navy.LabelManagement
	if err := s.db.WithContext(ctx).Preload("LabelValues").Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

// getNodeInfo 获取单个节点信息
func (s *NodeService) getNodeInfo(cluster string, nodeName string, specificLabels []navy.LabelManagement) (Node, error) {
	node, err := s.nodeManager.GetNode(cluster, nodeName)
	if err != nil {
		return Node{Name: nodeName, Function: ""}, err
	}
	if node == nil {
		return Node{Name: nodeName, Function: ""}, nil
	}

	nodeLabels := node.ObjectMeta.Labels
	return Node{
		Name:     nodeName,
		Function: matchFuncByLabel(nodeLabels, specificLabels),
		IP:       getIPv4Address(*node),
	}, nil
}

// matchFuncByLabel 根据标签匹配功能描述
func matchFuncByLabel(nodeLabels map[string]string, specificLabels []navy.LabelManagement) string {
	var function strings.Builder

	for _, label := range specificLabels {
		if val, ok := nodeLabels[label.Key]; ok {
			if label.Key == "Accelerator" {
				function.WriteString(label.Name)
				function.WriteString("-")
				function.WriteString(val)
				function.WriteString(" ")
			} else {
				for _, value := range label.LabelValues {
					if value.Value == val {
						function.WriteString(label.Name)
						function.WriteString(" ")
						break // 假设每个键只有一个匹配值
					}
				}
			}
		}
	}

	funcStr := function.String()
	if len(funcStr) == 0 {
		return ""
	}

	// 去除末尾空格
	trimmed := strings.TrimRight(funcStr, " ")
	return "(" + trimmed + ")"
}
