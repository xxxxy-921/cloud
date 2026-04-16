package license

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestProductHandler_Get_404(t *testing.T) {
	db := setupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{db: db},
		planRepo:         &PlanRepo{db: db},
		keyRepo:          &ProductKeyRepo{db: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	h := &ProductHandler{productSvc: productSvc, licenseSvc: &LicenseService{}}

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
	db := setupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{db: db},
		planRepo:         &PlanRepo{db: db},
		keyRepo:          &ProductKeyRepo{db: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	product, _ := productSvc.CreateProduct("Product", "prod-schema", "")
	h := &ProductHandler{productSvc: productSvc, licenseSvc: &LicenseService{}}

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
	db := setupTestDB(t)
	productSvc := &ProductService{
		productRepo:      &ProductRepo{db: db},
		planRepo:         &PlanRepo{db: db},
		keyRepo:          &ProductKeyRepo{db: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
	licenseSvc := newLicenseService(db)
	product, _ := productSvc.CreateProduct("Product", "prod-bulk", "")
	h := &ProductHandler{productSvc: productSvc, licenseSvc: licenseSvc}

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
