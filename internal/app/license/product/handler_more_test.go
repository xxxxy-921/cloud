package product

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

func setupProductHandlerEnv(t *testing.T) (*ProductHandler, *PlanHandler, *ProductService, *gin.Engine) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	planSvc := &PlanService{
		planRepo:    &PlanRepo{DB: db},
		productRepo: &ProductRepo{DB: db},
	}
	productHandler := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{}}
	planHandler := &PlanHandler{planSvc: planSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	r.POST("/products", productHandler.Create)
	r.GET("/products", productHandler.List)
	r.GET("/products/:id", productHandler.Get)
	r.PUT("/products/:id", productHandler.Update)
	r.PUT("/products/:id/schema", productHandler.UpdateSchema)
	r.PATCH("/products/:id/status", productHandler.UpdateStatus)
	r.POST("/products/:id/rotate-key", productHandler.RotateKey)
	r.GET("/products/:id/public-key", productHandler.GetPublicKey)
	r.GET("/products/:id/rotate-key-impact", productHandler.RotateKeyImpact)
	r.POST("/products/:id/bulk-reissue", productHandler.BulkReissue)
	r.POST("/products/:id/plans", planHandler.Create)
	r.PUT("/plans/:id", planHandler.Update)
	r.DELETE("/plans/:id", planHandler.Delete)
	r.PATCH("/plans/:id/default", planHandler.SetDefault)
	return productHandler, planHandler, productSvc, r
}

func TestProductAndPlanHandlersCoverCrudFlows(t *testing.T) {
	_, _, productSvc, r := setupProductHandlerEnv(t)

	t.Run("create list update status key flows", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"name":"Metis","code":"metis-http","description":"desc"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create product status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/products?keyword=metis-http&page=1&pageSize=20", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list product status = %d, want 200", w.Code)
		}

		body, _ := json.Marshal(map[string]any{"name": "Metis 2", "description": "desc2"})
		req = httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update product status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPatch, "/products/1/status", bytes.NewReader([]byte(`{"status":"published"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/products/1/rotate-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("rotate key status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/products/1/public-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("public key status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/products/1/rotate-key-impact", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("rotate key impact status = %d, want 200", w.Code)
		}
	})

	t.Run("list invalid paging rejected", func(t *testing.T) {
		for _, path := range []string{"/products?page=bad", "/products?pageSize=bad"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("path=%s status=%d want 400 body=%s", path, w.Code, w.Body.String())
			}
		}
	})

	t.Run("list invalid status rejected and all accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products?status=weird", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products?status=all", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=all = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products?status=ALL", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=ALL = %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("plan create update default delete flows", func(t *testing.T) {
		product, err := productSvc.CreateProduct("Plan Host", "plan-host", "desc")
		if err != nil {
			t.Fatalf("create plan host product: %v", err)
		}
		if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish plan host product: %v", err)
		}
		product, err = productSvc.GetProduct(product.ID)
		if err != nil {
			t.Fatalf("GetProduct: %v", err)
		}
		if product.Status != domain.StatusPublished {
			t.Fatalf("product status = %s, want published", product.Status)
		}

		req := httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/plans", bytes.NewReader([]byte(`{"name":"Basic","constraintValues":{},"sortOrder":1}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create plan status = %d, want 200, body=%s", w.Code, w.Body.String())
		}

		body, _ := json.Marshal(map[string]any{"name": "Basic Plus", "sortOrder": 2})
		req = httptest.NewRequest(http.MethodPut, "/plans/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update plan status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPatch, "/plans/1/default", bytes.NewReader([]byte(`{"isDefault":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("set default plan status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/plans/1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete plan status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid id is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/products/bad/status", bytes.NewReader([]byte(`{"status":"published"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid product id status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/plans/"+strconv.Itoa(999), nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing plan delete status = %d, want 404", w.Code)
		}
	})
}

func TestPlanHandlersRejectConflictsAndMissingResources(t *testing.T) {
	_, _, productSvc, r := setupProductHandlerEnv(t)

	product, err := productSvc.CreateProduct("Plan Guard", "plan-guard", "desc")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	schema := []byte(`[
		{"key":"seat","features":[{"key":"count","type":"number","min":1,"max":10}]}
	]`)
	if err := productSvc.UpdateConstraintSchema(product.ID, schema); err != nil {
		t.Fatalf("update schema: %v", err)
	}

	createPath := "/products/" + strconv.Itoa(int(product.ID)) + "/plans"
	req := httptest.NewRequest(http.MethodPost, createPath, bytes.NewReader([]byte(`{"name":"Starter","constraintValues":{"seat":{"count":2}},"sortOrder":1}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed create plan status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, createPath, bytes.NewReader([]byte(`{"name":"Starter","constraintValues":{"seat":{"count":3}},"sortOrder":2}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("duplicate plan create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/plans/1", bytes.NewReader([]byte(`{"constraintValues":{"seat":{"count":99}}}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid constraint update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/plans/999/default", bytes.NewReader([]byte(`{"isDefault":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing default plan status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/plans/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing delete plan status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, createPath, bytes.NewReader([]byte(`{"name":"   "}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank create plan status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/plans/1", bytes.NewReader([]byte(`{"name":"   "}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank update plan status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestProductHandlersRejectMissingResourcesAndPersistSchemaUpdates(t *testing.T) {
	_, _, productSvc, r := setupProductHandlerEnv(t)

	product, err := productSvc.CreateProduct("Schema Host", "schema-host", "desc")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	t.Run("get and update missing product", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing get status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/999", bytes.NewReader([]byte(`{"name":"ghost"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing update status = %d, want 404 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("schema update persists and missing rotate key is rejected", func(t *testing.T) {
		schema := `[{"key":"seat","label":"Seat","features":[{"key":"count","label":"Count","type":"number","min":1,"max":20}]}]`
		req := httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID))+"/schema", bytes.NewReader([]byte(`{"constraintSchema":`+schema+`}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("schema update status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		updated, err := productSvc.GetProduct(product.ID)
		if err != nil {
			t.Fatalf("get product after schema update: %v", err)
		}
		if string(updated.ConstraintSchema) != schema {
			t.Fatalf("constraint schema = %s, want %s", string(updated.ConstraintSchema), schema)
		}

		req = httptest.NewRequest(http.MethodPut, "/products/999/schema", bytes.NewReader([]byte(`{"constraintSchema":[]}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing schema update status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/bad/schema", bytes.NewReader([]byte(`{"constraintSchema":[]}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid schema id status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID))+"/schema", bytes.NewReader([]byte(`{"constraintSchema":`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid schema payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/products/999/rotate-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing rotate key status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

	})
}

func TestProductHandlersReturnUpdatedEntitiesAndRejectBadIDs(t *testing.T) {
	_, _, productSvc, r := setupProductHandlerEnv(t)

	product, err := productSvc.CreateProduct("Ops Host", "ops-host", "original")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	t.Run("get update rotate and public key expose persisted state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID)), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("get product status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"code":"ops-host"`)) {
			t.Fatalf("expected product code in get response, got %s", w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID)), bytes.NewReader([]byte(`{"name":"Ops Host Updated","description":"updated"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update product status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		updated, err := productSvc.GetProduct(product.ID)
		if err != nil {
			t.Fatalf("GetProduct after update: %v", err)
		}
		if updated.Name != "Ops Host Updated" || updated.Description != "updated" {
			t.Fatalf("unexpected updated product: %+v", updated)
		}

		beforeRotate, err := productSvc.GetPublicKey(product.ID)
		if err != nil {
			t.Fatalf("GetPublicKey before rotate: %v", err)
		}

		req = httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/rotate-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("rotate key status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID))+"/public-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("public key status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		afterRotate, err := productSvc.GetPublicKey(product.ID)
		if err != nil {
			t.Fatalf("GetPublicKey after rotate: %v", err)
		}
		if afterRotate.Version != beforeRotate.Version+1 {
			t.Fatalf("rotated key version = %d, want %d", afterRotate.Version, beforeRotate.Version+1)
		}
		if afterRotate.PublicKey == beforeRotate.PublicKey {
			t.Fatalf("expected rotated public key to change, got same key")
		}
	})

	t.Run("invalid ids are rejected across key endpoints", func(t *testing.T) {
		for _, path := range []string{
			"/products/bad",
			"/products/bad/rotate-key",
			"/products/bad/public-key",
		} {
			method := http.MethodGet
			if path == "/products/bad/rotate-key" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s invalid id status = %d, want 400 body=%s", path, w.Code, w.Body.String())
			}
		}
	})

	t.Run("bad payloads and missing resources are mapped consistently", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"name":`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid create payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID)), bytes.NewReader([]byte(`{"name":`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid update payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/products/"+strconv.Itoa(int(product.ID))+"/status", bytes.NewReader([]byte(`{"status":`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid status payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/products/999/status", bytes.NewReader([]byte(`{"status":"published"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing status update status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products/999/public-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing public key status = %d, want 404 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("blank business identifiers are rejected at entrypoints", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"name":"   ","code":"trim-http","description":"desc"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank create name status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"name":"Trim HTTP","code":"   ","description":"desc"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank create code status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID)), bytes.NewReader([]byte(`{"name":"   "}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank update name status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("missing current key is surfaced by rotate and public key endpoints", func(t *testing.T) {
		keyless, err := productSvc.CreateProduct("Keyless Host", "keyless-host", "desc")
		if err != nil {
			t.Fatalf("create keyless product: %v", err)
		}
		if err := productSvc.db.Where("product_id = ?", keyless.ID).Delete(&domain.ProductKey{}).Error; err != nil {
			t.Fatalf("delete current key: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(keyless.ID))+"/rotate-key", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing current key rotate status = %d, want 404 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(keyless.ID))+"/public-key", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("missing current key public key status = %d, want 404 body=%s", w.Code, w.Body.String())
		}
	})
}

func TestProductAndPlanHandlersGuardPayloadsAndDefaultSwitching(t *testing.T) {
	_, _, productSvc, r := setupProductHandlerEnv(t)

	product, err := productSvc.CreateProduct("Plan Ops", "plan-ops", "desc")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	createPath := "/products/" + strconv.Itoa(int(product.ID)) + "/plans"
	for _, body := range []string{
		`{"name":"Starter","constraintValues":{},"sortOrder":1}`,
		`{"name":"Growth","constraintValues":{},"sortOrder":2}`,
	} {
		req := httptest.NewRequest(http.MethodPost, createPath, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("seed create plan status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
	}

	t.Run("set default can switch and clear current default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/plans/1/default", bytes.NewReader([]byte(`{"isDefault":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("set first default status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/plans/2/default", bytes.NewReader([]byte(`{"isDefault":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("switch default status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/plans/2/default", bytes.NewReader([]byte(`{"isDefault":false}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("clear default status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		var count int64
		if err := productSvc.db.Model(&domain.Plan{}).Where("product_id = ? AND is_default = ?", product.ID, true).Count(&count).Error; err != nil {
			t.Fatalf("count default plans: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no default plans after clearing, got %d", count)
		}
	})

	t.Run("bad ids and payloads are rejected consistently", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products/bad/plans", bytes.NewReader([]byte(`{"name":"Ghost"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid product id create plan status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPut, "/plans/bad", bytes.NewReader([]byte(`{"name":"Ghost"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid plan id update status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPatch, "/plans/bad/default", bytes.NewReader([]byte(`{"isDefault":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid plan id default status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/products/bad/rotate-key-impact", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid rotate-key-impact id status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/products/bad/bulk-reissue", bytes.NewReader([]byte(`{"licenseIds":[1]}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid bulk-reissue id status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/bulk-reissue", bytes.NewReader([]byte(`{"licenseIds":"bad"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid bulk-reissue payload status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})
}
