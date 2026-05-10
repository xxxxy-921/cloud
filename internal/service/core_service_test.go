package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/pquerna/otp/totp"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/token"
	"metis/internal/repository"
)

func newCoreServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.SystemConfig{},
		&model.RefreshToken{},
		&model.UserConnection{},
		&model.AuthProvider{},
		&model.TwoFactorSecret{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func newCoreServiceInjector(db *gorm.DB) do.Injector {
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.ProvideValue(injector, token.NewBlacklist())
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, repository.NewUserConnection)
	do.Provide(injector, repository.NewAuthProvider)
	do.Provide(injector, repository.NewTwoFactorSecret)
	return injector
}

func seedCoreRole(t *testing.T, db *gorm.DB, code string) *model.Role {
	t.Helper()
	role := &model.Role{Name: code, Code: code}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	return role
}

func seedCoreUser(t *testing.T, db *gorm.DB, username string, roleID uint) *model.User {
	t.Helper()
	hash, err := token.HashPassword("Password123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{
		Username:          username,
		Password:          hash,
		Email:             username + "@example.com",
		RoleID:            roleID,
		IsActive:          true,
		PasswordChangedAt: ptrTime(time.Now()),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func ptrTime(v time.Time) *time.Time { return &v }

func TestSysConfigServiceCRUD(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewSysConfig(injector)
	if err != nil {
		t.Fatalf("NewSysConfig returned error: %v", err)
	}

	cfg := &model.SystemConfig{Key: "site.name", Value: "Metis"}
	if err := svc.Set(cfg); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	got, err := svc.Get("site.name")
	if err != nil || got.Value != "Metis" {
		t.Fatalf("unexpected Get result: %+v err=%v", got, err)
	}
	items, err := svc.List()
	if err != nil || len(items) != 1 {
		t.Fatalf("unexpected List result: len=%d err=%v", len(items), err)
	}
	if err := svc.Delete("site.name"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := svc.Get("site.name"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected repository.ErrNotFound, got %v", err)
	}
}

func TestUserConnectionServiceBindAndUnbind(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewUserConnection(injector)
	if err != nil {
		t.Fatalf("NewUserConnection returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)
	other := seedCoreUser(t, db, "bob", role.ID)

	if err := svc.Bind(user.ID, "github", "ext-1", "alice", "alice@example.com", ""); err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}
	if err := svc.Bind(user.ID, "github", "ext-2", "alice", "alice@example.com", ""); !errors.Is(err, ErrAlreadyBound) {
		t.Fatalf("expected ErrAlreadyBound, got %v", err)
	}
	if err := svc.Bind(other.ID, "github", "ext-1", "bob", "bob@example.com", ""); !errors.Is(err, ErrExternalIDBound) {
		t.Fatalf("expected ErrExternalIDBound, got %v", err)
	}

	items, err := svc.ListByUser(user.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one bound connection, got len=%d err=%v", len(items), err)
	}

	user.Password = ""
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("save user without password: %v", err)
	}
	if err := svc.Unbind(user.ID, "github"); !errors.Is(err, ErrLastLoginMethod) {
		t.Fatalf("expected ErrLastLoginMethod, got %v", err)
	}

	if err := svc.Bind(user.ID, "google", "ext-2", "alice", "alice@example.com", ""); err != nil {
		t.Fatalf("Bind google returned error: %v", err)
	}
	if err := svc.Unbind(user.ID, "github"); err != nil {
		t.Fatalf("Unbind returned error: %v", err)
	}
	if err := svc.Unbind(user.ID, "missing"); !errors.Is(err, ErrConnectionNotFound) {
		t.Fatalf("expected ErrConnectionNotFound, got %v", err)
	}
}

func TestAuthProviderServiceUpdateToggleAndBuild(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewAuthProvider(injector)
	if err != nil {
		t.Fatalf("NewAuthProvider returned error: %v", err)
	}

	provider := &model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      false,
		ClientID:     "old-id",
		ClientSecret: "old-secret",
		Scopes:       "read:user",
		CallbackURL:  "https://old.example.com/callback",
		SortOrder:    1,
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	updated, err := svc.Update("github", map[string]any{
		"displayName":  "GitHub OAuth",
		"clientId":     "new-id",
		"clientSecret": "••••••",
		"callbackUrl":  "https://new.example.com/callback",
		"sortOrder":    float64(9),
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.DisplayName != "GitHub OAuth" || updated.ClientID != "new-id" || updated.ClientSecret != "old-secret" || updated.SortOrder != 9 {
		t.Fatalf("unexpected updated provider: %+v", updated)
	}

	toggled, err := svc.Toggle("github")
	if err != nil {
		t.Fatalf("Toggle returned error: %v", err)
	}
	if !toggled.Enabled {
		t.Fatalf("expected provider to be enabled, got %+v", toggled)
	}

	enabled, err := svc.ListEnabled()
	if err != nil || len(enabled) != 1 {
		t.Fatalf("expected one enabled provider, got len=%d err=%v", len(enabled), err)
	}
	all, err := svc.ListAll()
	if err != nil || len(all) != 1 {
		t.Fatalf("expected one total provider, got len=%d err=%v", len(all), err)
	}
	found, err := svc.FindByKey("github")
	if err != nil || found.ID != provider.ID {
		t.Fatalf("unexpected FindByKey result: %+v err=%v", found, err)
	}

	oauthProvider, err := svc.BuildOAuthProvider(provider)
	if err != nil || oauthProvider == nil {
		t.Fatalf("expected OAuth provider to be built, got %v %v", oauthProvider, err)
	}
	if _, err := svc.BuildOAuthProvider(&model.AuthProvider{ProviderKey: "custom"}); err == nil {
		t.Fatal("expected unsupported provider to fail")
	}
}

func TestAuthProviderServiceUpdate_ReplacesSecretAndSupportsGoogle(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewAuthProvider(injector)
	if err != nil {
		t.Fatalf("NewAuthProvider returned error: %v", err)
	}

	provider := &model.AuthProvider{
		ProviderKey:  "google",
		DisplayName:  "Google",
		ClientID:     "old-id",
		ClientSecret: "old-secret",
		Scopes:       "openid",
		CallbackURL:  "https://old.example.com/callback",
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	updated, err := svc.Update("google", map[string]any{
		"clientSecret": "new-secret",
		"scopes":       "openid profile email",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.ClientSecret != "new-secret" || updated.Scopes != "openid profile email" {
		t.Fatalf("unexpected updated provider: %+v", updated)
	}

	oauthProvider, err := svc.BuildOAuthProvider(updated)
	if err != nil || oauthProvider == nil {
		t.Fatalf("expected google OAuth provider, got %v %v", oauthProvider, err)
	}
}

func TestAuthProviderServiceToggle_ReturnsErrorForMissingProvider(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewAuthProvider(injector)
	if err != nil {
		t.Fatalf("NewAuthProvider returned error: %v", err)
	}

	if _, err := svc.Toggle("missing"); err == nil {
		t.Fatal("expected missing provider toggle to fail")
	}
}

func TestSessionServiceListAndKick(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewSession(injector)
	if err != nil {
		t.Fatalf("NewSession returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)

	active := &model.RefreshToken{
		Token:          "active",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		IPAddress:      "10.0.0.1",
		UserAgent:      "Chrome",
		LastSeenAt:     time.Now(),
		AccessTokenJTI: "jti-1",
	}
	other := &model.RefreshToken{
		Token:          "other",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		IPAddress:      "10.0.0.2",
		UserAgent:      "Safari",
		LastSeenAt:     time.Now().Add(-time.Minute),
		AccessTokenJTI: "jti-2",
	}
	if err := db.Create(active).Error; err != nil {
		t.Fatalf("seed active token: %v", err)
	}
	if err := db.Create(other).Error; err != nil {
		t.Fatalf("seed other token: %v", err)
	}

	result, err := svc.ListSessions(1, 20, "jti-1")
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two sessions, got %+v", result)
	}
	if !result.Items[0].IsCurrent && !result.Items[1].IsCurrent {
		t.Fatalf("expected one current session, got %+v", result.Items)
	}

	if err := svc.KickSession(active.ID, "jti-1"); !errors.Is(err, ErrCannotKickSelf) {
		t.Fatalf("expected ErrCannotKickSelf, got %v", err)
	}
	if err := svc.KickSession(9999, "jti-1"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
	if err := svc.KickSession(other.ID, "jti-1"); err != nil {
		t.Fatalf("KickSession returned error: %v", err)
	}

	repo := do.MustInvoke[*repository.RefreshTokenRepo](injector)
	revoked, err := repo.FindByID(other.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if !revoked.Revoked {
		t.Fatalf("expected kicked session to be revoked, got %+v", revoked)
	}
	blacklist := do.MustInvoke[*token.TokenBlacklist](injector)
	if !blacklist.IsBlocked("jti-2") {
		t.Fatal("expected kicked session JTI to be blacklisted")
	}
	if err := svc.KickSession(other.ID, "jti-1"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected revoked session to be treated as missing, got %v", err)
	}
}

func TestTwoFactorServiceLifecycle(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewTwoFactor(injector)
	if err != nil {
		t.Fatalf("NewTwoFactor returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)

	setup, err := svc.Setup(user.ID)
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if setup.Secret == "" || setup.QRUri == "" {
		t.Fatalf("expected setup secret and QR URI, got %+v", setup)
	}
	if _, err := svc.Confirm(user.ID, "000000"); !errors.Is(err, ErrTwoFactorInvalidCode) {
		t.Fatalf("expected ErrTwoFactorInvalidCode, got %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode returned error: %v", err)
	}
	confirmed, err := svc.Confirm(user.ID, code)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if len(confirmed.BackupCodes) != 8 {
		t.Fatalf("expected eight backup codes, got %+v", confirmed.BackupCodes)
	}

	ok, err := svc.Verify(user.ID, code)
	if err != nil || !ok {
		t.Fatalf("expected TOTP verify to succeed, got ok=%v err=%v", ok, err)
	}
	backup := confirmed.BackupCodes[0]
	ok, err = svc.Verify(user.ID, backup)
	if err != nil || !ok {
		t.Fatalf("expected backup code verify to succeed, got ok=%v err=%v", ok, err)
	}

	repo := do.MustInvoke[*repository.TwoFactorSecretRepo](injector)
	stored, err := repo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID returned error: %v", err)
	}
	var codes []string
	if err := json.Unmarshal([]byte(stored.BackupCodes), &codes); err != nil {
		t.Fatalf("backup codes should stay as JSON: %v", err)
	}
	if len(codes) != 7 {
		t.Fatalf("expected consumed backup code to be removed, got %d codes", len(codes))
	}

	if err := svc.Disable(user.ID); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if _, err := svc.Verify(user.ID, "123456"); !errors.Is(err, ErrTwoFactorNotSetup) {
		t.Fatalf("expected ErrTwoFactorNotSetup after disable, got %v", err)
	}
	if _, err := svc.Setup(user.ID); err != nil {
		t.Fatalf("expected setup to work after disable, got %v", err)
	}
}

func TestTwoFactorServiceRejectsInvalidStates(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewTwoFactor(injector)
	if err != nil {
		t.Fatalf("NewTwoFactor returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)

	if _, err := svc.Confirm(user.ID, "123456"); !errors.Is(err, ErrTwoFactorNotSetup) {
		t.Fatalf("expected ErrTwoFactorNotSetup, got %v", err)
	}
	if ok, err := svc.Verify(user.ID, "123456"); !errors.Is(err, ErrTwoFactorNotSetup) || ok {
		t.Fatalf("expected verify to fail before setup, got ok=%v err=%v", ok, err)
	}
	if err := svc.Disable(user.ID); !errors.Is(err, ErrTwoFactorNotSetup) {
		t.Fatalf("expected ErrTwoFactorNotSetup on disable, got %v", err)
	}

	user.TwoFactorEnabled = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("save enabled user: %v", err)
	}
	if _, err := svc.Setup(user.ID); !errors.Is(err, ErrTwoFactorAlreadyEnabled) {
		t.Fatalf("expected ErrTwoFactorAlreadyEnabled, got %v", err)
	}
}

func TestTwoFactorService_MissingUserAndInvalidBackupCodeBranches(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewTwoFactor(injector)
	if err != nil {
		t.Fatalf("NewTwoFactor returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)

	if _, err := svc.Setup(999999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected missing user setup error, got %v", err)
	}
	if err := svc.Disable(999999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected missing user disable error, got %v", err)
	}

	setup, err := svc.Setup(user.ID)
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	missingUserCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode returned error: %v", err)
	}
	if err := db.Delete(&model.User{}, user.ID).Error; err != nil {
		t.Fatalf("delete user before confirm: %v", err)
	}
	if _, err := svc.Confirm(user.ID, missingUserCode); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected missing user confirm error, got %v", err)
	}

	user = seedCoreUser(t, db, "alice2", role.ID)
	setup, err = svc.Setup(user.ID)
	if err != nil {
		t.Fatalf("second Setup returned error: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode returned error: %v", err)
	}
	confirmed, err := svc.Confirm(user.ID, code)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}

	ok, err := svc.Verify(user.ID, "definitely-not-a-backup-code")
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if ok {
		t.Fatal("expected invalid backup code to fail verification")
	}
	if len(confirmed.BackupCodes) == 0 {
		t.Fatal("expected backup codes to be generated")
	}
}

func TestTwoFactorServiceSetup_ReplacesPendingSecretAndDisableWithoutStoredSecret(t *testing.T) {
	db := newCoreServiceDB(t)
	injector := newCoreServiceInjector(db)
	svc, err := NewTwoFactor(injector)
	if err != nil {
		t.Fatalf("NewTwoFactor returned error: %v", err)
	}
	role := seedCoreRole(t, db, "admin")
	user := seedCoreUser(t, db, "alice", role.ID)

	pending := &model.TwoFactorSecret{UserID: user.ID, Secret: "OLDSECRET"}
	if err := db.Create(pending).Error; err != nil {
		t.Fatalf("seed pending secret: %v", err)
	}

	setup, err := svc.Setup(user.ID)
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if setup.Secret == "" || setup.Secret == "OLDSECRET" {
		t.Fatalf("expected new secret to replace pending one, got %+v", setup)
	}

	var secrets []model.TwoFactorSecret
	if err := db.Where("user_id = ?", user.ID).Find(&secrets).Error; err != nil {
		t.Fatalf("reload secrets: %v", err)
	}
	if len(secrets) != 1 || secrets[0].Secret != setup.Secret {
		t.Fatalf("expected one replaced secret, got %+v", secrets)
	}

	user.TwoFactorEnabled = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("enable user 2FA: %v", err)
	}
	if err := db.Delete(&model.TwoFactorSecret{}, "user_id = ?", user.ID).Error; err != nil {
		t.Fatalf("delete stored secret: %v", err)
	}

	if err := svc.Disable(user.ID); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	reloaded, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.TwoFactorEnabled {
		t.Fatalf("expected 2FA to be disabled, got %+v", reloaded)
	}
}
