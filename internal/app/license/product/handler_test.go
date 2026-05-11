package product

import (
	"bytes"
	"encoding/json"
	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeLicenseOperations struct {
	impact      *domain.RotateKeyImpact
	impactErr   error
	bulkErr     error
	bulkResult  int
	lastProduct uint
	lastIDs     []uint
	lastUserID  uint
}

func (f fakeLicenseOperations) AssessKeyRotationImpact(productID uint) (*domain.RotateKeyImpact, error) {
	if f.impact != nil || f.impactErr != nil {
		return f.impact, f.impactErr
	}
	return &domain.RotateKeyImpact{}, nil
}

func (f fakeLicenseOperations) BulkReissueLicenses(productID uint, ids []uint, issuedBy uint) (int, error) {
	f.lastProduct = productID
	f.lastIDs = append([]uint(nil), ids...)
	f.lastUserID = issuedBy
	if f.bulkErr != nil {
		return 0, f.bulkErr
	}
	return f.bulkResult, nil
}

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestProductHandler_Get_404(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	h := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{}}

	r := setupGin()
	r.GET("/products/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestProductHandler_UpdateSchema_400(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, _ := productSvc.CreateProduct("domain.Product", "prod-schema", "")
	h := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{}}

	r := setupGin()
	r.PUT("/products/:id/schema", h.UpdateSchema)

	body, _ := json.Marshal(map[string]any{"constraintSchema": "not-valid-json"})
	req := httptest.NewRequest(http.MethodPut, "/products/"+strconv.Itoa(int(product.ID))+"/schema", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestProductHandler_BulkReissue_400(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, _ := productSvc.CreateProduct("domain.Product", "prod-bulk", "")
	h := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{bulkErr: domain.ErrBulkReissueTooMany}}

	r := setupGin()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	r.POST("/products/:id/bulk-reissue", h.BulkReissue)

	manyIDs := make([]uint, 101)
	for i := range manyIDs {
		manyIDs[i] = uint(i + 1)
	}
	body, _ := json.Marshal(map[string]any{"licenseIds": manyIDs})
	req := httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/bulk-reissue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestProductHandler_CreateAndStatusConflicts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	if _, err := productSvc.CreateProduct("Metis", "metis-dup", ""); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	h := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{}}

	r := setupGin()
	r.POST("/products", h.Create)
	r.PATCH("/products/:id/status", h.UpdateStatus)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"name":"Metis 2","code":"metis-dup","description":"dup"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("duplicate create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/products/1/status", bytes.NewReader([]byte(`{"status":"bogus"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid status transition status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestProductHandler_KeyLookupErrors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, err := productSvc.CreateProduct("Metis", "metis-key", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	var key domain.ProductKey
	if err := db.Where("product_id = ? AND is_current = ?", product.ID, true).First(&key).Error; err != nil {
		t.Fatalf("find current key: %v", err)
	}
	if err := db.Delete(&key).Error; err != nil {
		t.Fatalf("delete current key: %v", err)
	}

	h := &ProductHandler{productSvc: productSvc, licenseSvc: fakeLicenseOperations{}}
	r := setupGin()
	r.GET("/products/:id/public-key", h.GetPublicKey)
	r.GET("/products/:id/rotate-key-impact", h.RotateKeyImpact)

	req := httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID))+"/public-key", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing public key status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID))+"/rotate-key-impact", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("rotate key impact without key status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
}

func TestProductHandler_RotateKeyImpactAndBulkReissueHappyPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, err := productSvc.CreateProduct("Metis", "metis-ops", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	ops := fakeLicenseOperations{
		impact:     &domain.RotateKeyImpact{AffectedCount: 3, CurrentVersion: 2},
		bulkResult: 2,
	}
	h := &ProductHandler{productSvc: productSvc, licenseSvc: ops}

	r := setupGin()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(9))
		c.Next()
	})
	r.GET("/products/:id/rotate-key-impact", h.RotateKeyImpact)
	r.POST("/products/:id/bulk-reissue", h.BulkReissue)

	req := httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID))+"/rotate-key-impact", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("rotate key impact status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"affectedCount":3`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"currentVersion":2`)) {
		t.Fatalf("unexpected rotate key impact body: %s", w.Body.String())
	}

	body, _ := json.Marshal(map[string]any{"licenseIds": []uint{11, 12}})
	req = httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/bulk-reissue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("bulk reissue status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"reissued":2`)) {
		t.Fatalf("unexpected bulk reissue body: %s", w.Body.String())
	}
}

func TestProductHandler_RotateKeyImpactAndBulkReissueKeyErrors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, err := productSvc.CreateProduct("Metis", "metis-key-guard", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	h := &ProductHandler{
		productSvc: productSvc,
		licenseSvc: fakeLicenseOperations{
			impactErr: domain.ErrProductKeyNotFound,
			bulkErr:   domain.ErrProductKeyNotFound,
		},
	}
	r := setupGin()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(3))
		c.Next()
	})
	r.GET("/products/:id/rotate-key-impact", h.RotateKeyImpact)
	r.POST("/products/:id/bulk-reissue", h.BulkReissue)

	req := httptest.NewRequest(http.MethodGet, "/products/"+strconv.Itoa(int(product.ID))+"/rotate-key-impact", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("rotate key impact missing key status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/products/"+strconv.Itoa(int(product.ID))+"/bulk-reissue", bytes.NewReader([]byte(`{"licenseIds":[1]}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bulk reissue missing key status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}
