package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app"
	"metis/internal/config"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/token"
	"metis/internal/repository"
	"metis/internal/scheduler"
	"metis/internal/seed"
	"metis/internal/service"
)

var installTestOrgAppOnce sync.Once

type installTestOrgApp struct{}

func (a *installTestOrgApp) Name() string { return "install-test-org" }
func (a *installTestOrgApp) Models() []any {
	return []any{&testDepartment{}, &testPosition{}, &testUserPosition{}}
}
func (a *installTestOrgApp) Seed(db *gorm.DB, _ *casbin.Enforcer, _ bool) error {
	for _, code := range []string{"it", "headquarters"} {
		if err := db.Table("departments").Where("code = ?", code).FirstOrCreate(&testDepartment{Code: code}).Error; err != nil {
			return err
		}
	}
	for _, code := range []string{"it_admin", "db_admin", "network_admin", "security_admin", "ops_admin", "serial_reviewer"} {
		if err := db.Table("positions").Where("code = ?", code).FirstOrCreate(&testPosition{Code: code}).Error; err != nil {
			return err
		}
	}
	return nil
}
func (a *installTestOrgApp) Providers(do.Injector) {}
func (a *installTestOrgApp) Routes(*gin.RouterGroup) {}
func (a *installTestOrgApp) Tasks() []scheduler.TaskDef { return nil }

func registerInstallTestOrgApp() {
	installTestOrgAppOnce.Do(func() {
		app.Register(&installTestOrgApp{})
	})
}

type testDepartment struct {
	ID   uint   `gorm:"primaryKey"`
	Code string `gorm:"size:64"`
}

func (testDepartment) TableName() string { return "departments" }

type testPosition struct {
	ID   uint   `gorm:"primaryKey"`
	Code string `gorm:"size:64"`
}

func (testPosition) TableName() string { return "positions" }

type testUserPosition struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint
	DepartmentID uint
	PositionID   uint
	IsPrimary    bool
}

func (testUserPosition) TableName() string { return "user_positions" }

func newTestDBForInstallHandler(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Role{},
		&testDepartment{},
		&testPosition{},
		&testUserPosition{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func TestUpsertInstallAdmin_ReusesExistingUser(t *testing.T) {
	db := newTestDBForInstallHandler(t)

	role := model.Role{Name: "Admin", Code: model.RoleAdmin}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	oldPassword, err := token.HashPassword("OldPassword123!")
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	user := model.User{
		Username: "admin",
		Password: oldPassword,
		Email:    "old@example.com",
		RoleID:   999,
		IsActive: false,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	if err := seed.UpsertInstallAdmin(db, "admin", "NewPassword123!", "new@example.com", role.ID); err != nil {
		t.Fatalf("upsert install admin: %v", err)
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload updated user: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Fatalf("expected updated email, got %s", updated.Email)
	}
	if updated.RoleID != role.ID {
		t.Fatalf("expected role id %d, got %d", role.ID, updated.RoleID)
	}
	if !updated.IsActive {
		t.Fatal("expected user to be active")
	}
	if updated.PasswordChangedAt == nil {
		t.Fatal("expected password changed at to be set")
	}
	if !token.CheckPassword(updated.Password, "NewPassword123!") {
		t.Fatal("expected password to be replaced")
	}
}

func TestAssignInstallAdminOrgIdentity_WaitsForOrgSeedData(t *testing.T) {
	db := newTestDBForInstallHandler(t)

	user := model.User{Username: "admin", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := seed.AssignInstallAdminOrgIdentity(db, "admin"); err != nil {
		t.Fatalf("assign before org seed should not fail: %v", err)
	}

	var count int64
	if err := db.Table("user_positions").Count(&count).Error; err != nil {
		t.Fatalf("count user positions before seed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no user positions before org seed, got %d", count)
	}

	if err := db.Create(&testDepartment{Code: "it"}).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}
	if err := db.Create(&testPosition{Code: "it_admin"}).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	if err := seed.AssignInstallAdminOrgIdentity(db, "admin"); err != nil {
		t.Fatalf("assign after org seed: %v", err)
	}

	if err := db.Table("user_positions").Where("user_id = ? AND is_primary = ?", user.ID, true).Count(&count).Error; err != nil {
		t.Fatalf("count user positions after seed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user position after org seed, got %d", count)
	}
}

func TestAssignInstallAdminOrgIdentity_AssignsAllBuiltinITSMTestPositions(t *testing.T) {
	db := newTestDBForInstallHandler(t)

	user := model.User{Username: "admin", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	for _, code := range []string{"it", "headquarters"} {
		if err := db.Create(&testDepartment{Code: code}).Error; err != nil {
			t.Fatalf("create department %s: %v", code, err)
		}
	}
	for _, code := range []string{"it_admin", "db_admin", "network_admin", "security_admin", "ops_admin", "serial_reviewer"} {
		if err := db.Create(&testPosition{Code: code}).Error; err != nil {
			t.Fatalf("create position %s: %v", code, err)
		}
	}

	if err := seed.AssignInstallAdminOrgIdentity(db, "admin"); err != nil {
		t.Fatalf("assign admin identities: %v", err)
	}
	if err := seed.AssignInstallAdminOrgIdentity(db, "admin"); err != nil {
		t.Fatalf("repeat assign admin identities: %v", err)
	}

	var count int64
	if err := db.Table("user_positions").Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count user positions: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6 admin positions, got %d", count)
	}
	if err := db.Table("user_positions").Where("user_id = ? AND is_primary = ?", user.ID, true).Count(&count).Error; err != nil {
		t.Fatalf("count primary positions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 primary position, got %d", count)
	}

	var primary struct {
		DeptCode string
		PosCode  string
	}
	if err := db.Table("user_positions AS up").
		Select("d.code AS dept_code, p.code AS pos_code").
		Joins("JOIN departments AS d ON d.id = up.department_id").
		Joins("JOIN positions AS p ON p.id = up.position_id").
		Where("up.user_id = ? AND up.is_primary = ?", user.ID, true).
		Scan(&primary).Error; err != nil {
		t.Fatalf("load primary identity: %v", err)
	}
	if primary.DeptCode != "it" || primary.PosCode != "it_admin" {
		t.Fatalf("expected primary identity it/it_admin, got %s/%s", primary.DeptCode, primary.PosCode)
	}
}

func TestExecuteDoesNotMarkInstalledWhenConfigSaveFails(t *testing.T) {
	t.Setenv("TZ", "UTC")
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})
	blockedPath := filepath.Join(tmpDir, "missing", "config.yml")

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", blockedPath)
	do.ProvideValue(injector, token.NewBlacklist())

	db, err := database.Open("sqlite", "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open install db: %v", err)
	}
	t.Cleanup(func() { db.Shutdown() })

	do.ProvideValue(injector, &config.MetisConfig{DBDriver: "sqlite", DBDSN: "unused"})
	do.ProvideValue(injector, db)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, service.NewSysConfig)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewUser)

	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body, err := json.Marshal(map[string]any{
		"db_driver":      "sqlite",
		"site_name":      "Metis",
		"locale":         "zh-CN",
		"timezone":       "UTC",
		"admin_username": "admin",
		"admin_password": "Password123!",
		"admin_email":    "admin@example.com",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when config save fails, got %d body=%s", w.Code, w.Body.String())
	}

	var cfg model.SystemConfig
	err = db.DB.Where("\"key\" = ?", "app.installed").First(&cfg).Error
	if err == nil {
		t.Fatalf("expected app.installed to remain unset when config save fails")
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected record not found for app.installed, got %v", err)
	}
}

func TestExecuteReusesExistingAdminAndPersistsOTelDefaultsBeforeConfigFailure(t *testing.T) {
	t.Setenv("TZ", "UTC")
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	db, err := database.Open("sqlite", "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open install db: %v", err)
	}
	t.Cleanup(func() { db.Shutdown() })
	if err := database.AutoMigrateKernel(db.DB); err != nil {
		t.Fatalf("migrate kernel models: %v", err)
	}

	adminRole := &model.Role{Name: "管理员", Code: model.RoleAdmin, IsSystem: true}
	if err := db.DB.Create(adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	userRole := &model.Role{Name: "普通用户", Code: model.RoleUser}
	if err := db.DB.Create(userRole).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}
	oldPassword, err := token.HashPassword("OldPassword123!")
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	existingAdmin := &model.User{
		Username: "admin",
		Password: oldPassword,
		Email:    "old@example.com",
		RoleID:   userRole.ID,
		IsActive: false,
	}
	if err := db.DB.Create(existingAdmin).Error; err != nil {
		t.Fatalf("create existing admin: %v", err)
	}

	injector := do.New()
	r := gin.New()
	blockedPath := filepath.Join(tmpDir, "missing", "config.yml")
	do.ProvideNamedValue(injector, "configPath", blockedPath)
	do.ProvideValue(injector, token.NewBlacklist())
	do.ProvideValue(injector, &config.MetisConfig{DBDriver: "sqlite", DBDSN: "unused"})
	do.ProvideValue(injector, db)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, service.NewSysConfig)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewUser)

	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body, err := json.Marshal(map[string]any{
		"db_driver":              "sqlite",
		"site_name":              "Metis",
		"locale":                 "zh-CN",
		"timezone":               "Asia/Shanghai",
		"admin_username":         "admin",
		"admin_password":         "NewPassword123!",
		"admin_email":            "admin@example.com",
		"otel_enabled":           true,
		"otel_exporter_endpoint": "",
		"otel_service_name":      "",
		"otel_sample_rate":       "",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when config save fails, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`failed to write config`)) {
		t.Fatalf("expected config write failure body, got %s", w.Body.String())
	}

	var updated model.User
	if err := db.DB.Where("username = ?", "admin").First(&updated).Error; err != nil {
		t.Fatalf("reload admin user: %v", err)
	}
	if updated.Email != "admin@example.com" {
		t.Fatalf("expected updated admin email, got %s", updated.Email)
	}
	if updated.RoleID != adminRole.ID {
		t.Fatalf("expected admin role id %d, got %d", adminRole.ID, updated.RoleID)
	}
	if !updated.IsActive {
		t.Fatal("expected admin user to be active")
	}
	if !token.CheckPassword(updated.Password, "NewPassword123!") {
		t.Fatal("expected admin password to be updated")
	}

	for key, expected := range map[string]string{
		"otel.enabled":           "true",
		"otel.exporter_endpoint": "http://localhost:4318",
		"otel.service_name":      "metis",
		"otel.sample_rate":       "1.0",
		"system.locale":          "zh-CN",
		"system.timezone":        "Asia/Shanghai",
	} {
		var cfg model.SystemConfig
		if err := db.DB.Where("\"key\" = ?", key).First(&cfg).Error; err != nil {
			t.Fatalf("load config %s: %v", key, err)
		}
		if cfg.Value != expected {
			t.Fatalf("expected %s=%s, got %s", key, expected, cfg.Value)
		}
	}

	var installedCfg model.SystemConfig
	err = db.DB.Where("\"key\" = ?", "app.installed").First(&installedCfg).Error
	if err == nil {
		t.Fatalf("expected app.installed to remain unset when config save fails")
	}
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected record not found for app.installed, got %v", err)
	}
}

func TestInstallHandlerStatusReflectsInstalledFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.installed = true
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/install/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"installed":true`)) {
		t.Fatalf("expected installed=true body, got %s", w.Body.String())
	}
}

func TestInstallHandlerCheckDB_RejectsWhenInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.installed = true
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/check-db", bytes.NewReader([]byte(`{"driver":"sqlite"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerCheckDB_RejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/check-db", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerCheckDB_SkipsNonPostgresProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/check-db", bytes.NewReader([]byte(`{"driver":"sqlite"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"success":true`)) {
		t.Fatalf("expected success=true body, got %s", w.Body.String())
	}
}

func TestInstallHandlerCheckDB_PostgresFailureReturnsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/check-db", bytes.NewReader([]byte(`{"driver":"postgres","host":"127.0.0.1","port":1,"user":"u","password":"p","dbname":"db"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"success":false`)) {
		t.Fatalf("expected success=false body, got %s", w.Body.String())
	}
}

func TestInstallHandlerExecute_RejectsWhenInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.installed = true
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerExecute_RejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerExecute_PostgresConnectionFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body := `{"db_driver":"postgres","db_host":"127.0.0.1","db_port":1,"db_user":"u","db_password":"p","db_name":"db","site_name":"Metis","locale":"zh-CN","timezone":"UTC","admin_username":"admin","admin_password":"Password123!","admin_email":"admin@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerExecute_UserServiceInitFailure(t *testing.T) {
	t.Setenv("TZ", "UTC")
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", filepath.Join(tmpDir, "config.yml"))
	do.ProvideValue(injector, token.NewBlacklist())
	do.Provide(injector, func(do.Injector) (*service.UserService, error) {
		return nil, errors.New("user service boom")
	})

	db, err := database.Open("sqlite", "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open install db: %v", err)
	}
	t.Cleanup(func() { db.Shutdown() })

	do.ProvideValue(injector, &config.MetisConfig{DBDriver: "sqlite", DBDSN: "unused"})
	do.ProvideValue(injector, db)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, service.NewSysConfig)

	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body := `{"db_driver":"sqlite","site_name":"Metis","locale":"zh-CN","timezone":"UTC","admin_username":"admin","admin_password":"Password123!","admin_email":"admin@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`failed to initialize user service`)) {
		t.Fatalf("expected user service init failure body, got %s", w.Body.String())
	}
}

func TestInstallHandlerExecute_ReturnsRestartRequiredWhenHotSwitchFailsAfterInstall(t *testing.T) {
	t.Setenv("TZ", "UTC")
	gin.SetMode(gin.TestMode)
	registerInstallTestOrgApp()

	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	placeholderDB := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, placeholderDB)
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", filepath.Join(tmpDir, "config.yml"))
	do.Provide(injector, New)

	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body := `{
		"db_driver":"sqlite",
		"site_name":"Metis",
		"locale":"zh-CN",
		"timezone":"UTC",
		"admin_username":"admin",
		"admin_password":"Password123!",
		"admin_email":"admin@example.com",
		"otel_enabled":true,
		"otel_exporter_endpoint":"::bad-endpoint"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"restart_required":true`)) {
		t.Fatalf("expected restart_required response, got %s", w.Body.String())
	}

	db, err := database.Open("sqlite", "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open installed db: %v", err)
	}
	defer db.Shutdown()

	var installedCfg model.SystemConfig
	if err := db.DB.Where("\"key\" = ?", "app.installed").First(&installedCfg).Error; err != nil {
		t.Fatalf("expected app.installed to be set, got %v", err)
	}
	if installedCfg.Value != "true" {
		t.Fatalf("expected app.installed=true, got %s", installedCfg.Value)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "config.yml")); err != nil {
		t.Fatalf("expected config file to be written, got %v", err)
	}
}

func TestInstallHandlerExecute_SucceedsWithHotSwitch(t *testing.T) {
	t.Setenv("TZ", "UTC")
	gin.SetMode(gin.TestMode)
	registerInstallTestOrgApp()

	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	placeholderDB := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, placeholderDB)
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", filepath.Join(tmpDir, "config.yml"))
	do.Provide(injector, New)

	h := NewInstall(injector, r, func(do.Injector) {})
	h.RegisterInstallRoutes(r)

	body := `{
		"db_driver":"sqlite",
		"site_name":"Metis",
		"locale":"zh-CN",
		"timezone":"UTC",
		"admin_username":"admin",
		"admin_password":"Password123!",
		"admin_email":"admin@example.com",
		"otel_enabled":true,
		"otel_exporter_endpoint":"http://collector.example.com:4318",
		"otel_service_name":"metis-install-test",
		"otel_sample_rate":"0.5",
		"falkordb_addr":"127.0.0.1:6381",
		"falkordb_password":"falkor-secret",
		"falkordb_database":2,
		"clickhouse_dsn":"clickhouse://default:@127.0.0.1:9000/default"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/install/execute", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`restart_required`)) {
		t.Fatalf("expected full hot switch success, got %s", w.Body.String())
	}
	if !h.installed {
		t.Fatal("expected handler installed flag to be true")
	}

	db, err := database.Open("sqlite", "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open installed db: %v", err)
	}
	defer db.Shutdown()

	var installedCfg model.SystemConfig
	if err := db.DB.Where("\"key\" = ?", "app.installed").First(&installedCfg).Error; err != nil {
		t.Fatalf("expected app.installed to exist, got %v", err)
	}
	if installedCfg.Value != "true" {
		t.Fatalf("expected app.installed=true, got %s", installedCfg.Value)
	}

	var admin model.User
	if err := db.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatalf("expected admin user to exist, got %v", err)
	}
	if !admin.IsActive {
		t.Fatal("expected installed admin to be active")
	}
	cfgBytes, err := os.ReadFile(filepath.Join(tmpDir, "config.yml"))
	if err != nil {
		t.Fatalf("expected config file to be written, got %v", err)
	}
	cfgText := string(cfgBytes)
	for _, expected := range []string{
		"127.0.0.1:6381",
		"falkor-secret",
		"database: 2",
		"clickhouse://default:@127.0.0.1:9000/default",
	} {
		if !strings.Contains(cfgText, expected) {
			t.Fatalf("expected config file to contain %q, got %s", expected, cfgText)
		}
	}
	for key, expected := range map[string]string{
		"otel.enabled":           "true",
		"otel.exporter_endpoint": "http://collector.example.com:4318",
		"otel.service_name":      "metis-install-test",
		"otel.sample_rate":       "0.5",
	} {
		var cfg model.SystemConfig
		if err := db.DB.Where("\"key\" = ?", key).First(&cfg).Error; err != nil {
			t.Fatalf("expected %s to be stored in db, got %v", key, err)
		}
		if cfg.Value != expected {
			t.Fatalf("expected %s=%s, got %s", key, expected, cfg.Value)
		}
	}
}

func TestInstallHandlerHotSwitch_RejectsInvalidJWTSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := do.New()
	r := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	do.ProvideValue(injector, token.NewBlacklist())
	h := NewInstall(injector, r, func(do.Injector) {})

	cfg := &config.MetisConfig{JWTSecret: "not-hex", SecretKey: "secret"}
	err := h.hotSwitch(cfg, &database.DB{DB: newTestDBForInstallHandler(t)}, nil, "admin")
	if err == nil || err.Error() == "" {
		t.Fatal("expected invalid jwt secret error")
	}
}

func TestInstallHandlerHotSwitch_SucceedsWithKernelProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	engine := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	do.Provide(injector, New)

	adminRole := &model.Role{Name: "管理员", Code: model.RoleAdmin, Sort: 1, IsSystem: true, DataScope: model.DataScopeAll}
	if err := db.Create(adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := db.AutoMigrate(&testDepartment{}, &testPosition{}, &testUserPosition{}); err != nil {
		t.Fatalf("migrate org tables: %v", err)
	}
	admin := &model.User{Username: "admin", RoleID: adminRole.ID, IsActive: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	for _, code := range []string{"it", "headquarters"} {
		if err := db.Create(&testDepartment{Code: code}).Error; err != nil {
			t.Fatalf("create department %s: %v", code, err)
		}
	}
	for _, code := range []string{"it_admin", "db_admin", "network_admin", "security_admin", "ops_admin", "serial_reviewer"} {
		if err := db.Create(&testPosition{Code: code}).Error; err != nil {
			t.Fatalf("create position %s: %v", code, err)
		}
	}

	h := NewInstall(injector, engine, func(do.Injector) {})
	wrappedDB := do.MustInvoke[*database.DB](injector)
	enforcer := do.MustInvoke[*casbin.Enforcer](injector)
	schedulerEngine := do.MustInvoke[*scheduler.Engine](injector)

	cfg := &config.MetisConfig{
		JWTSecret: "3031323334353637383961626364656630313233343536373839616263646566",
		SecretKey: "test-secret-key",
	}

	if err := h.hotSwitch(cfg, wrappedDB, enforcer, "admin"); err != nil {
		t.Fatalf("expected hotSwitch success, got %v", err)
	}
	t.Cleanup(func() { schedulerEngine.Stop() })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/site-info", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected registered route to respond, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestInstallHandlerHotSwitch_FailsOnInvalidOTelConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	engine := gin.New()
	do.ProvideNamedValue(injector, "configPath", t.TempDir()+"/config.yml")
	do.Provide(injector, New)

	adminRole := &model.Role{Name: "管理员", Code: model.RoleAdmin, Sort: 1, IsSystem: true, DataScope: model.DataScopeAll}
	if err := db.Create(adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := db.AutoMigrate(&testDepartment{}, &testPosition{}, &testUserPosition{}); err != nil {
		t.Fatalf("migrate org tables: %v", err)
	}
	admin := &model.User{Username: "admin", RoleID: adminRole.ID, IsActive: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	for _, code := range []string{"it", "headquarters"} {
		if err := db.Create(&testDepartment{Code: code}).Error; err != nil {
			t.Fatalf("create department %s: %v", code, err)
		}
	}
	for _, code := range []string{"it_admin", "db_admin", "network_admin", "security_admin", "ops_admin", "serial_reviewer"} {
		if err := db.Create(&testPosition{Code: code}).Error; err != nil {
			t.Fatalf("create position %s: %v", code, err)
		}
	}
	for key, value := range map[string]string{
		"otel.enabled":           "true",
		"otel.exporter_endpoint": "::bad-endpoint",
		"otel.service_name":      "metis-test",
		"otel.sample_rate":       "1.0",
	} {
		if err := db.Create(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("create system config %s: %v", key, err)
		}
	}

	h := NewInstall(injector, engine, func(do.Injector) {})
	wrappedDB := do.MustInvoke[*database.DB](injector)
	enforcer := do.MustInvoke[*casbin.Enforcer](injector)

	cfg := &config.MetisConfig{
		JWTSecret: "3031323334353637383961626364656630313233343536373839616263646566",
		SecretKey: "test-secret-key",
	}

	err := h.hotSwitch(cfg, wrappedDB, enforcer, "admin")
	if err == nil || !strings.Contains(err.Error(), "initialize opentelemetry") {
		t.Fatalf("expected opentelemetry init failure, got %v", err)
	}
}

func TestFindAdminRole_NotFound(t *testing.T) {
	db := newTestDBForInstallHandler(t)

	_, err := findAdminRole(db)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found, got %v", err)
	}
}
