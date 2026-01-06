package services

import (
	"fmt"
	"testing"
	"time"

	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/features/systemsetting/services"
	"robusta-web/backend/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestRCAService_TriggerAndRateLimit(t *testing.T) {
	database := sharedservices.SetupTestDB()

	// Create required cluster
	cluster := &models.Cluster{Name: "cluster-1"}
	assert.NoError(t, database.Create(cluster).Error)
	settingService := services.NewSystemSettingService(database)
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)
	rcaService.SetAnalysisExecutor(func(*models.RCARun, *models.Alert) {})

	// 1. Enable Auto-RCA with strict rate limit (2 per 10 seconds)
	config := services.AutoRCAConfig{
		Enabled:   true,
		RateLimit: 2,
		Period:    10,
	}
	_, err := settingService.SetSetting(services.SettingKeyAutoRCA, config, "Strict Limit")
	assert.NoError(t, err)

	// Helper to create alert
	createAlert := func(i int) *models.Alert {
		alert := &models.Alert{
			Fingerprint: fmt.Sprintf("fp-%d", i),
			ClusterName: "cluster-1",
			Title:       fmt.Sprintf("Alert %d", i),
			Status:      "firing",
			Severity:    string(models.AlertSeverityHigh),
		}
		assert.NoError(t, database.Create(alert).Error)
		return alert
	}

	// 2. Trigger 3 alerts
	// Alert 1 -> Running
	run1, err := rcaService.TriggerRCAAnalysis(createAlert(1))
	assert.NoError(t, err)
	assert.Equal(t, string(models.RCAStatusPending), run1.Status) // Initial status is pending, then async becomes running

	// Alert 2 -> Running
	run2, err := rcaService.TriggerRCAAnalysis(createAlert(2))
	assert.NoError(t, err)
	assert.Equal(t, string(models.RCAStatusPending), run2.Status)

	// Alert 3 -> Queued (Rate limit exceeded)
	run3, err := rcaService.TriggerRCAAnalysis(createAlert(3))
	assert.NoError(t, err)
	assert.Equal(t, string(models.RCAStatusQueued), run3.Status)

	// 3. Verify DB state
	var runs []models.RCARun
	database.Order("created_at asc").Find(&runs)
	assert.Len(t, runs, 3)
	// Note: The async execution might change status of run1/run2 to Running quickly
	// But run3 must be Queued
	var queuedRun models.RCARun
	err = database.Where("id = ?", run3.ID).First(&queuedRun).Error
	assert.NoError(t, err)
	assert.Equal(t, string(models.RCAStatusQueued), queuedRun.Status)
}

func TestRCAService_QueueProcessing(t *testing.T) {
	database := sharedservices.SetupTestDB()

	// Create required cluster
	cluster := &models.Cluster{Name: "cluster-1"}
	assert.NoError(t, database.Create(cluster).Error)
	settingService := services.NewSystemSettingService(database)
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)
	rcaService.SetAnalysisExecutor(func(*models.RCARun, *models.Alert) {})

	// 1. Enable Auto-RCA
	config := services.AutoRCAConfig{Enabled: true, RateLimit: 10, Period: 60}
	_, err := settingService.SetSetting(services.SettingKeyAutoRCA, config, "")
	assert.NoError(t, err)

	// 2. Manually insert a Queued RCA
	alert := &models.Alert{
		Fingerprint: "fp-queued",
		ClusterName: "cluster-1",
		Title:       "Queued Alert",
		Severity:    string(models.AlertSeverityHigh),
	}
	database.Create(alert)

	queuedRun := &models.RCARun{
		AlertID: alert.ID,
		Status:  string(models.RCAStatusQueued),
	}
	database.Create(queuedRun)

	// 3. Trigger queue processing manually
	// Since processQueue runs in a loop with ticker, we can't easily control it.
	// But we can call processNextInQueue directly since it's exported (or we make it exported for test, or use the channel)
	// Wait, processNextInQueue is unexported.
	// We can send a signal to queueChan.

	// However, processQueue is running in a goroutine started by NewRCAService.
	// We can try to trigger it via the channel.
	rcaService.queueChan <- struct{}{}

	// 4. Wait for processing
	assert.Eventually(t, func() bool {
		var run models.RCARun
		database.Where("id = ?", queuedRun.ID).First(&run)
		// It should transition to Pending then Running/Completed
		return run.Status != string(models.RCAStatusQueued)
	}, 2*time.Second, 100*time.Millisecond)
}

func TestRCAService_Streaming(t *testing.T) {
	database := sharedservices.SetupTestDB()
	settingService := services.NewSystemSettingService(database)
	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)
	rcaService.SetAnalysisExecutor(func(*models.RCARun, *models.Alert) {})

	alertID := "1"

	// 1. Subscribe
	ch, unsubscribe := rcaService.Subscribe(alertID)
	defer unsubscribe()

	// 2. Broadcast
	go func() {
		rcaService.Broadcast(alertID, "message 1")
		rcaService.Broadcast(alertID, "message 2")
	}()

	// 3. Receive
	select {
	case msg := <-ch:
		assert.Equal(t, "message 1", msg)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message 1")
	}

	select {
	case msg := <-ch:
		assert.Equal(t, "message 2", msg)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message 2")
	}
}

func TestRCAService_SeverityFilter(t *testing.T) {
	database := sharedservices.SetupTestDB()

	// Create required cluster
	cluster := &models.Cluster{Name: "cluster-1"}
	assert.NoError(t, database.Create(cluster).Error)
	settingService := services.NewSystemSettingService(database)
	config := services.AutoRCAConfig{
		Enabled:           true,
		RateLimit:         10,
		Period:            60,
		AllowedSeverities: []models.AlertSeverity{models.AlertSeverityCritical},
	}
	_, err := settingService.SetSetting(services.SettingKeyAutoRCA, config, "")
	assert.NoError(t, err)

	rcaService := NewRCAService(database, nil, nil, settingService, nil, nil)
	rcaService.SetAnalysisExecutor(func(*models.RCARun, *models.Alert) {})

	warningAlert := &models.Alert{
		Fingerprint: "fp-warning",
		ClusterName: "cluster-1",
		Title:       "Warning Alert",
		Status:      "firing",
		Severity:    string(models.AlertSeverityWarning),
	}
	database.Create(warningAlert)

	_, err = rcaService.TriggerRCAAnalysis(warningAlert)
	assert.Error(t, err, "warning severity should be filtered out")

	criticalAlert := &models.Alert{
		Fingerprint: "fp-critical",
		ClusterName: "cluster-1",
		Title:       "Critical Alert",
		Status:      "firing",
		Severity:    string(models.AlertSeverityCritical),
	}
	database.Create(criticalAlert)

	run, err := rcaService.TriggerRCAAnalysis(criticalAlert)
	assert.NoError(t, err)
	assert.Equal(t, string(models.RCAStatusPending), run.Status)
}
