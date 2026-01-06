package services

import (
	"encoding/json"
	"sync"
	"time"
)

// SafeDrainEventManager handles broadcasting SSE events to clients
type SafeDrainEventManager struct {
	// listeners: map[drainID][]chan string
	listeners    map[string][]chan string
	listenersMux sync.RWMutex
}

func NewSafeDrainEventManager() *SafeDrainEventManager {
	return &SafeDrainEventManager{
		listeners: make(map[string][]chan string),
	}
}

// Subscribe returns a channel that receives serialized JSON messages for a specific drainID
// and a cleanup function to unsubscribe.
func (m *SafeDrainEventManager) Subscribe(drainID string) (chan string, func()) {
	ch := make(chan string, 50) // Buffered channel
	m.listenersMux.Lock()
	m.listeners[drainID] = append(m.listeners[drainID], ch)
	m.listenersMux.Unlock()

	return ch, func() {
		m.listenersMux.Lock()
		defer m.listenersMux.Unlock()

		subscribers := m.listeners[drainID]
		for i, sub := range subscribers {
			if sub == ch {
				// Remove by swapping with last element (if order doesn't matter) or slicing
				m.listeners[drainID] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}
		if len(m.listeners[drainID]) == 0 {
			delete(m.listeners, drainID)
		}
	}
}

// Broadcast sends a message to all subscribers of a drainID
func (m *SafeDrainEventManager) Broadcast(drainID string, msgType string, message string, data interface{}) {
	sseMsg := SSEMessage{
		Type:      msgType,
		DrainID:   drainID,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	jsonBytes, err := json.Marshal(sseMsg)
	if err != nil {
		return // Should handle error logging
	}
	payload := string(jsonBytes)

	m.listenersMux.RLock()
	defer m.listenersMux.RUnlock()

	for _, ch := range m.listeners[drainID] {
		select {
		case ch <- payload:
		default:
			// Drop message if channel is full to prevent blocking
		}
	}
}

// Helper methods for specific events

func (m *SafeDrainEventManager) SendLog(drainID, level, message string) {
	m.Broadcast(drainID, "log", message, map[string]interface{}{"level": level, "category": "drain"})
}

func (m *SafeDrainEventManager) SendProgress(drainID string, progress int, status string) {
	m.Broadcast(drainID, "progress", status, map[string]interface{}{
		"progress": progress,
		"step":     "evicting",
	})
}

func (m *SafeDrainEventManager) SendMigrationUpdate(drainID string, migration *DrainPodMigrationInfo) {
	m.Broadcast(drainID, "migration_update", "", map[string]interface{}{
		"migration": migration,
	})
}

func (m *SafeDrainEventManager) SendError(drainID, msg string) {
	m.Broadcast(drainID, "error", msg, nil)
}

// Auto-navy style, additional helpers (non-breaking)
func (m *SafeDrainEventManager) SendStarted(drainID, node string) {
	m.Broadcast(drainID, "started", "Drain started", map[string]interface{}{"node": node})
}

func (m *SafeDrainEventManager) SendCompleted(drainID string) {
	m.Broadcast(drainID, "completed", "Drain completed successfully", nil)
}

func (m *SafeDrainEventManager) SendCancelled(drainID string) {
	m.Broadcast(drainID, "canceled", "Drain canceled", nil)
}

func (m *SafeDrainEventManager) SendPodEvictionStarted(drainID string, migration *DrainPodMigrationInfo) {
	m.Broadcast(drainID, "pod_eviction_started", "Evicting pod", map[string]interface{}{
		"migration": migration,
		"resource":  map[string]interface{}{"kind": "Pod", "namespace": migration.SourcePod.Namespace, "name": migration.SourcePod.Name},
		"level":     "INFO",
		"category":  "pod",
	})
}

func (m *SafeDrainEventManager) SendPodEvictionSucceeded(drainID string, migration *DrainPodMigrationInfo) {
	m.Broadcast(drainID, "pod_eviction_succeeded", "Pod evicted", map[string]interface{}{
		"migration": migration,
		"resource":  map[string]interface{}{"kind": "Pod", "namespace": migration.SourcePod.Namespace, "name": migration.SourcePod.Name},
		"level":     "INFO",
		"category":  "pod",
	})
}

func (m *SafeDrainEventManager) SendPodEvictionFailed(drainID string, migration *DrainPodMigrationInfo, err string) {
	m.Broadcast(drainID, "pod_eviction_failed", err, map[string]interface{}{
		"migration": migration,
		"resource":  map[string]interface{}{"kind": "Pod", "namespace": migration.SourcePod.Namespace, "name": migration.SourcePod.Name},
		"level":     "ERROR",
		"category":  "pod",
		"error":     err,
	})
}

func (m *SafeDrainEventManager) SendPodMigrationIgnored(drainID string, migration *DrainPodMigrationInfo, reason string) {
	m.Broadcast(drainID, "pod_migration_ignored", reason, map[string]interface{}{
		"migration": migration,
		"resource":  map[string]interface{}{"kind": "Pod", "namespace": migration.SourcePod.Namespace, "name": migration.SourcePod.Name},
		"level":     "INFO",
		"category":  "pod",
		"reason":    reason,
	})
}
