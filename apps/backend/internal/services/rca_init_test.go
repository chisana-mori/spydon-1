package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// TestRCAService_InitializationWithConfig 测试RCA服务初始化时从配置读取速率限制
func TestRCAService_InitializationWithConfig(t *testing.T) {
	database := setupTestDB()
	settingService := NewSystemSettingService(database)

	// 1. 设置自定义配置
	customConfig := AutoRCAConfig{
		Enabled:   true,
		RateLimit: 20,
		Period:    120, // 20次/120秒 = 0.1667次/秒
	}
	_, err := settingService.SetSetting(SettingKeyAutoRCA, customConfig, "Custom Rate Limit")
	assert.NoError(t, err)

	// 2. 创建RCA服务（应该从数据库读取配置）
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)

	// 3. 验证令牌桶配置
	expectedRate := float64(customConfig.RateLimit) / float64(customConfig.Period)
	expectedBurst := customConfig.RateLimit

	rcaService.limiterMux.RLock()
	actualRate := rcaService.rateLimiter.Limit()
	actualBurst := rcaService.rateLimiter.Burst()
	rcaService.limiterMux.RUnlock()

	assert.Equal(t, rate.Limit(expectedRate), actualRate, "速率应该从配置中读取")
	assert.Equal(t, expectedBurst, actualBurst, "突发容量应该从配置中读取")
}

// TestRCAService_InitializationWithoutConfig 测试没有配置时使用默认值
func TestRCAService_InitializationWithoutConfig(t *testing.T) {
	database := setupTestDB()
	settingService := NewSystemSettingService(database)

	// 不设置任何配置，直接创建服务
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)

	// 验证使用默认配置：10次/60秒
	expectedRate := float64(10) / float64(60)
	expectedBurst := 10

	rcaService.limiterMux.RLock()
	actualRate := rcaService.rateLimiter.Limit()
	actualBurst := rcaService.rateLimiter.Burst()
	rcaService.limiterMux.RUnlock()

	assert.Equal(t, rate.Limit(expectedRate), actualRate, "应该使用默认速率")
	assert.Equal(t, expectedBurst, actualBurst, "应该使用默认突发容量")
}

// TestRCAService_DynamicConfigUpdate 测试运行时动态更新配置
func TestRCAService_DynamicConfigUpdate(t *testing.T) {
	database := setupTestDB()
	settingService := NewSystemSettingService(database)

	// 1. 初始配置
	initialConfig := AutoRCAConfig{
		Enabled:   true,
		RateLimit: 10,
		Period:    60,
	}
	_, err := settingService.SetSetting(SettingKeyAutoRCA, initialConfig, "Initial")
	assert.NoError(t, err)
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)

	// 2. 验证初始配置
	rcaService.limiterMux.RLock()
	initialRate := rcaService.rateLimiter.Limit()
	initialBurst := rcaService.rateLimiter.Burst()
	rcaService.limiterMux.RUnlock()

	assert.Equal(t, rate.Limit(float64(10)/float64(60)), initialRate)
	assert.Equal(t, 10, initialBurst)

	// 3. 更新配置
	newConfig := AutoRCAConfig{
		Enabled:   true,
		RateLimit: 30,
		Period:    60,
	}
	_, err = settingService.SetSetting(SettingKeyAutoRCA, newConfig, "Updated")
	assert.NoError(t, err)

	// 4. 触发配置更新（通过 checkRateLimit）
	allowed, err := rcaService.checkRateLimit(&newConfig)
	assert.NoError(t, err)
	assert.True(t, allowed) // 第一次应该允许

	// 5. 验证配置已更新
	rcaService.limiterMux.RLock()
	updatedRate := rcaService.rateLimiter.Limit()
	updatedBurst := rcaService.rateLimiter.Burst()
	rcaService.limiterMux.RUnlock()

	assert.Equal(t, rate.Limit(float64(30)/float64(60)), updatedRate)
	assert.Equal(t, 30, updatedBurst)
}
