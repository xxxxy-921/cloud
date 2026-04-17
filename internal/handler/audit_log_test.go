package handler

import (
	"encoding/json"
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

func newTestDBForAuditLogHandler(t *testing.T) *gorm.DB {
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

func seedAuditLogHandler(t *testing.T, db *gorm.DB, category model.AuditCategory, action, resource, summary, username string, createdAt time.Time) *model.AuditLog {
	t.Helper()
	log := &model.AuditLog{
		Category:  category,
		Action:    action,
		Resource:  resource,
		Summary:   summary,
		Username:  username,
		Level:     model.AuditLevelInfo,
		CreatedAt: createdAt,
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("seed audit log: %v", err)
	}
	return log
}

func newAuditLogHandlerForTest(t *testing.T, db *gorm.DB) *AuditLogHandler {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, service.NewAuditLog)
	svc := do.MustInvoke[*service.AuditLogService](injector)
	return &AuditLogHandler{auditSvc: svc}
}

func setupAuditLogRouter(h *AuditLogHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authed := r.Group("/api/v1")
	{
		authed.GET("/audit-logs", h.List)
	}
	return r
}

func TestAuditLogHandlerList_Success(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "user", "created alice", "alice", time.Now())
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("expected total 1, got %v", data["total"])
	}
}

func TestAuditLogHandlerList_MissingCategory(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "category is required" {
		t.Fatalf("expected 'category is required', got %v", resp["message"])
	}
}

func TestAuditLogHandlerList_InvalidCategory(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["message"] != "invalid category" {
		t.Fatalf("expected 'invalid category', got %v", resp["message"])
	}
}

func TestAuditLogHandlerList_KeywordFilter(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "", "created project alpha", "alice", time.Now())
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "", "created project beta", "bob", time.Now())
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation&keyword=alpha", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["summary"] != "created project alpha" {
		t.Fatalf("expected alpha summary, got %v", item["summary"])
	}
}

func TestAuditLogHandlerList_ActionFilter(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create_user", "", "summary", "alice", time.Now())
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "delete_user", "", "summary", "bob", time.Now())
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation&action=create_user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["action"] != "create_user" {
		t.Fatalf("expected create_user action, got %v", item["action"])
	}
}

func TestAuditLogHandlerList_ResourceFilter(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "user", "summary", "alice", time.Now())
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "role", "summary", "bob", time.Now())
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation&resource=user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["resource"] != "user" {
		t.Fatalf("expected user resource, got %v", item["resource"])
	}
}

func TestAuditLogHandlerList_DateRange(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "action", "", "january", "alice", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "action", "", "february", "bob", time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC))
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation&dateFrom=2024-01-01&dateTo=2024-01-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["summary"] != "january" {
		t.Fatalf("expected january summary, got %v", item["summary"])
	}
}

func TestAuditLogHandlerList_ResponseShape(t *testing.T) {
	db := newTestDBForAuditLogHandler(t)
	h := newAuditLogHandlerForTest(t, db)
	now := time.Now().Truncate(time.Second)
	seedAuditLogHandler(t, db, model.AuditCategoryOperation, "create", "user", "created alice", "alice", now)
	r := setupAuditLogRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?category=operation", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}
	data := resp["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("expected total 1, got %v", data["total"])
	}
	if data["page"].(float64) != 1 {
		t.Fatalf("expected page 1, got %v", data["page"])
	}
	if data["pageSize"].(float64) != 20 {
		t.Fatalf("expected pageSize 20, got %v", data["pageSize"])
	}
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if item["username"] != "alice" {
		t.Fatalf("expected username alice, got %v", item["username"])
	}
	if item["action"] != "create" {
		t.Fatalf("expected action create, got %v", item["action"])
	}
	if item["resource"] != "user" {
		t.Fatalf("expected resource user, got %v", item["resource"])
	}
	if item["summary"] != "created alice" {
		t.Fatalf("expected summary 'created alice', got %v", item["summary"])
	}
}
