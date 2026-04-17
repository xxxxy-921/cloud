package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/repository"
	"metis/internal/service"
)

func newTestDBForAuditMiddleware(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func newAuditMiddlewareForTest(t *testing.T, db *gorm.DB) gin.HandlerFunc {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, service.NewAuditLog)
	auditSvc := do.MustInvoke[*service.AuditLogService](injector)
	return Audit(auditSvc)
}

func setupAuditRouter(mw gin.HandlerFunc, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/test", handler)
	return r
}

func waitForAuditLogCount(t *testing.T, db *gorm.DB, expected int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int64
		db.Model(&model.AuditLog{}).Count(&count)
		if count >= expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for audit log count %d", expected)
}

func TestAuditMiddleware_RecordsOn2xxWithAction(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.Set("audit_action", "create_user")
		c.Set("audit_resource", "user")
		c.Set("audit_resource_id", "42")
		c.Set("audit_summary", "created user alice")
		c.Set("userId", uint(1))
		c.Set("userName", "alice")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	waitForAuditLogCount(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("find log: %v", err)
	}
	if log.Action != "create_user" {
		t.Fatalf("expected action create_user, got %s", log.Action)
	}
	if log.Resource != "user" || log.ResourceID != "42" {
		t.Fatalf("unexpected resource fields: %+v", log)
	}
	if log.Summary != "created user alice" {
		t.Fatalf("expected summary 'created user alice', got %s", log.Summary)
	}
	if log.Username != "alice" {
		t.Fatalf("expected username alice, got %s", log.Username)
	}
	if log.UserID == nil || *log.UserID != 1 {
		t.Fatalf("expected userID 1, got %v", log.UserID)
	}
	if log.Category != model.AuditCategoryOperation {
		t.Fatalf("expected category operation, got %s", log.Category)
	}
	if log.Level != model.AuditLevelInfo {
		t.Fatalf("expected level info, got %s", log.Level)
	}
}

func TestAuditMiddleware_SkipsOnNon2xx(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.Set("audit_action", "delete_user")
		c.AbortWithStatus(http.StatusForbidden)
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	time.Sleep(50 * time.Millisecond)
	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 logs for non-2xx, got %d", count)
	}
}

func TestAuditMiddleware_SkipsWhenNoAction(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	time.Sleep(50 * time.Millisecond)
	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 logs when no audit_action, got %d", count)
	}
}

func TestAuditMiddleware_UserIDZeroIgnored(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.Set("audit_action", "anonymous_action")
		c.Set("audit_summary", "anonymous")
		c.Set("userId", uint(0))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	waitForAuditLogCount(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("find log: %v", err)
	}
	if log.UserID != nil {
		t.Fatalf("expected nil userID for zero, got %v", *log.UserID)
	}
}

func TestAuditMiddleware_RecordsClientIPAndUserAgent(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.Set("audit_action", "create_user")
		c.Set("audit_summary", "created user")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	waitForAuditLogCount(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("find log: %v", err)
	}
	if log.IPAddress != "10.0.0.1" {
		t.Fatalf("expected IP 10.0.0.1, got %s", log.IPAddress)
	}
	if log.UserAgent != "TestAgent/1.0" {
		t.Fatalf("expected User-Agent TestAgent/1.0, got %s", log.UserAgent)
	}
}

func TestAuditMiddleware_HandlesNonStringAuditAction(t *testing.T) {
	db := newTestDBForAuditMiddleware(t)
	mw := newAuditMiddlewareForTest(t, db)
	handler := func(c *gin.Context) {
		c.Set("audit_action", 12345)
		c.Set("audit_summary", "action was int")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	r := setupAuditRouter(mw, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	waitForAuditLogCount(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("find log: %v", err)
	}
	if log.Action != "" {
		t.Fatalf("expected empty action for non-string, got %q", log.Action)
	}
	if log.Summary != "action was int" {
		t.Fatalf("expected summary 'action was int', got %s", log.Summary)
	}
}
