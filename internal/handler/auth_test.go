package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/oauth"
	"metis/internal/pkg/token"
	"metis/internal/repository"
	"metis/internal/service"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	cloned.Host = t.target.Host
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	return t.base.RoundTrip(cloned)
}

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
		&model.AuthProvider{},
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
	do.Provide(injector, repository.NewAuthProvider)
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, service.NewCasbin)
	do.Provide(injector, service.NewMenu)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewCaptcha)
	do.Provide(injector, service.NewAuth)
	do.Provide(injector, service.NewAuthProvider)
	do.Provide(injector, service.NewUserConnection)
	do.Provide(injector, service.NewUser)
	do.Provide(injector, service.NewAuditLog)

	authSvc := do.MustInvoke[*service.AuthService](injector)
	menuSvc := do.MustInvoke[*service.MenuService](injector)
	userSvc := do.MustInvoke[*service.UserService](injector)
	providerSvc := do.MustInvoke[*service.AuthProviderService](injector)
	connSvc := do.MustInvoke[*service.UserConnectionService](injector)
	auditSvc := do.MustInvoke[*service.AuditLogService](injector)

	return &AuthHandler{
		auth:        authSvc,
		userSvc:     userSvc,
		menuSvc:     menuSvc,
		providerSvc: providerSvc,
		connSvc:     connSvc,
		stateMgr:    oauth.NewStateManager(),
		auditSvc:    auditSvc,
	}
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	r.POST("/api/v1/auth/register", h.Register)
	r.GET("/api/v1/auth/registration-status", h.RegistrationStatus)
	r.POST("/api/v1/auth/logout", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.Logout(c)
	})
	r.POST("/api/v1/auth/refresh", h.Refresh)
	r.GET("/api/v1/auth/me", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.GetMe(c)
	})
	r.POST("/api/v1/auth/change-password", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.ChangePassword(c)
	})
	r.GET("/api/v1/auth/providers", h.ListProviders)
	r.GET("/api/v1/auth/oauth/:provider/initiate", h.InitiateOAuth)
	r.POST("/api/v1/auth/oauth/callback", h.OAuthCallback)
	r.GET("/api/v1/auth/connections", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.ListConnections(c)
	})
	r.GET("/api/v1/auth/oauth/:provider/bind/initiate", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.InitiateBind(c)
	})
	r.POST("/api/v1/auth/oauth/bind/callback", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.BindCallback(c)
	})
	r.DELETE("/api/v1/auth/connections/:provider", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.Unbind(c)
	})
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

func TestAuthHandlerLogin_InvalidBody(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
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

func TestAuthHandlerLogin_SpecialErrorBranches(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	passwordHash, err := token.HashPassword("Password123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	lockedUntil := time.Now().Add(time.Hour)
	lockedUser := &model.User{
		Username:     "locked",
		Password:     passwordHash,
		Email:        "locked@example.com",
		RoleID:       role.ID,
		IsActive:     true,
		LockedUntil:  &lockedUntil,
	}
	if err := db.Create(lockedUser).Error; err != nil {
		t.Fatalf("seed locked user: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"locked","password":"Password123!"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusLocked {
		t.Fatalf("expected 423 for locked user, got %d: %s", w.Code, w.Body.String())
	}

	disabledUser := &model.User{
		Username: "disabled",
		Password: passwordHash,
		Email:    "disabled@example.com",
		RoleID:   role.ID,
	}
	if err := db.Create(disabledUser).Error; err != nil {
		t.Fatalf("seed disabled user: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", disabledUser.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	disabledReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"disabled","password":"Password123!"}`)))
	disabledReq.Header.Set("Content-Type", "application/json")
	disabledW := httptest.NewRecorder()
	r.ServeHTTP(disabledW, disabledReq)
	if disabledW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for disabled user, got %d: %s", disabledW.Code, disabledW.Body.String())
	}

	captchaUser := &model.User{
		Username: "captcha",
		Password: passwordHash,
		Email:    "captcha@example.com",
		RoleID:   role.ID,
		IsActive: true,
	}
	if err := db.Create(captchaUser).Error; err != nil {
		t.Fatalf("seed captcha user: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.captcha_provider", Value: "image"}).Error; err != nil {
		t.Fatalf("seed captcha provider: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"captcha","password":"Password123!"}`)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing captcha, got %d: %s", w2.Code, w2.Body.String())
	}

	twoFactorUser := &model.User{
		Username:          "otp",
		Password:          passwordHash,
		Email:             "otp@example.com",
		RoleID:            role.ID,
		IsActive:          true,
		TwoFactorEnabled:  true,
	}
	if err := db.Create(twoFactorUser).Error; err != nil {
		t.Fatalf("seed 2fa user: %v", err)
	}
	if err := db.Model(&model.SystemConfig{}).Where("\"key\" = ?", "security.captcha_provider").Update("value", "none").Error; err != nil {
		t.Fatalf("disable captcha provider: %v", err)
	}
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"otp","password":"Password123!"}`)))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for 2FA flow, got %d: %s", w3.Code, w3.Body.String())
	}
	if !bytes.Contains(w3.Body.Bytes(), []byte(`"needsTwoFactor":true`)) {
		t.Fatalf("expected 2FA payload, got %s", w3.Body.String())
	}
}

func TestAuthHandlerRegistrationStatus(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/registration-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"registrationOpen":true`)) {
		t.Fatalf("expected registrationOpen=true, got %s", w.Body.String())
	}
}

func TestAuthHandlerRegister_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.Role{Name: "User", Code: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration_open: %v", err)
	}

	body := `{"username":"newuser","password":"Password123!","email":"new@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var user model.User
	if err := db.Where("username = ?", "newuser").First(&user).Error; err != nil {
		t.Fatalf("find registered user: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %s", user.Email)
	}
}

func TestAuthHandlerLogout_SuccessWritesAuditLog(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{"refreshToken":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	waitForAuthAuditLog(t, db, 1, 100*time.Millisecond)
	var log model.AuditLog
	if err := db.Where("action = ?", "logout").First(&log).Error; err != nil {
		t.Fatalf("find logout audit log: %v", err)
	}
	if log.UserID == nil || *log.UserID != 1 {
		t.Fatalf("expected logout user ID 1, got %v", log.UserID)
	}
}

func TestAuthHandlerLogout_InvalidBody(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerRefresh_InvalidToken(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{"refreshToken":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerGetMe_ReturnsUserPayload(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	user := &model.User{Username: "alice", Email: "alice@example.com", RoleID: role.ID, IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"username":"alice"`)) {
		t.Fatalf("expected alice in payload, got %s", w.Body.String())
	}
}

func TestAuthHandlerGetMe_MissingUser(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for missing user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerChangePassword_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	passwordHash, err := token.HashPassword("OldPassword123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{
		Username: "alice",
		Password: passwordHash,
		Email:    "alice@example.com",
		RoleID:   role.ID,
		IsActive: true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	body := `{"oldPassword":"OldPassword123!","newPassword":"NewPassword123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !token.CheckPassword(updated.Password, "NewPassword123!") {
		t.Fatal("expected password updated")
	}
}

func TestAuthHandlerChangePassword_ErrorBranches(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}

	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	passwordHash, err := token.HashPassword("OldPassword123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{Username: "alice", Password: passwordHash, RoleID: role.ID, IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{"oldPassword":"wrong","newPassword":"NewPassword123!"}`)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong old password, got %d: %s", w2.Code, w2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{"oldPassword":"OldPassword123!","newPassword":"short"}`)))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for weak new password, got %d: %s", w3.Code, w3.Body.String())
	}

	t.Run("missing user", func(t *testing.T) {
		db2 := newTestDBForAuthHandler(t)
		h2 := newAuthHandlerForTest(t, db2)
		r2 := setupAuthRouter(h2)
		req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(`{"oldPassword":"OldPassword123!","newPassword":"NewPassword123!"}`)))
		req4.Header.Set("Content-Type", "application/json")
		w4 := httptest.NewRecorder()
		r2.ServeHTTP(w4, req4)
		if w4.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 for missing user, got %d: %s", w4.Code, w4.Body.String())
		}
	})
}

func TestAuthHandlerRegister_ClosedAndDuplicate(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.Role{Name: "User", Code: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	closedReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{"username":"user1","password":"Password123!","email":"u1@example.com"}`)))
	closedReq.Header.Set("Content-Type", "application/json")
	closedResp := httptest.NewRecorder()
	r.ServeHTTP(closedResp, closedReq)
	if closedResp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when registration closed, got %d: %s", closedResp.Code, closedResp.Body.String())
	}

	if err := db.Create(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration_open: %v", err)
	}
	if err := db.Create(&model.User{Username: "user1", RoleID: 1, IsActive: true}).Error; err != nil {
		t.Fatalf("seed existing user: %v", err)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{"username":"user1","password":"Password123!","email":"u1@example.com"}`)))
	dupReq.Header.Set("Content-Type", "application/json")
	dupResp := httptest.NewRecorder()
	r.ServeHTTP(dupResp, dupReq)
	if dupResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate username, got %d: %s", dupResp.Code, dupResp.Body.String())
	}
}

func TestAuthHandlerRegister_InvalidBody(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerRefresh_InvalidBody(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerLoginAndRefresh_InternalErrors(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{"login", "/api/v1/auth/login", `{"username":"alice","password":"Password123!"}`},
		{"refresh", "/api/v1/auth/refresh", `{"refreshToken":"some-token"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("expected %s => 500, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandlerUpdateProfile_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/profile", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.UpdateProfile(c)
	})

	role := &model.Role{Name: "user", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	user := &model.User{Username: "alice", Email: "alice@example.com", RoleID: role.ID, IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/profile", bytes.NewReader([]byte(`{"locale":"zh-CN","timezone":"Asia/Shanghai"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.Locale != "zh-CN" || updated.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected updated locale/timezone, got %+v", updated)
	}
}

func TestAuthHandlerListProviders_ReturnsEnabledOnly(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.AuthProvider{ProviderKey: "github", DisplayName: "GitHub", Enabled: true}).Error; err != nil {
		t.Fatalf("seed enabled provider: %v", err)
	}
	if err := db.Create(&model.AuthProvider{ProviderKey: "google", DisplayName: "Google", Enabled: false}).Error; err != nil {
		t.Fatalf("seed disabled provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"providerKey":"github"`)) {
		t.Fatalf("expected github provider in payload, got %s", w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`"providerKey":"google"`)) {
		t.Fatalf("did not expect disabled provider in payload, got %s", w.Body.String())
	}
}

func TestAuthHandlerListProviders_Empty(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"data":[]`)) {
		t.Fatalf("expected empty provider list, got %s", w.Body.String())
	}
}

func TestAuthHandlerListProviders_InternalError(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerInitiateOAuth_ProviderUnavailable(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerInitiateOAuth_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"authURL"`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"state"`)) {
		t.Fatalf("expected auth URL payload, got %s", w.Body.String())
	}
}

func TestAuthHandlerInitiateOAuth_UnsupportedProviderConfig(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "custom",
		DisplayName:  "Custom",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed custom provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/custom/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for unsupported provider config, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerOAuthCallback_InvalidBody(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerListConnections_Empty(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/connections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"data":[]`)) {
		t.Fatalf("expected empty connections, got %s", w.Body.String())
	}
}

func TestAuthHandlerListConnections_WithRecords(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.UserConnection{
		UserID:        1,
		Provider:      "github",
		ExternalID:    "42",
		ExternalName:  "alice",
		ExternalEmail: "alice@example.com",
	}).Error; err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/connections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"provider":"github"`)) {
		t.Fatalf("expected github connection, got %s", w.Body.String())
	}
}

func TestAuthHandlerListConnections_InternalError(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/connections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerInitiateBind_ProviderUnavailable(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/bind/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerInitiateBind_SuccessAndAlreadyBound(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/bind/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.UserConnection{
		UserID:     1,
		Provider:   "github",
		ExternalID: "42",
	}).Error; err != nil {
		t.Fatalf("seed bound connection: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/bind/initiate", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when already bound, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAuthHandlerInitiateBind_UnsupportedProviderConfig(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "custom",
		DisplayName:  "Custom",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed custom provider: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/custom/bind/initiate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for unsupported provider config, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerBindCallback_InvalidState(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(`{"provider":"github","code":"abc","state":"bad-state"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerBindCallback_InvalidBindState(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	state, err := h.stateMgr.Generate("github")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"abc","state":"%s"}`, state))))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-bind state, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerUnbind_NotFound(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/connections/github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerUnbind_LastLoginMethod(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.Role{Name: "user", Code: "user"}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice", RoleID: 1, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserConnection{UserID: 1, Provider: "github", ExternalID: "42"}).Error; err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/connections/github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when removing last login method, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerUnbind_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.Role{Name: "user", Code: "user"}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	passwordHash, err := token.HashPassword("Password123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice", Password: passwordHash, RoleID: 1, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserConnection{UserID: 1, Provider: "github", ExternalID: "42"}).Error; err != nil {
		t.Fatalf("seed github connection: %v", err)
	}
	if err := db.Create(&model.UserConnection{UserID: 1, Provider: "google", ExternalID: "99"}).Error; err != nil {
		t.Fatalf("seed google connection: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/connections/github", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	if err := db.Model(&model.UserConnection{}).Where("user_id = ? AND provider = ?", 1, "github").Count(&count).Error; err != nil {
		t.Fatalf("count github connection: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected github connection removed, got count=%d", count)
	}
}

func TestAuthHandlerOAuthCallback_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.Role{Name: "User", Code: model.RoleUser}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":42,"login":"octocat","email":"octo@example.com","avatar_url":"https://example.com/octo.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthSrv.Close()

	targetURL, err := url.Parse(oauthSrv.URL)
	if err != nil {
		t.Fatalf("parse oauth server url: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{target: targetURL}}

	state, err := h.stateMgr.Generate("github")
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	body := fmt.Sprintf(`{"provider":"github","code":"code-1","state":"%s"}`, state)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(body)))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"accessToken"`)) {
		t.Fatalf("expected access token in response, got %s", w.Body.String())
	}

	var conn model.UserConnection
	if err := db.Where("provider = ? AND external_id = ?", "github", "42").First(&conn).Error; err != nil {
		t.Fatalf("expected OAuth connection to be created: %v", err)
	}
}

func TestAuthHandlerBindCallback_Success(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "User", Code: model.RoleUser}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice", Password: "hashed", RoleID: role.ID, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":99,"login":"bindcat","email":"bind@example.com","avatar_url":"https://example.com/bind.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthSrv.Close()

	targetURL, err := url.Parse(oauthSrv.URL)
	if err != nil {
		t.Fatalf("parse oauth server url: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{target: targetURL}}

	state, err := h.stateMgr.GenerateForBind("github", 1)
	if err != nil {
		t.Fatalf("generate bind state: %v", err)
	}
	body := fmt.Sprintf(`{"provider":"github","code":"code-1","state":"%s"}`, state)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(body)))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var conn model.UserConnection
	if err := db.Where("user_id = ? AND provider = ? AND external_id = ?", 1, "github", "99").First(&conn).Error; err != nil {
		t.Fatalf("expected bound connection to be created: %v", err)
	}
}

func TestAuthHandlerBindCallback_ConflictBranches(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "User", Code: model.RoleUser}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice", Password: "hashed", RoleID: role.ID, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user 1: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 2}, Username: "bob", Password: "hashed", RoleID: role.ID, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user 2: %v", err)
	}
	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":99,"login":"bindcat","email":"bind@example.com","avatar_url":"https://example.com/bind.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthSrv.Close()

	targetURL, err := url.Parse(oauthSrv.URL)
	if err != nil {
		t.Fatalf("parse oauth server url: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{target: targetURL}}

	if err := db.Create(&model.UserConnection{UserID: 1, Provider: "github", ExternalID: "1"}).Error; err != nil {
		t.Fatalf("seed already bound connection: %v", err)
	}
	state1, err := h.stateMgr.GenerateForBind("github", 1)
	if err != nil {
		t.Fatalf("generate bind state1: %v", err)
	}
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code-1","state":"%s"}`, state1))))
	req1 = req1.WithContext(context.WithValue(req1.Context(), oauth2.HTTPClient, client))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for already bound provider, got %d: %s", w1.Code, w1.Body.String())
	}

	if err := db.Delete(&model.UserConnection{}, "user_id = ? AND provider = ?", 1, "github").Error; err != nil {
		t.Fatalf("delete prior connection: %v", err)
	}
	if err := db.Create(&model.UserConnection{UserID: 2, Provider: "github", ExternalID: "99"}).Error; err != nil {
		t.Fatalf("seed external id conflict: %v", err)
	}
	state2, err := h.stateMgr.GenerateForBind("github", 1)
	if err != nil {
		t.Fatalf("generate bind state2: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code-1","state":"%s"}`, state2))))
	req2 = req2.WithContext(context.WithValue(req2.Context(), oauth2.HTTPClient, client))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 for external id conflict, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAuthHandlerUpdateProfile_ErrorBranches(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/profile", func(c *gin.Context) {
		c.Set("userId", uint(1))
		h.UpdateProfile(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/profile", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/profile", bytes.NewReader([]byte(`{"locale":"zh-CN"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for missing user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerRegister_PasswordViolationAndMissingDefaultRole(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	if err := db.Create(&model.SystemConfig{Key: "security.registration_open", Value: "true"}).Error; err != nil {
		t.Fatalf("seed registration_open: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "security.password_min_length", Value: "12"}).Error; err != nil {
		t.Fatalf("seed password policy: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{"username":"shorty","password":"short","email":"s@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for password violation, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.SystemConfig{Key: "security.default_role_code", Value: "missing-role"}).Error; err != nil {
		t.Fatalf("seed missing default role code: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{"username":"user2","password":"VeryLongPass123!","email":"u2@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for missing default role, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerOAuthCallback_ErrorBranches(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "User", Code: model.RoleUser}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	if err := db.Create(&model.User{Username: "disabled", Email: "disabled@example.com", RoleID: role.ID, IsActive: false}).Error; err != nil {
		t.Fatalf("seed disabled user: %v", err)
	}
	var disabledUser model.User
	if err := db.Where("username = ?", "disabled").First(&disabledUser).Error; err != nil {
		t.Fatalf("reload disabled user: %v", err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", disabledUser.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("mark disabled user inactive: %v", err)
	}
	if err := db.Create(&model.UserConnection{UserID: disabledUser.ID, Provider: "github", ExternalID: "42"}).Error; err != nil {
		t.Fatalf("seed disabled user connection: %v", err)
	}

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":43,"login":"ok","email":"ok@example.com","avatar_url":"https://example.com/c.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthSrv.Close()

	targetURL, err := url.Parse(oauthSrv.URL)
	if err != nil {
		t.Fatalf("parse oauth server url: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{target: targetURL}}

	state, _ := h.stateMgr.Generate("github")
	if err := db.Model(&model.AuthProvider{}).Where("provider_key = ?", "github").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable provider: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unavailable provider, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "custom",
		DisplayName:  "Custom",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed custom provider: %v", err)
	}
	customState, _ := h.stateMgr.Generate("custom")
	customReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"custom","code":"code","state":"%s"}`, customState))))
	customReq.Header.Set("Content-Type", "application/json")
	customResp := httptest.NewRecorder()
	r.ServeHTTP(customResp, customReq)
	if customResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for unsupported oauth provider config, got %d: %s", customResp.Code, customResp.Body.String())
	}

	if err := db.Model(&model.AuthProvider{}).Where("provider_key = ?", "github").Update("enabled", true).Error; err != nil {
		t.Fatalf("re-enable provider: %v", err)
	}
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad exchange", http.StatusBadRequest)
	}))
	defer failSrv.Close()
	failURL, err := url.Parse(failSrv.URL)
	if err != nil {
		t.Fatalf("parse fail oauth server url: %v", err)
	}
	failClient := &http.Client{Transport: &rewriteTransport{target: failURL}}
	state, _ = h.stateMgr.Generate("github")
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, failClient))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for exchange failure, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.User{Username: "taken", Email: "taken@example.com", RoleID: role.ID, IsActive: true}).Error; err != nil {
		t.Fatalf("seed conflict user: %v", err)
	}
	conflictSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":41,"login":"conflict","email":"taken@example.com","avatar_url":"https://example.com/a.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer conflictSrv.Close()
	conflictURL, err := url.Parse(conflictSrv.URL)
	if err != nil {
		t.Fatalf("parse conflict oauth server url: %v", err)
	}
	conflictClient := &http.Client{Transport: &rewriteTransport{target: conflictURL}}
	state, _ = h.stateMgr.Generate("github")
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, conflictClient))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for email conflict, got %d: %s", w.Code, w.Body.String())
	}

	disabledSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"gh-token","token_type":"bearer"}`)
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":42,"login":"disabled","email":"disabled@example.com","avatar_url":"https://example.com/b.png"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer disabledSrv.Close()
	disabledURL, err := url.Parse(disabledSrv.URL)
	if err != nil {
		t.Fatalf("parse disabled oauth server url: %v", err)
	}
	disabledClient := &http.Client{Transport: &rewriteTransport{target: disabledURL}}
	state, _ = h.stateMgr.Generate("github")
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"code","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, disabledClient))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for disabled account, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerBindCallback_ProviderUnavailableAndExchangeFailure(t *testing.T) {
	db := newTestDBForAuthHandler(t)
	h := newAuthHandlerForTest(t, db)
	r := setupAuthRouter(h)

	role := &model.Role{Name: "User", Code: model.RoleUser}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "alice", Password: "hashed", RoleID: role.ID, IsActive: true}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       "read:user user:email",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			http.Error(w, "bad exchange", http.StatusBadRequest)
			return
		}
		http.NotFound(w, r)
	}))
	defer oauthSrv.Close()
	targetURL, err := url.Parse(oauthSrv.URL)
	if err != nil {
		t.Fatalf("parse oauth server url: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{target: targetURL}}

	state, _ := h.stateMgr.GenerateForBind("github", 1)
	if err := db.Model(&model.AuthProvider{}).Where("provider_key = ?", "github").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable provider: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"abc","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for provider unavailable, got %d: %s", w.Code, w.Body.String())
	}

	if err := db.Create(&model.AuthProvider{
		ProviderKey:  "custom",
		DisplayName:  "Custom",
		Enabled:      true,
		ClientID:     "cid",
		ClientSecret: "secret",
		CallbackURL:  "https://app.example.com/callback",
	}).Error; err != nil {
		t.Fatalf("seed custom provider: %v", err)
	}
	customState, _ := h.stateMgr.GenerateForBind("custom", 1)
	customReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"custom","code":"abc","state":"%s"}`, customState))))
	customReq.Header.Set("Content-Type", "application/json")
	customResp := httptest.NewRecorder()
	r.ServeHTTP(customResp, customReq)
	if customResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for unsupported bind provider config, got %d: %s", customResp.Code, customResp.Body.String())
	}

	if err := db.Model(&model.AuthProvider{}).Where("provider_key = ?", "github").Update("enabled", true).Error; err != nil {
		t.Fatalf("re-enable provider: %v", err)
	}
	state, _ = h.stateMgr.GenerateForBind("github", 1)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/bind/callback", bytes.NewReader([]byte(fmt.Sprintf(`{"provider":"github","code":"abc","state":"%s"}`, state))))
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for exchange failure, got %d: %s", w.Code, w.Body.String())
	}
}
