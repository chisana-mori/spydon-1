package nodesync

import (
	"context"
	"fmt"
	"time"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"
)

// SyncAllClusters 全量同步所有集群的节点状态
// 遍历所有管理的集群，获取其实时节点列表，并更新到 DB
func (m *Manager) SyncAllClusters(ctx context.Context) {
	// 等待一小段时间让控制器启动和缓存同步
	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second):
	}

	logger.S().Info("开始全量同步所有集群节点状态...")

	m.mu.RLock()
	clusterNames := make([]string, 0, len(m.controllers))
	for name := range m.controllers {
		clusterNames = append(clusterNames, name)
	}
	m.mu.RUnlock()

	for _, name := range clusterNames {
		if err := m.syncCluster(ctx, name); err != nil {
			logger.S().Errorw("同步集群节点失败", "cluster", name, "error", err)
		}
	}

	logger.S().Info("所有集群节点状态全量同步完成")
}

func (m *Manager) syncCluster(ctx context.Context, clusterName string) error {
	logger.S().Infow("正在同步集群节点...", "cluster", clusterName)

	nodes, err := m.ListNodes(clusterName)
	if err != nil {
		return err
	}

	// 获取 Cluster ID
	var cluster models.Cluster
	if err := m.mainDB.Where("clustername = ?", clusterName).First(&cluster).Error; err != nil {
		return fmt.Errorf("查询集群信息失败: %w", err)
	}

	activeNodeNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		activeNodeNames = append(activeNodeNames, node.Name)
		// 使用 nodesync 包中已有的 UpdateDeviceFromNode 函数
		if err := UpdateDeviceFromNode(ctx, m.mainDB, clusterName, &node); err != nil {
			// 单个失败不中断整体
			logger.S().Errorw("同步单节点状态失败", "cluster", clusterName, "node", node.Name, "error", err)
		}
	}

	// 对齐状态：清理数据库中有但实际集群中没有的节点关联信息
	if err := CleanOrphanDevices(ctx, m.mainDB, int(cluster.ID), activeNodeNames); err != nil {
		return fmt.Errorf("清理孤儿节点失败: %w", err)
	}

	logger.S().Infow("集群节点同步完成", "cluster", clusterName, "node_count", len(nodes))
	return nil
}
