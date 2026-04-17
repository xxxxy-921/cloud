package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/identity"
	"metis/internal/repository"
	"metis/internal/service"
)

func newTestDBForIdentitySourceHandler(t *testing.T) *gorm.DB {
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

func seedIdentitySourceHandler(t *testing.T, db *gorm.DB, name, sourceType, config, domains string, enabled bool) *model.IdentitySource {
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

func newIdentitySourceHandlerForTest(t *testing.T, db *gorm.DB) *IdentitySourceHandler {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewIdentitySource)
	do.Provide(injector, service.NewIdentitySource)
	svc := do.MustInvoke[*service.IdentitySourceService](injector)
	svc.TestOIDCFn = func(ctx context.Context, issuerURL string) error { return nil }
	svc.TestLDAPFn = func(cfg *model.LDAPConfig) error { return nil }
	svc.LDAPAuthFn = func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error) {
		return &identity.LDAPAuthResult{DN: "cn=user", Username: "user"}, nil
	}
	return &IdentitySourceHandler{svc: svc}
}

func setupIdentitySourceRouter(h *IdentitySourceHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authed := r.Group("/api/v1")
	{
		authed.GET("/identity-sources", h.List)
		authed.POST("/identity-sources", h.Create)
		authed.PUT("/identity-sources/:id", h.Update)
		authed.DELETE("/identity-sources/:id", h.Delete)
		authed.PUT("/identity-sources/:id/toggle", h.Toggle)
		authed.POST("/identity-sources/:id/test", h.TestConnection)
	}
	return r
}

func TestIdentitySourceHandlerList(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seedIdentitySourceHandler(t, db, "Okta", "oidc", `{}`, "", true)
	r := setupIdentitySourceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity-sources", nil)
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
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data))
	}
}

func TestIdentitySourceHandlerCreate_Success(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	r := setupIdentitySourceRouter(h)

	body := `{"name":"Okta","type":"oidc","config":{"issuerUrl":"https://example.com","clientId":"id"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity-sources", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["id"].(float64) == 0 {
		t.Fatal("expected id in response")
	}
	if data["name"] != "Okta" {
		t.Fatalf("expected name Okta, got %v", data["name"])
	}
}

func TestIdentitySourceHandlerCreate_UnsupportedType(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	r := setupIdentitySourceRouter(h)

	body := `{"name":"SAML","type":"saml","config":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity-sources", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIdentitySourceHandlerCreate_DomainConflict(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seedIdentitySourceHandler(t, db, "Existing", "oidc", `{}`, "example.com", true)
	r := setupIdentitySourceRouter(h)

	body := `{"name":"New","type":"oidc","config":{},"domains":"example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity-sources", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIdentitySourceHandlerUpdate_Success(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seeded := seedIdentitySourceHandler(t, db, "Old", "oidc", `{}`, "", true)
	r := setupIdentitySourceRouter(h)

	body := `{"name":"New","config":{}}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/identity-sources/%d", seeded.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "New" {
		t.Fatalf("expected name New, got %v", data["name"])
	}
}

func TestIdentitySourceHandlerUpdate_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	r := setupIdentitySourceRouter(h)

	body := `{"name":"New","config":{}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/identity-sources/9999", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIdentitySourceHandlerDelete_Success(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seeded := seedIdentitySourceHandler(t, db, "Delete", "oidc", `{}`, "", true)
	r := setupIdentitySourceRouter(h)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/identity-sources/%d", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIdentitySourceHandlerDelete_NotFound(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	r := setupIdentitySourceRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/identity-sources/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIdentitySourceHandlerToggle(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seeded := seedIdentitySourceHandler(t, db, "Toggle", "oidc", `{}`, "", true)
	r := setupIdentitySourceRouter(h)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/identity-sources/%d/toggle", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["enabled"].(bool) {
		t.Fatal("expected disabled after toggle")
	}
}

func TestIdentitySourceHandlerTestConnection(t *testing.T) {
	db := newTestDBForIdentitySourceHandler(t)
	h := newIdentitySourceHandlerForTest(t, db)
	seeded := seedIdentitySourceHandler(t, db, "Okta", "oidc", `{"issuerUrl":"https://example.com"}`, "", true)
	r := setupIdentitySourceRouter(h)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/identity-sources/%d/test", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if !data["success"].(bool) {
		t.Fatalf("expected success true, got %v", data)
	}
	if data["message"] != "OIDC discovery successful" {
		t.Fatalf("expected success message, got %v", data["message"])
	}
}
