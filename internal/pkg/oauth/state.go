package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

const stateTTL = 10 * time.Minute

// StateMeta holds metadata associated with an OAuth state token.
type StateMeta struct {
	Provider  string
	BindMode  bool // true if this is a bind (not login) flow
	UserID    uint // only set if BindMode is true
	CreatedAt time.Time
}

// StateManager manages OAuth state tokens with TTL expiry.
type StateManager struct {
	states sync.Map
	done   chan struct{}
}

func NewStateManager() *StateManager {
	sm := &StateManager{done: make(chan struct{})}
	go sm.cleanup()
	return sm
}

// Generate creates a new random state token and stores it with metadata.
func (sm *StateManager) Generate(provider string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	sm.states.Store(state, &StateMeta{
		Provider:  provider,
		CreatedAt: time.Now(),
	})
	return state, nil
}

// GenerateForBind creates a state token for a bind (account linking) flow.
func (sm *StateManager) GenerateForBind(provider string, userID uint) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	sm.states.Store(state, &StateMeta{
		Provider:  provider,
		BindMode:  true,
		UserID:    userID,
		CreatedAt: time.Now(),
	})
	return state, nil
}

// Validate checks and consumes a state token, returning its metadata.
func (sm *StateManager) Validate(state string) (*StateMeta, error) {
	val, ok := sm.states.LoadAndDelete(state)
	if !ok {
		return nil, fmt.Errorf("invalid state")
	}
	meta := val.(*StateMeta)
	if time.Since(meta.CreatedAt) > stateTTL {
		return nil, fmt.Errorf("state expired")
	}
	return meta, nil
}

// Stop terminates the cleanup goroutine.
func (sm *StateManager) Stop() {
	close(sm.done)
}

func (sm *StateManager) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-sm.done:
			return
		case <-ticker.C:
			now := time.Now()
			sm.states.Range(func(key, value any) bool {
				meta := value.(*StateMeta)
				if now.Sub(meta.CreatedAt) > stateTTL {
					sm.states.Delete(key)
				}
				return true
			})
		}
	}
}
