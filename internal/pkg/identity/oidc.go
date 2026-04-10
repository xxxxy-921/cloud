package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"metis/internal/model"
)

// OIDCProvider wraps a go-oidc Provider with cached metadata and helper methods.
type OIDCProvider struct {
	provider *gooidc.Provider
	config   *model.OIDCConfig
	sourceID uint
	cachedAt time.Time
}

const oidcCacheTTL = 1 * time.Hour

var (
	oidcProviders   = make(map[uint]*OIDCProvider)
	oidcProvidersMu sync.RWMutex
)

// GetOIDCProvider returns a cached or freshly-discovered OIDC provider.
func GetOIDCProvider(ctx context.Context, sourceID uint, cfg *model.OIDCConfig) (*OIDCProvider, error) {
	oidcProvidersMu.RLock()
	if cached, ok := oidcProviders[sourceID]; ok && time.Since(cached.cachedAt) < oidcCacheTTL {
		oidcProvidersMu.RUnlock()
		return cached, nil
	}
	oidcProvidersMu.RUnlock()

	provider, err := gooidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed for %s: %w", cfg.IssuerURL, err)
	}

	op := &OIDCProvider{
		provider: provider,
		config:   cfg,
		sourceID: sourceID,
		cachedAt: time.Now(),
	}

	oidcProvidersMu.Lock()
	oidcProviders[sourceID] = op
	oidcProvidersMu.Unlock()

	return op, nil
}

// OAuth2Config builds an oauth2.Config from the OIDC config.
func (op *OIDCProvider) OAuth2Config() *oauth2.Config {
	scopes := op.config.Scopes
	if len(scopes) == 0 {
		scopes = []string{gooidc.ScopeOpenID, "profile", "email"}
	}
	return &oauth2.Config{
		ClientID:     op.config.ClientID,
		ClientSecret: op.config.ClientSecret,
		Endpoint:     op.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  op.config.CallbackURL,
	}
}

// Verifier returns an ID token verifier for this provider.
func (op *OIDCProvider) Verifier() *gooidc.IDTokenVerifier {
	return op.provider.Verifier(&gooidc.Config{ClientID: op.config.ClientID})
}

// AuthURL builds an authorization URL with PKCE if configured.
func (op *OIDCProvider) AuthURL(state string, pkce *PKCEParams) string {
	cfg := op.OAuth2Config()
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if pkce != nil {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", pkce.Challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	return cfg.AuthCodeURL(state, opts...)
}

// ExchangeCode exchanges an authorization code for tokens.
func (op *OIDCProvider) ExchangeCode(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error) {
	cfg := op.OAuth2Config()
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}
	return cfg.Exchange(ctx, code, opts...)
}

// VerifyIDToken extracts and verifies the ID token from an OAuth2 token.
func (op *OIDCProvider) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*gooidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in response")
	}
	return op.Verifier().Verify(ctx, rawIDToken)
}

// OIDCClaims holds standard claims extracted from an OIDC ID token.
type OIDCClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// ExtractClaims extracts standard claims from an ID token.
func ExtractClaims(idToken *gooidc.IDToken) (*OIDCClaims, error) {
	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}
	return &claims, nil
}

// PKCEParams holds PKCE code_verifier and code_challenge.
type PKCEParams struct {
	Verifier  string
	Challenge string
}

// GeneratePKCE creates a PKCE code_verifier and S256 code_challenge.
func GeneratePKCE() (*PKCEParams, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	return &PKCEParams{Verifier: verifier, Challenge: challenge}, nil
}

// TestOIDCDiscovery tests that OIDC discovery succeeds for the given issuer URL.
func TestOIDCDiscovery(ctx context.Context, issuerURL string) error {
	_, err := gooidc.NewProvider(ctx, issuerURL)
	return err
}

// ClearOIDCProviderCache removes a cached provider (e.g., after config update).
func ClearOIDCProviderCache(sourceID uint) {
	oidcProvidersMu.Lock()
	delete(oidcProviders, sourceID)
	oidcProvidersMu.Unlock()
}
