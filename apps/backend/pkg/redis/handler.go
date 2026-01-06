package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client abstracts Redis capabilities for the drain module, facilitating unit testing
type Client interface {
	Set(key string, value interface{})
	SetWithExpireTime(key string, value string, expiry time.Duration)
	AcquireLock(key string, value string, expiry time.Duration) (bool, error)
	Get(key string) string
	Delete(key string)
	Pub(channel string, message string) error
	Subscribe(channel string) *goredis.PubSub
	ScanKeys(pattern string) ([]string, error)
	SAdd(key string, members ...string) error
	SMembers(key string) ([]string, error)
	Close() error
}

// Handler is the Redis client implementation
type Handler struct {
	client *goredis.Client
	ctx    context.Context
}

// NewHandler creates a new Redis handler
func NewHandler(redisURL string, poolSize int) (*Handler, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	if poolSize > 0 {
		opts.PoolSize = poolSize
	}

	client := goredis.NewClient(opts)
	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Handler{
		client: client,
		ctx:    ctx,
	}, nil
}

// Set stores a key-value pair without expiration
func (h *Handler) Set(key string, value interface{}) {
	h.client.Set(h.ctx, key, value, 0)
}

// SetWithExpireTime stores a key-value pair with expiration
func (h *Handler) SetWithExpireTime(key string, value string, expiry time.Duration) {
	h.client.Set(h.ctx, key, value, expiry)
}

// AcquireLock attempts to acquire a distributed lock
func (h *Handler) AcquireLock(key string, value string, expiry time.Duration) (bool, error) {
	return h.client.SetNX(h.ctx, key, value, expiry).Result()
}

// Get retrieves a value by key
func (h *Handler) Get(key string) string {
	val, err := h.client.Get(h.ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return ""
	}
	if err != nil {
		return ""
	}
	return val
}

// Delete removes a key
func (h *Handler) Delete(key string) {
	h.client.Del(h.ctx, key)
}

// Pub publishes a message to a channel
func (h *Handler) Pub(channel string, message string) error {
	return h.client.Publish(h.ctx, channel, message).Err()
}

// Subscribe subscribes to a channel
func (h *Handler) Subscribe(channel string) *goredis.PubSub {
	return h.client.Subscribe(h.ctx, channel)
}

// ScanKeys scans keys matching a pattern
func (h *Handler) ScanKeys(pattern string) ([]string, error) {
	var allKeys []string
	var cursor uint64

	for {
		keys, nextCursor, err := h.client.Scan(h.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}

// SAdd adds members to a set
func (h *Handler) SAdd(key string, members ...string) error {
	args := make([]interface{}, len(members))
	for i, m := range members {
		args[i] = m
	}
	return h.client.SAdd(h.ctx, key, args...).Err()
}

// SMembers returns all members of a set
func (h *Handler) SMembers(key string) ([]string, error) {
	return h.client.SMembers(h.ctx, key).Result()
}

// Close closes the Redis connection
func (h *Handler) Close() error {
	return h.client.Close()
}

// Ensure Handler implements Client interface
var _ Client = (*Handler)(nil)
