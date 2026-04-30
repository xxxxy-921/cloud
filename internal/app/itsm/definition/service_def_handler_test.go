package definition

import (
	. "metis/internal/app/itsm/domain"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"metis/internal/handler"
)

func TestServiceDefHandlerCreate_Returns400ForMissingCatalog(t *testing.T) {
	db := newTestDB(t)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"VPN","code":"vpn","catalogId":999,"engineType":"classic"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerUpdate_ClearsSLAWithExplicitNull(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	slaID := uint(42)
	service, err := svc.Create(&ServiceDefinition{Name: "VPN", Code: "vpn", CatalogID: root.ID, EngineType: "classic", SLAID: &slaID})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), []byte(`{"slaId":null}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	updated, err := svc.Get(service.ID)
	if err != nil {
		t.Fatalf("get updated service: %v", err)
	}
	if updated.SLAID != nil {
		t.Fatalf("expected slaId to be cleared, got %v", *updated.SLAID)
	}
}

func TestServiceDefHandlerUpdate_LeavesSLAWhenFieldOmitted(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	slaID := uint(42)
	service, err := svc.Create(&ServiceDefinition{Name: "VPN", Code: "vpn", CatalogID: root.ID, EngineType: "classic", SLAID: &slaID})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), []byte(`{"description":"updated"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	updated, err := svc.Get(service.ID)
	if err != nil {
		t.Fatalf("get updated service: %v", err)
	}
	if updated.SLAID == nil || *updated.SLAID != slaID {
		t.Fatalf("expected slaId to remain %d, got %v", slaID, updated.SLAID)
	}
}

func TestServiceDefHandlerList_ParsesEngineTypeFilter(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if _, err := svc.Create(&ServiceDefinition{Name: "Classic", Code: "classic", CatalogID: root.ID, EngineType: "classic"}); err != nil {
		t.Fatalf("create classic: %v", err)
	}
	if _, err := svc.Create(&ServiceDefinition{Name: "Smart", Code: "smart", CatalogID: root.ID, EngineType: "smart"}); err != nil {
		t.Fatalf("create smart: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services", h.List)
	}, http.MethodGet, "/services?engineType=smart", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeResponseBody[handler.R](t, rec)
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", resp.Data)
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 filtered item, got %#v", data["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok || first["engineType"] != "smart" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
}
