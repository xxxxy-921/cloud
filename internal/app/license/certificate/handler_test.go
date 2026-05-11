package certificate

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"metis/internal/app/license/domain"
	licenseepkg "metis/internal/app/license/licensee"
	productpkg "metis/internal/app/license/product"
	"metis/internal/app/license/testutil"
)

func setupLicenseHandlerEnv(t *testing.T) (*LicenseHandler, *productpkg.ProductService, *licenseepkg.LicenseeService, *LicenseService, *gin.Engine) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)
	handler := &LicenseHandler{licenseSvc: licenseSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	r.POST("/licenses", handler.Issue)
	r.GET("/licenses", handler.List)
	r.GET("/licenses/:id", handler.Get)
	r.PATCH("/licenses/:id/revoke", handler.Revoke)
	r.GET("/licenses/:id/export", handler.Export)
	r.POST("/licenses/:id/renew", handler.Renew)
	r.POST("/licenses/:id/upgrade", handler.Upgrade)
	r.POST("/licenses/:id/suspend", handler.Suspend)
	r.POST("/licenses/:id/reactivate", handler.Reactivate)
	r.POST("/registrations", handler.CreateRegistration)
	r.GET("/registrations", handler.ListRegistrations)
	r.POST("/registrations/generate", handler.GenerateRegistration)

	return handler, productSvc, licenseeSvc, licenseSvc, r
}

func TestLicenseHandlerLifecycleAndRegistrationEndpoints(t *testing.T) {
	_, productSvc, licenseeSvc, licenseSvc, r := setupLicenseHandlerEnv(t)

	product, err := productSvc.CreateProduct("Metis", "metis-handler", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}
	var primaryLicenseID uint

	t.Run("create and generate registrations", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"productId":1,"licenseeId":1,"code":"RG-HANDLER-1"}`))
		req := httptest.NewRequest(http.MethodPost, "/registrations", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create registration status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/registrations/generate", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("generate registration status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/registrations?available=true&page=1&pageSize=20", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list registration status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/registrations/generate", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("generate registration with refs status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"productId":1`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"licenseeId":1`)) {
			t.Fatalf("expected generated registration refs in body, got %s", w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"code":"   "}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create blank registration code status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"code":"RG-`)) {
			t.Fatalf("expected blank registration code to auto generate RG- prefix, body=%s", w.Body.String())
		}
	})

	t.Run("issue and list and get license", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "Basic",
			"registrationCode": "RG-HANDLER-1",
			"validFrom":        time.Now().Add(-time.Hour).Format(time.RFC3339),
		})
		req := httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("issue status = %d, want 200, body=%s", w.Code, w.Body.String())
		}

		items, total, err := licenseSvc.ListLicenses(LicenseListParams{Keyword: "RG-HANDLER-1", Page: 1, PageSize: 10})
		if err != nil || total != 1 {
			t.Fatalf("load issued license by registration: total=%d err=%v items=%+v", total, err, items)
		}
		primaryLicenseID = items[0].ID

		req = httptest.NewRequest(http.MethodGet, "/licenses?keyword=RG-HANDLER-1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list licenses status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10), nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("get license status = %d, want 200", w.Code)
		}
	})

	t.Run("issue can auto create missing registration when enabled", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"productId":              product.ID,
			"licenseeId":             licensee.ID,
			"planName":               "AutoCreate",
			"registrationCode":       "RG-HANDLER-AUTO-1",
			"autoCreateRegistration": true,
			"validFrom":              time.Now().Add(-time.Hour).Format(time.RFC3339),
		})
		req := httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("auto-create issue status = %d, want 200, body=%s", w.Code, w.Body.String())
		}

		reg, err := licenseSvc.regRepo.FindByCode("RG-HANDLER-AUTO-1")
		if err != nil {
			t.Fatalf("load auto-created registration: %v", err)
		}
		if reg.Source != "manual_input" || reg.BoundLicenseID == nil {
			t.Fatalf("unexpected auto-created registration: %+v", reg)
		}
	})

	t.Run("issue rejects blank registration code even when auto create is enabled", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"productId":              product.ID,
			"licenseeId":             licensee.ID,
			"planName":               "AutoCreate",
			"registrationCode":       "   ",
			"autoCreateRegistration": true,
			"validFrom":              time.Now().Add(-time.Hour).Format(time.RFC3339),
		})
		req := httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("blank registration auto-create issue status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("renew export suspend reactivate revoke", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/renew", bytes.NewReader([]byte(`{"validUntil":"2030-01-01"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("renew status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/export?format=v1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("export status = %d, want 200", w.Code)
		}
		if got := w.Header().Get("Content-Disposition"); got == "" {
			t.Fatal("expected Content-Disposition header")
		}

		req = httptest.NewRequest(http.MethodGet, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/export?format=V2", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("export uppercase v2 status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/export?format=weird", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid export format status = %d, want 400 body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/suspend", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("suspend status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/reactivate", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("reactivate status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPatch, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/revoke", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("revoke status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPatch, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/revoke", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("double revoke status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/suspend", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("suspend revoked status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/reactivate", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("reactivate revoked status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/export?format=v1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("revoked export status = %d, want 400", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/renew", bytes.NewReader([]byte(`{"validUntil":"2031-01-01"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("renew revoked status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("upgrade bad date and invalid id are rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/licenses/bad/reactivate", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid id status = %d, want 400", w.Code)
		}

		body, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "Pro",
			"registrationCode": "RG-HANDLER-2",
			"validFrom":        "bad-date",
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses/1/upgrade", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad upgrade date status = %d, want 400", w.Code)
		}

		body, _ = json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "MissingSource",
			"registrationCode": "RG-HANDLER-404",
			"validFrom":        time.Now().Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses/999/upgrade", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("missing upgrade source status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("upgrade issues replacement license and rebinds registration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1,"code":"RG-HANDLER-3"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create upgrade registration status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		body, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "Growth",
			"registrationCode": "RG-HANDLER-3",
			"validFrom":        time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("issue upgrade source license status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		sourceItems, total, err := licenseSvc.ListLicenses(LicenseListParams{Keyword: "RG-HANDLER-3", Page: 1, PageSize: 10})
		if err != nil || total != 1 {
			t.Fatalf("load source license by registration: total=%d err=%v items=%+v", total, err, sourceItems)
		}
		sourceID := sourceItems[0].ID

		req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1,"code":"RG-HANDLER-4"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create replacement registration status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		upgradeBody, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "Enterprise",
			"registrationCode": "RG-HANDLER-4",
			"validFrom":        time.Now().Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(sourceID), 10)+"/upgrade", bytes.NewReader(upgradeBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("upgrade status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		upgradedItems, total, err := licenseSvc.ListLicenses(LicenseListParams{Keyword: "RG-HANDLER-4", Page: 1, PageSize: 10})
		if err != nil || total != 1 {
			t.Fatalf("load upgraded license by registration: total=%d err=%v items=%+v", total, err, upgradedItems)
		}
		detail, err := licenseSvc.GetLicense(upgradedItems[0].ID)
		if err != nil {
			t.Fatalf("load upgraded license: %v", err)
		}
		if detail.License.PlanName != "Enterprise" || detail.License.OriginalLicenseID == nil || *detail.License.OriginalLicenseID != sourceID {
			t.Fatalf("unexpected upgraded license detail: %+v", detail.License)
		}
		original, err := licenseSvc.GetLicense(sourceID)
		if err != nil {
			t.Fatalf("load original upgraded-from license: %v", err)
		}
		if original.License.LifecycleStatus != domain.LicenseLifecycleRevoked {
			t.Fatalf("expected original license revoked after upgrade, got %+v", original.License)
		}
	})

	t.Run("upgrade preserves suspended lifecycle on replacement license", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1,"code":"RG-HANDLER-UPG-SUSP-1"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create suspended-upgrade registration status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		body, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "SuspendSource",
			"registrationCode": "RG-HANDLER-UPG-SUSP-1",
			"validFrom":        time.Now().Add(-time.Hour).Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("issue suspended upgrade source status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		sourceItems, total, err := licenseSvc.ListLicenses(LicenseListParams{Keyword: "RG-HANDLER-UPG-SUSP-1", Page: 1, PageSize: 10})
		if err != nil || total != 1 {
			t.Fatalf("load suspended upgrade source by registration: total=%d err=%v items=%+v", total, err, sourceItems)
		}
		sourceID := sourceItems[0].ID
		if err := licenseSvc.SuspendLicense(sourceID, 11); err != nil {
			t.Fatalf("suspend source before upgrade: %v", err)
		}

		upgradeBody, _ := json.Marshal(map[string]any{
			"productId":        product.ID,
			"licenseeId":       licensee.ID,
			"planName":         "SuspendTarget",
			"registrationCode": "RG-HANDLER-UPG-SUSP-1",
			"validFrom":        time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(sourceID), 10)+"/upgrade", bytes.NewReader(upgradeBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("upgrade suspended source status = %d, want 200 body=%s", w.Code, w.Body.String())
		}

		upgradedItems, total, err := licenseSvc.ListLicenses(LicenseListParams{Keyword: "SuspendTarget", Page: 1, PageSize: 10})
		if err != nil || total != 1 {
			t.Fatalf("load suspended upgraded license: total=%d err=%v items=%+v", total, err, upgradedItems)
		}
		upgradedDetail, err := licenseSvc.GetLicense(upgradedItems[0].ID)
		if err != nil {
			t.Fatalf("load suspended upgraded detail: %v", err)
		}
		if upgradedDetail.License.LifecycleStatus != domain.LicenseLifecycleSuspended {
			t.Fatalf("expected upgraded license to stay suspended, got %+v", upgradedDetail.License)
		}
		if upgradedDetail.License.SuspendedBy == nil || *upgradedDetail.License.SuspendedBy != 11 {
			t.Fatalf("expected upgraded suspension metadata preserved, got %+v", upgradedDetail.License)
		}
	})

	t.Run("service level helpers still work", func(t *testing.T) {
		impact, err := licenseSvc.AssessKeyRotationImpact(product.ID)
		if err != nil {
			t.Fatalf("AssessKeyRotationImpact: %v", err)
		}
		if impact.CurrentVersion == 0 {
			t.Fatal("expected current version to be set")
		}
	})

	t.Run("available registrations exclude bound codes and upgrade rejects revoked source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/registrations?available=true&page=1&pageSize=50", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("available registrations status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if bytes.Contains([]byte(body), []byte(`RG-HANDLER-1`)) || bytes.Contains([]byte(body), []byte(`RG-HANDLER-AUTO-1`)) {
			t.Fatalf("expected bound registrations to be filtered from available list, body=%s", body)
		}

		upgradeBody, _ := json.Marshal(map[string]any{
			"productId":              product.ID,
			"licenseeId":             licensee.ID,
			"planName":               "RevokedUpgrade",
			"registrationCode":       "RG-HANDLER-UPG-REVOKED",
			"autoCreateRegistration": true,
			"validFrom":              time.Now().Format(time.RFC3339),
		})
		req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(primaryLicenseID), 10)+"/upgrade", bytes.NewReader(upgradeBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("upgrade revoked source status = %d, want 400 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("list filters by lifecycle status and scoped ids", func(t *testing.T) {
		suspendedReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-HANDLER-LIST-SUSPEND",
		})
		if err != nil {
			t.Fatalf("create suspended registration: %v", err)
		}
		suspendedLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "ListSuspended",
			RegistrationCode: suspendedReg.Code,
			ValidFrom:        time.Now().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue suspended list license: %v", err)
		}
		if err := licenseSvc.SuspendLicense(suspendedLicense.ID, 2); err != nil {
			t.Fatalf("suspend list license: %v", err)
		}

		otherLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Scoped Other"})
		if err != nil {
			t.Fatalf("create scoped other licensee: %v", err)
		}
		otherReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &otherLicensee.ID,
			Code:       "RG-HANDLER-LIST-OTHER",
		})
		if err != nil {
			t.Fatalf("create other scoped registration: %v", err)
		}
		if _, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       otherLicensee.ID,
			PlanName:         "ScopedOther",
			RegistrationCode: otherReg.Code,
			ValidFrom:        time.Now().Add(-time.Hour),
			IssuedBy:         1,
		}); err != nil {
			t.Fatalf("issue other scoped license: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/licenses?productId="+strconv.FormatUint(uint64(product.ID), 10)+"&licenseeId="+strconv.FormatUint(uint64(licensee.ID), 10)+"&lifecycleStatus="+domain.LicenseLifecycleSuspended+"&keyword=RG-HANDLER-LIST-SUSPEND&page=1&pageSize=20", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("filtered license list status = %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"total":1`)) || !bytes.Contains(w.Body.Bytes(), []byte(`RG-HANDLER-LIST-SUSPEND`)) {
			t.Fatalf("expected suspended scoped license only, body=%s", w.Body.String())
		}
		if bytes.Contains(w.Body.Bytes(), []byte(`RG-HANDLER-LIST-OTHER`)) {
			t.Fatalf("expected other licensee record to be filtered out, body=%s", w.Body.String())
		}
	})

	t.Run("list rejects explicit zero scoped ids", func(t *testing.T) {
		for _, path := range []string{
			"/licenses?productId=0",
			"/licenses?licenseeId=0",
			"/registrations?productId=0",
			"/registrations?licenseeId=0",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s status = %d, want 400 body=%s", path, w.Code, w.Body.String())
			}
		}
	})
}

func TestLicenseHandlerDateParsers(t *testing.T) {
	got, err := parseLicenseDate("2030-01-02")
	if err != nil || got.Year() != 2030 || got.Month() != time.January {
		t.Fatalf("parse date = %v, %v", got, err)
	}
	opt, err := parseOptionalLicenseDate(nil)
	if err != nil || opt != nil {
		t.Fatalf("parse nil optional date = %v, %v", opt, err)
	}
	value := "2030-02-03T10:20:30Z"
	opt, err = parseOptionalLicenseDate(&value)
	if err != nil || opt == nil || opt.Year() != 2030 {
		t.Fatalf("parse optional date = %v, %v", opt, err)
	}
}

func TestLicenseHandlerRejectsInvalidDatesAndMissingResources(t *testing.T) {
	_, productSvc, licenseeSvc, licenseSvc, r := setupLicenseHandlerEnv(t)

	product, err := productSvc.CreateProduct("Metis", "metis-handler-guards", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":1,"licenseeId":1,"planName":"Basic","registrationCode":"RG-GUARD-001","validFrom":"bad-date"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid issue date status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid issue payload status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":999999,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"MissingProduct","registrationCode":"RG-GUARD-MISSING-PRODUCT","validFrom":"2030-01-01T00:00:00Z","autoCreateRegistration":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing product issue status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":999999,"planName":"MissingLicensee","registrationCode":"RG-GUARD-MISSING-LICENSEE","validFrom":"2030-01-01T00:00:00Z","autoCreateRegistration":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing licensee issue status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"BadWindow","registrationCode":"RG-GUARD-BAD-WINDOW","validFrom":"2030-01-02T00:00:00Z","validUntil":"2030-01-01T00:00:00Z","autoCreateRegistration":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid issue window status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/999/renew", bytes.NewReader([]byte(`{"validUntil":"2030-01-01"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing renew status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/1/renew", bytes.NewReader([]byte(`{"validUntil":`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid renew payload status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/1/renew", bytes.NewReader([]byte(`{"validUntil":"not-a-date"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid renew date status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-GUARD-RENEW-BAD-WINDOW",
	})
	if err != nil {
		t.Fatalf("create renew registration: %v", err)
	}
	renewValidFrom, err := parseLicenseDate("2030-01-02T00:00:00Z")
	if err != nil {
		t.Fatalf("parse renew validFrom: %v", err)
	}
	renewValidUntil, err := parseLicenseDate("2030-01-03T00:00:00Z")
	if err != nil {
		t.Fatalf("parse renew validUntil: %v", err)
	}
	renewTarget, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Renew Guard",
		RegistrationCode: reg.Code,
		ValidFrom:        renewValidFrom,
		ValidUntil:       ptrTime(renewValidUntil),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue renew target: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(renewTarget.ID), 10)+"/renew", bytes.NewReader([]byte(`{"validUntil":"2030-01-02T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid renew window status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing get status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses/not-a-number", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid get id status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses/999/export?format=v1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing export status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?productId=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses productId status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?page=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses page status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?pageSize=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses pageSize status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?licenseeId=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses licenseeId status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?status=weird", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses status status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?lifecycleStatus=weird", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid licenses lifecycleStatus status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?status=all&lifecycleStatus=all", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("licenses status=all lifecycleStatus=all status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses?status=ALL&lifecycleStatus=ALL", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("licenses status=ALL lifecycleStatus=ALL status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/licenses/999/revoke", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing revoke status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/999/suspend", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing suspend status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/999/reactivate", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing reactivate status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/licenses/not-a-number/revoke", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid revoke id status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/licenses/not-a-number/export?format=v1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid export id status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/not-a-number/suspend", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid suspend id status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/not-a-number/reactivate", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid reactivate id status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"code":"RG-GUARD-REG-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create registration status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"code":"RG-GUARD-REG-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("duplicate registration status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/registrations?productId="+strconv.FormatUint(uint64(product.ID), 10)+"&licenseeId="+strconv.FormatUint(uint64(licensee.ID), 10)+"&page=1&pageSize=20", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("filtered registration list status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":"RG-GUARD-REG-1"`)) {
		t.Fatalf("expected filtered registration list to contain created code, body=%s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/registrations?productId=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid registrations productId status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/registrations?page=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid registrations page status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/registrations?pageSize=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid registrations pageSize status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/registrations?licenseeId=bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid registrations licenseeId status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":999999,"code":"RG-GUARD-MISSING-PRODUCT-REG"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing product registration create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/registrations/generate", bytes.NewReader([]byte(`{"licenseeId":999999}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing licensee registration generate status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/registrations/generate", bytes.NewReader([]byte(`{"productId":"bad","licenseeId":1}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid generate registration payload status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	unpublished, err := productSvc.CreateProduct("Draft", "draft-handler-guard", "")
	if err != nil {
		t.Fatalf("create unpublished product: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/registrations", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(unpublished.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"code":"RG-GUARD-DRAFT-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create unpublished registration status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(unpublished.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"Draft","registrationCode":"RG-GUARD-DRAFT-1","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("issue unpublished product status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	boundReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-GUARD-BOUND-1",
	})
	if err != nil {
		t.Fatalf("create bound registration: %v", err)
	}
	if _, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Bound",
		RegistrationCode: boundReg.Code,
		ValidFrom:        time.Now().Add(-time.Hour),
		IssuedBy:         1,
	}); err != nil {
		t.Fatalf("issue bound registration seed license: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"BoundAgain","registrationCode":"RG-GUARD-BOUND-1","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("issue already bound registration status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	foreignProduct, err := productSvc.CreateProduct("Foreign Product", "foreign-product", "")
	if err != nil {
		t.Fatalf("create foreign product: %v", err)
	}
	if err := productSvc.UpdateStatus(foreignProduct.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish foreign product: %v", err)
	}
	foreignLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Foreign Licensee"})
	if err != nil {
		t.Fatalf("create foreign licensee: %v", err)
	}
	foreignReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &foreignProduct.ID,
		LicenseeID: &foreignLicensee.ID,
		Code:       "RG-GUARD-FOREIGN-1",
	})
	if err != nil {
		t.Fatalf("create foreign registration: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/licenses", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"ForeignScope","registrationCode":"`+foreignReg.Code+`","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("issue foreign-scoped registration status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	upgradeSourceReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-GUARD-UPGRADE-SRC-1",
	})
	if err != nil {
		t.Fatalf("create upgrade source registration: %v", err)
	}
	upgradeSource, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "UpgradeSource",
		RegistrationCode: upgradeSourceReg.Code,
		ValidFrom:        time.Now().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue upgrade source license: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(upgradeSource.ID), 10)+"/upgrade", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"BadUntil","registrationCode":"RG-UPGRADE-BAD-UNTIL","validFrom":"2030-01-01T00:00:00Z","validUntil":"bad-date"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upgrade invalid validUntil status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(upgradeSource.ID), 10)+"/upgrade", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"MissingRegistration","registrationCode":"RG-UPGRADE-MISSING-REG","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upgrade missing registration status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(upgradeSource.ID), 10)+"/upgrade", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"BadWindow","registrationCode":"RG-UPGRADE-BAD-WINDOW","validFrom":"2030-01-02T00:00:00Z","validUntil":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upgrade invalid window status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(upgradeSource.ID), 10)+"/upgrade", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(product.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(licensee.ID), 10)+`,"planName":"ForeignScope","registrationCode":"RG-GUARD-FOREIGN-1","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upgrade foreign-scoped registration status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	crossScopeProduct, err := productSvc.CreateProduct("Cross Scope", "prod-upgrade-cross-scope", "")
	if err != nil {
		t.Fatalf("create cross-scope product: %v", err)
	}
	if err := productSvc.UpdateStatus(crossScopeProduct.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish cross-scope product: %v", err)
	}
	crossScopeLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Cross Scope Corp"})
	if err != nil {
		t.Fatalf("create cross-scope licensee: %v", err)
	}
	crossScopeReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &crossScopeProduct.ID,
		LicenseeID: &crossScopeLicensee.ID,
		Code:       "RG-UPGRADE-CROSS-SCOPE-001",
	})
	if err != nil {
		t.Fatalf("create cross-scope registration: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(upgradeSource.ID), 10)+"/upgrade", bytes.NewReader([]byte(`{"productId":`+strconv.FormatUint(uint64(crossScopeProduct.ID), 10)+`,"licenseeId":`+strconv.FormatUint(uint64(crossScopeLicensee.ID), 10)+`,"planName":"CrossScope","registrationCode":"`+crossScopeReg.Code+`","validFrom":"2030-01-01T00:00:00Z"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upgrade cross-scope target status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	renewableReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-GUARD-RENEW-1",
	})
	if err != nil {
		t.Fatalf("create renewable registration: %v", err)
	}
	renewable, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Renewable",
		RegistrationCode: renewableReg.Code,
		ValidFrom:        time.Now().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue renewable license: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(renewable.ID), 10)+"/reactivate", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("reactivate non-suspended status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	currentKey, err := licenseSvc.keyRepo.FindCurrentByProductID(product.ID)
	if err != nil {
		t.Fatalf("find current product key: %v", err)
	}
	if err := licenseSvc.db.Delete(&domain.ProductKey{}, currentKey.ID).Error; err != nil {
		t.Fatalf("delete current product key: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/licenses/"+strconv.FormatUint(uint64(renewable.ID), 10)+"/renew", bytes.NewReader([]byte(`{"validUntil":"2032-01-01"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("renew missing key status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}
