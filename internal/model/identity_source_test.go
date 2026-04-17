package model

import (
	"encoding/json"
	"testing"
)

func TestIdentitySourceToResponse_OIDC_MasksSecret(t *testing.T) {
	source := IdentitySource{
		ID:       1,
		Name:     "Okta",
		Type:     "oidc",
		Config:   `{"issuerUrl":"https://example.com","clientId":"id","clientSecret":"super-secret"}`,
	}
	resp := source.ToResponse()
	var cfg OIDCConfig
	if err := json.Unmarshal(resp.Config, &cfg); err != nil {
		t.Fatalf("unmarshal oidc config: %v", err)
	}
	if cfg.ClientSecret != IdentitySecretMask {
		t.Fatalf("expected masked secret, got %q", cfg.ClientSecret)
	}
	if cfg.IssuerURL != "https://example.com" {
		t.Fatalf("expected issuer URL preserved, got %q", cfg.IssuerURL)
	}
}

func TestIdentitySourceToResponse_LDAP_MasksSecret(t *testing.T) {
	source := IdentitySource{
		ID:     2,
		Name:   "AD",
		Type:   "ldap",
		Config: `{"serverUrl":"ldap://localhost","bindPassword":"super-secret","searchBase":"dc=example,dc=com"}`,
	}
	resp := source.ToResponse()
	var cfg LDAPConfig
	if err := json.Unmarshal(resp.Config, &cfg); err != nil {
		t.Fatalf("unmarshal ldap config: %v", err)
	}
	if cfg.BindPassword != IdentitySecretMask {
		t.Fatalf("expected masked secret, got %q", cfg.BindPassword)
	}
	if cfg.ServerURL != "ldap://localhost" {
		t.Fatalf("expected server URL preserved, got %q", cfg.ServerURL)
	}
}

func TestIdentitySourceToResponse_UnknownType_Passthrough(t *testing.T) {
	source := IdentitySource{
		ID:     3,
		Name:   "Unknown",
		Type:   "saml",
		Config: `{"foo":"bar"}`,
	}
	resp := source.ToResponse()
	if string(resp.Config) != `{"foo":"bar"}` {
		t.Fatalf("expected passthrough config, got %s", string(resp.Config))
	}
}

func TestDefaultLDAPAttributeMapping(t *testing.T) {
	m := DefaultLDAPAttributeMapping()
	if m["username"] != "uid" {
		t.Fatalf("expected username->uid, got %q", m["username"])
	}
	if m["email"] != "mail" {
		t.Fatalf("expected email->mail, got %q", m["email"])
	}
	if m["display_name"] != "cn" {
		t.Fatalf("expected display_name->cn, got %q", m["display_name"])
	}
}
