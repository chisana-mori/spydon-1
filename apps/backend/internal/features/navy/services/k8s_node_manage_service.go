package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/pkg/nodesync"

	corev1 "k8s.io/api/core/v1"
)

// K8sNodeManageService K8s 节点标签/污点管理服务
// 提供实时的节点信息获取和修改功能，与数据库查询分离
type K8sNodeManageService struct {
	mainDB          *db.Database
	nodesyncManager *nodesync.Manager
}

// NewK8sNodeManageService 创建 K8s 节点管理服务
func NewK8sNodeManageService(mainDB *db.Database, nodesyncManager *nodesync.Manager) *K8sNodeManageService {
	return &K8sNodeManageService{
		mainDB:          mainDB,
		nodesyncManager: nodesyncManager,
	}
}

// NodeLabelInfo 节点标签信息
type NodeLabelInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NodeTaintInfo 节点污点信息
type NodeTaintInfo struct {
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	Effect    string     `json:"effect"` // NoSchedule, PreferNoSchedule, NoExecute
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// NodeLabelTaintResponse 节点标签和污点响应
type NodeLabelTaintResponse struct {
	NodeName    string          `json:"nodeName"`
	ClusterName string          `json:"clusterName"`
	ClusterID   uint            `json:"clusterId"`
	Labels      []NodeLabelInfo `json:"labels"`
	Taints      []NodeTaintInfo `json:"taints"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Conditions  []string        `json:"conditions"` // 节点状态条件
}

// getClusterIDByName 通过集群名称获取集群 ID
func (s *K8sNodeManageService) getClusterIDByName(ctx context.Context, clusterName string) (uint, error) {
	var cluster models.Cluster
	if err := s.mainDB.WithContext(ctx).Where("clustername = ?", clusterName).First(&cluster).Error; err != nil {
		return 0, fmt.Errorf("集群 %s 不存在: %w", clusterName, err)
	}
	return cluster.ID, nil
}

// GetNodeLabelsAndTaints 获取节点的实时标签和污点 (通过 clusterName 和 ciCode)
func (s *K8sNodeManageService) GetNodeLabelsAndTaints(ctx context.Context, clusterName, ciCode string) (*NodeLabelTaintResponse, error) {
	// ciCode 可能是 nodeName 或 IP，尝试用 nodeName 查找
	node, err := s.nodesyncManager.GetNode(clusterName, ciCode)
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 如果用 ciCode 找不到，可能 ciCode 是 IP，需要遍历查找
	if node == nil {
		nodes, listErr := s.nodesyncManager.ListNodes(clusterName)
		if listErr != nil {
			return nil, fmt.Errorf("列出节点失败: %w", listErr)
		}
		for _, n := range nodes {
			// 检查 nodeName 或 hostIP 是否匹配
			if strings.EqualFold(n.Name, ciCode) {
				node = &n
				break
			}
			// 检查所有 InternalIP
			for _, addr := range n.Status.Addresses {
				if addr.Type == corev1.NodeInternalIP && strings.EqualFold(addr.Address, ciCode) {
					node = &n
					break
				}
			}
			if node != nil {
				break
			}
		}
	}

	if node == nil {
		return nil, fmt.Errorf("节点 %s 不存在于集群 %s", ciCode, clusterName)
	}

	// 查询 clusterID 用于响应
	clusterID, err := s.getClusterIDByName(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	return s.nodeToResponse(node, clusterName, clusterID), nil
}

// nodeToResponse 将 K8s Node 转换为响应结构
func (s *K8sNodeManageService) nodeToResponse(node *corev1.Node, clusterName string, clusterID uint) *NodeLabelTaintResponse {
	// 转换标签
	labels := make([]NodeLabelInfo, 0, len(node.Labels))
	for k, v := range node.Labels {
		labels = append(labels, NodeLabelInfo{Key: k, Value: v})
	}

	// 转换污点
	taints := make([]NodeTaintInfo, 0, len(node.Spec.Taints))
	for _, t := range node.Spec.Taints {
		var timeAdded *time.Time
		if t.TimeAdded != nil {
			ts := t.TimeAdded.Time
			timeAdded = &ts
		}
		taints = append(taints, NodeTaintInfo{
			Key:       t.Key,
			Value:     t.Value,
			Effect:    string(t.Effect),
			TimeAdded: timeAdded,
		})
	}

	// 获取节点状态条件
	conditions := make([]string, 0)
	for _, c := range node.Status.Conditions {
		if c.Status == corev1.ConditionTrue {
			conditions = append(conditions, string(c.Type))
		}
	}

	return &NodeLabelTaintResponse{
		NodeName:    node.Name,
		ClusterName: clusterName,
		ClusterID:   clusterID,
		Labels:      labels,
		Taints:      taints,
		UpdatedAt:   time.Now(),
		Conditions:  conditions,
	}
}

// ListClusterNodes 列出集群的所有节点及其标签/污点 (通过 clusterName)
func (s *K8sNodeManageService) ListClusterNodes(ctx context.Context, clusterName string) ([]NodeLabelTaintResponse, error) {
	clusterID, err := s.getClusterIDByName(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	nodes, err := s.nodesyncManager.ListNodes(clusterName)
	if err != nil {
		return nil, err
	}

	responses := make([]NodeLabelTaintResponse, len(nodes))
	for i, node := range nodes {
		responses[i] = *s.nodeToResponse(&node, clusterName, clusterID)
	}

	return responses, nil
}
