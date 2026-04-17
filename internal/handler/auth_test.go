package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/token"
	"metis/internal/repository"
	"metis/internal/service"
)

func newTestDBForAuthHandler(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.SystemConfig{},
		&model.RefreshToken{},
		&model.UserConnection{},
		&model.Menu{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func newAuthHandlerForTest(t *testing.T, db *gorm.DB) *AuthHandler {
	t.Helper()
	wrapped := &database.DB{DB: db}
	injector := do.New()
	do.ProvideValue(injector, wrapped)

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
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, service.NewCasbin)
	do.Provide(injector, service.NewMenu)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewCaptcha)
	do.Provide(injector, service.NewAuth)
	do.Provide(injector, service.NewAuditLog)

	authSvc := do.MustInvoke[*service.AuthService](injector)
	auditSvc := do.MustInvoke[*service.AuditLogService](injector)

	return &AuthHandler{
		auth:     authSvc,
		auditSvc: auditSvc,
	}
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	return r
}

func waitForAuthAuditLog(t *testing.T, db *gorm.DB, expected int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int64
		db.Model(&model.AuditLog{}).Where("category = ?", model.AuditCategoryAuth).Count(&count)
		if count >= expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for auth audit log count %d", expected)
}

func TestAuthHandlerLogin_Failure_AuditLog(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	body := `{"username":"nonexistent","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	waitForAuthAuditLog(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.Where("category = ?", model.AuditCategoryAuth).First(&log).Error; err != nil {
		t.Fatalf("find audit log: %v", err)
	}
	if log.Action != "login_failed" {
		t.Fatalf("expected action login_failed, got %s", log.Action)
	}
	if log.Username != "nonexistent" {
		t.Fatalf("expected username nonexistent, got %s", log.Username)
	}
	if log.Level != model.AuditLevelWarn {
		t.Fatalf("expected level warn, got %s", log.Level)
	}
	if log.Category != model.AuditCategoryAuth {
		t.Fatalf("expected category auth, got %s", log.Category)
	}
	if log.IPAddress == "" {
		t.Fatal("expected IPAddress to be set")
	}
	if log.Detail == nil || *log.Detail == "" {
		t.Fatal("expected Detail to contain failure reason")
	}
}

func TestAuthHandlerLogin_Success_AuditLog(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	// Seed a user with a known password
	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	passwordHash, _ := token.HashPassword("Password123!")
	user := &model.User{
		Username:     "alice",
		Password:     passwordHash,
		Email:        "alice@example.com",
		RoleID:       role.ID,
		IsActive:     true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	body := `{"username":"alice","password":"Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	waitForAuthAuditLog(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.Where("category = ?", model.AuditCategoryAuth).First(&log).Error; err != nil {
		t.Fatalf("find audit log: %v", err)
	}
	if log.Action != "login_success" {
		t.Fatalf("expected action login_success, got %s", log.Action)
	}
	if log.Username != "alice" {
		t.Fatalf("expected username alice, got %s", log.Username)
	}
	if log.Level != model.AuditLevelInfo {
		t.Fatalf("expected level info, got %s", log.Level)
	}
	if log.UserID == nil || *log.UserID != user.ID {
		t.Fatalf("expected userID %d, got %v", user.ID, log.UserID)
	}
}
