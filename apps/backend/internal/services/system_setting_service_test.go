package services

import (
	"testing"

	"robusta-web/backend/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestSystemSettingService_GetSet(t *testing.T) {
	database := setupTestDB()
	service := NewSystemSettingService(database)

	// 1. Test SetSetting
	setting, err := service.SetSetting("test_key", map[string]string{"foo": "bar"}, "test description")
	assert.NoError(t, err)
	assert.NotNil(t, setting)
	assert.Equal(t, "test_key", setting.SettingKey)
	assert.Equal(t, "test description", setting.Description)

	// 2. Test GetSetting
	got, err := service.GetSetting("test_key")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, setting.ID, got.ID)
	assert.JSONEq(t, `{"foo":"bar"}`, string(got.Value))

	// 3. Test Update
	updated, err := service.SetSetting("test_key", map[string]string{"foo": "baz"}, "updated")
	assert.NoError(t, err)
	assert.Equal(t, "updated", updated.Description)

	gotUpdated, err := service.GetSetting("test_key")
	assert.NoError(t, err)
	assert.JSONEq(t, `{"foo":"baz"}`, string(gotUpdated.Value))
}

func TestSystemSettingService_AutoRCAConfig(t *testing.T) {
	database := setupTestDB()
	service := NewSystemSettingService(database)

	// 1. Default config
	config, _, err := service.GetAutoRCAConfig()
	assert.NoError(t, err)
	assert.False(t, config.Enabled) // Default is false
	assert.Equal(t, 10, config.RateLimit)
	assert.ElementsMatch(t, defaultAllowedSeverities(), config.AllowedSeverities)

	// 2. Set config
	newConfig := AutoRCAConfig{
		Enabled:           true,
		RateLimit:         5,
		Period:            30,
		AllowedSeverities: []models.AlertSeverity{models.AlertSeverityCritical, models.AlertSeverityHigh},
	}
	_, err = service.SetSetting(SettingKeyAutoRCA, newConfig, "Auto RCA Config")
	assert.NoError(t, err)

	// 3. Get updated config
	gotConfig, _, err := service.GetAutoRCAConfig()
	assert.NoError(t, err)
	assert.True(t, gotConfig.Enabled)
	assert.Equal(t, 5, gotConfig.RateLimit)
	assert.Equal(t, 30, gotConfig.Period)
	assert.ElementsMatch(t, newConfig.AllowedSeverities, gotConfig.AllowedSeverities)
}
