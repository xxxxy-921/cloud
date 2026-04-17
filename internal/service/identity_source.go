package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/identity"
	"metis/internal/pkg/token"
	"metis/internal/repository"
)

var (
	ErrUnsupportedType = errors.New("error.identity.unsupported_type")
	ErrSourceNotFound  = errors.New("error.identity.not_found")
)

// ExternalAuthResult contains the external identity info from a successful
// external authentication (e.g. LDAP). It is returned to the caller so that
// JIT provisioning can be handled by AuthService.ProvisionExternalUser.
type ExternalAuthResult struct {
	Provider         string // e.g. "ldap_3"
	ExternalID       string // e.g. DN for LDAP
	Email            string
	DisplayName      string
	AvatarURL        string
	Username         string
	DefaultRoleID    uint
	ConflictStrategy string
}

// DomainCheckResult is the result of checking an email domain against identity sources.
type DomainCheckResult struct {
	SourceID uint   `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ForceSso bool   `json:"forceSso"`
}

type IdentitySourceService struct {
	repo *repository.IdentitySourceRepo
	db   *database.DB

	TestOIDCFn  func(ctx context.Context, issuerURL string) error
	TestLDAPFn  func(cfg *model.LDAPConfig) error
	LDAPAuthFn  func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error)
}

func NewIdentitySource(i do.Injector) (*IdentitySourceService, error) {
	return &IdentitySourceService{
		repo: do.MustInvoke[*repository.IdentitySourceRepo](i),
		db:   do.MustInvoke[*database.DB](i),
	}, nil
}

// ---------------------------------------------------------------------------
// CRUD operations
// ---------------------------------------------------------------------------

func (s *IdentitySourceService) List() ([]model.IdentitySourceResponse, error) {
	sources, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	resp := make([]model.IdentitySourceResponse, 0, len(sources))
	for _, src := range sources {
		resp = append(resp, src.ToResponse())
	}
	return resp, nil
}

func (s *IdentitySourceService) Create(source *model.IdentitySource, rawConfig json.RawMessage) error {
	if source.Type != "oidc" && source.Type != "ldap" {
		return ErrUnsupportedType
	}

	if err := s.repo.CheckDomainConflict(source.Domains, 0); err != nil {
		return err
	}

	configJSON, err := s.encryptConfig(source.Type, rawConfig)
	if err != nil {
		return fmt.Errorf("encrypt config: %w", err)
	}
	source.Config = configJSON

	return s.repo.Create(source)
}

func (s *IdentitySourceService) Update(id uint, source *model.IdentitySource, rawConfig json.RawMessage) (*model.IdentitySourceResponse, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceNotFound
		}
		return nil, err
	}

	if err := s.repo.CheckDomainConflict(source.Domains, id); err != nil {
		return nil, err
	}

	existing.Name = source.Name
	existing.Domains = source.Domains
	existing.ForceSso = source.ForceSso
	existing.DefaultRoleID = source.DefaultRoleID
	existing.ConflictStrategy = source.ConflictStrategy
	existing.SortOrder = source.SortOrder

	configJSON, err := s.encryptConfigPreserving(existing.Type, rawConfig, existing.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt config: %w", err)
	}
	existing.Config = configJSON

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	resp := existing.ToResponse()
	return &resp, nil
}

func (s *IdentitySourceService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSourceNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}

func (s *IdentitySourceService) Toggle(id uint) (*model.IdentitySourceResponse, error) {
	source, err := s.repo.Toggle(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceNotFound
		}
		return nil, err
	}
	resp := source.ToResponse()
	return &resp, nil
}

// TestConnection tests connectivity for the identity source.
func (s *IdentitySourceService) TestConnection(id uint) (bool, string) {
	source, err := s.repo.FindByID(id)
	if err != nil {
		return false, "identity source not found"
	}

	switch source.Type {
	case "oidc":
		return s.runTestOIDC(source)
	case "ldap":
		return s.runTestLDAP(source)
	default:
		return false, "unsupported type"
	}
}

func (s *IdentitySourceService) runTestOIDC(source *model.IdentitySource) (bool, string) {
	var cfg model.OIDCConfig
	if err := json.Unmarshal([]byte(source.Config), &cfg); err != nil {
		return false, "invalid OIDC config: " + err.Error()
	}

	if cfg.ClientSecret != "" {
		decrypted, err := token.Decrypt(s.db.DB, cfg.ClientSecret)
		if err == nil {
			cfg.ClientSecret = decrypted
		}
	}

	if cfg.IssuerURL == "" {
		return false, "issuer URL is empty"
	}

	ctx := context.Background()
	testFn := s.TestOIDCFn
	if testFn == nil {
		testFn = identity.TestOIDCDiscovery
	}
	if err := testFn(ctx, cfg.IssuerURL); err != nil {
		return false, "OIDC discovery failed: " + err.Error()
	}

	return true, "OIDC discovery successful"
}

func (s *IdentitySourceService) runTestLDAP(source *model.IdentitySource) (bool, string) {
	var cfg model.LDAPConfig
	if err := json.Unmarshal([]byte(source.Config), &cfg); err != nil {
		return false, "invalid LDAP config: " + err.Error()
	}

	if cfg.BindPassword != "" {
		decrypted, err := token.Decrypt(s.db.DB, cfg.BindPassword)
		if err == nil {
			cfg.BindPassword = decrypted
		}
	}

	if cfg.ServerURL == "" {
		return false, "server URL is empty"
	}

	testFn := s.TestLDAPFn
	if testFn == nil {
		testFn = identity.TestLDAPConnection
	}
	if err := testFn(&cfg); err != nil {
		return false, "LDAP bind failed: " + err.Error()
	}

	return true, "LDAP bind successful"
}

// ---------------------------------------------------------------------------
// Encryption helpers
// ---------------------------------------------------------------------------

func (s *IdentitySourceService) encryptConfig(sourceType string, raw json.RawMessage) (string, error) {
	switch sourceType {
	case "oidc":
		var cfg model.OIDCConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", err
		}
		if cfg.ClientSecret != "" {
			encrypted, err := token.Encrypt(s.db.DB, cfg.ClientSecret)
			if err != nil {
				return "", err
			}
			cfg.ClientSecret = encrypted
		}
		b, err := json.Marshal(cfg)
		return string(b), err

	case "ldap":
		var cfg model.LDAPConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", err
		}
		if cfg.AttributeMapping == nil {
			cfg.AttributeMapping = model.DefaultLDAPAttributeMapping()
		}
		if cfg.BindPassword != "" {
			encrypted, err := token.Encrypt(s.db.DB, cfg.BindPassword)
			if err != nil {
				return "", err
			}
			cfg.BindPassword = encrypted
		}
		b, err := json.Marshal(cfg)
		return string(b), err

	default:
		return string(raw), nil
	}
}

func (s *IdentitySourceService) encryptConfigPreserving(sourceType string, raw json.RawMessage, existingConfig string) (string, error) {
	switch sourceType {
	case "oidc":
		var cfg model.OIDCConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", err
		}
		if cfg.ClientSecret == model.IdentitySecretMask {
			var existing model.OIDCConfig
			if err := json.Unmarshal([]byte(existingConfig), &existing); err == nil {
				cfg.ClientSecret = existing.ClientSecret
			}
		} else if cfg.ClientSecret != "" {
			encrypted, err := token.Encrypt(s.db.DB, cfg.ClientSecret)
			if err != nil {
				return "", err
			}
			cfg.ClientSecret = encrypted
		}
		b, err := json.Marshal(cfg)
		return string(b), err

	case "ldap":
		var cfg model.LDAPConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", err
		}
		if cfg.AttributeMapping == nil {
			cfg.AttributeMapping = model.DefaultLDAPAttributeMapping()
		}
		if cfg.BindPassword == model.IdentitySecretMask {
			var existing model.LDAPConfig
			if err := json.Unmarshal([]byte(existingConfig), &existing); err == nil {
				cfg.BindPassword = existing.BindPassword
			}
		} else if cfg.BindPassword != "" {
			encrypted, err := token.Encrypt(s.db.DB, cfg.BindPassword)
			if err != nil {
				return "", err
			}
			cfg.BindPassword = encrypted
		}
		b, err := json.Marshal(cfg)
		return string(b), err

	default:
		return string(raw), nil
	}
}

// GetDecryptedConfig returns a source with its config secrets decrypted (for internal use).
func (s *IdentitySourceService) GetDecryptedConfig(id uint) (*model.IdentitySource, any, error) {
	source, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSourceNotFound
		}
		return nil, nil, err
	}

	switch source.Type {
	case "oidc":
		var cfg model.OIDCConfig
		if err := json.Unmarshal([]byte(source.Config), &cfg); err != nil {
			return source, nil, err
		}
		if cfg.ClientSecret != "" {
			decrypted, err := token.Decrypt(s.db.DB, cfg.ClientSecret)
			if err != nil {
				return source, nil, fmt.Errorf("decrypt client_secret: %w", err)
			}
			cfg.ClientSecret = decrypted
		}
		return source, &cfg, nil

	case "ldap":
		var cfg model.LDAPConfig
		if err := json.Unmarshal([]byte(source.Config), &cfg); err != nil {
			return source, nil, err
		}
		if cfg.BindPassword != "" {
			decrypted, err := token.Decrypt(s.db.DB, cfg.BindPassword)
			if err != nil {
				return source, nil, fmt.Errorf("decrypt bind_password: %w", err)
			}
			cfg.BindPassword = decrypted
		}
		return source, &cfg, nil

	default:
		return source, nil, nil
	}
}

// FindByDomain looks up an enabled identity source by email domain.
func (s *IdentitySourceService) FindByDomain(domain string) (*model.IdentitySource, error) {
	return s.repo.FindByDomain(domain)
}

// ---------------------------------------------------------------------------
// Authentication (formerly in Authenticator)
// ---------------------------------------------------------------------------

// AuthenticateByPassword tries LDAP bind authentication for the given credentials.
// Returns an ExternalAuthResult that the caller can pass to AuthService.ProvisionExternalUser.
func (s *IdentitySourceService) AuthenticateByPassword(username, password string) (*ExternalAuthResult, error) {
	sources, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	for _, source := range sources {
		if !source.Enabled || source.Type != "ldap" {
			continue
		}

		var cfg model.LDAPConfig
		if err := json.Unmarshal([]byte(source.Config), &cfg); err != nil {
			slog.Error("identity: invalid LDAP config", "sourceId", source.ID, "error", err)
			continue
		}

		if cfg.BindPassword != "" {
			decrypted, err := token.Decrypt(s.db.DB, cfg.BindPassword)
			if err != nil {
				slog.Error("identity: decrypt bind password failed", "sourceId", source.ID, "error", err)
				continue
			}
			cfg.BindPassword = decrypted
		}

		authFn := s.LDAPAuthFn
		if authFn == nil {
			authFn = identity.LDAPAuthenticate
		}
		result, err := authFn(&cfg, username, password)
		if err != nil {
			slog.Debug("identity: LDAP auth failed", "sourceId", source.ID, "username", username, "error", err)
			continue
		}

		providerName := fmt.Sprintf("ldap_%d", source.ID)
		preferredUsername := result.Username
		if preferredUsername == "" {
			preferredUsername = fmt.Sprintf("%s_%s", providerName, result.DN)
		}

		return &ExternalAuthResult{
			Provider:         providerName,
			ExternalID:       result.DN,
			Email:            result.Email,
			DisplayName:      result.DisplayName,
			AvatarURL:        result.Avatar,
			Username:         preferredUsername,
			DefaultRoleID:    source.DefaultRoleID,
			ConflictStrategy: source.ConflictStrategy,
		}, nil
	}

	return nil, errors.New("error.identity.ldap_auth_failed")
}

// CheckDomain checks if the email domain matches an enabled identity source.
func (s *IdentitySourceService) CheckDomain(email string) (*DomainCheckResult, error) {
	domain := ExtractDomain(email)
	if domain == "" {
		return nil, errors.New("error.identity.invalid_email")
	}

	source, err := s.repo.FindByDomain(domain)
	if err != nil {
		return nil, err
	}

	return &DomainCheckResult{
		SourceID: source.ID,
		Name:     source.Name,
		Type:     source.Type,
		ForceSso: source.ForceSso,
	}, nil
}

// IsForcedSSO checks if the user's email domain requires SSO login.
func (s *IdentitySourceService) IsForcedSSO(email string) bool {
	domain := ExtractDomain(email)
	if domain == "" {
		return false
	}

	source, err := s.repo.FindByDomain(domain)
	if err != nil {
		return false
	}
	return source.ForceSso
}

// ExtractDomain extracts the domain part from an email address.
func ExtractDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}
