package nodesync

import (
	"context"
	"errors"
	"strings"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"
	"robusta-web/backend/pkg/logger"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
)

// UpdateDeviceFromNode 根据节点信息更新 device 表
// 通过 nodename 匹配 ci_code 进行关联
// 注意：仅更新 cluster 名称和 k8s_status，不更新 cluster_id 和 role（由其他系统管理）
func UpdateDeviceFromNode(ctx context.Context, navyDB *db.NavyDatabase, clusterName string, node *corev1.Node) error {
	nodeName := node.Name
	_, k8sStatus := ExtractNodeInfo(node)

	// 通过 ci_code 查找设备
	var device navy.Device
	// 1. 尝试精确匹配 ci_code (此时 nodeName 已转为大写)
	if err := navyDB.WithContext(ctx).Where("ci_code = ?", nodeName).First(&device).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 2. 尝试通过 IP 匹配 (InternalIP)
		foundByIP := false
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				// 注意：IP 匹配可能存在风险，但在 ci_code 匹配失败时作为 fallback 是合理的
				// 假设 IP 是唯一的
				if err := navyDB.WithContext(ctx).Where("ip = ?", addr.Address).First(&device).Error; err == nil {
					foundByIP = true
					logger.S().Infow("通过 IP 匹配到设备", "cluster", clusterName, "node", nodeName, "ip", addr.Address, "device", device.CICode)
					break
				}
			}
		}

		if !foundByIP {
			// 设备不存在，跳过（不新建，不算错误）
			logger.S().Debugw("未找到匹配设备，跳过",
				"cluster", clusterName,
				"nodename", nodeName)
			return nil
		}
	}

	// 比较后仅在有变化时更新（不更新 cluster_id 和 role）
	updates := map[string]interface{}{}
	if device.Cluster != clusterName {
		updates["cluster"] = clusterName
	}
	if device.K8sStatus != k8sStatus {
		updates["k8s_status"] = k8sStatus
	}

	if len(updates) == 0 {
		// 无变化，跳过更新
		logger.S().Debugw("设备集群信息未变化，跳过更新",
			"device_id", device.ID,
			"ci_code", nodeName,
			"cluster", clusterName,
			"k8s_status", k8sStatus)
		return nil
	}

	if err := navyDB.WithContext(ctx).Model(&navy.Device{}).
		Where("id = ?", device.ID).
		Updates(updates).Error; err != nil {
		return err
	}

	logger.S().Infow("设备集群信息已更新",
		"device_id", device.ID,
		"ci_code", nodeName,
		"cluster", clusterName,
		"k8s_status", k8sStatus)

	return nil
}

// ClearDeviceClusterInfo 清除设备的集群关联信息
// 当节点从集群中删除时调用
func ClearDeviceClusterInfo(ctx context.Context, navyDB *db.NavyDatabase, ciCode string) error {
	result := navyDB.WithContext(ctx).Model(&navy.Device{}).
		Where("ci_code = ?", ciCode).
		Updates(map[string]interface{}{
			"cluster":    "",
			"cluster_id": 0,
			"k8s_status": "",
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.S().Infow("设备集群关联已清除", "ci_code", ciCode)
	}

	return nil
}

// CleanOrphanDevices 清理孤儿设备
// 对于指定集群，清除那些在集群中找不到对应节点的设备的集群关联信息
func CleanOrphanDevices(ctx context.Context, navyDB *db.NavyDatabase, clusterID int, activeNodeNames []string) error {
	if len(activeNodeNames) == 0 {
		// 没有活跃节点，清除该集群所有设备的关联
		return navyDB.WithContext(ctx).Model(&navy.Device{}).
			Where("cluster_id = ?", clusterID).
			Updates(map[string]interface{}{
				"cluster":    "",
				"cluster_id": 0,
				"k8s_status": "",
			}).Error
	}

	// 清除不在活跃节点列表中的设备
	return navyDB.WithContext(ctx).Model(&navy.Device{}).
		Where("cluster_id = ? AND ci_code NOT IN ?", clusterID, activeNodeNames).
		Updates(map[string]interface{}{
			"cluster":    "",
			"cluster_id": 0,
			"k8s_status": "",
		}).Error
}

// ExtractNodeInfo 从 Node 对象提取同步所需信息
func ExtractNodeInfo(node *corev1.Node) (role, k8sStatus string) {
	// 提取角色：从 node-role.kubernetes.io/* 标签
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			roleName := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if roleName != "" {
				if role != "" {
					role += ","
				}
				role += roleName
			}
		}
	}

	// 提取状态
	k8sStatus = getNodeStatus(node)

	return role, k8sStatus
}

// getNodeStatus 获取节点状态
// 优先级: Unschedulable > Ready > NotReady
func getNodeStatus(node *corev1.Node) string {
	// 优先判断是否被标记为不可调度
	if node.Spec.Unschedulable {
		return "Unschedulable"
	}

	// 检查 Ready condition
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}

	return "Unknown"
}
