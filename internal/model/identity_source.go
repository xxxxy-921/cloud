package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// IdentitySource represents an external identity provider (OIDC or LDAP).
type IdentitySource struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
	Name             string         `json:"name" gorm:"size:128;not null"`
	Type             string         `json:"type" gorm:"size:16;not null"`             // "oidc" or "ldap"
	Enabled          bool           `json:"enabled" gorm:"not null;default:false"`
	Domains          string         `json:"domains" gorm:"size:512"`                   // comma-separated email domains
	ForceSso         bool           `json:"forceSso" gorm:"not null;default:false"`
	DefaultRoleID    uint           `json:"defaultRoleId" gorm:"not null;default:0"`
	ConflictStrategy string         `json:"conflictStrategy" gorm:"size:16;not null;default:fail"` // "link" or "fail"
	Config           string         `json:"-" gorm:"type:text"`                        // JSON-encoded OIDCConfig or LDAPConfig
	SortOrder        int            `json:"sortOrder" gorm:"not null;default:0"`
}

// OIDCConfig holds OIDC-specific configuration stored as JSON in Config field.
type OIDCConfig struct {
	IssuerURL    string   `json:"issuerUrl"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"` // AES-256-GCM encrypted at rest
	Scopes       []string `json:"scopes"`
	UsePKCE      bool     `json:"usePkce"`
	CallbackURL  string   `json:"callbackUrl"`
}

// LDAPConfig holds LDAP-specific configuration stored as JSON in Config field.
type LDAPConfig struct {
	ServerURL        string            `json:"serverUrl"`
	BindDN           string            `json:"bindDn"`
	BindPassword     string            `json:"bindPassword"` // AES-256-GCM encrypted at rest
	SearchBase       string            `json:"searchBase"`
	UserFilter       string            `json:"userFilter"` // e.g. "(uid={{username}})"
	UseTLS           bool              `json:"useTls"`
	SkipVerify       bool              `json:"skipVerify"`
	AttributeMapping map[string]string `json:"attributeMapping"` // keys: username, email, display_name, avatar
}

// DefaultLDAPAttributeMapping returns the default LDAP→User attribute mapping.
func DefaultLDAPAttributeMapping() map[string]string {
	return map[string]string{
		"username":     "uid",
		"email":        "mail",
		"display_name": "cn",
	}
}

// IdentitySourceResponse is the safe representation for API responses.
type IdentitySourceResponse struct {
	ID               uint            `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	Enabled          bool            `json:"enabled"`
	Domains          string          `json:"domains"`
	ForceSso         bool            `json:"forceSso"`
	DefaultRoleID    uint            `json:"defaultRoleId"`
	ConflictStrategy string          `json:"conflictStrategy"`
	Config           json.RawMessage `json:"config"`
	SortOrder        int             `json:"sortOrder"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

const IdentitySecretMask = "••••••"

// ToResponse converts an IdentitySource to a safe API response with masked secrets.
func (s *IdentitySource) ToResponse() IdentitySourceResponse {
	resp := IdentitySourceResponse{
		ID:               s.ID,
		Name:             s.Name,
		Type:             s.Type,
		Enabled:          s.Enabled,
		Domains:          s.Domains,
		ForceSso:         s.ForceSso,
		DefaultRoleID:    s.DefaultRoleID,
		ConflictStrategy: s.ConflictStrategy,
		SortOrder:        s.SortOrder,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}

	switch s.Type {
	case "oidc":
		var cfg OIDCConfig
		if err := json.Unmarshal([]byte(s.Config), &cfg); err == nil {
			if cfg.ClientSecret != "" {
				cfg.ClientSecret = IdentitySecretMask
			}
			resp.Config, _ = json.Marshal(cfg)
		}
	case "ldap":
		var cfg LDAPConfig
		if err := json.Unmarshal([]byte(s.Config), &cfg); err == nil {
			if cfg.BindPassword != "" {
				cfg.BindPassword = IdentitySecretMask
			}
			resp.Config, _ = json.Marshal(cfg)
		}
	default:
		resp.Config = json.RawMessage(s.Config)
	}

	return resp
}
