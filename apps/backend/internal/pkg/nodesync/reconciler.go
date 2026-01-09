package nodesync

import (
	"context"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeReconciler Node 资源的 reconciler
type NodeReconciler struct {
	clusterName string
	database    *db.Database
	client      client.Client
}

// Reconcile 处理 Node 事件
func (r *NodeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	nodeName := req.Name

	// 获取 Node 对象
	var node corev1.Node
	if err := r.client.Get(ctx, req.NamespacedName, &node); err != nil {
		if errors.IsNotFound(err) {
			// 节点被删除，清除设备的集群关联
			logger.S().Infow("节点已删除，清除设备集群关联",
				"cluster", r.clusterName,
				"node", nodeName)
			if clearErr := ClearDeviceClusterInfo(ctx, r.database, nodeName); clearErr != nil {
				logger.S().Errorw("清除设备集群关联失败",
					"cluster", r.clusterName,
					"node", nodeName,
					"error", clearErr)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 同步节点信息到 device 表
	if err := UpdateDeviceFromNode(ctx, r.database, r.clusterName, &node); err != nil {
		logger.S().Warnw("同步节点到设备表失败",
			"cluster", r.clusterName,
			"node", nodeName,
			"error", err)
		// 不返回错误，避免无限重试
		return ctrl.Result{}, nil
	}

	logger.S().Debugw("节点同步成功",
		"cluster", r.clusterName,
		"node", nodeName)

	return ctrl.Result{}, nil
}
