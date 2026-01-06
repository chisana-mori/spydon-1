package services

import (
	"encoding/json"
	"fmt"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"sync"
	"time"
)

// DrainStateManager manages drain operation state
type DrainStateManager struct {
	redisHandler sharedservices.RedisClient
	memCache     sync.Map
	ttl          time.Duration
}

// NewDrainStateManager creates a new DrainStateManager
func NewDrainStateManager(redisHandler sharedservices.RedisClient) *DrainStateManager {
	return &DrainStateManager{
		redisHandler: redisHandler,
		ttl:          24 * time.Hour,
	}
}

// GetDrainState retrieves the state for a given drainID
func (dsm *DrainStateManager) GetDrainState(drainID string) (*DrainState, error) {
	if cached, found := dsm.memCache.Load(drainID); found {
		if state, ok := cached.(*DrainState); ok {
			key := dsm.getStateKey(drainID)
			if dsm.redisHandler.Get(key) != "" {
				return state, nil
			}
			dsm.memCache.Delete(drainID)
		}
	}

	key := dsm.getStateKey(drainID)
	val := dsm.redisHandler.Get(key)
	if val == "" {
		return nil, nil
	}

	var state DrainState
	err := json.Unmarshal([]byte(val), &state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse drain state: %w", err)
	}

	dsm.memCache.Store(drainID, &state)
	return &state, nil
}

// SaveDrainState persists drain state
func (dsm *DrainStateManager) SaveDrainState(drainID string, state *DrainState) error {
	state.LastUpdate = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	key := dsm.getStateKey(drainID)
	dsm.redisHandler.SetWithExpireTime(key, string(data), dsm.ttl)
	dsm.memCache.Store(drainID, state)

	return nil
}

// DeleteDrainState removes drain state
func (dsm *DrainStateManager) DeleteDrainState(drainID string) error {
	key := dsm.getStateKey(drainID)
	dsm.redisHandler.Delete(key)
	dsm.memCache.Delete(drainID)
	return nil
}

// IsDrainActive checks if a drain is active
func (dsm *DrainStateManager) IsDrainActive(drainID string) bool {
	key := dsm.getStateKey(drainID)
	return dsm.redisHandler.Get(key) != ""
}

// GetAllActiveDrains returns all active drain IDs
func (dsm *DrainStateManager) GetAllActiveDrains() ([]string, error) {
	pattern := "drain:state:*"
	keys, err := dsm.redisHandler.ScanKeys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get active drains: %w", err)
	}

	drainIDs := make([]string, len(keys))
	for i, key := range keys {
		if len(key) > 12 {
			drainIDs[i] = key[12:]
		}
	}

	return drainIDs, nil
}

// UpdateDrainStatus updates the status of a drain
func (dsm *DrainStateManager) UpdateDrainStatus(drainID, status string) error {
	state, err := dsm.GetDrainState(drainID)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("drain state not found: %s", drainID)
	}

	state.Status = status
	state.LastUpdate = time.Now()

	return dsm.SaveDrainState(drainID, state)
}

// getStateKey generates the Redis key for state
func (dsm *DrainStateManager) getStateKey(drainID string) string {
	return fmt.Sprintf("drain:state:%s", drainID)
}

// StateManager defines the interface for drain state management
type StateManager interface {
	GetDrainState(drainID string) (*DrainState, error)
	SaveDrainState(drainID string, state *DrainState) error
	DeleteDrainState(drainID string) error
	IsDrainActive(drainID string) bool
	UpdateDrainStatus(drainID, status string) error
}

var _ StateManager = (*DrainStateManager)(nil)
