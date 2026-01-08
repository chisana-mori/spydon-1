package redis_test

import (
	"testing"

	"robusta-web/backend/pkg/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_SingletonPattern(t *testing.T) {
	factory1 := redis.GetFactory()
	factory2 := redis.GetFactory()

	// 验证单例模式：两次调用应该返回同一个实例
	assert.Same(t, factory1, factory2, "GetFactory() should return the same instance")
}

func TestFactory_SetupAndGetClient(t *testing.T) {
	// 注意: 这个测试需要真实的 Redis 实例，或者使用 mock
	// 这里假设本地有 Redis 运行在 localhost:6379
	t.Skip("Skipping integration test, requires Redis instance")

	factory := redis.GetFactory()
	defer factory.Close()

	// Setup 应该成功
	err := factory.Setup("redis://localhost:6379", 5)
	require.NoError(t, err)

	// 验证已初始化
	assert.True(t, factory.IsSetup())

	// 获取客户端应该成功
	client, err := factory.GetClient()
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestFactory_GetClient_BeforeSetup(t *testing.T) {
	// 创建一个新的 factory 用于测试（绕过单例）
	factory := &redis.Factory{}

	// 在 Setup 之前调用 GetClient 应该返回错误
	client, err := factory.GetClient()
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "not setup")
}

func TestFactory_IsSetup(t *testing.T) {
	factory := &redis.Factory{}

	// 初始状态应该是未初始化
	assert.False(t, factory.IsSetup())
}

func TestFactory_Setup_AlreadySetup(t *testing.T) {
	t.Skip("Skipping integration test, requires Redis instance")

	factory := &redis.Factory{}
	defer factory.Close()

	// 第一次 Setup
	err := factory.Setup("redis://localhost:6379", 5)
	require.NoError(t, err)

	// 第二次 Setup 应该失败
	err = factory.Setup("redis://localhost:6379", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already setup")
}

func TestFactory_MustGetClient_Panic(t *testing.T) {
	factory := &redis.Factory{}

	// MustGetClient 在未初始化时应该 panic
	assert.Panics(t, func() {
		factory.MustGetClient()
	})
}

func TestFactory_Close_NotSetup(t *testing.T) {
	factory := &redis.Factory{}

	// 关闭未初始化的工厂不应该报错
	err := factory.Close()
	assert.NoError(t, err)
}

func TestFactory_Close_AfterSetup(t *testing.T) {
	t.Skip("Skipping integration test, requires Redis instance")

	factory := &redis.Factory{}

	err := factory.Setup("redis://localhost:6379", 5)
	require.NoError(t, err)

	// 关闭应该成功
	err = factory.Close()
	assert.NoError(t, err)

	// 关闭后应该是未初始化状态
	assert.False(t, factory.IsSetup())

	// 关闭后 GetClient 应该返回错误
	_, err = factory.GetClient()
	assert.Error(t, err)
}
