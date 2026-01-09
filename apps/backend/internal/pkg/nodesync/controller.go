package nodesync

import (
	"context"
	"fmt"
	"strings"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// Controller 单集群节点同步控制器
type Controller struct {
	clusterName string
	database    *db.Database
	mgr         manager.Manager
	cancel      context.CancelFunc
}

// NewController 创建集群控制器
func NewController(cluster *models.Cluster, database *db.Database) (*Controller, error) {
	// 从 KubeConfig 创建 REST 配置
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config))
	if err != nil {
		return nil, fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}

	// 创建 controller-runtime manager
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		// 只 watch Node 资源，减少内存占用
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{},
		},
		// 禁用 metrics 和 health probe
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("创建 manager 失败: %w", err)
	}

	return &Controller{
		clusterName: cluster.Name,
		database:    database,
		mgr:         mgr,
	}, nil
}

// Start 启动控制器
func (c *Controller) Start(ctx context.Context) error {
	// 注册 Node reconciler
	reconciler := &NodeReconciler{
		clusterName: c.clusterName,
		database:    c.database,
		client:      c.mgr.GetClient(),
	}

	if err := ctrl.NewControllerManagedBy(c.mgr).
		Named(fmt.Sprintf("node-%s", strings.ToLower(c.clusterName))).
		For(&corev1.Node{}).
		Complete(reconciler); err != nil {
		return fmt.Errorf("注册 reconciler 失败: %w", err)
	}

	logger.S().Infow("集群控制器开始运行", "cluster", c.clusterName)

	// 启动 manager（阻塞直到 ctx 取消）
	if err := c.mgr.Start(ctx); err != nil {
		return fmt.Errorf("manager 运行失败: %w", err)
	}

	return nil
}

// Stop 停止控制器
func (c *Controller) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// GetClient 获取 K8s client（用于测试）
func (c *Controller) GetClient() client.Client {
	return c.mgr.GetClient()
}

// GetNode 从缓存读取指定节点
func (c *Controller) GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	var node corev1.Node
	if err := c.mgr.GetClient().Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}
