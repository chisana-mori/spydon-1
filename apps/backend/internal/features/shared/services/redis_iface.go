package services

import (
	"time"

	"robusta-web/backend/pkg/redis"

	goredis "github.com/redis/go-redis/v9"
)

// RedisClient abstracts Redis capabilities for the drain module
type RedisClient interface {
	Set(key string, value interface{})
	SetWithExpireTime(key string, value string, expiry time.Duration)
	AcquireLock(key string, value string, expiry time.Duration) (bool, error)
	Get(key string) string
	Delete(key string)
	Pub(channel string, message string) error
	Subscribe(channel string) *goredis.PubSub
	ScanKeys(pattern string) ([]string, error)
	SAdd(key string, members ...string) error
	SRem(key string, members ...string) error
	SMembers(key string) ([]string, error)
}

// Ensure redis.Handler satisfies RedisClient
var _ RedisClient = (*redis.Handler)(nil)
