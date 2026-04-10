package token

import (
	"sync"
	"time"
)

// TokenBlacklist maintains an in-memory set of revoked JWT IDs (jti).
// Entries auto-expire when the original access token would have expired.
type TokenBlacklist struct {
	mu    sync.RWMutex
	items map[string]time.Time // jti → access token expiration time
}

func NewBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		items: make(map[string]time.Time),
	}
}

// Add marks a JWT ID as blocked until the given expiration time.
func (b *TokenBlacklist) Add(jti string, expiresAt time.Time) {
	if jti == "" {
		return
	}
	b.mu.Lock()
	b.items[jti] = expiresAt
	b.mu.Unlock()
}

// IsBlocked checks if a JWT ID is in the blacklist.
// Performs lazy cleanup of expired entries.
func (b *TokenBlacklist) IsBlocked(jti string) bool {
	b.mu.RLock()
	exp, ok := b.items[jti]
	b.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		// Lazy cleanup
		b.mu.Lock()
		delete(b.items, jti)
		b.mu.Unlock()
		return false
	}
	return true
}

// Cleanup removes all expired entries from the blacklist.
func (b *TokenBlacklist) Cleanup() int {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	removed := 0
	for jti, exp := range b.items {
		if now.After(exp) {
			delete(b.items, jti)
			removed++
		}
	}
	return removed
}

// Count returns the number of entries currently in the blacklist.
func (b *TokenBlacklist) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.items)
}
