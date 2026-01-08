package redis_test

import (
	"fmt"

	"robusta-web/backend/pkg/redis"
)

// 示例 1: 应用启动时初始化
func ExampleFactory_Setup() {
	factory := redis.GetFactory()

	// 初始化 Redis 连接
	err := factory.Setup("redis://localhost:6379", 10)
	if err != nil {
		fmt.Printf("Setup failed: %v\n", err)
		return
	}
	defer factory.Close()

	fmt.Println("Redis factory initialized successfully")
	// Output: Redis factory initialized successfully
}

// 示例 2: 获取客户端并使用（安全方式）
func ExampleFactory_GetClient() {
	factory := redis.GetFactory()

	// 获取客户端
	client, err := factory.GetClient()
	if err != nil {
		fmt.Printf("Failed to get client: %v\n", err)
		return
	}

	// 使用客户端
	client.Set("example:key", "example value")
	value := client.Get("example:key")
	fmt.Printf("Value: %s\n", value)
}

// 示例 3: 检查工厂是否已初始化
func ExampleFactory_IsSetup() {
	factory := redis.GetFactory()

	if factory.IsSetup() {
		fmt.Println("Factory is ready")
	} else {
		fmt.Println("Factory not initialized, call Setup() first")
	}
	// Output: Factory not initialized, call Setup() first
}

// 示例 4: 在服务中使用（推荐模式）
type UserService struct {
	redis redis.Client
}

func NewUserService() (*UserService, error) {
	client, err := redis.GetFactory().GetClient()
	if err != nil {
		return nil, fmt.Errorf("redis not available: %w", err)
	}

	return &UserService{
		redis: client,
	}, nil
}

func (s *UserService) CacheUserData(userID string, data string) {
	key := fmt.Sprintf("user:%s", userID)
	s.redis.Set(key, data)
}

func (s *UserService) GetCachedUserData(userID string) string {
	key := fmt.Sprintf("user:%s", userID)
	return s.redis.Get(key)
}

// 示例 5: 快速使用（确保已初始化）
func ExampleFactory_MustGetClient() {
	// 注意：仅在确保 Redis 已初始化时使用
	// 否则会 panic

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic caught: Redis not initialized")
		}
	}()

	client := redis.GetFactory().MustGetClient()
	client.Set("quick:key", "quick value")

	// Output: Panic caught: Redis not initialized
}
