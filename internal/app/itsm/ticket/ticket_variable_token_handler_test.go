package ticket

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	appcore "metis/internal/app"
	"metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	"metis/internal/app/itsm/sla"
	"metis/internal/database"
	"metis/internal/model"
)

func newVariableTokenRouter(t *testing.T, db *gorm.DB, orgResolver appcore.OrgResolver) *gin.Engine {
	t.Helper()

	injector := do.New()
	wrapped := &database.DB{DB: db}
	resolver := engine.NewParticipantResolver(orgResolver)
	do.ProvideValue(injector, wrapped)
	if orgResolver != nil {
		do.ProvideValue[appcore.OrgResolver](injector, orgResolver)
	}
	do.Provide(injector, NewTicketRepo)
	do.Provide(injector, NewTimelineRepo)
	do.Provide(injector, definition.NewServiceDefRepo)
	do.Provide(injector, sla.NewSLATemplateRepo)
	do.Provide(injector, sla.NewPriorityRepo)
	do.ProvideValue(injector, engine.NewClassicEngine(resolver, nil, nil))
	do.ProvideValue(injector, engine.NewSmartEngine(submissionTestDecisionExecutor{}, nil, nil, resolver, &submissionTestSubmitter{db: db}, nil))
	do.Provide(injector, NewTicketService)
	do.Provide(injector, NewVariableRepository)
	do.Provide(injector, NewVariableService)
	do.Provide(injector, NewVariableHandler)
	do.Provide(injector, NewTokenRepository)
	do.Provide(injector, NewTokenHandler)

	variableHandler := do.MustInvoke[*VariableHandler](injector)
	tokenHandler := do.MustInvoke[*TokenHandler](injector)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID, _ := strconv.ParseUint(c.GetHeader("X-User-ID"), 10, 64)
		c.Set("userId", uint(userID))
		roleCode := c.GetHeader("X-User-Role")
		if roleCode == "" {
			roleCode = model.RoleUser
		}
		c.Set("userRole", roleCode)
		c.Next()
	})
	router.GET("/tickets/:id/variables", variableHandler.List)
	router.PUT("/tickets/:id/variables/:key", variableHandler.Update)
	router.GET("/tickets/:id/tokens", tokenHandler.List)
	return router
}

func performTicketAuxJSONRequest(t *testing.T, router *gin.Engine, method, path string, body []byte, userID uint, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
	req.Header.Set("X-User-Role", role)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestTicketVariableAndTokenHandlersManageWorkflowState(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ProcessVariable{}, &ExecutionToken{}); err != nil {
		t.Fatalf("migrate variable/token tables: %v", err)
	}
	if err := db.Exec("CREATE TABLE operator_positions (user_id INTEGER NOT NULL, position_id INTEGER NOT NULL)").Error; err != nil {
		t.Fatalf("create operator_positions: %v", err)
	}
	if err := db.Exec("CREATE TABLE operator_departments (user_id INTEGER NOT NULL, department_id INTEGER NOT NULL)").Error; err != nil {
		t.Fatalf("create operator_departments: %v", err)
	}

	seedTicketHandlerUsers(t, db)
	service, priority := seedClassicServiceForHandler(t, db)
	activeTicket, _ := seedTicketHandlerTickets(t, db, service, priority)
	router := newVariableTokenRouter(t, db, &rootDBOrgResolver{db: db})

	if err := db.Create(&ProcessVariable{
		TicketID:  activeTicket.ID,
		ScopeID:   "root",
		Key:       "requester",
		Value:     "alice",
		ValueType: ValueTypeString,
		Source:    "form",
	}).Error; err != nil {
		t.Fatalf("create process variable: %v", err)
	}
	if err := db.Create(&ExecutionToken{
		TicketID:  activeTicket.ID,
		NodeID:    "approve-manager",
		Status:    engine.TokenWaiting,
		TokenType: engine.TokenMain,
	}).Error; err != nil {
		t.Fatalf("create execution token: %v", err)
	}

	varPath := "/tickets/" + strconv.FormatUint(uint64(activeTicket.ID), 10) + "/variables"
	rec := performTicketHandlerRequest(t, router, http.MethodGet, varPath, 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("list variables status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"key":"requester"`) || !strings.Contains(rec.Body.String(), `"value":"alice"`) {
		t.Fatalf("unexpected variable list response: %s", rec.Body.String())
	}

	updatePath := varPath + "/requester"
	rec = performTicketAuxJSONRequest(t, router, http.MethodPut, updatePath, []byte(`{"value":"not-a-number","valueType":"number"}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid variable update to fail, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketAuxJSONRequest(t, router, http.MethodPut, updatePath, []byte(`{"value":{"name":"alice","roles":["vpn"]},"valueType":"json"}`), 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected valid variable update to succeed, got status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated ProcessVariable
	if err := db.Where("ticket_id = ? AND scope_id = ? AND key = ?", activeTicket.ID, "root", "requester").First(&updated).Error; err != nil {
		t.Fatalf("reload updated variable: %v", err)
	}
	if updated.ValueType != ValueTypeJSON || !strings.Contains(updated.Value, `"roles":["vpn"]`) || updated.Source != "manual:20" {
		t.Fatalf("unexpected updated variable: %+v", updated)
	}

	tokenPath := "/tickets/" + strconv.FormatUint(uint64(activeTicket.ID), 10) + "/tokens"
	rec = performTicketHandlerRequest(t, router, http.MethodGet, tokenPath, 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tokens status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"nodeId":"approve-manager"`) || !strings.Contains(rec.Body.String(), `"status":"waiting"`) {
		t.Fatalf("unexpected token list response: %s", rec.Body.String())
	}

	t.Run("variable and token handlers reject invalid ids and forbidden access", func(t *testing.T) {
		rec := performTicketHandlerRequest(t, router, http.MethodGet, "/tickets/bad/variables", 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid variable ticket id status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerRequest(t, router, http.MethodGet, "/tickets/bad/tokens", 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid token ticket id status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerRequest(t, router, http.MethodGet, varPath, 50, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("forbidden variable list status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerRequest(t, router, http.MethodGet, tokenPath, 50, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("forbidden token list status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("variable update maps missing variable and invalid payload types", func(t *testing.T) {
		missingPath := varPath + "/missing"
		rec := performTicketAuxJSONRequest(t, router, http.MethodPut, missingPath, []byte(`{"value":"x"}`), 20, model.RoleUser)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("missing variable update status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketAuxJSONRequest(t, router, http.MethodPut, updatePath, []byte(`{"value":tru`), 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid variable body status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketAuxJSONRequest(t, router, http.MethodPut, updatePath, []byte(`{"value":"not-bool","valueType":"boolean"}`), 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid boolean variable update status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketAuxJSONRequest(t, router, http.MethodPut, updatePath, []byte(`{"value":"{bad}","valueType":"json"}`), 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid json variable update status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
