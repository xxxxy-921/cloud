package licensee

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
)

func setupLicenseeHandler(t *testing.T) (*LicenseeHandler, *LicenseeService) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	svc := &LicenseeService{repo: &LicenseeRepo{DB: db}}
	return &LicenseeHandler{svc: svc}, svc
}

func TestLicenseeHandlerCRUDAndStatusContracts(t *testing.T) {
	h, svc := setupLicenseeHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/licensees", h.Create)
	r.GET("/licensees", h.List)
	r.GET("/licensees/:id", h.Get)
	r.PUT("/licensees/:id", h.Update)
	r.PATCH("/licensees/:id/status", h.UpdateStatus)

	t.Run("create and list", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"name":"Acme","notes":"vip"}`))
		req := httptest.NewRequest(http.MethodPost, "/licensees", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list status = %d, want 200", w.Code)
		}
	})

	t.Run("list invalid paging rejected", func(t *testing.T) {
		for _, path := range []string{"/licensees?page=bad", "/licensees?pageSize=bad"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("path=%s status=%d want 400 body=%s", path, w.Code, w.Body.String())
			}
		}
	})

	t.Run("list invalid status rejected and all accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/licensees?status=weird", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees?status=all", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=all = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees?status=ALL", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=ALL = %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("create invalid payload rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/licensees", bytes.NewReader([]byte(`{"name":`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid create payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("create duplicate name rejected", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"name":"Acme","notes":"dup"}`))
		req := httptest.NewRequest(http.MethodPost, "/licensees", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("duplicate create status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("blank business identifiers are rejected at entrypoints", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/licensees", bytes.NewReader([]byte(`{"name":"   ","notes":"x"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank create status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	created, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Beta", Notes: "old"})
	if err != nil {
		t.Fatalf("create service licensee: %v", err)
	}

	t.Run("get invalid and not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/licensees/bad", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid get status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees/999", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing get status = %d, want 404", w.Code)
		}
	})

	t.Run("get returns created licensee response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/licensees/"+strconv.Itoa(int(created.ID)), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("get created status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"name":"Beta"`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"code":"`)) {
			t.Fatalf("expected licensee response payload, body=%s", w.Body.String())
		}
	})

	t.Run("update duplicate name rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Acme"})
		req := httptest.NewRequest(http.MethodPut, "/licensees/"+strconv.Itoa(int(created.ID)), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("duplicate update status = %d, want 400", w.Code)
		}
	})

	t.Run("update invalid id and bad payload rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/licensees/bad", bytes.NewReader([]byte(`{"notes":"x"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid update id status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/licensees/"+strconv.Itoa(int(created.ID)), bytes.NewReader([]byte(`{"name":`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid update payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("update persists fields and list filters by keyword and status", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Beta Prime", "notes": "fresh"})
		req := httptest.NewRequest(http.MethodPut, "/licensees/"+strconv.Itoa(int(created.ID)), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		updated, err := svc.GetLicensee(created.ID)
		if err != nil {
			t.Fatalf("reload updated licensee: %v", err)
		}
		if updated.Name != "Beta Prime" || updated.Notes != "fresh" {
			t.Fatalf("unexpected updated licensee: %+v", updated)
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees?keyword=Prime&status=active", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("filtered list status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"total":1`)) || !bytes.Contains(w.Body.Bytes(), []byte(`Beta Prime`)) {
			t.Fatalf("expected filtered active licensee in body, got %s", w.Body.String())
		}
		if bytes.Contains(w.Body.Bytes(), []byte(`Acme`)) {
			t.Fatalf("expected keyword filter to exclude Acme, body=%s", w.Body.String())
		}
	})

	t.Run("update trims business identifiers and rejects blanks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/licensees/"+strconv.Itoa(int(created.ID)), bytes.NewReader([]byte(`{"name":"   "}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank update status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/licensees/"+strconv.Itoa(int(created.ID)), bytes.NewReader([]byte(`{"name":"  Beta Trimmed  "}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("trimmed update status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		updated, err := svc.GetLicensee(created.ID)
		if err != nil {
			t.Fatalf("reload trimmed licensee: %v", err)
		}
		if updated.Name != "Beta Trimmed" {
			t.Fatalf("trimmed update name = %q, want %q", updated.Name, "Beta Trimmed")
		}
	})

	t.Run("status transition and invalid transition", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"status":"archived"}`))
		req := httptest.NewRequest(http.MethodPatch, "/licensees/"+strconv.Itoa(int(created.ID))+"/status", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("archive status = %d, want 200", w.Code)
		}

		body = bytes.NewReader([]byte(`{"status":"archived"}`))
		req = httptest.NewRequest(http.MethodPatch, "/licensees/"+strconv.Itoa(int(created.ID))+"/status", body)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("repeat archive status = %d, want 400", w.Code)
		}

		updated, err := svc.GetLicensee(created.ID)
		if err != nil {
			t.Fatalf("get updated licensee: %v", err)
		}
		if updated.Status != domain.LicenseeStatusArchived {
			t.Fatalf("status = %q, want archived", updated.Status)
		}

		req = httptest.NewRequest(http.MethodGet, "/licensees?status=archived", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("archived filter list status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`Beta Trimmed`)) {
			t.Fatalf("expected archived licensee in filtered list, body=%s", w.Body.String())
		}
	})

	t.Run("update and status missing resource mappings", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"notes": "fresh"})
		req := httptest.NewRequest(http.MethodPut, "/licensees/999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing update status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/licensees/999/status", bytes.NewReader([]byte(`{"status":"archived"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing status update status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/licensees/bad/status", bytes.NewReader([]byte(`{"status":"archived"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status update id = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/licensees/"+strconv.Itoa(int(created.ID))+"/status", bytes.NewReader([]byte(`{"status":`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status update payload = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/licensees/"+strconv.Itoa(int(created.ID))+"/status", bytes.NewReader([]byte(`{"status":"disabled"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status transition = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/licensees/"+strconv.Itoa(int(created.ID))+"/status", bytes.NewReader([]byte(`{"status":"active"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("reactivate status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})
}
