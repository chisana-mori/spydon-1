package redis

import (
	"fmt"
	"robusta-web/backend/pkg/logger"
	"sync"

	"go.uber.org/zap"
)

// Factory 是 Redis 客户端工厂，负责管理全局 Redis 客户端实例
// 使用单例模式，确保整个应用共享同一个 Redis 连接池
type Factory struct {
	client   Client
	mu       sync.RWMutex
	isSetup  bool
	redisURL string
	poolSize int
}

var (
	globalFactory *Factory
	factoryOnce   sync.Once
)

// GetFactory 获取全局 Redis 工厂实例（单例）
func GetFactory() *Factory {
	factoryOnce.Do(func() {
		globalFactory = &Factory{}
	})
	return globalFactory
}

// Setup 初始化 Redis 客户端（应在应用启动时调用一次）
// redisURL: Redis 连接 URL，支持单机或集群模式
// poolSize: 连接池大小，0 表示使用默认值
func (f *Factory) Setup(redisURL, password string, poolSize int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isSetup {
		return fmt.Errorf("redis factory already setup")
	}

	handler, err := createClient(redisURL, password, poolSize)
	logger.L().Info("redis地址为", zap.String("redis", redisURL))
	if err != nil {
		return fmt.Errorf("failed to create redis handler: %w", err)
	}

	f.client = handler
	f.redisURL = redisURL
	f.poolSize = poolSize
	f.isSetup = true

	return nil
}

// GetClient 获取 Redis 客户端实例
// 在使用前必须先调用 Setup 初始化
func (f *Factory) GetClient() (Client, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.isSetup {
		return nil, fmt.Errorf("redis factory not setup, call Setup() first")
	}

	return f.client, nil
}

// IsSetup 检查工厂是否已初始化
func (f *Factory) IsSetup() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.isSetup
}

// Close 关闭 Redis 客户端连接（通常在应用关闭时调用）
func (f *Factory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isSetup {
		return nil
	}

	if err := f.client.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}

	f.client = nil
	f.isSetup = false

	return nil
}

// MustGetClient 获取 Redis 客户端，如果未初始化则 panic
// 仅在确保已初始化的场景使用
func (f *Factory) MustGetClient() Client {
	client, err := f.GetClient()
	if err != nil {
		panic(fmt.Sprintf("redis client not available: %v", err))
	}
	return client
}
