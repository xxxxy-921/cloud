package sla

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"metis/internal/app/itsm/domain"
	"metis/internal/database"
	"metis/internal/model"
)

func setupSLAHandlers(t *testing.T) (*PriorityHandler, *SLATemplateHandler, *EscalationRuleHandler, *gin.Engine, *database.DB) {
	t.Helper()
	db := &database.DB{DB: newTestDB(t)}
	if err := db.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate message channels: %v", err)
	}
	prioritySvc := &PriorityService{repo: &PriorityRepo{db: db}}
	slaSvc := &SLATemplateService{repo: &SLATemplateRepo{db: db}, db: db}
	escalationSvc := &EscalationRuleService{repo: &EscalationRuleRepo{db: db}, db: db}

	priorityHandler := &PriorityHandler{svc: prioritySvc}
	slaHandler := &SLATemplateHandler{svc: slaSvc}
	escalationHandler := &EscalationRuleHandler{svc: escalationSvc, db: db}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/priorities", priorityHandler.Create)
	r.GET("/priorities", priorityHandler.List)
	r.PUT("/priorities/:id", priorityHandler.Update)
	r.DELETE("/priorities/:id", priorityHandler.Delete)
	r.POST("/slas", slaHandler.Create)
	r.GET("/slas", slaHandler.List)
	r.PUT("/slas/:id", slaHandler.Update)
	r.DELETE("/slas/:id", slaHandler.Delete)
	r.POST("/slas/:id/escalations", escalationHandler.Create)
	r.GET("/slas/:id/escalations", escalationHandler.List)
	r.PUT("/slas/:id/escalations/:escalationId", escalationHandler.Update)
	r.DELETE("/slas/:id/escalations/:escalationId", escalationHandler.Delete)
	r.GET("/escalation-notification-channels", escalationHandler.NotificationChannels)
	return priorityHandler, slaHandler, escalationHandler, r, db
}

func TestSLAHandlersCoverCrudAndStatusMappings(t *testing.T) {
	_, _, _, r, db := setupSLAHandlers(t)

	t.Run("priority create list update delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"P1","code":"P1","value":1,"color":"#ef4444"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create priority status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/priorities", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list priorities status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPut, "/priorities/1", bytes.NewReader([]byte(`{"name":"P1 updated"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update priority status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/priorities/1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete priority status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/priorities/bad", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad priority id status = %d, want 400", w.Code)
		}
	})

	t.Run("sla create list update delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"Standard","code":"std","responseMinutes":5,"resolutionMinutes":30}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create sla status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/slas", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list sla status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPut, "/slas/1", bytes.NewReader([]byte(`{"name":"Standard 2"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update sla status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/slas/1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete sla status = %d, want 200", w.Code)
		}
	})

	t.Run("escalation create list update delete and notification channels", func(t *testing.T) {
		priority := domain.Priority{Name: "P2", Code: "P2", Value: 2, Color: "#0ea5e9", IsActive: true}
		if err := db.Create(&priority).Error; err != nil {
			t.Fatalf("create priority for escalation: %v", err)
		}
		sla := domain.SLATemplate{Name: "Escalation SLA", Code: "esc", ResponseMinutes: 10, ResolutionMinutes: 60, IsActive: true}
		if err := db.Create(&sla).Error; err != nil {
			t.Fatalf("create sla for escalation: %v", err)
		}
		channel := model.MessageChannel{Name: "Email", Type: "smtp", Config: `{}`, Enabled: true}
		if err := db.Create(&channel).Error; err != nil {
			t.Fatalf("create channel: %v", err)
		}

		body := `{"triggerType":"response_timeout","level":1,"waitMinutes":5,"actionType":"notify","targetConfig":{"recipients":[{"type":"user","value":"1"}],"channelId":` + strconv.Itoa(int(channel.ID)) + `}}`
		req := httptest.NewRequest(http.MethodPost, "/slas/"+strconv.Itoa(int(sla.ID))+"/escalations", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create escalation status = %d, want 200, body=%s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/slas/"+strconv.Itoa(int(sla.ID))+"/escalations", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list escalations status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPut, "/slas/"+strconv.Itoa(int(sla.ID))+"/escalations/1", bytes.NewReader([]byte(`{"waitMinutes":10}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("update escalation status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/escalation-notification-channels", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("notification channels status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/slas/"+strconv.Itoa(int(sla.ID))+"/escalations/1", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("delete escalation status = %d, want 200", w.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/slas/bad/escalations", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad sla id status = %d, want 400", w.Code)
		}
	})
}

func TestSLAHandlersRejectConflictsAndReferencedResources(t *testing.T) {
	_, _, _, r, db := setupSLAHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"P1","code":"P1","value":1,"color":"#ef4444"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed priority status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"P1 copy","code":"P1","value":2,"color":"#0ea5e9"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate priority create status = %d, want 409", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/priorities/999", bytes.NewReader([]byte(`{"name":"missing"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing priority update status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/priorities/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing priority delete status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"Standard","code":"std","responseMinutes":5,"resolutionMinutes":30}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed sla status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"Standard Copy","code":"std","responseMinutes":10,"resolutionMinutes":40}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate sla create status = %d, want 409", w.Code)
	}

	slaID := uint(1)
	if err := db.Create(&domain.ServiceDefinition{
		Name:         "VPN Access",
		Code:         "vpn-access",
		CatalogID:    1,
		EngineType:   "classic",
		SLAID:        &slaID,
		WorkflowJSON: domain.JSONField(`{"nodes":[],"edges":[]}`),
		IsActive:     true,
	}).Error; err != nil {
		t.Fatalf("create active service reference: %v", err)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1", bytes.NewReader([]byte(`{"isActive":false}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("deactivate referenced sla status = %d, want 400", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/slas/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete referenced sla status = %d, want 400", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/999", bytes.NewReader([]byte(`{"name":"missing"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing sla update status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/slas/999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing sla delete status = %d, want 404", w.Code)
	}

	channel := model.MessageChannel{Name: "Email", Type: "smtp", Config: `{}`, Enabled: true}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	disabled := model.MessageChannel{Name: "Disabled", Type: "smtp", Config: `{}`, Enabled: false}
	if err := db.Create(&disabled).Error; err != nil {
		t.Fatalf("create disabled channel: %v", err)
	}
	if err := db.Model(&disabled).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable channel: %v", err)
	}

	createEscalation := func(body string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/slas/1/escalations", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	okBody := `{"triggerType":"response_timeout","level":1,"waitMinutes":5,"actionType":"notify","targetConfig":{"recipients":[{"type":"user","value":"1"}],"channelId":` + strconv.Itoa(int(channel.ID)) + `}}`
	if code := createEscalation(okBody); code != http.StatusOK {
		t.Fatalf("seed escalation status = %d, want 200", code)
	}
	if code := createEscalation(okBody); code != http.StatusConflict {
		t.Fatalf("duplicate escalation create status = %d, want 409", code)
	}

	invalidTargetBody := `{"triggerType":"resolution_timeout","level":2,"waitMinutes":10,"actionType":"notify","targetConfig":{"recipients":[{"type":"user","value":"1"}]}}`
	if code := createEscalation(invalidTargetBody); code != http.StatusBadRequest {
		t.Fatalf("invalid escalation target status = %d, want 400", code)
	}

	invalidRuleBody := `{"triggerType":"response_timeout","level":1,"waitMinutes":-1,"actionType":"notify","targetConfig":{"recipients":[{"type":"user","value":"1"}],"channelId":` + strconv.Itoa(int(channel.ID)) + `}}`
	if code := createEscalation(invalidRuleBody); code != http.StatusBadRequest {
		t.Fatalf("invalid escalation wait_minutes status = %d, want 400", code)
	}

	req = httptest.NewRequest(http.MethodPost, "/slas/999999/escalations", bytes.NewReader([]byte(okBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("create escalation for missing sla status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/slas/999999/escalations", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("list escalation for missing sla status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1/escalations/999", bytes.NewReader([]byte(`{"waitMinutes":15}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing escalation update status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1/escalations/bad", bytes.NewReader([]byte(`{"waitMinutes":15}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad escalation id update status = %d, want 400", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/escalation-notification-channels", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("notification channels status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Email") || strings.Contains(body, "Disabled") {
		t.Fatalf("unexpected notification channel response: %s", body)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1/escalations/1", bytes.NewReader([]byte(`{"waitMinutes":-1}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid escalation update wait_minutes status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1/escalations/1", bytes.NewReader([]byte(`{"targetConfig":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid escalation target update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	secondSLA := domain.SLATemplate{Name: "Second SLA", Code: "esc-2", ResponseMinutes: 20, ResolutionMinutes: 80, IsActive: true}
	if err := db.Create(&secondSLA).Error; err != nil {
		t.Fatalf("create second sla: %v", err)
	}
	foreignRule := domain.EscalationRule{
		SLAID:        secondSLA.ID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  25,
		ActionType:   "notify",
		TargetConfig: domain.JSONField(`{"recipients":[{"type":"user","value":"1"}],"channelId":` + strconv.Itoa(int(channel.ID)) + `}`),
		IsActive:     true,
	}
	if err := db.Create(&foreignRule).Error; err != nil {
		t.Fatalf("create foreign escalation: %v", err)
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1/escalations/"+strconv.FormatUint(uint64(foreignRule.ID), 10), bytes.NewReader([]byte(`{"waitMinutes":99}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-sla escalation update status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	var reloadedForeign domain.EscalationRule
	if err := db.First(&reloadedForeign, foreignRule.ID).Error; err != nil {
		t.Fatalf("reload foreign escalation after update attempt: %v", err)
	}
	if reloadedForeign.WaitMinutes != foreignRule.WaitMinutes {
		t.Fatalf("expected foreign escalation to remain unchanged, got %+v", reloadedForeign)
	}

	req = httptest.NewRequest(http.MethodDelete, "/slas/1/escalations/"+strconv.FormatUint(uint64(foreignRule.ID), 10), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-sla escalation delete status = %d, want 404 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/slas/1/escalations/bad", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad escalation id delete status = %d, want 400", w.Code)
	}

	if err := db.First(&reloadedForeign, foreignRule.ID).Error; err != nil {
		t.Fatalf("expected foreign escalation to survive delete attempt, got %v", err)
	}
}

func TestSLAHandlersRejectInvalidDurationsAndBadPayloads(t *testing.T) {
	_, _, _, r, _ := setupSLAHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"Broken","code":"broken","responseMinutes":0,"resolutionMinutes":30}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid sla duration create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"bad"`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid priority payload status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"Broken Priority","code":"BROKEN","value":0,"color":"#000"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid priority value create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"Valid","code":"valid","responseMinutes":5,"resolutionMinutes":30}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed valid sla status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1", bytes.NewReader([]byte(`{"resolutionMinutes":0}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid sla duration update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"P1","code":"P1","value":1,"color":"#ef4444"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("seed priority status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/priorities/1", bytes.NewReader([]byte(`{"value":0}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid priority value update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/priorities", bytes.NewReader([]byte(`{"name":"   ","code":" P2 ","value":1,"color":"#ef4444"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank priority identifier create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/priorities/1", bytes.NewReader([]byte(`{"name":"   "}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank priority identifier update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/slas", bytes.NewReader([]byte(`{"name":"   ","code":" std2 ","responseMinutes":5,"resolutionMinutes":30}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank sla identifier create status = %d, want 400 body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/slas/1", bytes.NewReader([]byte(`{"code":"   "}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank sla identifier update status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}
