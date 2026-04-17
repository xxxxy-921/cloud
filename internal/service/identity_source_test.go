package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/identity"
	"metis/internal/repository"
)

func newTestDBForIdentitySourceService(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.IdentitySource{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedIdentitySourceService(t *testing.T, db *gorm.DB, name, sourceType, config, domains string, enabled bool) *model.IdentitySource {
	t.Helper()
	s := &model.IdentitySource{
		Name:    name,
		Type:    sourceType,
		Config:  config,
		Domains: domains,
		Enabled: enabled,
	}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("seed identity source: %v", err)
	}
	return s
}

func newIdentitySourceServiceForTest(t *testing.T, db *gorm.DB) *IdentitySourceService {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewIdentitySource)
	do.Provide(injector, NewIdentitySource)
	return do.MustInvoke[*IdentitySourceService](injector)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceCreate_OIDC_EncryptsSecret(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	raw := json.RawMessage(`{"issuerUrl":"https://example.com","clientId":"id","clientSecret":"super-secret"}`)
	src := &model.IdentitySource{Name: "Okta", Type: "oidc"}
	if err := svc.Create(src, raw); err != nil {
		t.Fatalf("create: %v", err)
	}

	var stored model.IdentitySource
	if err := db.First(&stored, src.ID).Error; err != nil {
		t.Fatalf("find stored: %v", err)
	}
	var cfg model.OIDCConfig
	if err := json.Unmarshal([]byte(stored.Config), &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.ClientSecret == "" || cfg.ClientSecret == "super-secret" {
		t.Fatalf("expected encrypted secret, got %q", cfg.ClientSecret)
	}
}

func TestIdentitySourceServiceCreate_LDAP_FillsDefaultsAndEncryptsPassword(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	raw := json.RawMessage(`{"serverUrl":"ldap://localhost","bindPassword":"secret","searchBase":"dc=example,dc=com"}`)
	src := &model.IdentitySource{Name: "AD", Type: "ldap"}
	if err := svc.Create(src, raw); err != nil {
		t.Fatalf("create: %v", err)
	}

	var stored model.IdentitySource
	if err := db.First(&stored, src.ID).Error; err != nil {
		t.Fatalf("find stored: %v", err)
	}
	var cfg model.LDAPConfig
	if err := json.Unmarshal([]byte(stored.Config), &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.BindPassword == "" || cfg.BindPassword == "secret" {
		t.Fatalf("expected encrypted password, got %q", cfg.BindPassword)
	}
	if cfg.AttributeMapping == nil || cfg.AttributeMapping["username"] != "uid" {
		t.Fatalf("expected default attribute mapping, got %v", cfg.AttributeMapping)
	}
}

func TestIdentitySourceServiceCreate_UnsupportedType(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	src := &model.IdentitySource{Name: "SAML", Type: "saml"}
	if err := svc.Create(src, json.RawMessage(`{}`)); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestIdentitySourceServiceCreate_DomainConflict(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seedIdentitySourceService(t, db, "Existing", "oidc", `{}`, "example.com", true)

	src := &model.IdentitySource{Name: "New", Type: "oidc", Domains: "example.com"}
	if err := svc.Create(src, json.RawMessage(`{}`)); !errors.Is(err, repository.ErrDomainConflict) {
		t.Fatalf("expected ErrDomainConflict, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceUpdate_Success(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Old", "oidc", `{"issuerUrl":"https://old.com","clientId":"old"}`, "", true)

	raw := json.RawMessage(`{"issuerUrl":"https://new.com","clientId":"new"}`)
	updated := &model.IdentitySource{Name: "New", Domains: "new.com", SortOrder: 5}
	resp, err := svc.Update(seeded.ID, updated, raw)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Name != "New" {
		t.Fatalf("expected name New, got %s", resp.Name)
	}

	var stored model.IdentitySource
	if err := db.First(&stored, seeded.ID).Error; err != nil {
		t.Fatalf("find stored: %v", err)
	}
	if stored.Domains != "new.com" {
		t.Fatalf("expected domains updated, got %s", stored.Domains)
	}
}

func TestIdentitySourceServiceUpdate_PreservesMaskedOIDCSecret(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Okta", "oidc", `{"issuerUrl":"https://example.com","clientId":"id","clientSecret":"encrypted-secret"}`, "", true)

	raw := json.RawMessage(`{"issuerUrl":"https://example.com","clientId":"id","clientSecret":"` + model.IdentitySecretMask + `"}`)
	updated := &model.IdentitySource{Name: "Okta"}
	if _, err := svc.Update(seeded.ID, updated, raw); err != nil {
		t.Fatalf("update: %v", err)
	}

	var stored model.IdentitySource
	if err := db.First(&stored, seeded.ID).Error; err != nil {
		t.Fatalf("find stored: %v", err)
	}
	var cfg model.OIDCConfig
	if err := json.Unmarshal([]byte(stored.Config), &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.ClientSecret != "encrypted-secret" {
		t.Fatalf("expected secret preserved, got %q", cfg.ClientSecret)
	}
}

func TestIdentitySourceServiceUpdate_PreservesMaskedLDAPPassword(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost","bindPassword":"encrypted-pw"}`, "", true)

	raw := json.RawMessage(`{"serverUrl":"ldap://localhost","bindPassword":"` + model.IdentitySecretMask + `"}`)
	updated := &model.IdentitySource{Name: "AD"}
	if _, err := svc.Update(seeded.ID, updated, raw); err != nil {
		t.Fatalf("update: %v", err)
	}

	var stored model.IdentitySource
	if err := db.First(&stored, seeded.ID).Error; err != nil {
		t.Fatalf("find stored: %v", err)
	}
	var cfg model.LDAPConfig
	if err := json.Unmarshal([]byte(stored.Config), &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.BindPassword != "encrypted-pw" {
		t.Fatalf("expected password preserved, got %q", cfg.BindPassword)
	}
}

func TestIdentitySourceServiceUpdate_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	_, err := svc.Update(9999, &model.IdentitySource{Name: "X"}, json.RawMessage(`{}`))
	if !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("expected ErrSourceNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete / Toggle
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceDelete_Success(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Delete", "oidc", `{}`, "", true)

	if err := svc.Delete(seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	var count int64
	db.Model(&model.IdentitySource{}).Where("id = ?", seeded.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected record deleted, got count %d", count)
	}
}

func TestIdentitySourceServiceDelete_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	if err := svc.Delete(9999); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestIdentitySourceServiceToggle_Success(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Toggle", "oidc", `{}`, "", true)

	resp, err := svc.Toggle(seeded.ID)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if resp.Enabled {
		t.Fatal("expected disabled after toggle")
	}
}

func TestIdentitySourceServiceToggle_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	_, err := svc.Toggle(9999)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("expected ErrSourceNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestConnection
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceTestConnection_OIDCSuccess(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Okta", "oidc", `{"issuerUrl":"https://example.com"}`, "", true)
	svc.TestOIDCFn = func(ctx context.Context, issuerURL string) error { return nil }

	ok, msg := svc.TestConnection(seeded.ID)
	if !ok || msg != "OIDC discovery successful" {
		t.Fatalf("unexpected result: ok=%v msg=%q", ok, msg)
	}
}

func TestIdentitySourceServiceTestConnection_OIDCFailure(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Okta", "oidc", `{"issuerUrl":"https://example.com"}`, "", true)
	svc.TestOIDCFn = func(ctx context.Context, issuerURL string) error { return errors.New("discovery error") }

	ok, msg := svc.TestConnection(seeded.ID)
	if ok || msg != "OIDC discovery failed: discovery error" {
		t.Fatalf("unexpected result: ok=%v msg=%q", ok, msg)
	}
}

func TestIdentitySourceServiceTestConnection_LDAPSuccess(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost"}`, "", true)
	svc.TestLDAPFn = func(cfg *model.LDAPConfig) error { return nil }

	ok, msg := svc.TestConnection(seeded.ID)
	if !ok || msg != "LDAP bind successful" {
		t.Fatalf("unexpected result: ok=%v msg=%q", ok, msg)
	}
}

func TestIdentitySourceServiceTestConnection_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)

	ok, msg := svc.TestConnection(9999)
	if ok || msg != "identity source not found" {
		t.Fatalf("unexpected result: ok=%v msg=%q", ok, msg)
	}
}

// ---------------------------------------------------------------------------
// AuthenticateByPassword
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceAuthenticateByPassword_Success(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost"}`, "", true)
	seeded.DefaultRoleID = 7
	seeded.ConflictStrategy = "link"
	db.Save(seeded)

	svc.LDAPAuthFn = func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error) {
		return &identity.LDAPAuthResult{
			DN:          "cn=user,dc=example,dc=com",
			Username:    "user",
			Email:       "user@example.com",
			DisplayName: "User Name",
			Avatar:      "https://avatar",
		}, nil
	}

	result, err := svc.AuthenticateByPassword("user", "pass")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	if result.Provider != "ldap_1" {
		t.Fatalf("expected provider ldap_1, got %s", result.Provider)
	}
	if result.Email != "user@example.com" {
		t.Fatalf("expected email, got %s", result.Email)
	}
	if result.DefaultRoleID != 7 {
		t.Fatalf("expected default role 7, got %d", result.DefaultRoleID)
	}
	if result.ConflictStrategy != "link" {
		t.Fatalf("expected link strategy, got %s", result.ConflictStrategy)
	}
}

func TestIdentitySourceServiceAuthenticateByPassword_AllFail(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost"}`, "", true)

	svc.LDAPAuthFn = func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error) {
		return nil, errors.New("bind failed")
	}

	_, err := svc.AuthenticateByPassword("user", "pass")
	if err == nil || err.Error() != "error.identity.ldap_auth_failed" {
		t.Fatalf("expected ldap_auth_failed error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CheckDomain / IsForcedSSO / ExtractDomain
// ---------------------------------------------------------------------------

func TestIdentitySourceServiceCheckDomain(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Okta", "oidc", `{}`, "example.com", true)
	seeded.ForceSso = true
	db.Save(seeded)

	result, err := svc.CheckDomain("user@example.com")
	if err != nil {
		t.Fatalf("check domain: %v", err)
	}
	if result.SourceID != seeded.ID {
		t.Fatalf("expected source ID %d, got %d", seeded.ID, result.SourceID)
	}
	if !result.ForceSso {
		t.Fatal("expected forceSso true")
	}
}

func TestIdentitySourceServiceIsForcedSSO(t *testing.T) {
	db := newTestDBForIdentitySourceService(t)
	svc := newIdentitySourceServiceForTest(t, db)
	seeded := seedIdentitySourceService(t, db, "Okta", "oidc", `{}`, "example.com", true)
	seeded.ForceSso = true
	db.Save(seeded)

	if !svc.IsForcedSSO("user@example.com") {
		t.Fatal("expected forced SSO")
	}
	if svc.IsForcedSSO("user@other.com") {
		t.Fatal("expected not forced SSO")
	}
}

func TestExtractDomain(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"user@EXAMPLE.COM", "example.com"},
		{"user@example.com", "example.com"},
		{"invalid", ""},
		{"@example.com", ""},
		{"user@", ""},
	}
	for _, c := range cases {
		got := ExtractDomain(c.input)
		if got != c.expected {
			t.Fatalf("ExtractDomain(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}
