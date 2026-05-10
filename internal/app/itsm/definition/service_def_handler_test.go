package definition

import (
	"encoding/json"
	"errors"
	"fmt"
	. "metis/internal/app/itsm/domain"
	"net/http"
	"strconv"
	"strings"
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

func TestServiceDefHandlerCreate_Returns400ForInvalidSLA(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}
	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"VPN","code":"vpn","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"classic","slaId":999}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerUpdate_Returns400ForInactiveSLA(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}
	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := svc.Create(&ServiceDefinition{Name: "VPN", Code: "vpn", CatalogID: root.ID, EngineType: "classic"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	inactiveSLA := SLATemplate{Name: "停用 SLA", Code: "inactive-sla", ResponseMinutes: 1, ResolutionMinutes: 5, IsActive: false}
	if err := db.Create(&inactiveSLA).Error; err != nil {
		t.Fatalf("create inactive SLA: %v", err)
	}
	if err := db.Model(&inactiveSLA).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate SLA: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), []byte(`{"slaId":`+strconv.FormatUint(uint64(inactiveSLA.ID), 10)+`}`))

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
	sla := SLATemplate{Name: "标准 SLA", Code: "standard-sla", ResponseMinutes: 1, ResolutionMinutes: 5, IsActive: true}
	if err := db.Create(&sla).Error; err != nil {
		t.Fatalf("create SLA: %v", err)
	}
	slaID := sla.ID
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
	sla := SLATemplate{Name: "标准 SLA", Code: "standard-sla", ResponseMinutes: 1, ResolutionMinutes: 5, IsActive: true}
	if err := db.Create(&sla).Error; err != nil {
		t.Fatalf("create SLA: %v", err)
	}
	slaID := sla.ID
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

func TestServiceDefHandlerList_ClampsPageSizeAndReturnsSummaryItems(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-list", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	for i := 0; i < 105; i++ {
		_, err := svc.Create(&ServiceDefinition{
			Name:              fmt.Sprintf("Service %03d", i),
			Code:              fmt.Sprintf("svc-%03d", i),
			CatalogID:         root.ID,
			EngineType:        "smart",
			IntakeFormSchema:  JSONField(`{"fields":[{"key":"huge"}]}`),
			CollaborationSpec: "large runtime collaboration spec",
			AgentConfig:       JSONField(`{"temperature":0.1}`),
			KnowledgeBaseIDs:  JSONField(`[1,2,3]`),
		})
		if err != nil {
			t.Fatalf("create service %d: %v", i, err)
		}
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services", h.List)
	}, http.MethodGet, "/services?pageSize=100000", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := len(body.Data.Items); got != 100 {
		t.Fatalf("expected pageSize to be clamped to 100, got %d items", got)
	}
	heavyKeys := []string{"workflowJson", "intakeFormSchema", "agentConfig", "knowledgeBaseIds", "collaborationSpec"}
	for _, key := range heavyKeys {
		if _, exists := body.Data.Items[0][key]; exists {
			t.Fatalf("list summary item must not include heavy field %q: %#v", key, body.Data.Items[0])
		}
	}
}

func TestServiceDefHandlerList_RejectsInvalidFiltersAndPaging(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-list-invalid", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if _, err := svc.Create(&ServiceDefinition{
		Name:              "VPN",
		Code:              "svc-list-invalid",
		CatalogID:         root.ID,
		EngineType:        "smart",
		CollaborationSpec: "spec",
	}); err != nil {
		t.Fatalf("create service: %v", err)
	}

	for _, path := range []string{
		"/services?catalogId=bad",
		"/services?rootCatalogId=bad",
		"/services?page=bad",
		"/services?pageSize=bad",
		"/services?engineType=weird",
		"/services?isActive=maybe",
	} {
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.GET("/services", h.List)
		}, http.MethodGet, path, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s expected 400, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestServiceDefHandlerGetHealthCheckAndDelete(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-get-delete", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := svc.Create(&ServiceDefinition{
		Name:       "Classic VPN",
		Code:       "classic-vpn-get",
		CatalogID:  root.ID,
		EngineType: "classic",
		WorkflowJSON: JSONField(`{
			"nodes":[
				{"id":"start","type":"start","label":"开始"},
				{"id":"process","type":"process","data":{"label":"处理","participants":[{"type":"user","value":"1"}]}},
				{"id":"end","type":"end","label":"结束"}
			],
			"edges":[
				{"id":"e1","source":"start","target":"process"},
				{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}},
				{"id":"e3","source":"process","target":"end","data":{"outcome":"rejected"}}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	getPath := "/services/" + strconv.FormatUint(uint64(service.ID), 10)
	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id", h.Get)
	}, http.MethodGet, getPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"classic-vpn-get"`) {
		t.Fatalf("unexpected get response: %s", rec.Body.String())
	}

	healthPath := getPath + "/health-check"
	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/health-check", h.HealthCheck)
	}, http.MethodGet, healthPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected health-check 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"pass"`) {
		t.Fatalf("unexpected health-check response: %s", rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/services/:id", h.Delete)
	}, http.MethodDelete, getPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := svc.Get(service.ID); !errors.Is(err, ErrServiceDefNotFound) {
		t.Fatalf("expected deleted service to be gone, got %v", err)
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id", h.Get)
	}, http.MethodGet, getPath, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected deleted get 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerRejectsConflictsAndInvalidQueries(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-conflicts", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	if _, err := svc.Create(&ServiceDefinition{Name: "VPN", Code: "svc-conflict", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"}); err != nil {
		t.Fatalf("seed service: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"VPN 2","code":"svc-conflict","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"smart","collaborationSpec":"spec"}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate code create to return 409, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services", h.List)
	}, http.MethodGet, "/services?catalogId=1&rootCatalogId=1", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected mutually exclusive query to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/health-check", h.HealthCheck)
	}, http.MethodGet, "/services/999/health-check", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing health-check to return 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id", h.Get)
	}, http.MethodGet, "/services/bad", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid get id to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/services/:id/health-check", h.HealthCheck)
	}, http.MethodGet, "/services/bad/health-check", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid health-check id to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.DELETE("/services/:id", h.Delete)
	}, http.MethodDelete, "/services/bad", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid delete id to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerRejectsBlankBusinessIdentifiers(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-blank-service", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"   ","code":"vpn-blank","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"smart","collaborationSpec":"spec"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected blank create name to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"VPN","code":"   ","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"smart","collaborationSpec":"spec"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected blank create code to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	service, err := svc.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-update-blank", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), []byte(`{"name":"   "}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected blank update name to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.PUT("/services/:id", h.Update)
	}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), []byte(`{"code":"   "}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected blank update code to return 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerAllowsManualEngineAndRejectsUnknownEngine(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-handler-manual", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"人工受理服务","code":"manual-handler-service","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"manual"}`))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"engineType":"manual"`) {
		t.Fatalf("manual create status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performJSONRequest(t, func(r *gin.Engine) {
		r.POST("/services", h.Create)
	}, http.MethodPost, "/services", []byte(`{"name":"非法服务","code":"invalid-engine-service","catalogId":`+strconv.FormatUint(uint64(root.ID), 10)+`,"engineType":"weird"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown engine create status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceDefHandlerCreateUpdateAndDeleteContracts(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	svc := newServiceDefServiceForTest(t, db)
	h := &ServiceDefHandler{svc: svc}

	root, err := catSvc.Create("Root", "root-contracts", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	otherCatalog, err := catSvc.Create("Other", "other-contracts", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create other catalog: %v", err)
	}

	t.Run("create persists classic service shape", func(t *testing.T) {
		body := []byte(`{
			"name":"Classic VPN",
			"code":"classic-vpn-contract",
			"catalogId":` + strconv.FormatUint(uint64(root.ID), 10) + `,
			"engineType":"classic",
			"description":"vpn for contracts",
			"workflowJson":{"nodes":[{"id":"start","type":"start"},{"id":"end","type":"end"}],"edges":[{"id":"e1","source":"start","target":"end","data":{"default":true}}]},
			"sortOrder":7
		}`)
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.POST("/services", h.Create)
		}, http.MethodPost, "/services", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected create 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"code":"classic-vpn-contract"`) {
			t.Fatalf("unexpected create response: %s", rec.Body.String())
		}

		created, total, err := svc.List(ServiceDefListParams{Keyword: "classic-vpn-contract", Page: 1, PageSize: 10})
		if err != nil || total != 1 || len(created) != 1 {
			t.Fatalf("load created service total=%d err=%v items=%+v", total, err, created)
		}
		if created[0].Description != "vpn for contracts" || created[0].SortOrder != 7 || created[0].CatalogID != root.ID {
			t.Fatalf("unexpected created service: %+v", created[0])
		}
	})

	t.Run("update persists fields and handles missing resource", func(t *testing.T) {
		service, err := svc.Create(&ServiceDefinition{
			Name:              "Smart VPN",
			Code:              "smart-vpn-contract",
			CatalogID:         root.ID,
			EngineType:        "smart",
			Description:       "old",
			CollaborationSpec: "old spec",
		})
		if err != nil {
			t.Fatalf("create service: %v", err)
		}

		body := []byte(`{
			"name":"Smart VPN Updated",
			"description":"new desc",
			"catalogId":` + strconv.FormatUint(uint64(otherCatalog.ID), 10) + `,
			"sortOrder":13,
			"isActive":false,
			"collaborationSpec":"updated spec"
		}`)
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.PUT("/services/:id", h.Update)
		}, http.MethodPut, "/services/"+strconv.FormatUint(uint64(service.ID), 10), body)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected update 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"name":"Smart VPN Updated"`) || !strings.Contains(rec.Body.String(), `"isActive":false`) {
			t.Fatalf("unexpected update response: %s", rec.Body.String())
		}

		updated, err := svc.Get(service.ID)
		if err != nil {
			t.Fatalf("reload updated service: %v", err)
		}
		if updated.Name != "Smart VPN Updated" || updated.Description != "new desc" || updated.CatalogID != otherCatalog.ID || updated.SortOrder != 13 || updated.IsActive || updated.CollaborationSpec != "updated spec" {
			t.Fatalf("unexpected updated service: %+v", updated)
		}

		rec = performJSONRequest(t, func(r *gin.Engine) {
			r.PUT("/services/:id", h.Update)
		}, http.MethodPut, "/services/999999", []byte(`{"name":"ghost"}`))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected missing update 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete missing resource maps to 404", func(t *testing.T) {
		rec := performJSONRequest(t, func(r *gin.Engine) {
			r.DELETE("/services/:id", h.Delete)
		}, http.MethodDelete, "/services/999999", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected missing delete 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}
