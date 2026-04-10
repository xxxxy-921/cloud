package identity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

const ssoStateTTL = 10 * time.Minute

// SSOStateMeta holds SSO state data stored during the auth flow.
type SSOStateMeta struct {
	SourceID     uint
	CodeVerifier string // PKCE code_verifier
	CreatedAt    time.Time
}

// SSOStateManager manages SSO state tokens (separate from kernel OAuth StateManager).
type SSOStateManager struct {
	states sync.Map
	done   chan struct{}
}

func NewSSOStateManager() *SSOStateManager {
	sm := &SSOStateManager{done: make(chan struct{})}
	go sm.cleanup()
	return sm
}

// Generate creates a state token storing source ID and optional PKCE verifier.
func (sm *SSOStateManager) Generate(sourceID uint, codeVerifier string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate SSO state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	sm.states.Store(state, &SSOStateMeta{
		SourceID:     sourceID,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now(),
	})
	return state, nil
}

// Validate checks and consumes a state token, returning its metadata.
func (sm *SSOStateManager) Validate(state string) (*SSOStateMeta, error) {
	val, ok := sm.states.LoadAndDelete(state)
	if !ok {
		return nil, fmt.Errorf("invalid or expired state")
	}
	meta := val.(*SSOStateMeta)
	if time.Since(meta.CreatedAt) > ssoStateTTL {
		return nil, fmt.Errorf("state expired")
	}
	return meta, nil
}

func (sm *SSOStateManager) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-sm.done:
			return
		case <-ticker.C:
			now := time.Now()
			sm.states.Range(func(key, value any) bool {
				meta := value.(*SSOStateMeta)
				if now.Sub(meta.CreatedAt) > ssoStateTTL {
					sm.states.Delete(key)
				}
				return true
			})
		}
	}
}
