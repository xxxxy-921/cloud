package definition

import (
	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newServiceActionServiceForTest(t *testing.T, db *gorm.DB, serviceDefs *ServiceDefService) *ServiceActionService {
	t.Helper()
	return &ServiceActionService{
		repo:        &ServiceActionRepo{db: &database.DB{DB: db}},
		serviceDefs: serviceDefs,
	}
}

func TestServiceActionHandlerUpdateAndDelete_RequireParentServiceMatch(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	serviceA, err := serviceDefs.Create(&ServiceDefinition{Name: "A", Code: "svc-a", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service A: %v", err)
	}
	serviceB, err := serviceDefs.Create(&ServiceDefinition{Name: "B", Code: "svc-b", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service B: %v", err)
	}
	actionB, err := actionSvc.Create(&ServiceAction{
		Name:       "Notify",
		Code:       "notify",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/hook","method":"POST","timeout":30,"retries":3}`),
		ServiceID:  serviceB.ID,
	})
	if err != nil {
		t.Fatalf("create action B: %v", err)
	}

	updatePath := "/services/" + strconv.FormatUint(uint64(serviceA.ID), 10) + "/actions/" + strconv.FormatUint(uint64(actionB.ID), 10)
	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id/actions/:actionId", h.Update)
	}, http.MethodPut, updatePath, []byte(`{"name":"cross-service-update"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-service update to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	deletePath := "/services/" + strconv.FormatUint(uint64(serviceA.ID), 10) + "/actions/" + strconv.FormatUint(uint64(actionB.ID), 10)
	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/services/:id/actions/:actionId", h.Delete)
	}, http.MethodDelete, deletePath, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-service delete to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceActionHandlerCreate_ValidatesHTTPConfig(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "Webhook", Code: "webhook", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "non-http action type", body: `{"name":"Bad","code":"bad-type","actionType":"shell","configJson":{"url":"https://example.com"}}`},
		{name: "invalid url", body: `{"name":"Bad","code":"bad-url","actionType":"http","configJson":{"url":"ftp://example.com"}}`},
		{name: "invalid method", body: `{"name":"Bad","code":"bad-method","actionType":"http","configJson":{"url":"https://example.com","method":"TRACE"}}`},
		{name: "timeout too large", body: `{"name":"Bad","code":"bad-timeout","actionType":"http","configJson":{"url":"https://example.com","timeout":121}}`},
		{name: "retries too large", body: `{"name":"Bad","code":"bad-retries","actionType":"http","configJson":{"url":"https://example.com","retries":6}}`},
		{name: "header injection", body: `{"name":"Bad","code":"bad-header","actionType":"http","configJson":{"url":"https://example.com","headers":{"X-Test":"ok\nbad"}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/services/" + strconv.FormatUint(uint64(service.ID), 10) + "/actions"
			rec := performJSONRequest(t, func(r *gin.Engine) {
				r.POST("/services/:id/actions", h.Create)
			}, http.MethodPost, path, []byte(tt.body))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestServiceActionHandlerCreate_RejectsBlankBusinessIdentifiers(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root-blank-action", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "Webhook", Code: "webhook-blank", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	path := "/services/" + strconv.FormatUint(uint64(service.ID), 10) + "/actions"
	for name, body := range map[string]string{
		"blank name": `{"name":"   ","code":"notify-http","actionType":"http","configJson":{"url":"https://example.com"}}`,
		"blank code": `{"name":"Notify","code":"   ","actionType":"http","configJson":{"url":"https://example.com"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := performJSONRequest(t, func(r *gin.Engine) {
				r.POST("/services/:id/actions", h.Create)
			}, http.MethodPost, path, []byte(body))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestServiceActionHandlerList_ReturnsActionsScopedToService(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root-list-actions", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	serviceA, err := serviceDefs.Create(&ServiceDefinition{Name: "A", Code: "svc-list-a", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service A: %v", err)
	}
	serviceB, err := serviceDefs.Create(&ServiceDefinition{Name: "B", Code: "svc-list-b", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service B: %v", err)
	}
	if _, err := actionSvc.Create(&ServiceAction{Name: "Notify A", Code: "notify-a", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/a"}`), ServiceID: serviceA.ID}); err != nil {
		t.Fatalf("create action A: %v", err)
	}
	if _, err := actionSvc.Create(&ServiceAction{Name: "Notify B", Code: "notify-b", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/b"}`), ServiceID: serviceB.ID}); err != nil {
		t.Fatalf("create action B: %v", err)
	}

	path := "/services/" + strconv.FormatUint(uint64(serviceA.ID), 10) + "/actions"
	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/actions", h.List)
	}, http.MethodGet, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"notify-a"`) || strings.Contains(rec.Body.String(), `"code":"notify-b"`) {
		t.Fatalf("unexpected service action list response: %s", rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/actions", h.List)
	}, http.MethodGet, "/services/not-a-number/actions", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid id to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceActionHandlerRejectsMissingServiceAndDuplicateCodes(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root-action-conflict", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "A", Code: "svc-action-conflict", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if _, err := actionSvc.Create(&ServiceAction{Name: "Notify", Code: "notify", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/hook"}`), ServiceID: service.ID}); err != nil {
		t.Fatalf("seed action: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/actions", h.List)
	}, http.MethodGet, "/services/999/actions", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing service list to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	path := "/services/" + strconv.FormatUint(uint64(service.ID), 10) + "/actions"
	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services/:id/actions", h.Create)
	}, http.MethodPost, path, []byte(`{"name":"Notify Again","code":"notify","actionType":"http","configJson":{"url":"https://example.com/dup"}}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate action code to return 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceActionHandlerUpdateAndDeleteSuccessContracts(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)
	h := &ServiceActionHandler{svc: actionSvc}

	root, err := catSvc.Create("Root", "root-action-update", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "A", Code: "svc-action-update", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	action, err := actionSvc.Create(&ServiceAction{
		Name:       "Notify",
		Code:       "notify",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/hook","method":"POST","timeout":30,"retries":3}`),
		ServiceID:  service.ID,
	})
	if err != nil {
		t.Fatalf("create action: %v", err)
	}

	updatePath := "/services/" + strconv.FormatUint(uint64(service.ID), 10) + "/actions/" + strconv.FormatUint(uint64(action.ID), 10)
	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id/actions/:actionId", h.Update)
	}, http.MethodPut, updatePath, []byte(`{"name":"Notify Updated","configJson":{"url":"https://example.com/new","method":"POST","timeout":60,"retries":2},"isActive":false}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected update success, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"Notify Updated"`) || !strings.Contains(rec.Body.String(), `"isActive":false`) {
		t.Fatalf("unexpected update response body=%s", rec.Body.String())
	}

	reloaded, err := actionSvc.GetByService(service.ID, action.ID)
	if err != nil {
		t.Fatalf("reload action: %v", err)
	}
	if reloaded.Name != "Notify Updated" || reloaded.IsActive || !strings.Contains(string(reloaded.ConfigJSON), "https://example.com/new") {
		t.Fatalf("unexpected reloaded action: %+v", reloaded)
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id/actions/:actionId", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/actions/bad", []byte(`{"name":"oops"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid action id to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id/actions/:actionId", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/actions/999999", []byte(`{"name":"ghost"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing action update to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	deletePath := "/services/" + strconv.FormatUint(uint64(service.ID), 10) + "/actions/" + strconv.FormatUint(uint64(action.ID), 10)
	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/services/:id/actions/:actionId", h.Delete)
	}, http.MethodDelete, deletePath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected delete success, got %d body=%s", rec.Code, rec.Body.String())
	}

	if _, err := actionSvc.GetByService(service.ID, action.ID); err == nil {
		t.Fatal("expected deleted action to be unavailable")
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/services/:id/actions/:actionId", h.Delete)
	}, http.MethodDelete, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/actions/999999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing action delete to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}
