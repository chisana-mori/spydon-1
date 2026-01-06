package services

import (
	"testing"
	"time"

	"robusta-web/backend/internal/features/systemsetting/services"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// TestTokenBucketRateLimit 测试令牌桶速率限制行为
func TestTokenBucketRateLimit(t *testing.T) {
	// 配置：每10秒允许2个请求
	rateLimit := 2
	period := 10
	ratePerSecond := float64(rateLimit) / float64(period)
	burst := rateLimit

	limiter := rate.NewLimiter(rate.Limit(ratePerSecond), burst)

	// 1. 前2个请求应该立即通过（突发容量）
	assert.True(t, limiter.Allow(), "第1个请求应该通过")
	assert.True(t, limiter.Allow(), "第2个请求应该通过")

	// 2. 第3个请求应该被拒绝（超过突发容量）
	assert.False(t, limiter.Allow(), "第3个请求应该被拒绝")

	// 3. 等待一段时间后，应该有新的令牌
	// 每5秒恢复1个令牌
	time.Sleep(6 * time.Second)
	assert.True(t, limiter.Allow(), "等待后应该有新令牌")
}

// TestRateLimiterUpdate 测试动态更新速率限制
func TestRateLimiterUpdate(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(6*time.Second), 2)

	// 初始配置：10次/60秒
	config1 := &services.AutoRCAConfig{
		RateLimit: 10,
		Period:    60,
	}

	ratePerSecond1 := float64(config1.RateLimit) / float64(config1.Period)
	limiter.SetLimit(rate.Limit(ratePerSecond1))
	limiter.SetBurst(config1.RateLimit)

	assert.Equal(t, rate.Limit(ratePerSecond1), limiter.Limit())
	assert.Equal(t, config1.RateLimit, limiter.Burst())

	// 更新配置：5次/30秒
	config2 := &services.AutoRCAConfig{
		RateLimit: 5,
		Period:    30,
	}

	ratePerSecond2 := float64(config2.RateLimit) / float64(config2.Period)
	limiter.SetLimit(rate.Limit(ratePerSecond2))
	limiter.SetBurst(config2.RateLimit)

	assert.Equal(t, rate.Limit(ratePerSecond2), limiter.Limit())
	assert.Equal(t, config2.RateLimit, limiter.Burst())
}

// BenchmarkRateLimit 对比数据库查询和令牌桶的性能
func BenchmarkRateLimit(b *testing.B) {
	b.Run("TokenBucket", func(b *testing.B) {
		limiter := rate.NewLimiter(rate.Every(6*time.Second), 10)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = limiter.Allow()
		}
	})

	// 注意：数据库查询版本的基准测试需要真实的数据库连接
	// 这里仅作为对比参考，实际性能差异会非常明显
	// 令牌桶：O(1) 纳秒级
	// 数据库查询：O(n) 毫秒级
}
