package oauth

import (
	"context"
	"strings"
)

// OAuthUserInfo holds user info obtained from an OAuth provider.
type OAuthUserInfo struct {
	ID        string
	Name      string
	Email     string
	AvatarURL string
}

// OAuthProvider defines the interface for OAuth providers.
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthUserInfo, error)
}

// splitScopes splits a space-separated scopes string into a slice.
func splitScopes(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}
