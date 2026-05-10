package catalog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	. "metis/internal/app/itsm/domain"
	"metis/internal/handler"
)

func TestCatalogHandlerCreate_Returns409ForDuplicateCode(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	if _, err := svc.Create("Root", "root", "", "", nil, 10); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/catalogs", h.Create)
	}, http.MethodPost, "/catalogs", []byte(`{"name":"Other","code":"root"}`))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}

	resp := decodeResponseBody[handler.R](t, rec)
	if resp.Message == "" {
		t.Fatalf("expected error message")
	}
}

func TestCatalogHandlerUpdate_Returns400ForSelfParent(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	body := []byte(`{"parentId":` + itoa(root.ID) + `}`)
	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/catalogs/:id", h.Update)
	}, http.MethodPut, "/catalogs/"+itoa(root.ID), body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCatalogHandlerTreeAndServiceCounts(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 5)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := db.Create(&ServiceDefinition{Name: "VPN", Code: "vpn", CatalogID: child.ID, EngineType: "classic", IsActive: true}).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/catalogs/tree", h.Tree)
	}, http.MethodGet, "/catalogs/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tree status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"root"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"child"`)) {
		t.Fatalf("expected root/child tree response, body=%s", rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/catalogs/service-counts", h.ServiceCounts)
	}, http.MethodGet, "/catalogs/service-counts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("service counts status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("expected total count 1, body=%s", rec.Body.String())
	}
}

func TestCatalogHandlerDeleteValidatesBusinessConstraints(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 10)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/catalogs/:id", h.Delete)
	}, http.MethodDelete, "/catalogs/"+itoa(root.ID), nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete root with child status=%d body=%s", rec.Code, rec.Body.String())
	}

	serviceCatalog, err := svc.Create("Leaf", "leaf", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	if err := db.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-delete", CatalogID: serviceCatalog.ID, EngineType: "classic", IsActive: true}).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/catalogs/:id", h.Delete)
	}, http.MethodDelete, "/catalogs/"+itoa(serviceCatalog.ID), nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete catalog with service status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/catalogs/:id", h.Delete)
	}, http.MethodDelete, "/catalogs/"+itoa(child.ID), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete leaf child status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/catalogs/:id", h.Delete)
	}, http.MethodDelete, "/catalogs/999999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCatalogHandlerCreateAndUpdateConstraintMappings(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 20)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	t.Run("create rejects missing parent and third level", func(t *testing.T) {
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.POST("/catalogs", h.Create)
		}, http.MethodPost, "/catalogs", []byte(`{"name":"Ghost","code":"ghost","parentId":999}`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("create missing parent status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performJSONRequest(t, func(r *gin.Engine) {
			r.POST("/catalogs", h.Create)
		}, http.MethodPost, "/catalogs", []byte(`{"name":"Grand","code":"grand","parentId":`+itoa(child.ID)+`}`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("create third level status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("update persists payload and rejects missing catalog", func(t *testing.T) {
		body := []byte(`{"name":"Root Updated","description":"updated","icon":"grid","sortOrder":99,"isActive":false}`)
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.PUT("/catalogs/:id", h.Update)
		}, http.MethodPut, "/catalogs/"+itoa(root.ID), body)
		if rec.Code != http.StatusOK {
			t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"name":"Root Updated"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"isActive":false`)) {
			t.Fatalf("unexpected update body=%s", rec.Body.String())
		}

		rec = performJSONRequest(t, func(r *gin.Engine) {
			r.PUT("/catalogs/:id", h.Update)
		}, http.MethodPut, "/catalogs/999999", []byte(`{"name":"ghost"}`))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("update missing status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestCatalogHandlerTreeServiceCountsAndBadIDs(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)
	h := &CatalogHandler{svc: svc}

	t.Run("tree and service counts return empty payloads for empty state", func(t *testing.T) {
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.GET("/catalogs/tree", h.Tree)
		}, http.MethodGet, "/catalogs/tree", nil)
		if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"data":[]`)) {
			t.Fatalf("empty tree status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performJSONRequest(t, func(r *gin.Engine) {
			r.GET("/catalogs/service-counts", h.ServiceCounts)
		}, http.MethodGet, "/catalogs/service-counts", nil)
		if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":0`)) {
			t.Fatalf("empty service counts status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("update and delete reject bad ids", func(t *testing.T) {
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.PUT("/catalogs/:id", h.Update)
		}, http.MethodPut, "/catalogs/bad", []byte(`{"name":"ghost"}`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("update bad id status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performJSONRequest(t, func(r *gin.Engine) {
			r.DELETE("/catalogs/:id", h.Delete)
		}, http.MethodDelete, "/catalogs/bad", nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("delete bad id status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func performJSONRequest(t *testing.T, routes func(*gin.Engine), method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	r := gin.New()
	routes(r)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	return rec
}

func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
