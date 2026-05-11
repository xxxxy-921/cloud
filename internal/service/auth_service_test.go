package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/identity"
	"metis/internal/pkg/token"
	"metis/internal/repository"
)

func newTestDBForAuthService(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.SystemConfig{},
		&model.RefreshToken{},
		&model.UserConnection{},
		&model.IdentitySource{},
		&model.Menu{},
		&model.RoleDeptScope{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func newAuthServiceForTest(t *testing.T, db *gorm.DB) (*AuthService, do.Injector) {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	enforcer, err := casbinpkg.NewEnforcerWithDB(db)
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	do.ProvideValue(injector, enforcer)
	do.ProvideValue(injector, token.NewBlacklist())
	do.ProvideValue(injector, []byte("test-jwt-secret"))

	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, repository.NewUserConnection)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewRole)
	do.Provide(injector, repository.NewMenu)
	do.Provide(injector, NewCasbin)
	do.Provide(injector, NewMenu)
	do.Provide(injector, NewSettings)
	do.Provide(injector, NewCaptcha)
	do.Provide(injector, NewAuth)

	return do.MustInvoke[*AuthService](injector), injector
}

func TestNewAuth_WithIdentityService(t *testing.T) {
	db := newTestDBForAuthService(t)
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	enforcer, err := casbinpkg.NewEnforcerWithDB(db)
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	do.ProvideValue(injector, enforcer)
	do.ProvideValue(injector, token.NewBlacklist())
	do.ProvideValue(injector, []byte("test-jwt-secret"))

	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, repository.NewUserConnection)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewRole)
	do.Provide(injector, repository.NewMenu)
	do.Provide(injector, repository.NewIdentitySource)
	do.Provide(injector, NewCasbin)
	do.Provide(injector, NewMenu)
	do.Provide(injector, NewSettings)
	do.Provide(injector, NewCaptcha)
	do.Provide(injector, NewIdentitySource)

	svc, err := NewAuth(injector)
	if err != nil {
		t.Fatalf("NewAuth returned error: %v", err)
	}
	if svc.identitySvc == nil {
		t.Fatal("expected NewAuth to wire optional identity service when available")
	}
}

func seedAuthRole(t *testing.T, db *gorm.DB, code string) *model.Role {
	t.Helper()
	role := &model.Role{Name: code, Code: code}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	return role
}

func seedAuthUser(t *testing.T, db *gorm.DB, username string, role *model.Role, password string) *model.User {
	t.Helper()
	var hash string
	if password != "" {
		var err error
		hash, err = token.HashPassword(password)
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
	}
	now := time.Now()
	user := &model.User{
		Username:          username,
		Password:          hash,
		Email:             username + "@example.com",
		RoleID:            role.ID,
		IsActive:          true,
		PasswordChangedAt: &now,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func TestAuthServiceProvisionExternalUser(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	existing := seedAuthUser(t, db, "alice", role, "Password123!")
	conn := &model.UserConnection{
		UserID:        existing.ID,
		Provider:      "github",
		ExternalID:    "42",
		ExternalName:  "Old",
		ExternalEmail: "old@example.com",
		AvatarURL:     "old.png",
	}
	if err := db.Create(conn).Error; err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	user, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:    "github",
		ExternalID:  "42",
		Email:       "alice@example.com",
		DisplayName: "Alice Updated",
		AvatarURL:   "new.png",
	})
	if err != nil {
		t.Fatalf("ProvisionExternalUser returned error: %v", err)
	}
	if user.ID != existing.ID {
		t.Fatalf("expected existing user, got %+v", user)
	}
	var refreshed model.UserConnection
	if err := db.First(&refreshed, conn.ID).Error; err != nil {
		t.Fatalf("reload connection: %v", err)
	}
	if refreshed.ExternalName != "Alice Updated" || refreshed.AvatarURL != "new.png" {
		t.Fatalf("expected connection metadata refresh, got %+v", refreshed)
	}

	linked, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:         "google",
		ExternalID:       "g-1",
		Email:            existing.Email,
		DisplayName:      "Alice",
		ConflictStrategy: "link",
	})
	if err != nil {
		t.Fatalf("expected link strategy to succeed, got %v", err)
	}
	if linked.ID != existing.ID {
		t.Fatalf("expected linked existing user, got %+v", linked)
	}

	if _, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:   "oidc_1",
		ExternalID: "sub-1",
		Email:      existing.Email,
	}); !errors.Is(err, ErrEmailConflict) {
		t.Fatalf("expected ErrEmailConflict, got %v", err)
	}

	created, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:          "ldap_1",
		ExternalID:        "uid-1",
		Email:             "new@example.com",
		DisplayName:       "New User",
		PreferredUsername: "alice",
	})
	if err != nil {
		t.Fatalf("expected new external user to be created, got %v", err)
	}
	if created.Username != "ldap_1_uid-1" {
		t.Fatalf("expected username fallback, got %+v", created)
	}

	disabled := seedAuthUser(t, db, "disabled", role, "Password123!")
	disabled.IsActive = false
	if err := db.Save(disabled).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	disabledConn := &model.UserConnection{
		UserID:     disabled.ID,
		Provider:   "github",
		ExternalID: "disabled-ext",
	}
	if err := db.Create(disabledConn).Error; err != nil {
		t.Fatalf("seed disabled connection: %v", err)
	}
	if _, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:   "github",
		ExternalID: "disabled-ext",
		Email:      "disabled@example.com",
	}); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("expected ErrAccountDisabled from existing disabled connection, got %v", err)
	}

	implicitRoleUser, err := svc.ProvisionExternalUser(ExternalUserParams{
		Provider:   "oidc_9",
		ExternalID: "sub-9",
		Email:      "implicit-role@example.com",
	})
	if err != nil {
		t.Fatalf("expected implicit default role create, got %v", err)
	}
	if implicitRoleUser.RoleID != role.ID {
		t.Fatalf("expected fallback user role id %d, got %d", role.ID, implicitRoleUser.RoleID)
	}
}

func TestAuthServiceLoginAndTokenFlows(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, injector := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")
	if err := db.Create(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "1"}).Error; err != nil {
		t.Fatalf("seed session limit config: %v", err)
	}
	if err := db.Create(&model.Menu{Name: "Users", Type: model.MenuTypeMenu, Permission: "system:user:list"}).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	casbinSvc := do.MustInvoke[*CasbinService](injector)
	if err := casbinSvc.SetPoliciesForRole(role.Code, [][]string{{role.Code, "system:user:list", "GET"}}); err != nil {
		t.Fatalf("seed casbin policy: %v", err)
	}

	pair, err := svc.Login("alice", "Password123!", "", "", "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || len(pair.Permissions) != 1 {
		t.Fatalf("unexpected token pair: %+v", pair)
	}

	refreshed, err := svc.RefreshTokens(pair.RefreshToken, "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("RefreshTokens returned error: %v", err)
	}
	if refreshed.RefreshToken == pair.RefreshToken {
		t.Fatalf("expected rotated refresh token, got %+v", refreshed)
	}
	if _, err := svc.RefreshTokens(pair.RefreshToken, "127.0.0.1", "Chrome"); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse on reused token, got %v", err)
	}

	if err := svc.Logout(refreshed.RefreshToken); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if _, err := svc.RefreshTokens("missing", "127.0.0.1", "Chrome"); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}

	expired := &model.RefreshToken{Token: "expired", UserID: user.ID, ExpiresAt: time.Now().Add(-time.Minute)}
	if err := db.Create(expired).Error; err != nil {
		t.Fatalf("seed expired token: %v", err)
	}
	if _, err := svc.RefreshTokens("expired", "127.0.0.1", "Chrome"); !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}

	disabledToken := &model.RefreshToken{Token: "disabled", UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(disabledToken).Error; err != nil {
		t.Fatalf("seed disabled token: %v", err)
	}
	user.IsActive = false
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, err := svc.RefreshTokens("disabled", "127.0.0.1", "Chrome"); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("expected ErrAccountDisabled for disabled user refresh, got %v", err)
	}
	user.IsActive = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("re-enable user: %v", err)
	}

	got, err := svc.GetCurrentUser(user.ID)
	if err != nil || got.ID != user.ID {
		t.Fatalf("unexpected GetCurrentUser result: %+v err=%v", got, err)
	}

	conns, err := svc.GetUserConnections(user.ID)
	if err != nil || len(conns) != 0 {
		t.Fatalf("expected no connections, got len=%d err=%v", len(conns), err)
	}
}

func TestAuthServiceChangePasswordAndRegister(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	active := &model.RefreshToken{
		Token:          "active",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		AccessTokenJTI: "jti-1",
	}
	if err := db.Create(active).Error; err != nil {
		t.Fatalf("seed active token: %v", err)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.password_min_length", Value: "8"}).Error; err != nil {
		t.Fatalf("seed password config: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration config: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.default_role_code", Value: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed default role config: %v", err)
	}

	if err := svc.ChangePassword(user.ID, "wrong", "NewPassword123!"); !errors.Is(err, ErrOldPasswordWrong) {
		t.Fatalf("expected ErrOldPasswordWrong, got %v", err)
	}
	if err := svc.ChangePassword(user.ID, "Password123!", "NewPassword123!"); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}

	refreshedUser, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !token.CheckPassword(refreshedUser.Password, "NewPassword123!") {
		t.Fatalf("expected password to be updated, got %+v", refreshedUser)
	}

	blacklist := svc.blacklist
	if !blacklist.IsBlocked("jti-1") {
		t.Fatal("expected old access token to be blacklisted")
	}

	pair, err := svc.Register("bob", "Password123!", "bob@example.com", "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected registration token pair, got %+v", pair)
	}

	if _, err := svc.Register("bob", "Password123!", "bob2@example.com", "127.0.0.1", "Chrome"); !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected ErrUsernameExists, got %v", err)
	}
	if !svc.IsRegistrationOpen() {
		t.Fatal("expected registration to be open")
	}

	if err := db.Save(&model.SystemConfig{Key: "security.default_role_code", Value: ""}).Error; err != nil {
		t.Fatalf("clear default role config: %v", err)
	}
	pair, err = svc.Register("carol", "Password123!", "carol@example.com", "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("Register with empty default role code returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected registration token pair for fallback default role, got %+v", pair)
	}
}

func TestAuthServiceChangePassword_ErrorBranches(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Save(&model.SystemConfig{Key: "security.password_min_length", Value: "12"}).Error; err != nil {
		t.Fatalf("seed password config: %v", err)
	}
	if err := svc.ChangePassword(user.ID, "Password123!", "short"); !errors.Is(err, ErrPasswordViolation) {
		t.Fatalf("expected ErrPasswordViolation, got %v", err)
	}
	if err := svc.ChangePassword(999999, "Password123!", "LongEnough123!"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestAuthServiceLoginGuardsAndTwoFactor(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	user.IsActive = false
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, err := svc.Login("alice", "Password123!", "", "", "", ""); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("expected ErrAccountDisabled, got %v", err)
	}

	user.IsActive = true
	locked := time.Now().Add(time.Hour)
	user.LockedUntil = &locked
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("lock user: %v", err)
	}
	if _, err := svc.Login("alice", "Password123!", "", "", "", ""); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}

	user.LockedUntil = nil
	user.TwoFactorEnabled = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("enable 2fa user: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	pair, err := svc.Login("alice", "Password123!", "", "", "", "")
	if err != nil {
		t.Fatalf("expected 2FA login to return token, got %v", err)
	}
	if !pair.NeedsTwoFactor || pair.TwoFactorToken == "" {
		t.Fatalf("expected 2FA challenge, got %+v", pair)
	}

	user.TwoFactorEnabled = false
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("disable 2fa user: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.require_two_factor", Value: "true"}).Error; err != nil {
		t.Fatalf("seed require 2fa config: %v", err)
	}
	pair, err = svc.Login("alice", "Password123!", "", "", "", "")
	if err != nil {
		t.Fatalf("expected login with forced setup to succeed, got %v", err)
	}
	if !pair.RequireTwoFactorSetup {
		t.Fatalf("expected RequireTwoFactorSetup, got %+v", pair)
	}

	user.IsActive = false
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("disable user again: %v", err)
	}
	if _, err := svc.GenerateTokenPairByID(user.ID, "", ""); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("expected GenerateTokenPairByID to reject disabled user, got %v", err)
	}
}

func TestAuthServiceExternalAuthAndOAuthFlows(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	if err := db.Create(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "2"}).Error; err != nil {
		t.Fatalf("seed session limit config: %v", err)
	}

	identitySvc := newIdentitySourceServiceForTest(t, db)
	src := seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost"}`, "", true)
	src.DefaultRoleID = role.ID
	src.ConflictStrategy = "link"
	if err := db.Save(src).Error; err != nil {
		t.Fatalf("save identity source: %v", err)
	}
	identitySvc.LDAPAuthFn = func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error) {
		return &identity.LDAPAuthResult{
			DN:          "cn=ldap,dc=example,dc=com",
			Username:    "ldap-user",
			Email:       "ldap@example.com",
			DisplayName: "LDAP User",
		}, nil
	}
	svc.identitySvc = identitySvc

	pair, err := svc.Login("ldap-user", "Password123!", "", "", "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("external Login returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair from external auth, got %+v", pair)
	}

	var externalUser model.User
	if err := db.Where("email = ?", "ldap@example.com").First(&externalUser).Error; err != nil {
		t.Fatalf("expected provisioned external user: %v", err)
	}

	oauthPair, err := svc.OAuthLogin("github", "ext-1", "GH User", "gh@example.com", "avatar.png", "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("OAuthLogin returned error: %v", err)
	}
	if oauthPair.AccessToken == "" || oauthPair.RefreshToken == "" {
		t.Fatalf("expected oauth token pair, got %+v", oauthPair)
	}
}

func TestAuthServiceHandleFailedLoginAndBlacklistUserTokens(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Create(&model.SystemConfig{Key: "security.login_max_attempts", Value: "2"}).Error; err != nil {
		t.Fatalf("seed login max attempts: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.login_lockout_minutes", Value: "15"}).Error; err != nil {
		t.Fatalf("seed lockout minutes: %v", err)
	}

	svc.handleFailedLogin(user.ID)
	svc.handleFailedLogin(user.ID)

	lockedUser, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user after failed login: %v", err)
	}
	if lockedUser.FailedLoginAttempts < 2 {
		t.Fatalf("expected failed attempts >= 2, got %d", lockedUser.FailedLoginAttempts)
	}
	if lockedUser.LockedUntil == nil {
		t.Fatal("expected user to be locked after threshold")
	}

	active1 := &model.RefreshToken{
		Token:          "active-1",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		AccessTokenJTI: "jti-a",
	}
	active2 := &model.RefreshToken{
		Token:          "active-2",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		AccessTokenJTI: "jti-b",
	}
	if err := db.Create(active1).Error; err != nil {
		t.Fatalf("seed active1: %v", err)
	}
	if err := db.Create(active2).Error; err != nil {
		t.Fatalf("seed active2: %v", err)
	}

	svc.BlacklistUserTokens(user.ID)
	if !svc.blacklist.IsBlocked("jti-a") || !svc.blacklist.IsBlocked("jti-b") {
		t.Fatal("expected active token JTIs to be blacklisted")
	}
	if svc.blacklist.IsBlocked("") {
		t.Fatal("expected blank JTI to remain ignored")
	}

	expired := &model.RefreshToken{
		Token:          "expired-jti",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(-time.Minute),
		AccessTokenJTI: "jti-expired",
	}
	blank := &model.RefreshToken{
		Token:     "blank-jti",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(expired).Error; err != nil {
		t.Fatalf("seed expired jti token: %v", err)
	}
	if err := db.Create(blank).Error; err != nil {
		t.Fatalf("seed blank jti token: %v", err)
	}
	svc.BlacklistUserTokens(user.ID)
	if svc.blacklist.IsBlocked("jti-expired") {
		t.Fatal("expected expired token JTI to stay unblocked")
	}
}

func TestAuthServiceGenerateTokenPairByID_EnforcesConcurrentLimit(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Create(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "1"}).Error; err != nil {
		t.Fatalf("seed session limit config: %v", err)
	}

	oldest := &model.RefreshToken{
		Token:          "oldest",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		LastSeenAt:     time.Now().Add(-2 * time.Hour),
		AccessTokenJTI: "jti-oldest",
	}
	newer := &model.RefreshToken{
		Token:          "newer",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		LastSeenAt:     time.Now().Add(-1 * time.Hour),
		AccessTokenJTI: "jti-newer",
	}
	if err := db.Create(oldest).Error; err != nil {
		t.Fatalf("seed oldest token: %v", err)
	}
	if err := db.Create(newer).Error; err != nil {
		t.Fatalf("seed newer token: %v", err)
	}

	pair, err := svc.GenerateTokenPairByID(user.ID, "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("GenerateTokenPairByID: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair, got %+v", pair)
	}

	oldestReloaded, err := svc.refreshTokenRepo.FindByID(oldest.ID)
	if err != nil {
		t.Fatalf("reload oldest token: %v", err)
	}
	if !oldestReloaded.Revoked {
		t.Fatal("expected oldest token to be revoked by concurrent limit")
	}
	if !svc.blacklist.IsBlocked("jti-oldest") {
		t.Fatal("expected oldest access token JTI to be blacklisted")
	}
}

func TestAuthServiceGenerateTokenPair_UsesConfiguredSessionTimeout(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Create(&model.SystemConfig{Key: "security.session_timeout_minutes", Value: "30"}).Error; err != nil {
		t.Fatalf("seed session timeout config: %v", err)
	}

	before := time.Now()
	pair, err := svc.GenerateTokenPair(user, "127.0.0.1", "Chrome")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair, got %+v", pair)
	}

	rt, err := svc.refreshTokenRepo.FindByToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("load refresh token: %v", err)
	}
	minExpected := before.Add(29 * time.Minute)
	maxExpected := before.Add(31 * time.Minute)
	if rt.ExpiresAt.Before(minExpected) || rt.ExpiresAt.After(maxExpected) {
		t.Fatalf("expected refresh token expiry near 30 minutes, got %v", rt.ExpiresAt)
	}
	if rt.IPAddress != "127.0.0.1" || rt.UserAgent != "Chrome" {
		t.Fatalf("expected session metadata persisted, got %+v", rt)
	}
}

func TestAuthServiceGenerateTokenPairAndRefresh_InternalErrors(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")
	if err := db.Create(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	tokenRow := &model.RefreshToken{
		Token:     "refresh-internal",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(tokenRow).Error; err != nil {
		t.Fatalf("seed refresh token: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	if _, err := svc.Login("alice", "Password123!", "", "", "", ""); err == nil {
		t.Fatal("expected Login to return db error after close")
	}
	if _, err := svc.RefreshTokens("refresh-internal", "", ""); err == nil {
		t.Fatal("expected RefreshTokens to return db error after close")
	}
	if _, err := svc.GenerateTokenPair(user, "", ""); err == nil {
		t.Fatal("expected GenerateTokenPair to fail when refresh token create fails")
	}
}

func TestAuthServiceRefreshTokens_UserMissing(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)

	tokenRow := &model.RefreshToken{
		Token:     "orphan-refresh",
		UserID:    999999,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(tokenRow).Error; err != nil {
		t.Fatalf("seed orphan refresh token: %v", err)
	}

	if _, err := svc.RefreshTokens("orphan-refresh", "", ""); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound for orphan refresh token, got %v", err)
	}
}

func TestAuthServiceHandleFailedLoginAndBlacklistUserTokens_DBErrors(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	svc.handleFailedLogin(user.ID)
	svc.BlacklistUserTokens(user.ID)
}

func TestAuthServiceLogin_InvalidPasswordTracksFailures(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Create(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.login_max_attempts", Value: "2"}).Error; err != nil {
		t.Fatalf("seed login max attempts: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.login_lockout_minutes", Value: "15"}).Error; err != nil {
		t.Fatalf("seed login lockout minutes: %v", err)
	}

	if _, err := svc.Login("alice", "wrong-password", "", "", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Login("alice", "wrong-password", "", "", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on second failure, got %v", err)
	}

	reloaded, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts < 2 {
		t.Fatalf("expected failed attempts >= 2, got %d", reloaded.FailedLoginAttempts)
	}
	if reloaded.LockedUntil == nil {
		t.Fatal("expected user to be locked after repeated failures")
	}
}

func TestAuthServiceLogin_SuccessResetsFailedAttempts(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Save(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha config: %v", err)
	}
	user.FailedLoginAttempts = 3
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("seed failed attempts: %v", err)
	}

	pair, err := svc.Login("alice", "Password123!", "", "", "", "")
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair, got %+v", pair)
	}

	reloaded, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 0 {
		t.Fatalf("expected failed attempts reset, got %d", reloaded.FailedLoginAttempts)
	}
}

func TestAuthServiceLogin_RejectsCaptchaForcedSSOAndOAuthOnlyUsers(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Save(&model.SystemConfig{Key: "security.captcha_provider", Value: "image"}).Error; err != nil {
		t.Fatalf("seed captcha provider: %v", err)
	}
	if _, err := svc.Login("alice", "Password123!", "", "", "", ""); !errors.Is(err, ErrCaptchaRequired) {
		t.Fatalf("expected ErrCaptchaRequired, got %v", err)
	}

	captcha, err := svc.captchaSvc.Generate()
	if err != nil {
		t.Fatalf("Generate captcha returned error: %v", err)
	}
	answer := svc.captchaSvc.store.Get(captcha.ID, false)
	if _, err := svc.Login("alice", "Password123!", captcha.ID, "wrong", "", ""); !errors.Is(err, ErrCaptchaInvalid) {
		t.Fatalf("expected ErrCaptchaInvalid, got %v", err)
	}

	captcha, err = svc.captchaSvc.Generate()
	if err != nil {
		t.Fatalf("Generate second captcha returned error: %v", err)
	}
	answer = svc.captchaSvc.store.Get(captcha.ID, false)

	identitySvc := newIdentitySourceServiceForTest(t, db)
	source := seedIdentitySourceService(t, db, "Okta", "oidc", `{"issuerUrl":"https://example.com"}`, "example.com", true)
	source.ForceSso = true
	if err := db.Save(source).Error; err != nil {
		t.Fatalf("enable force sso: %v", err)
	}
	svc.identitySvc = identitySvc
	if _, err := svc.Login("alice", "Password123!", captcha.ID, answer, "", ""); !errors.Is(err, ErrForcedSSO) {
		t.Fatalf("expected ErrForcedSSO, got %v", err)
	}

	user.Password = ""
	user.Email = "alice@other.com"
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("update oauth-only user: %v", err)
	}
	captcha, err = svc.captchaSvc.Generate()
	if err != nil {
		t.Fatalf("Generate third captcha returned error: %v", err)
	}
	answer = svc.captchaSvc.store.Get(captcha.ID, false)
	if _, err := svc.Login("alice", "Password123!", captcha.ID, answer, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for oauth-only user, got %v", err)
	}
}

func TestAuthServiceLoginAndOAuthLogin_ExternalFailures(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	if err := db.Save(&model.SystemConfig{Key: "security.captcha_provider", Value: "none"}).Error; err != nil {
		t.Fatalf("seed captcha provider: %v", err)
	}

	identitySvc := newIdentitySourceServiceForTest(t, db)
	src := seedIdentitySourceService(t, db, "AD", "ldap", `{"serverUrl":"ldap://localhost"}`, "", true)
	src.DefaultRoleID = role.ID
	if err := db.Save(src).Error; err != nil {
		t.Fatalf("save identity source: %v", err)
	}
	svc.identitySvc = identitySvc

	if _, err := svc.Login("missing", "bad-password", "", "", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials when external auth fails, got %v", err)
	}

	disabled := seedAuthUser(t, db, "disabled", role, "Password123!")
	disabled.IsActive = false
	if err := db.Save(disabled).Error; err != nil {
		t.Fatalf("disable local user: %v", err)
	}
	if err := db.Create(&model.UserConnection{
		UserID:     disabled.ID,
		Provider:   "github",
		ExternalID: "ext-disabled",
	}).Error; err != nil {
		t.Fatalf("seed connection: %v", err)
	}
	if _, err := svc.OAuthLogin("github", "ext-disabled", "Disabled", "disabled@example.com", "", "", ""); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("expected ErrAccountDisabled from OAuthLogin, got %v", err)
	}

	conflictUser := seedAuthUser(t, db, "taken", role, "Password123!")
	identitySvc.LDAPAuthFn = func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error) {
		return &identity.LDAPAuthResult{
			DN:          "cn=taken,dc=example,dc=com",
			Username:    "taken-ldap",
			Email:       conflictUser.Email,
			DisplayName: "Taken",
		}, nil
	}
	if _, err := svc.Login("missing", "Password123!", "", "", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on external conflict, got %v", err)
	}
}

func TestAuthServiceRegister_RejectsClosedPolicyViolationAndMissingDefaultRole(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)

	if _, err := svc.Register("bob", "Password123!", "bob@example.com", "", ""); !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("expected ErrRegistrationClosed, got %v", err)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration config: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.password_min_length", Value: "12"}).Error; err != nil {
		t.Fatalf("seed password config: %v", err)
	}
	if _, err := svc.Register("bob", "short", "bob@example.com", "", ""); !errors.Is(err, ErrPasswordViolation) {
		t.Fatalf("expected ErrPasswordViolation, got %v", err)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.default_role_code", Value: "missing"}).Error; err != nil {
		t.Fatalf("seed default role config: %v", err)
	}
	if _, err := svc.Register("bob", "LongEnough123!", "bob@example.com", "", ""); !errors.Is(err, ErrDefaultRoleNotFound) {
		t.Fatalf("expected ErrDefaultRoleNotFound, got %v", err)
	}
}

func TestAuthServiceGetMaxConcurrentSessions_FallsBackForInvalidValues(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)

	if got := svc.getMaxConcurrentSessions(); got != 5 {
		t.Fatalf("expected default 5 when config missing, got %d", got)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "not-a-number"}).Error; err != nil {
		t.Fatalf("seed invalid config: %v", err)
	}
	if got := svc.getMaxConcurrentSessions(); got != 5 {
		t.Fatalf("expected default 5 for invalid config, got %d", got)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "-1"}).Error; err != nil {
		t.Fatalf("seed negative config: %v", err)
	}
	if got := svc.getMaxConcurrentSessions(); got != 5 {
		t.Fatalf("expected default 5 for negative config, got %d", got)
	}

	if err := db.Save(&model.SystemConfig{Key: "security.max_concurrent_sessions", Value: "0"}).Error; err != nil {
		t.Fatalf("seed zero config: %v", err)
	}
	if got := svc.getMaxConcurrentSessions(); got != 0 {
		t.Fatalf("expected zero to disable limit, got %d", got)
	}
}

func TestAuthServiceHandleFailedLogin_NoLockoutWhenDisabled(t *testing.T) {
	db := newTestDBForAuthService(t)
	svc, _ := newAuthServiceForTest(t, db)
	role := seedAuthRole(t, db, model.RoleUser)
	user := seedAuthUser(t, db, "alice", role, "Password123!")

	if err := db.Save(&model.SystemConfig{Key: "security.login_max_attempts", Value: "0"}).Error; err != nil {
		t.Fatalf("seed disabled login max attempts: %v", err)
	}
	if err := db.Save(&model.SystemConfig{Key: "security.login_lockout_minutes", Value: "15"}).Error; err != nil {
		t.Fatalf("seed lockout minutes: %v", err)
	}

	svc.handleFailedLogin(user.ID)

	reloaded, err := svc.userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 1 {
		t.Fatalf("expected failed attempts incremented, got %d", reloaded.FailedLoginAttempts)
	}
	if reloaded.LockedUntil != nil {
		t.Fatalf("expected no lockout when disabled, got %v", reloaded.LockedUntil)
	}
}
