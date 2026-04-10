package service

import (
	"fmt"

	"github.com/samber/do/v2"

	"metis/internal/model"
	"metis/internal/pkg/oauth"
	"metis/internal/repository"
)

type AuthProviderService struct {
	repo *repository.AuthProviderRepo
}

func NewAuthProvider(i do.Injector) (*AuthProviderService, error) {
	return &AuthProviderService{
		repo: do.MustInvoke[*repository.AuthProviderRepo](i),
	}, nil
}

func (s *AuthProviderService) ListEnabled() ([]model.AuthProvider, error) {
	return s.repo.FindAllEnabled()
}

func (s *AuthProviderService) ListAll() ([]model.AuthProvider, error) {
	return s.repo.FindAll()
}

func (s *AuthProviderService) FindByKey(key string) (*model.AuthProvider, error) {
	return s.repo.FindByKey(key)
}

func (s *AuthProviderService) Update(key string, updates map[string]any) (*model.AuthProvider, error) {
	p, err := s.repo.FindByKey(key)
	if err != nil {
		return nil, err
	}

	if v, ok := updates["displayName"].(string); ok {
		p.DisplayName = v
	}
	if v, ok := updates["clientId"].(string); ok {
		p.ClientID = v
	}
	if v, ok := updates["clientSecret"].(string); ok && v != "••••••" && v != "" {
		p.ClientSecret = v
	}
	if v, ok := updates["scopes"].(string); ok {
		p.Scopes = v
	}
	if v, ok := updates["callbackUrl"].(string); ok {
		p.CallbackURL = v
	}
	if v, ok := updates["sortOrder"].(float64); ok {
		p.SortOrder = int(v)
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AuthProviderService) Toggle(key string) (*model.AuthProvider, error) {
	p, err := s.repo.FindByKey(key)
	if err != nil {
		return nil, err
	}
	p.Enabled = !p.Enabled
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// BuildOAuthProvider creates an OAuthProvider from database config.
func (s *AuthProviderService) BuildOAuthProvider(p *model.AuthProvider) (oauth.OAuthProvider, error) {
	switch p.ProviderKey {
	case "github":
		return oauth.NewGitHub(p.ClientID, p.ClientSecret, p.CallbackURL, p.Scopes), nil
	case "google":
		return oauth.NewGoogle(p.ClientID, p.ClientSecret, p.CallbackURL, p.Scopes), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", p.ProviderKey)
	}
}
