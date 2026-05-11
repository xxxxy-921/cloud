package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/channel"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/repository"
	"metis/internal/service"
)

func newTestDBForChannelHandler(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

type stubChannelDriver struct{}

func (d *stubChannelDriver) Send(config map[string]any, payload channel.Payload) error {
	return nil
}

func (d *stubChannelDriver) Test(config map[string]any) error {
	return nil
}

type failingChannelDriver struct{}

func (d *failingChannelDriver) Send(config map[string]any, payload channel.Payload) error {
	return errors.New("send failed")
}

func (d *failingChannelDriver) Test(config map[string]any) error {
	return errors.New("test failed")
}

func newChannelHandlerForTest(t *testing.T, db *gorm.DB, driver channel.Driver) *ChannelHandler {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewMessageChannel)
	do.Provide(injector, service.NewMessageChannel)
	svc := do.MustInvoke[*service.MessageChannelService](injector)
	if driver != nil {
		svc.DriverResolver = func(string) (channel.Driver, error) {
			return driver, nil
		}
	}
	return &ChannelHandler{svc: svc}
}

func setupChannelRouter(h *ChannelHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authed := r.Group("/api/v1")
	{
		authed.GET("/channels", h.List)
		authed.POST("/channels", h.Create)
		authed.GET("/channels/:id", h.Get)
		authed.PUT("/channels/:id", h.Update)
		authed.DELETE("/channels/:id", h.Delete)
		authed.PUT("/channels/:id/toggle", h.Toggle)
		authed.POST("/channels/:id/test", h.Test)
		authed.POST("/channels/:id/send-test", h.SendTest)
	}
	return r
}

func seedChannel(t *testing.T, db *gorm.DB, name, channelType, config string, enabled bool) *model.MessageChannel {
	t.Helper()
	ch := &model.MessageChannel{Name: name, Type: channelType, Config: config, Enabled: enabled}
	if err := db.Create(ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	return ch
}

func TestChannelHandlerList(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seedChannel(t, db, "SMTP", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels?page=1&pageSize=10", nil)
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
}

func TestChannelHandlerGet_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "SMTP", "email", `{"password":"secret"}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "SMTP" {
		t.Fatalf("expected name SMTP, got %v", data["name"])
	}
}

func TestChannelHandlerGet_NotFound(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestChannelHandlerGet_InvalidID(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelHandlerCreate_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	body := `{"name":"SMTP","type":"email","config":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", bytes.NewReader([]byte(body)))
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
	if resp["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", resp["code"])
	}
}

func TestChannelHandlerCreate_InvalidType(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, nil)
	r := setupChannelRouter(h)

	body := `{"name":"SMTP","type":"unknown","config":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelHandlerCreate_InvalidBody(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChannelHandlerUpdate_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "Old", "email", `{}`, true)
	r := setupChannelRouter(h)

	body := `{"name":"New","config":"{}"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), bytes.NewReader([]byte(body)))
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

func TestChannelHandlerUpdate_NotFound(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	body := `{"name":"New","config":"{}"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/9999", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestChannelHandlerUpdate_InvalidIDAndBody(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "SMTP", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/abc", bytes.NewReader([]byte(`{"name":"New","config":"{}"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChannelHandlerDelete_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "Delete", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestChannelHandlerDelete_NotFound(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestChannelHandlerDelete_InvalidID(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelHandlerToggle(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "Toggle", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%d/toggle", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["enabled"].(bool) {
		t.Fatal("expected channel to be disabled after toggle")
	}
}

func TestChannelHandlerToggle_NotFound(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/9999/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestChannelHandlerToggle_InvalidID(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/abc/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelHandlerTest_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "Test", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/test", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if !data["success"].(bool) {
		t.Fatalf("expected success true, got %v", data)
	}
}

func TestChannelHandlerTest_Failure(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &failingChannelDriver{})
	seeded := seedChannel(t, db, "Test", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/test", seeded.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["success"].(bool) {
		t.Fatal("expected success false")
	}
	if data["error"] != "test failed" {
		t.Fatalf("expected error message, got %v", data)
	}
}

func TestChannelHandlerTest_InvalidIDAndNotFound(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/abc/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/channels/9999/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing channel, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChannelHandlerSendTest_Success(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "Send", "email", `{}`, true)
	r := setupChannelRouter(h)

	body := `{"to":"to@example.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/send-test", seeded.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
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
}

func TestChannelHandlerSendTest_FailurePaths(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &failingChannelDriver{})
	seeded := seedChannel(t, db, "Send", "email", `{}`, true)
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/send-test", seeded.ID), bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/channels/9999/send-test", bytes.NewReader([]byte(`{"to":"to@example.com","subject":"Hello","body":"World"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/send-test", seeded.ID), bytes.NewReader([]byte(`{"to":"to@example.com","subject":"Hello","body":"World"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["success"].(bool) {
		t.Fatalf("expected success false, got %v", data)
	}
}

func TestChannelHandlerSendTest_InvalidID(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	r := setupChannelRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/abc/send-test", bytes.NewReader([]byte(`{"to":"to@example.com","subject":"Hello","body":"World"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChannelHandler_InternalErrors(t *testing.T) {
	db := newTestDBForChannelHandler(t)
	h := newChannelHandlerForTest(t, db, &stubChannelDriver{})
	seeded := seedChannel(t, db, "SMTP", "email", `{}`, true)
	r := setupChannelRouter(h)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		code   int
	}{
		{"list", http.MethodGet, "/api/v1/channels?page=1&pageSize=10", "", http.StatusInternalServerError},
		{"get", http.MethodGet, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), "", http.StatusInternalServerError},
		{"update", http.MethodPut, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), `{"name":"New","config":"{}"}`, http.StatusInternalServerError},
		{"delete", http.MethodDelete, fmt.Sprintf("/api/v1/channels/%d", seeded.ID), "", http.StatusInternalServerError},
		{"toggle", http.MethodPut, fmt.Sprintf("/api/v1/channels/%d/toggle", seeded.ID), "", http.StatusInternalServerError},
		{"test", http.MethodPost, fmt.Sprintf("/api/v1/channels/%d/test", seeded.ID), "", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("expected %s => %d, got %d: %s", tc.name, tc.code, w.Code, w.Body.String())
			}
			if tc.name == "test" {
				var resp map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				data := resp["data"].(map[string]any)
				if data["success"].(bool) {
					t.Fatalf("expected failure payload, got %v", data)
				}
			}
		})
	}
}
