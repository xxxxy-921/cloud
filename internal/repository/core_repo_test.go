package repository

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

func newCoreRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Role{},
		&model.RoleDeptScope{},
		&model.User{},
		&model.Menu{},
		&model.SystemConfig{},
		&model.RefreshToken{},
		&model.MessageChannel{},
		&model.IdentitySource{},
		&model.UserConnection{},
		&model.AuthProvider{},
		&model.TwoFactorSecret{},
		&model.Notification{},
		&model.NotificationRead{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func newCoreRepoInjector(db *gorm.DB) do.Injector {
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	return injector
}

func TestSysConfigRepoCRUD(t *testing.T) {
	db := newCoreRepoTestDB(t)
	repo, err := NewSysConfig(newCoreRepoInjector(db))
	if err != nil {
		t.Fatalf("NewSysConfig returned error: %v", err)
	}

	cfg := &model.SystemConfig{Key: "system.logo", Value: "logo.png", Remark: "logo"}
	if err := repo.Set(cfg); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := repo.Get("system.logo")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Value != "logo.png" {
		t.Fatalf("expected saved value, got %+v", got)
	}

	items, err := repo.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one config, got %d", len(items))
	}

	if err := repo.Delete("system.logo"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if err := repo.Delete("system.logo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deleting missing key, got %v", err)
	}
}

func TestNewAuditLogRepo_Constructs(t *testing.T) {
	db := newCoreRepoTestDB(t)
	repo, err := NewAuditLog(newCoreRepoInjector(db))
	if err != nil {
		t.Fatalf("NewAuditLog returned error: %v", err)
	}
	if repo == nil || repo.db == nil {
		t.Fatalf("expected audit log repo to be wired, got %+v", repo)
	}
}

func TestMessageChannelRepoLifecycleAndMaskConfig(t *testing.T) {
	db := newCoreRepoTestDB(t)
	repo, err := NewMessageChannel(newCoreRepoInjector(db))
	if err != nil {
		t.Fatalf("NewMessageChannel returned error: %v", err)
	}

	first := &model.MessageChannel{Name: "SMTP", Type: "email", Config: `{"password":"secret","host":"smtp.example.com"}`, Enabled: true}
	second := &model.MessageChannel{Name: "Webhook", Type: "webhook", Config: `{"url":"https://example.com"}`, Enabled: false}
	if err := repo.Create(first); err != nil {
		t.Fatalf("Create first returned error: %v", err)
	}
	if err := repo.Create(second); err != nil {
		t.Fatalf("Create second returned error: %v", err)
	}

	got, err := repo.FindByID(first.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got.Name != "SMTP" {
		t.Fatalf("expected SMTP, got %+v", got)
	}

	items, total, err := repo.List(ListParams{Keyword: "SMTP", Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one filtered result, got total=%d items=%d", total, len(items))
	}

	first.Name = "SMTP2"
	if err := repo.Update(first); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	toggled, err := repo.ToggleEnabled(first.ID)
	if err != nil {
		t.Fatalf("ToggleEnabled returned error: %v", err)
	}
	if toggled.Enabled {
		t.Fatalf("expected toggle to disable channel, got %+v", toggled)
	}

	if err := repo.Delete(second.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if err := repo.Delete(9999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}

	if masked := MaskConfig(first.Config); masked == first.Config {
		t.Fatalf("expected sensitive config to be masked, got %s", masked)
	}
	if plain := MaskConfig("not-json"); plain != "not-json" {
		t.Fatalf("expected invalid JSON to stay unchanged, got %s", plain)
	}
}

func TestIdentitySourceRepoDomainAndToggle(t *testing.T) {
	db := newCoreRepoTestDB(t)
	repo, err := NewIdentitySource(newCoreRepoInjector(db))
	if err != nil {
		t.Fatalf("NewIdentitySource returned error: %v", err)
	}

	oidc := &model.IdentitySource{
		Name:    "Corp OIDC",
		Type:    "oidc",
		Enabled: true,
		Domains: "Example.com, corp.example.com",
		Config:  `{"issuerUrl":"https://issuer.example.com","clientSecret":"secret"}`,
	}
	ldap := &model.IdentitySource{
		Name:      "Corp LDAP",
		Type:      "ldap",
		Enabled:   false,
		Domains:   "legacy.example.com",
		SortOrder: 1,
		Config:    `{"serverUrl":"ldaps://ldap.example.com","bindPassword":"secret"}`,
	}
	if err := repo.Create(oidc); err != nil {
		t.Fatalf("Create oidc returned error: %v", err)
	}
	if err := repo.Create(ldap); err != nil {
		t.Fatalf("Create ldap returned error: %v", err)
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two sources, got %d", len(list))
	}

	byID, err := repo.FindByID(oidc.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if byID.Name != "Corp OIDC" {
		t.Fatalf("unexpected source: %+v", byID)
	}

	byDomain, err := repo.FindByDomain(" corp.example.com ")
	if err != nil {
		t.Fatalf("FindByDomain returned error: %v", err)
	}
	if byDomain.ID != oidc.ID {
		t.Fatalf("expected OIDC source for domain lookup, got %+v", byDomain)
	}
	if _, err := repo.FindByDomain("missing.example.com"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found for missing domain, got %v", err)
	}

	if err := repo.CheckDomainConflict("example.com", oidc.ID); err != nil {
		t.Fatalf("expected excluded source to be ignored, got %v", err)
	}
	if err := repo.CheckDomainConflict("legacy.example.com", 0); !errors.Is(err, ErrDomainConflict) {
		t.Fatalf("expected ErrDomainConflict, got %v", err)
	}
	if err := repo.CheckDomainConflict(" , ", 0); err != nil {
		t.Fatalf("blank domains should be ignored, got %v", err)
	}

	oidc.Name = "Corp OIDC 2"
	if err := repo.Update(oidc); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	toggled, err := repo.Toggle(ldap.ID)
	if err != nil {
		t.Fatalf("Toggle returned error: %v", err)
	}
	if !toggled.Enabled {
		t.Fatalf("expected toggle to enable source, got %+v", toggled)
	}

	if err := repo.Delete(ldap.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestIdentitySourceResponseMasksSecrets(t *testing.T) {
	oidc := (&model.IdentitySource{
		Type:   "oidc",
		Config: `{"issuerUrl":"https://issuer.example.com","clientSecret":"secret"}`,
	}).ToResponse()
	if string(oidc.Config) == `{"issuerUrl":"https://issuer.example.com","clientSecret":"secret"}` {
		t.Fatalf("expected OIDC secret to be masked, got %s", oidc.Config)
	}

	ldap := (&model.IdentitySource{
		Type:   "ldap",
		Config: `{"serverUrl":"ldaps://ldap.example.com","bindPassword":"secret"}`,
	}).ToResponse()
	if string(ldap.Config) == `{"serverUrl":"ldaps://ldap.example.com","bindPassword":"secret"}` {
		t.Fatalf("expected LDAP secret to be masked, got %s", ldap.Config)
	}

	raw := (&model.IdentitySource{Type: "custom", Config: `{"raw":true}`}).ToResponse()
	if string(raw.Config) != `{"raw":true}` {
		t.Fatalf("expected unknown type config to pass through, got %s", raw.Config)
	}
}

func TestUserConnectionRepoLifecycle(t *testing.T) {
	db := newCoreRepoTestDB(t)
	repo, err := NewUserConnection(newCoreRepoInjector(db))
	if err != nil {
		t.Fatalf("NewUserConnection returned error: %v", err)
	}

	first := &model.UserConnection{UserID: 1, Provider: "github", ExternalID: "ext-1", ExternalName: "alice"}
	second := &model.UserConnection{UserID: 2, Provider: "google", ExternalID: "ext-2", ExternalName: "bob"}
	if err := repo.Create(first); err != nil {
		t.Fatalf("Create first returned error: %v", err)
	}
	if err := repo.Create(second); err != nil {
		t.Fatalf("Create second returned error: %v", err)
	}

	byUser, err := repo.FindByUserID(1)
	if err != nil || len(byUser) != 1 {
		t.Fatalf("expected one connection for user 1, got len=%d err=%v", len(byUser), err)
	}
	if empty, err := repo.FindByUserIDs(nil); err != nil || empty != nil {
		t.Fatalf("expected nil result for empty IDs, got %+v err=%v", empty, err)
	}
	byUsers, err := repo.FindByUserIDs([]uint{1, 2})
	if err != nil || len(byUsers) != 2 {
		t.Fatalf("expected two connections, got len=%d err=%v", len(byUsers), err)
	}

	byProvider, err := repo.FindByProviderAndExternalID("github", "ext-1")
	if err != nil || byProvider.UserID != 1 {
		t.Fatalf("unexpected provider lookup result: %+v err=%v", byProvider, err)
	}
	byUserProvider, err := repo.FindByUserAndProvider(2, "google")
	if err != nil || byUserProvider.ExternalID != "ext-2" {
		t.Fatalf("unexpected user/provider lookup result: %+v err=%v", byUserProvider, err)
	}

	first.ExternalName = "alice-updated"
	if err := repo.Update(first); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	count, err := repo.CountByUserID(1)
	if err != nil || count != 1 {
		t.Fatalf("expected one connection for user 1, got count=%d err=%v", count, err)
	}

	if err := repo.Delete(first.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestAuthProviderAndTwoFactorRepos(t *testing.T) {
	db := newCoreRepoTestDB(t)
	injector := newCoreRepoInjector(db)
	authRepo, err := NewAuthProvider(injector)
	if err != nil {
		t.Fatalf("NewAuthProvider returned error: %v", err)
	}
	twoFactorRepo, err := NewTwoFactorSecret(injector)
	if err != nil {
		t.Fatalf("NewTwoFactorSecret returned error: %v", err)
	}

	first := &model.AuthProvider{ProviderKey: "github", DisplayName: "GitHub", Enabled: true, SortOrder: 2}
	second := &model.AuthProvider{ProviderKey: "google", DisplayName: "Google", Enabled: false, SortOrder: 1}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("seed first auth provider: %v", err)
	}
	if err := db.Create(second).Error; err != nil {
		t.Fatalf("seed second auth provider: %v", err)
	}

	found, err := authRepo.FindByKey("github")
	if err != nil || found.DisplayName != "GitHub" {
		t.Fatalf("unexpected auth provider lookup: %+v err=%v", found, err)
	}
	enabled, err := authRepo.FindAllEnabled()
	if err != nil || len(enabled) != 1 || enabled[0].ProviderKey != "github" {
		t.Fatalf("unexpected enabled providers: %+v err=%v", enabled, err)
	}
	all, err := authRepo.FindAll()
	if err != nil || len(all) != 2 || all[0].ProviderKey != "google" {
		t.Fatalf("expected sort-order ascending providers, got %+v err=%v", all, err)
	}
	second.Enabled = true
	if err := authRepo.Update(second); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	secret := &model.TwoFactorSecret{UserID: 9, Secret: "base32", BackupCodes: `["a","b"]`}
	if err := twoFactorRepo.Create(secret); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	foundSecret, err := twoFactorRepo.FindByUserID(9)
	if err != nil || foundSecret.Secret != "base32" {
		t.Fatalf("unexpected 2FA secret lookup: %+v err=%v", foundSecret, err)
	}
	secret.BackupCodes = `["c"]`
	if err := twoFactorRepo.Update(secret); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if err := twoFactorRepo.DeleteByUserID(9); err != nil {
		t.Fatalf("DeleteByUserID returned error: %v", err)
	}
	if err := twoFactorRepo.DeleteByUserID(9); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestUserRoleMenuAndRefreshTokenRepos(t *testing.T) {
	db := newCoreRepoTestDB(t)
	injector := newCoreRepoInjector(db)
	roleRepo, _ := NewRole(injector)
	userRepo, _ := NewUser(injector)
	menuRepo, _ := NewMenu(injector)
	refreshRepo, _ := NewRefreshToken(injector)

	role := &model.Role{Name: "管理员", Code: "admin", Sort: 2, DataScope: model.DataScopeCustom}
	if err := roleRepo.Create(role); err != nil {
		t.Fatalf("Create role returned error: %v", err)
	}
	if err := roleRepo.SetCustomDeptIDs(role.ID, []uint{3, 4}); err != nil {
		t.Fatalf("SetCustomDeptIDs returned error: %v", err)
	}
	role.Name = "超级管理员"
	if err := roleRepo.Update(role); err != nil {
		t.Fatalf("Update role returned error: %v", err)
	}
	foundRole, err := roleRepo.FindByID(role.ID)
	if err != nil || foundRole.Code != "admin" {
		t.Fatalf("unexpected role lookup: %+v err=%v", foundRole, err)
	}
	byCode, err := roleRepo.FindByCode("admin")
	if err != nil || byCode.ID != role.ID {
		t.Fatalf("unexpected role by code: %+v err=%v", byCode, err)
	}
	withScope, deptIDs, err := roleRepo.FindByIDWithDeptScope(role.ID)
	if err != nil || withScope.ID != role.ID || len(deptIDs) != 2 {
		t.Fatalf("unexpected role scope result: role=%+v deptIDs=%v err=%v", withScope, deptIDs, err)
	}
	scope, customIDs, err := roleRepo.GetScopeByCode("admin")
	if err != nil || scope != model.DataScopeCustom || len(customIDs) != 2 {
		t.Fatalf("unexpected scope lookup: scope=%s ids=%v err=%v", scope, customIDs, err)
	}
	if missingScope, missingIDs, err := roleRepo.GetScopeByCode("missing"); err != nil || missingScope != model.DataScopeAll || missingIDs != nil {
		t.Fatalf("expected missing scope to fall back to all, got scope=%s ids=%v err=%v", missingScope, missingIDs, err)
	}
	listedRoles, totalRoles, err := roleRepo.List(0, 0)
	if err != nil || totalRoles != 1 || len(listedRoles) != 1 {
		t.Fatalf("unexpected role list: total=%d len=%d err=%v", totalRoles, len(listedRoles), err)
	}
	if exists, err := roleRepo.ExistsByCode("admin"); err != nil || !exists {
		t.Fatalf("expected role code to exist, got exists=%v err=%v", exists, err)
	}

	passwordHash := "hashed"
	user := &model.User{Username: "alice", Password: passwordHash, Email: "alice@example.com", RoleID: role.ID, IsActive: true}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user returned error: %v", err)
	}
	second := &model.User{Username: "bob", Email: "bob@example.com", RoleID: role.ID, IsActive: false}
	if err := userRepo.Create(second); err != nil {
		t.Fatalf("Create second user returned error: %v", err)
	}
	foundUser, err := userRepo.FindByUsername("alice")
	if err != nil || foundUser.Role.ID != role.ID {
		t.Fatalf("unexpected username lookup: %+v err=%v", foundUser, err)
	}
	if byID, err := userRepo.FindByID(user.ID); err != nil || byID.Username != "alice" {
		t.Fatalf("unexpected user ID lookup: %+v err=%v", byID, err)
	}
	if _, err := userRepo.FindByIDWithManager(user.ID); err != nil {
		t.Fatalf("FindByIDWithManager returned error: %v", err)
	}
	if _, err := userRepo.FindByEmail("alice@example.com"); err != nil {
		t.Fatalf("FindByEmail returned error: %v", err)
	}
	active := true
	result, err := userRepo.List(ListParams{Keyword: "alice", IsActive: &active, Page: 0, PageSize: 0})
	if err != nil || result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected user list result: %+v err=%v", result, err)
	}
	if exists, err := userRepo.ExistsByUsername("alice"); err != nil || !exists {
		t.Fatalf("expected username to exist, got exists=%v err=%v", exists, err)
	}
	if err := userRepo.IncrementFailedAttempts(user.ID); err != nil {
		t.Fatalf("IncrementFailedAttempts returned error: %v", err)
	}
	if err := userRepo.LockUser(user.ID, time.Minute); err != nil {
		t.Fatalf("LockUser returned error: %v", err)
	}
	attempts, err := userRepo.GetFailedAttempts(user.ID)
	if err != nil || attempts != 1 {
		t.Fatalf("unexpected failed attempts: %d err=%v", attempts, err)
	}
	if err := userRepo.UnlockUser(user.ID); err != nil {
		t.Fatalf("UnlockUser returned error: %v", err)
	}
	if err := userRepo.ResetFailedAttempts(user.ID); err != nil {
		t.Fatalf("ResetFailedAttempts returned error: %v", err)
	}
	second.Email = "bob2@example.com"
	if err := userRepo.Update(second); err != nil {
		t.Fatalf("Update user returned error: %v", err)
	}
	if count, err := roleRepo.CountUsersByRoleID(role.ID); err != nil || count != 2 {
		t.Fatalf("unexpected CountUsersByRoleID result: %d err=%v", count, err)
	}

	root := &model.Menu{Name: "系统管理", Type: model.MenuTypeDirectory, Permission: "system", Sort: 1}
	if err := menuRepo.Create(root); err != nil {
		t.Fatalf("Create root menu returned error: %v", err)
	}
	child := &model.Menu{Name: "用户管理", Type: model.MenuTypeMenu, ParentID: &root.ID, Permission: "system:user:list", Sort: 2}
	if err := menuRepo.Create(child); err != nil {
		t.Fatalf("Create child menu returned error: %v", err)
	}
	foundMenu, err := menuRepo.FindByID(child.ID)
	if err != nil || foundMenu.Permission != "system:user:list" {
		t.Fatalf("unexpected menu lookup: %+v err=%v", foundMenu, err)
	}
	allMenus, err := menuRepo.FindAll()
	if err != nil || len(allMenus) != 2 {
		t.Fatalf("unexpected FindAll result: len=%d err=%v", len(allMenus), err)
	}
	rootMenus, err := menuRepo.FindByParentID(nil)
	if err != nil || len(rootMenus) != 1 {
		t.Fatalf("unexpected root menus: len=%d err=%v", len(rootMenus), err)
	}
	childMenus, err := menuRepo.FindByParentID(&root.ID)
	if err != nil || len(childMenus) != 1 {
		t.Fatalf("unexpected child menus: len=%d err=%v", len(childMenus), err)
	}
	permMenu, err := menuRepo.FindByPermission("system:user:list")
	if err != nil || permMenu.ID != child.ID {
		t.Fatalf("unexpected FindByPermission result: %+v err=%v", permMenu, err)
	}
	tree, err := menuRepo.GetTree()
	if err != nil || len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Fatalf("unexpected menu tree: %+v err=%v", tree, err)
	}
	if hasChildren, err := menuRepo.HasChildren(root.ID); err != nil || !hasChildren {
		t.Fatalf("expected root to have children, got %v err=%v", hasChildren, err)
	}
	if err := menuRepo.UpdateSorts([]SortItem{{ID: child.ID, Sort: 1}}); err != nil {
		t.Fatalf("UpdateSorts returned error: %v", err)
	}
	if menus, err := menuRepo.FindByIDs([]uint{root.ID, child.ID}); err != nil || len(menus) != 2 {
		t.Fatalf("unexpected FindByIDs result: len=%d err=%v", len(menus), err)
	}
	if emptyMenus, err := menuRepo.FindByIDs(nil); err != nil || len(emptyMenus) != 0 {
		t.Fatalf("expected empty FindByIDs result, got len=%d err=%v", len(emptyMenus), err)
	}
	if err := menuRepo.Update(child); err != nil {
		t.Fatalf("Update menu returned error: %v", err)
	}

	rtA := &model.RefreshToken{
		Token:          "active",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		LastSeenAt:     time.Now().Add(-time.Minute),
		AccessTokenJTI: "jti-1",
	}
	rtB := &model.RefreshToken{
		Token:          "active-2",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		LastSeenAt:     time.Now(),
		AccessTokenJTI: "jti-2",
	}
	rtExpired := &model.RefreshToken{
		Token:      "expired",
		UserID:     user.ID,
		ExpiresAt:  time.Now().Add(-time.Hour),
		LastSeenAt: time.Now().Add(-2 * time.Hour),
	}
	for _, rt := range []*model.RefreshToken{rtA, rtB, rtExpired} {
		if err := refreshRepo.Create(rt); err != nil {
			t.Fatalf("Create refresh token returned error: %v", err)
		}
	}
	if _, err := refreshRepo.FindValid("active"); err != nil {
		t.Fatalf("FindValid returned error: %v", err)
	}
	if _, err := refreshRepo.FindByToken("active-2"); err != nil {
		t.Fatalf("FindByToken returned error: %v", err)
	}
	if err := refreshRepo.Revoke("active"); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}
	if err := refreshRepo.RevokeAllForUser(user.ID); err != nil {
		t.Fatalf("RevokeAllForUser returned error: %v", err)
	}
	if activeTokens, err := refreshRepo.GetActiveByUserID(user.ID); err != nil || len(activeTokens) != 0 {
		t.Fatalf("expected no active tokens after revoke-all, got len=%d err=%v", len(activeTokens), err)
	}
	rtC := &model.RefreshToken{
		Token:          "active-3",
		UserID:         user.ID,
		ExpiresAt:      time.Now().Add(time.Hour),
		LastSeenAt:     time.Now(),
		AccessTokenJTI: "jti-3",
	}
	if err := refreshRepo.Create(rtC); err != nil {
		t.Fatalf("Create active-3 returned error: %v", err)
	}
	sessions, totalSessions, err := refreshRepo.GetActiveSessions(1, 20)
	if err != nil || totalSessions != 1 || len(sessions) != 1 {
		t.Fatalf("unexpected active sessions: total=%d len=%d err=%v", totalSessions, len(sessions), err)
	}
	jtis, err := refreshRepo.GetActiveTokenJTIsByUserID(user.ID)
	if err != nil || len(jtis) != 1 || jtis[0] != "jti-3" {
		t.Fatalf("unexpected active JTIs: %v err=%v", jtis, err)
	}
	if foundRT, err := refreshRepo.FindByID(rtC.ID); err != nil || foundRT.Token != "active-3" {
		t.Fatalf("unexpected FindByID result: %+v err=%v", foundRT, err)
	}
	if err := refreshRepo.RevokeByID(rtC.ID); err != nil {
		t.Fatalf("RevokeByID returned error: %v", err)
	}
	if _, err := refreshRepo.DeleteExpiredTokens(0); err != nil {
		t.Fatalf("DeleteExpiredTokens returned error: %v", err)
	}

	if err := menuRepo.Delete(child.ID); err != nil {
		t.Fatalf("Delete child menu returned error: %v", err)
	}
	if err := userRepo.Delete(second.ID); err != nil {
		t.Fatalf("Delete second user returned error: %v", err)
	}
	if err := roleRepo.Delete(role.ID); err != nil {
		t.Fatalf("Delete role returned error: %v", err)
	}

	allScopeRole := &model.Role{Name: "访客", Code: "guest", Sort: 3, DataScope: model.DataScopeAll}
	if err := roleRepo.Create(allScopeRole); err != nil {
		t.Fatalf("Create all-scope role returned error: %v", err)
	}
	if scope, deptIDs, err := roleRepo.GetScopeByCode("guest"); err != nil || scope != model.DataScopeAll || deptIDs != nil {
		t.Fatalf("expected all-scope role to return nil dept IDs, got scope=%s deptIDs=%v err=%v", scope, deptIDs, err)
	}
}

func TestNotificationRepoLifecycle(t *testing.T) {
	db := newCoreRepoTestDB(t)
	injector := newCoreRepoInjector(db)
	repo, err := NewNotification(injector)
	if err != nil {
		t.Fatalf("NewNotification returned error: %v", err)
	}
	user := &model.User{Username: "alice", RoleID: 1}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	creatorID := user.ID
	targetID := user.ID
	broadcast := &model.Notification{
		Type:       model.NotificationTypeAnnouncement,
		Source:     "system",
		Title:      "Broadcast",
		Content:    "hello",
		TargetType: model.NotificationTargetAll,
		CreatedBy:  &creatorID,
	}
	targeted := &model.Notification{
		Type:       "notice",
		Source:     "system",
		Title:      "Targeted",
		Content:    "private",
		TargetType: model.NotificationTargetUser,
		TargetID:   &targetID,
		CreatedBy:  &creatorID,
	}
	if err := repo.Create(broadcast); err != nil {
		t.Fatalf("Create broadcast returned error: %v", err)
	}
	if err := repo.Create(targeted); err != nil {
		t.Fatalf("Create targeted returned error: %v", err)
	}
	found, err := repo.FindByID(broadcast.ID)
	if err != nil || found.Title != "Broadcast" {
		t.Fatalf("unexpected FindByID result: %+v err=%v", found, err)
	}
	broadcast.Content = "updated"
	if err := repo.Update(broadcast); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	items, total, err := repo.ListForUser(user.ID, ListParams{Keyword: "a", Page: 0, PageSize: 0})
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("unexpected ListForUser result: total=%d len=%d err=%v", total, len(items), err)
	}
	if unread, err := repo.CountUnreadForUser(user.ID); err != nil || unread != 2 {
		t.Fatalf("unexpected unread count: %d err=%v", unread, err)
	}
	if err := repo.MarkAsRead(targeted.ID, user.ID); err != nil {
		t.Fatalf("MarkAsRead returned error: %v", err)
	}
	if err := repo.MarkAsRead(targeted.ID, user.ID); err != nil {
		t.Fatalf("MarkAsRead idempotent call returned error: %v", err)
	}
	if unread, err := repo.CountUnreadForUser(user.ID); err != nil || unread != 1 {
		t.Fatalf("unexpected unread count after mark read: %d err=%v", unread, err)
	}
	if err := repo.MarkAllAsRead(user.ID); err != nil {
		t.Fatalf("MarkAllAsRead returned error: %v", err)
	}
	if unread, err := repo.CountUnreadForUser(user.ID); err != nil || unread != 0 {
		t.Fatalf("expected all notifications read, got unread=%d err=%v", unread, err)
	}
	announcements, totalAnnouncements, err := repo.ListAnnouncements(ListParams{Keyword: "Broad", Page: 1, PageSize: 20})
	if err != nil || totalAnnouncements != 1 || len(announcements) != 1 {
		t.Fatalf("unexpected ListAnnouncements result: total=%d len=%d err=%v", totalAnnouncements, len(announcements), err)
	}
	if announcements[0].CreatorUsername != "alice" {
		t.Fatalf("expected creator username alice, got %+v", announcements[0])
	}
	if err := repo.Delete(targeted.ID); err != nil {
		t.Fatalf("Delete targeted returned error: %v", err)
	}
	if err := repo.Delete(9999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}
