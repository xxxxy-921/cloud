package ticket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

type ticketHandlerFixture struct {
	handler       *TicketHandler
	router        *gin.Engine
	db            *gorm.DB
	activeTicket  Ticket
	activeActID   uint
	historyTicket Ticket
}

func newTicketHandlerFixture(t *testing.T) ticketHandlerFixture {
	t.Helper()

	db := newTestDB(t)
	if err := db.Exec("CREATE TABLE operator_positions (user_id INTEGER NOT NULL, position_id INTEGER NOT NULL)").Error; err != nil {
		t.Fatalf("create operator_positions: %v", err)
	}
	if err := db.Exec("CREATE TABLE operator_departments (user_id INTEGER NOT NULL, department_id INTEGER NOT NULL)").Error; err != nil {
		t.Fatalf("create operator_departments: %v", err)
	}

	seedTicketHandlerUsers(t, db)
	service, priority := seedClassicServiceForHandler(t, db)
	activeTicket, historyTicket := seedTicketHandlerTickets(t, db, service, priority)

	orgResolver := &rootDBOrgResolver{db: db}
	handler := newTicketHandler(t, db, orgResolver)
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
	router.GET("/tickets", handler.List)
	router.GET("/tickets/mine", handler.Mine)
	router.GET("/tickets/pending-approvals", handler.PendingApprovals)
	router.GET("/tickets/approval-history", handler.ApprovalHistory)
	router.GET("/tickets/:id", handler.Get)
	router.GET("/tickets/:id/timeline", handler.Timeline)
	router.GET("/tickets/:id/activities", handler.Activities)
	router.POST("/tickets/:id/progress", handler.Progress)
	router.POST("/tickets/:id/signal", handler.Signal)
	router.POST("/tickets/:id/override-jump", handler.OverrideJump)
	router.POST("/tickets/:id/override-reassign", handler.OverrideReassign)
	router.PUT("/tickets/:id/assign", handler.Assign)
	router.POST("/tickets/:id/claim", handler.Claim)
	router.POST("/tickets/:id/transfer", handler.Transfer)
	router.POST("/tickets/:id/delegate", handler.Delegate)
	router.PUT("/tickets/:id/sla/pause", handler.SLAPause)
	router.PUT("/tickets/:id/sla/resume", handler.SLAResume)
	router.POST("/tickets/:id/recover", handler.Recover)
	router.POST("/tickets/:id/retry-ai", handler.RetryAI)
	router.GET("/tickets/decision-quality", handler.DecisionQuality)

	return ticketHandlerFixture{
		handler:       handler,
		router:        router,
		db:            db,
		activeTicket:  activeTicket,
		activeActID:   *activeTicket.CurrentActivityID,
		historyTicket: historyTicket,
	}
}

func newTicketHandler(t *testing.T, db *gorm.DB, orgResolver appcore.OrgResolver) *TicketHandler {
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
	do.Provide(injector, NewTimelineService)
	do.Provide(injector, NewTicketHandler)
	return do.MustInvoke[*TicketHandler](injector)
}

func seedTicketHandlerUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	users := []model.User{
		{BaseModel: model.BaseModel{ID: 10}, Username: "requester", Email: "requester@example.com", IsActive: true},
		{BaseModel: model.BaseModel{ID: 20}, Username: "assignee", Email: "assignee@example.com", IsActive: true},
		{BaseModel: model.BaseModel{ID: 30}, Username: "history", Email: "history@example.com", IsActive: true},
		{BaseModel: model.BaseModel{ID: 99}, Username: "admin", Email: "admin@example.com", IsActive: true},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Username, err)
		}
	}
}

func seedClassicServiceForHandler(t *testing.T, db *gorm.DB) (ServiceDefinition, Priority) {
	t.Helper()
	priority := Priority{Name: "普通", Code: "normal-handler", Value: 3, Color: "#64748b", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}
	catalog := ServiceCatalog{Name: "账号权限", Code: "account-handler", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:       "VPN 开通",
		Code:       "vpn-handler",
		CatalogID:  catalog.ID,
		EngineType: "classic",
		IsActive:   true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	return service, priority
}

func seedTicketHandlerTickets(t *testing.T, db *gorm.DB, service ServiceDefinition, priority Priority) (Ticket, Ticket) {
	t.Helper()
	activeTicket := Ticket{
		Code:        "TICK-HANDLER-001",
		Title:       "VPN 开通申请",
		Description: "生产环境 VPN 开通",
		ServiceID:   service.ID,
		EngineType:  service.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: 10,
		Source:      TicketSourceCatalog,
	}
	if err := db.Create(&activeTicket).Error; err != nil {
		t.Fatalf("create active ticket: %v", err)
	}
	activeActivity := TicketActivity{
		TicketID:     activeTicket.ID,
		Name:         "网络管理员处理",
		ActivityType: "process",
		Status:       AssignmentPending,
	}
	if err := db.Create(&activeActivity).Error; err != nil {
		t.Fatalf("create active activity: %v", err)
	}
	if err := db.Model(&activeTicket).Update("current_activity_id", activeActivity.ID).Error; err != nil {
		t.Fatalf("bind current activity: %v", err)
	}
	activeTicket.CurrentActivityID = &activeActivity.ID
	assigneeID := uint(20)
	if err := db.Create(&TicketAssignment{
		TicketID:   activeTicket.ID,
		ActivityID: activeActivity.ID,
		UserID:     &assigneeID,
		AssigneeID: &assigneeID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create pending assignment: %v", err)
	}
	if err := db.Create(&TicketTimeline{
		TicketID:   activeTicket.ID,
		OperatorID: assigneeID,
		EventType:  "assigned",
		Message:    "已分配给网络管理员",
		Details:    JSONField(`{"step":"assign"}`),
	}).Error; err != nil {
		t.Fatalf("create active timeline: %v", err)
	}

	historyTicket := Ticket{
		Code:        "TICK-HANDLER-002",
		Title:       "VPN 权限回收",
		Description: "离职权限回收",
		ServiceID:   service.ID,
		EngineType:  service.EngineType,
		Status:      TicketStatusCompleted,
		Outcome:     TicketOutcomeFulfilled,
		PriorityID:  priority.ID,
		RequesterID: 10,
		Source:      TicketSourceCatalog,
		FinishedAt:  ptrTime(time.Now()),
	}
	if err := db.Create(&historyTicket).Error; err != nil {
		t.Fatalf("create history ticket: %v", err)
	}
	historyActivity := TicketActivity{
		TicketID:          historyTicket.ID,
		Name:              "网络管理员审批",
		ActivityType:      "approve",
		Status:            "completed",
		TransitionOutcome: TicketOutcomeApproved,
		FinishedAt:        ptrTime(time.Now()),
	}
	if err := db.Create(&historyActivity).Error; err != nil {
		t.Fatalf("create history activity: %v", err)
	}
	if err := db.Create(&TicketAssignment{
		TicketID:   historyTicket.ID,
		ActivityID: historyActivity.ID,
		AssigneeID: &assigneeID,
		Status:     AssignmentApproved,
		FinishedAt: ptrTime(time.Now()),
	}).Error; err != nil {
		t.Fatalf("create history assignment: %v", err)
	}

	return activeTicket, historyTicket
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func performTicketHandlerRequest(t *testing.T, router *gin.Engine, method, path string, userID uint, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
	req.Header.Set("X-User-Role", role)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func performTicketHandlerJSONRequest(t *testing.T, router *gin.Engine, method, path string, body []byte, userID uint, role string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
	req.Header.Set("X-User-Role", role)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestTicketHandlerListAndMineExposeVisibleTickets(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets?keyword=VPN&page=1&pageSize=10", 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":2`) {
		t.Fatalf("expected total=2 in list response, body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"serviceName":"VPN 开通"`) {
		t.Fatalf("expected service name in list response, body=%s", rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/mine?status=active&page=1&pageSize=10", 10, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("mine status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":1`) {
		t.Fatalf("expected active mine total=1, body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), fixture.activeTicket.Code) {
		t.Fatalf("expected active ticket in mine response, body=%s", rec.Body.String())
	}
}

func TestTicketHandlerGetTimelineActivitiesAndApprovalViews(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	if err := fixture.db.Create(&TicketTimeline{
		TicketID:   fixture.activeTicket.ID,
		OperatorID: 0,
		EventType:  "system_notice",
		Message:    "系统自动流转",
	}).Error; err != nil {
		t.Fatalf("create system timeline: %v", err)
	}

	rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/"+strconv.FormatUint(uint64(fixture.activeTicket.ID), 10), 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"requesterName":"requester"`) {
		t.Fatalf("expected requester name in get response, body=%s", rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/"+strconv.FormatUint(uint64(fixture.activeTicket.ID), 10)+"/timeline", 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("timeline status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"operatorName":"assignee"`) {
		t.Fatalf("expected operator name in timeline response, body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"operatorName":"系统"`) {
		t.Fatalf("expected system timeline row to render 系统 operatorName, body=%s", rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/"+strconv.FormatUint(uint64(fixture.activeTicket.ID), 10)+"/activities", 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("activities status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Data []TicketActivity `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode activities: %v", err)
	}
	if len(payload.Data) != 1 || !payload.Data[0].CanAct {
		t.Fatalf("expected one actionable activity, got %+v", payload.Data)
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/pending-approvals?page=1&pageSize=10", 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("pending approvals status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":1`) || !strings.Contains(rec.Body.String(), fixture.activeTicket.Code) {
		t.Fatalf("unexpected pending approvals response: %s", rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/approval-history?page=1&pageSize=10", 20, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("approval history status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":1`) || !strings.Contains(rec.Body.String(), fixture.historyTicket.Code) {
		t.Fatalf("unexpected approval history response: %s", rec.Body.String())
	}
}

func TestTicketHandlerMineFiltersByDateAndActivitiesHonorVisibility(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	oldDate := time.Now().AddDate(0, 0, -3)
	if err := fixture.db.Model(&fixture.activeTicket).Update("created_at", oldDate).Error; err != nil {
		t.Fatalf("backdate active ticket: %v", err)
	}
	recentDate := time.Now()
	if err := fixture.db.Model(&fixture.historyTicket).Update("created_at", recentDate).Error; err != nil {
		t.Fatalf("refresh history ticket created_at: %v", err)
	}

	start := recentDate.Format("2006-01-02")
	rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/mine?page=1&pageSize=10&startDate="+start, 10, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("mine with date filter status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), fixture.activeTicket.Code) || !strings.Contains(rec.Body.String(), fixture.historyTicket.Code) {
		t.Fatalf("expected date-filtered mine list to include only recent ticket, body=%s", rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/"+strconv.FormatUint(uint64(fixture.activeTicket.ID), 10)+"/activities", 50, model.RoleUser)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected invisible activities request to return 403, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTicketHandlerRejectsInvisibleAndInvalidTargets(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/not-a-number", 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/"+strconv.FormatUint(uint64(fixture.activeTicket.ID), 10), 50, model.RoleUser)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("forbidden get status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerRequest(t, fixture.router, http.MethodGet, "/tickets/999999/timeline", 20, model.RoleUser)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing timeline status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTicketHandlerLifecycleGuardMappings(t *testing.T) {
	fixture := newTicketHandlerFixture(t)
	fixture.router.PUT("/tickets/:id/cancel", fixture.handler.Cancel)
	fixture.router.POST("/tickets/:id/withdraw", fixture.handler.Withdraw)

	terminal := Ticket{
		Code:        "TICK-HANDLER-TERMINAL",
		Title:       "终态工单",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusCompleted,
		Outcome:     TicketOutcomeFulfilled,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
		FinishedAt:  ptrTime(time.Now()),
	}
	if err := fixture.db.Create(&terminal).Error; err != nil {
		t.Fatalf("create terminal ticket: %v", err)
	}

	noAccessTicket := Ticket{
		Code:        "TICK-HANDLER-NOACCESS",
		Title:       "无权限处理",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&noAccessTicket).Error; err != nil {
		t.Fatalf("create no-access ticket: %v", err)
	}
	noAccessActivity := TicketActivity{
		TicketID:     noAccessTicket.ID,
		Name:         "待处理活动",
		ActivityType: engine.NodeProcess,
		Status:       AssignmentPending,
	}
	if err := fixture.db.Create(&noAccessActivity).Error; err != nil {
		t.Fatalf("create no-access activity: %v", err)
	}
	if err := fixture.db.Model(&noAccessTicket).Update("current_activity_id", noAccessActivity.ID).Error; err != nil {
		t.Fatalf("bind no-access activity: %v", err)
	}
	assignee20 := uint(20)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   noAccessTicket.ID,
		ActivityID: noAccessActivity.ID,
		UserID:     &assignee20,
		AssigneeID: &assignee20,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create no-access assignment: %v", err)
	}

	claimedTicket := Ticket{
		Code:        "TICK-HANDLER-CLAIMED",
		Title:       "已认领撤回",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&claimedTicket).Error; err != nil {
		t.Fatalf("create claimed ticket: %v", err)
	}
	claimedActivity := TicketActivity{
		TicketID:     claimedTicket.ID,
		Name:         "已认领活动",
		ActivityType: engine.NodeProcess,
		Status:       AssignmentInProgress,
	}
	if err := fixture.db.Create(&claimedActivity).Error; err != nil {
		t.Fatalf("create claimed activity: %v", err)
	}
	claimedAt := time.Now().Add(-5 * time.Minute)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   claimedTicket.ID,
		ActivityID: claimedActivity.ID,
		UserID:     &assignee20,
		AssigneeID: &assignee20,
		Status:     AssignmentInProgress,
		IsCurrent:  true,
		ClaimedAt:  &claimedAt,
	}).Error; err != nil {
		t.Fatalf("create claimed assignment: %v", err)
	}

	pausedTicket := Ticket{
		Code:        "TICK-HANDLER-PAUSED",
		Title:       "已暂停 SLA",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
		SLAPausedAt: ptrTime(time.Now().Add(-10 * time.Minute)),
	}
	if err := fixture.db.Create(&pausedTicket).Error; err != nil {
		t.Fatalf("create paused ticket: %v", err)
	}
	notPausedTicket := Ticket{
		Code:        "TICK-HANDLER-NOT-PAUSED",
		Title:       "未暂停 SLA",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&notPausedTicket).Error; err != nil {
		t.Fatalf("create not-paused ticket: %v", err)
	}

	t.Run("withdraw maps requester and claimed guards", func(t *testing.T) {
		path := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/withdraw"
		rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, path, []byte(`{"reason":"not mine"}`), 20, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("withdraw non-requester status=%d body=%s", rec.Code, rec.Body.String())
		}

		path = "/tickets/" + strconv.FormatUint(uint64(claimedTicket.ID), 10) + "/withdraw"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, path, []byte(`{"reason":"too late"}`), claimedTicket.RequesterID, model.RoleUser)
		if rec.Code != http.StatusConflict {
			t.Fatalf("withdraw claimed status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("cancel and sla handlers reject terminal or conflicting states", func(t *testing.T) {
		cancelPath := "/tickets/" + strconv.FormatUint(uint64(terminal.ID), 10) + "/cancel"
		rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, cancelPath, []byte(`{"reason":"done"}`), 99, model.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("cancel terminal status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, "/tickets/not-a-number/cancel", []byte(`{"reason":"bad id"}`), 99, model.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("cancel invalid id status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, "/tickets/999999/cancel", []byte(`{"reason":"missing"}`), 99, model.RoleAdmin)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cancel missing ticket status=%d body=%s", rec.Code, rec.Body.String())
		}

		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, cancelPath, []byte(`{"reason":`), 99, model.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("cancel invalid payload status=%d body=%s", rec.Code, rec.Body.String())
		}

		pausePath := "/tickets/" + strconv.FormatUint(uint64(pausedTicket.ID), 10) + "/sla/pause"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, pausePath, nil, 99, model.RoleAdmin)
		if rec.Code != http.StatusConflict {
			t.Fatalf("pause already-paused status=%d body=%s", rec.Code, rec.Body.String())
		}

		resumePath := "/tickets/" + strconv.FormatUint(uint64(notPausedTicket.ID), 10) + "/sla/resume"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, resumePath, nil, 99, model.RoleAdmin)
		if rec.Code != http.StatusConflict {
			t.Fatalf("resume not-paused status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("claim transfer and delegate map assignment access and terminal guards", func(t *testing.T) {
		claimPath := "/tickets/" + strconv.FormatUint(uint64(noAccessTicket.ID), 10) + "/claim"
		rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, claimPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(noAccessActivity.ID), 10)+`}`), 30, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("claim no-active-assignment status=%d body=%s", rec.Code, rec.Body.String())
		}

		transferPath := "/tickets/" + strconv.FormatUint(uint64(noAccessTicket.ID), 10) + "/transfer"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, transferPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(noAccessActivity.ID), 10)+`,"targetUserId":30}`), 30, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("transfer no-active-assignment status=%d body=%s", rec.Code, rec.Body.String())
		}

		delegatePath := "/tickets/" + strconv.FormatUint(uint64(noAccessTicket.ID), 10) + "/delegate"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, delegatePath, []byte(`{"activityId":`+strconv.FormatUint(uint64(noAccessActivity.ID), 10)+`,"targetUserId":30}`), 30, model.RoleUser)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("delegate no-active-assignment status=%d body=%s", rec.Code, rec.Body.String())
		}

		claimTerminal := "/tickets/" + strconv.FormatUint(uint64(terminal.ID), 10) + "/claim"
		rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, claimTerminal, []byte(`{"activityId":1}`), 20, model.RoleUser)
		if rec.Code != http.StatusConflict {
			t.Fatalf("claim terminal status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestTicketHandlerListAndMonitorHonorCompoundFilters(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	if err := fixture.db.Exec("INSERT INTO user_positions (user_id, position_id, department_id) VALUES (?, ?, ?), (?, ?, ?)", 10, 1001, 501, 30, 1002, 999).Error; err != nil {
		t.Fatalf("seed user_positions: %v", err)
	}

	excludedRequester := uint(30)
	excludedTicket := Ticket{
		Code:        "TICK-HANDLER-MONITOR-EXCLUDED",
		Title:       "不在部门数据域",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: excludedRequester,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&excludedTicket).Error; err != nil {
		t.Fatalf("create excluded ticket: %v", err)
	}

	rec := performTicketHandlerRequest(
		t,
		fixture.router,
		http.MethodGet,
		"/tickets?status=terminal&serviceId="+strconv.FormatUint(uint64(fixture.activeTicket.ServiceID), 10)+
			"&priorityId="+strconv.FormatUint(uint64(fixture.activeTicket.PriorityID), 10)+
			"&requesterId="+strconv.FormatUint(uint64(fixture.activeTicket.RequesterID), 10)+
			"&engineType="+fixture.activeTicket.EngineType+
			"&page=1&pageSize=10",
		20,
		model.RoleUser,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":1`) || !strings.Contains(rec.Body.String(), fixture.historyTicket.Code) {
		t.Fatalf("expected filtered list to return only history ticket, body=%s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), fixture.activeTicket.Code) {
		t.Fatalf("expected active ticket to be excluded by terminal filter, body=%s", rec.Body.String())
	}

	gin.SetMode(gin.TestMode)
	monitorRouter := gin.New()
	monitorRouter.Use(func(c *gin.Context) {
		deptScope := []uint{501}
		c.Set("userId", uint(10))
		c.Set("userRole", model.RoleUser)
		c.Set("deptScope", &deptScope)
		c.Next()
	})
	monitorRouter.GET("/tickets/monitor", fixture.handler.Monitor)

	monitorReq := httptest.NewRequest(
		http.MethodGet,
		"/tickets/monitor?status=active&serviceId="+strconv.FormatUint(uint64(fixture.activeTicket.ServiceID), 10)+
			"&priorityId="+strconv.FormatUint(uint64(fixture.activeTicket.PriorityID), 10)+
			"&engineType="+fixture.activeTicket.EngineType+
			"&page=1&pageSize=20",
		nil,
	)
	monitorRec := httptest.NewRecorder()
	monitorRouter.ServeHTTP(monitorRec, monitorReq)
	if monitorRec.Code != http.StatusOK {
		t.Fatalf("filtered monitor status=%d body=%s", monitorRec.Code, monitorRec.Body.String())
	}
	if !strings.Contains(monitorRec.Body.String(), `"total":1`) || !strings.Contains(monitorRec.Body.String(), fixture.activeTicket.Code) {
		t.Fatalf("expected monitor response to include scoped active ticket only, body=%s", monitorRec.Body.String())
	}
	if strings.Contains(monitorRec.Body.String(), excludedTicket.Code) || strings.Contains(monitorRec.Body.String(), fixture.historyTicket.Code) {
		t.Fatalf("expected monitor response to exclude out-of-scope or terminal tickets, body=%s", monitorRec.Body.String())
	}
}

func TestTicketHandlerListAndMonitorRejectInvalidFilterIDs(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	for _, path := range []string{
		"/tickets?priorityId=bad",
		"/tickets?serviceId=bad",
		"/tickets?assigneeId=bad",
		"/tickets?requesterId=bad",
		"/tickets?status=weird",
		"/tickets?engineType=weird",
		"/tickets?page=bad",
		"/tickets?pageSize=bad",
	} {
		rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, path, 99, model.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("list invalid filter path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}

	gin.SetMode(gin.TestMode)
	monitorRouter := gin.New()
	monitorRouter.Use(func(c *gin.Context) {
		deptScope := []uint{501}
		c.Set("userId", uint(10))
		c.Set("userRole", model.RoleUser)
		c.Set("deptScope", &deptScope)
		c.Next()
	})
	monitorRouter.GET("/tickets/monitor", fixture.handler.Monitor)

	for _, path := range []string{
		"/tickets/monitor?priorityId=bad",
		"/tickets/monitor?serviceId=bad",
		"/tickets/monitor?status=weird",
		"/tickets/monitor?engineType=weird",
		"/tickets/monitor?riskLevel=weird",
		"/tickets/monitor?metricCode=unknown_metric",
		"/tickets/monitor?page=bad",
		"/tickets/monitor?pageSize=bad",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		monitorRouter.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("monitor invalid filter path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTicketHandlerDecisionQualityRejectsInvalidFilters(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	tests := []struct {
		name string
		path string
	}{
		{name: "invalid windowDays", path: "/tickets/decision-quality?windowDays=bad"},
		{name: "out-of-range windowDays zero", path: "/tickets/decision-quality?windowDays=0"},
		{name: "out-of-range windowDays too large", path: "/tickets/decision-quality?windowDays=181"},
		{name: "invalid dimension", path: "/tickets/decision-quality?dimension=weird"},
		{name: "invalid serviceId", path: "/tickets/decision-quality?serviceId=bad"},
		{name: "invalid departmentId", path: "/tickets/decision-quality?departmentId=bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, tt.path, 99, model.RoleAdmin)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTicketHandlerMineAndApprovalViewsRejectInvalidPagingAndDates(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	for _, path := range []string{
		"/tickets/mine?status=weird",
		"/tickets/mine?page=bad",
		"/tickets/mine?pageSize=bad",
		"/tickets/mine?startDate=bad-date",
		"/tickets/mine?endDate=bad-date",
		"/tickets/pending-approvals?page=bad",
		"/tickets/pending-approvals?pageSize=bad",
		"/tickets/approval-history?page=bad",
		"/tickets/approval-history?pageSize=bad",
	} {
		rec := performTicketHandlerRequest(t, fixture.router, http.MethodGet, path, 20, model.RoleUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s expected 400, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTicketHandlerLifecycleActionsUpdateAssignmentsAndSLA(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	assignPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/assign"
	rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, assignPath, []byte(`{"assigneeId":30}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("assign status=%d body=%s", rec.Code, rec.Body.String())
	}

	var assigned Ticket
	if err := fixture.db.First(&assigned, fixture.activeTicket.ID).Error; err != nil {
		t.Fatalf("reload assigned ticket: %v", err)
	}
	if assigned.AssigneeID == nil || *assigned.AssigneeID != 30 || assigned.Status != TicketStatusWaitingHuman {
		t.Fatalf("unexpected assigned ticket: %+v", assigned)
	}
	var reassignedCurrent TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND is_current = ?", fixture.activeTicket.ID, fixture.activeActID, true).First(&reassignedCurrent).Error; err != nil {
		t.Fatalf("reload reassigned assignment: %v", err)
	}
	if reassignedCurrent.ParticipantType != "user" || reassignedCurrent.UserID == nil || *reassignedCurrent.UserID != 30 || reassignedCurrent.AssigneeID == nil || *reassignedCurrent.AssigneeID != 30 {
		t.Fatalf("expected assign to retarget current assignment, got %+v", reassignedCurrent)
	}
	if reassignedCurrent.Status != AssignmentPending || reassignedCurrent.ClaimedAt != nil {
		t.Fatalf("expected assign to reset current assignment to pending, got %+v", reassignedCurrent)
	}

	competingUserID := uint(20)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   fixture.activeTicket.ID,
		ActivityID: fixture.activeActID,
		UserID:     &competingUserID,
		AssigneeID: &competingUserID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create competing assignment: %v", err)
	}

	claimPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/claim"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, claimPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(fixture.activeActID), 10)+`}`), 30, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("claim status=%d body=%s", rec.Code, rec.Body.String())
	}

	var claimed TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", fixture.activeTicket.ID, fixture.activeActID, 30).First(&claimed).Error; err != nil {
		t.Fatalf("load claimed assignment: %v", err)
	}
	if claimed.Status != AssignmentInProgress || claimed.ClaimedAt == nil {
		t.Fatalf("unexpected claimed assignment: %+v", claimed)
	}
	var competing TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", fixture.activeTicket.ID, fixture.activeActID, competingUserID).First(&competing).Error; err != nil {
		t.Fatalf("load competing assignment: %v", err)
	}
	if competing.Status != AssignmentClaimedByOther {
		t.Fatalf("expected competing assignment to be claimed_by_other, got %+v", competing)
	}

	assignNoActiveTicket := Ticket{
		Code:        "TICK-HANDLER-ASSIGN-NO-ACTIVE",
		Title:       "指派缺失待办",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&assignNoActiveTicket).Error; err != nil {
		t.Fatalf("create assign-no-active ticket: %v", err)
	}
	assignNoActiveActivity := TicketActivity{TicketID: assignNoActiveTicket.ID, Name: "无待办活动", ActivityType: engine.NodeProcess, Status: AssignmentPending}
	if err := fixture.db.Create(&assignNoActiveActivity).Error; err != nil {
		t.Fatalf("create assign-no-active activity: %v", err)
	}
	if err := fixture.db.Model(&assignNoActiveTicket).Update("current_activity_id", assignNoActiveActivity.ID).Error; err != nil {
		t.Fatalf("bind assign-no-active activity: %v", err)
	}
	assignNoActivePath := "/tickets/" + strconv.FormatUint(uint64(assignNoActiveTicket.ID), 10) + "/assign"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, assignNoActivePath, []byte(`{"assigneeId":30}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("assign without active assignment status=%d body=%s", rec.Code, rec.Body.String())
	}
	var assignNoActiveReloaded Ticket
	if err := fixture.db.First(&assignNoActiveReloaded, assignNoActiveTicket.ID).Error; err != nil {
		t.Fatalf("reload assign-no-active ticket: %v", err)
	}
	if assignNoActiveReloaded.AssigneeID != nil || assignNoActiveReloaded.Status != TicketStatusWaitingHuman {
		t.Fatalf("assign without active assignment should not mutate ticket, got %+v", assignNoActiveReloaded)
	}

	transferTicket := Ticket{
		Code:        "TICK-HANDLER-TRANSFER",
		Title:       "转办测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&transferTicket).Error; err != nil {
		t.Fatalf("create transfer ticket: %v", err)
	}
	transferActivity := TicketActivity{TicketID: transferTicket.ID, Name: "转办活动", ActivityType: engine.NodeProcess, Status: AssignmentPending}
	if err := fixture.db.Create(&transferActivity).Error; err != nil {
		t.Fatalf("create transfer activity: %v", err)
	}
	if err := fixture.db.Model(&transferTicket).Update("current_activity_id", transferActivity.ID).Error; err != nil {
		t.Fatalf("bind transfer activity: %v", err)
	}
	operatorID := uint(20)
	targetUserID := uint(30)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   transferTicket.ID,
		ActivityID: transferActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create transfer assignment: %v", err)
	}

	transferPath := "/tickets/" + strconv.FormatUint(uint64(transferTicket.ID), 10) + "/transfer"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, transferPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(transferActivity.ID), 10)+`,"targetUserId":30}`), operatorID, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("transfer status=%d body=%s", rec.Code, rec.Body.String())
	}
	var transferredOld TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", transferTicket.ID, transferActivity.ID, operatorID).First(&transferredOld).Error; err != nil {
		t.Fatalf("load original transferred assignment: %v", err)
	}
	if transferredOld.Status != AssignmentTransferred || transferredOld.IsCurrent {
		t.Fatalf("expected original transfer assignment status, got %+v", transferredOld)
	}
	var transferredNew TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", transferTicket.ID, transferActivity.ID, targetUserID).First(&transferredNew).Error; err != nil {
		t.Fatalf("load new transfer assignment: %v", err)
	}
	if transferredNew.TransferFrom == nil || *transferredNew.TransferFrom != transferredOld.ID || transferredNew.Status != AssignmentPending || !transferredNew.IsCurrent {
		t.Fatalf("unexpected new transfer assignment: %+v", transferredNew)
	}

	delegateTicket := Ticket{
		Code:        "TICK-HANDLER-DELEGATE",
		Title:       "委派测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&delegateTicket).Error; err != nil {
		t.Fatalf("create delegate ticket: %v", err)
	}
	delegateActivity := TicketActivity{TicketID: delegateTicket.ID, Name: "委派活动", ActivityType: engine.NodeProcess, Status: AssignmentPending}
	if err := fixture.db.Create(&delegateActivity).Error; err != nil {
		t.Fatalf("create delegate activity: %v", err)
	}
	if err := fixture.db.Model(&delegateTicket).Update("current_activity_id", delegateActivity.ID).Error; err != nil {
		t.Fatalf("bind delegate activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   delegateTicket.ID,
		ActivityID: delegateActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create delegate assignment: %v", err)
	}

	delegatePath := "/tickets/" + strconv.FormatUint(uint64(delegateTicket.ID), 10) + "/delegate"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, delegatePath, []byte(`{"activityId":`+strconv.FormatUint(uint64(delegateActivity.ID), 10)+`,"targetUserId":30}`), operatorID, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("delegate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var delegatedOld TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", delegateTicket.ID, delegateActivity.ID, operatorID).First(&delegatedOld).Error; err != nil {
		t.Fatalf("load original delegated assignment: %v", err)
	}
	if delegatedOld.Status != AssignmentDelegated || delegatedOld.IsCurrent {
		t.Fatalf("expected original delegate assignment status, got %+v", delegatedOld)
	}
	var delegatedNew TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", delegateTicket.ID, delegateActivity.ID, targetUserID).First(&delegatedNew).Error; err != nil {
		t.Fatalf("load new delegated assignment: %v", err)
	}
	if delegatedNew.DelegatedFrom == nil || *delegatedNew.DelegatedFrom != delegatedOld.ID || delegatedNew.Status != AssignmentPending {
		t.Fatalf("unexpected new delegate assignment: %+v", delegatedNew)
	}

	inProgressTicket := Ticket{
		Code:        "TICK-HANDLER-TRANSFER-INPROGRESS",
		Title:       "转办认领中测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&inProgressTicket).Error; err != nil {
		t.Fatalf("create in-progress transfer ticket: %v", err)
	}
	inProgressActivity := TicketActivity{TicketID: inProgressTicket.ID, Name: "认领中转办活动", ActivityType: engine.NodeProcess, Status: engine.ActivityInProgress}
	if err := fixture.db.Create(&inProgressActivity).Error; err != nil {
		t.Fatalf("create in-progress transfer activity: %v", err)
	}
	if err := fixture.db.Model(&inProgressTicket).Update("current_activity_id", inProgressActivity.ID).Error; err != nil {
		t.Fatalf("bind in-progress transfer activity: %v", err)
	}
	claimedAt := time.Now().Add(-time.Minute)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   inProgressTicket.ID,
		ActivityID: inProgressActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentInProgress,
		IsCurrent:  true,
		ClaimedAt:  &claimedAt,
	}).Error; err != nil {
		t.Fatalf("create in-progress transfer assignment: %v", err)
	}

	inProgressTransferPath := "/tickets/" + strconv.FormatUint(uint64(inProgressTicket.ID), 10) + "/transfer"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, inProgressTransferPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(inProgressActivity.ID), 10)+`,"targetUserId":30}`), operatorID, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("transfer in-progress status=%d body=%s", rec.Code, rec.Body.String())
	}

	var inProgressTransferredOld TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", inProgressTicket.ID, inProgressActivity.ID, operatorID).First(&inProgressTransferredOld).Error; err != nil {
		t.Fatalf("load in-progress original transferred assignment: %v", err)
	}
	if inProgressTransferredOld.Status != AssignmentTransferred || inProgressTransferredOld.IsCurrent {
		t.Fatalf("expected in-progress original transfer assignment status, got %+v", inProgressTransferredOld)
	}

	var inProgressTransferredNew TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", inProgressTicket.ID, inProgressActivity.ID, targetUserID).First(&inProgressTransferredNew).Error; err != nil {
		t.Fatalf("load in-progress new transfer assignment: %v", err)
	}
	if inProgressTransferredNew.TransferFrom == nil || *inProgressTransferredNew.TransferFrom != inProgressTransferredOld.ID || inProgressTransferredNew.Status != AssignmentPending || !inProgressTransferredNew.IsCurrent {
		t.Fatalf("unexpected in-progress new transfer assignment: %+v", inProgressTransferredNew)
	}

	inProgressDelegateTicket := Ticket{
		Code:        "TICK-HANDLER-DELEGATE-INPROGRESS",
		Title:       "委派认领中测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&inProgressDelegateTicket).Error; err != nil {
		t.Fatalf("create in-progress delegate ticket: %v", err)
	}
	inProgressDelegateActivity := TicketActivity{TicketID: inProgressDelegateTicket.ID, Name: "认领中委派活动", ActivityType: engine.NodeProcess, Status: engine.ActivityInProgress}
	if err := fixture.db.Create(&inProgressDelegateActivity).Error; err != nil {
		t.Fatalf("create in-progress delegate activity: %v", err)
	}
	if err := fixture.db.Model(&inProgressDelegateTicket).Update("current_activity_id", inProgressDelegateActivity.ID).Error; err != nil {
		t.Fatalf("bind in-progress delegate activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   inProgressDelegateTicket.ID,
		ActivityID: inProgressDelegateActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentInProgress,
		IsCurrent:  true,
		ClaimedAt:  &claimedAt,
	}).Error; err != nil {
		t.Fatalf("create in-progress delegate assignment: %v", err)
	}

	inProgressDelegatePath := "/tickets/" + strconv.FormatUint(uint64(inProgressDelegateTicket.ID), 10) + "/delegate"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, inProgressDelegatePath, []byte(`{"activityId":`+strconv.FormatUint(uint64(inProgressDelegateActivity.ID), 10)+`,"targetUserId":30}`), operatorID, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("delegate in-progress status=%d body=%s", rec.Code, rec.Body.String())
	}

	var inProgressDelegatedOld TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", inProgressDelegateTicket.ID, inProgressDelegateActivity.ID, operatorID).First(&inProgressDelegatedOld).Error; err != nil {
		t.Fatalf("load in-progress original delegated assignment: %v", err)
	}
	if inProgressDelegatedOld.Status != AssignmentDelegated || inProgressDelegatedOld.IsCurrent {
		t.Fatalf("expected in-progress original delegate assignment status, got %+v", inProgressDelegatedOld)
	}

	var inProgressDelegatedNew TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", inProgressDelegateTicket.ID, inProgressDelegateActivity.ID, targetUserID).First(&inProgressDelegatedNew).Error; err != nil {
		t.Fatalf("load in-progress new delegate assignment: %v", err)
	}
	if inProgressDelegatedNew.DelegatedFrom == nil || *inProgressDelegatedNew.DelegatedFrom != inProgressDelegatedOld.ID || inProgressDelegatedNew.Status != AssignmentPending || !inProgressDelegatedNew.IsCurrent {
		t.Fatalf("unexpected in-progress new delegate assignment: %+v", inProgressDelegatedNew)
	}

	cancelTicket := Ticket{
		Code:        "TICK-HANDLER-CANCEL",
		Title:       "取消测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&cancelTicket).Error; err != nil {
		t.Fatalf("create cancel ticket: %v", err)
	}
	cancelActivity := TicketActivity{TicketID: cancelTicket.ID, Name: "取消活动", ActivityType: engine.NodeProcess, Status: engine.ActivityPending}
	if err := fixture.db.Create(&cancelActivity).Error; err != nil {
		t.Fatalf("create cancel activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   cancelTicket.ID,
		ActivityID: cancelActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create cancel assignment: %v", err)
	}

	cancelPath := "/tickets/" + strconv.FormatUint(uint64(cancelTicket.ID), 10) + "/cancel"
	fixture.router.PUT("/tickets/:id/cancel", fixture.handler.Cancel)
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, cancelPath, []byte(`{"reason":"无需处理"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cancelled Ticket
	if err := fixture.db.First(&cancelled, cancelTicket.ID).Error; err != nil {
		t.Fatalf("reload cancelled ticket: %v", err)
	}
	if cancelled.Status != TicketStatusCancelled || cancelled.Outcome != TicketOutcomeCancelled || cancelled.FinishedAt == nil {
		t.Fatalf("unexpected cancelled ticket: %+v", cancelled)
	}
	if cancelled.AssigneeID != nil {
		t.Fatalf("expected cancel to clear assignee, got %+v", cancelled)
	}

	var cancelledActivity TicketActivity
	if err := fixture.db.First(&cancelledActivity, cancelActivity.ID).Error; err != nil {
		t.Fatalf("reload cancelled activity: %v", err)
	}
	if cancelledActivity.Status != engine.ActivityCancelled || cancelledActivity.FinishedAt == nil {
		t.Fatalf("expected cancel to close activity, got %+v", cancelledActivity)
	}

	var cancelledAssignment TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ?", cancelTicket.ID, cancelActivity.ID).First(&cancelledAssignment).Error; err != nil {
		t.Fatalf("reload cancelled assignment: %v", err)
	}
	if cancelledAssignment.Status != engine.ActivityCancelled {
		t.Fatalf("expected cancel to close assignment, got %+v", cancelledAssignment)
	}

	withdrawTicket := Ticket{
		Code:        "TICK-HANDLER-WITHDRAW",
		Title:       "撤回测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&withdrawTicket).Error; err != nil {
		t.Fatalf("create withdraw ticket: %v", err)
	}
	withdrawActivity := TicketActivity{TicketID: withdrawTicket.ID, Name: "撤回活动", ActivityType: engine.NodeProcess, Status: engine.ActivityPending}
	if err := fixture.db.Create(&withdrawActivity).Error; err != nil {
		t.Fatalf("create withdraw activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   withdrawTicket.ID,
		ActivityID: withdrawActivity.ID,
		UserID:     &operatorID,
		AssigneeID: &operatorID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create withdraw assignment: %v", err)
	}

	withdrawPath := "/tickets/" + strconv.FormatUint(uint64(withdrawTicket.ID), 10) + "/withdraw"
	fixture.router.PUT("/tickets/:id/withdraw", fixture.handler.Withdraw)
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, withdrawPath, []byte(`{"reason":"申请有误"}`), withdrawTicket.RequesterID, model.RoleUser)
	if rec.Code != http.StatusOK {
		t.Fatalf("withdraw status=%d body=%s", rec.Code, rec.Body.String())
	}
	var withdrawn Ticket
	if err := fixture.db.First(&withdrawn, withdrawTicket.ID).Error; err != nil {
		t.Fatalf("reload withdrawn ticket: %v", err)
	}
	if withdrawn.Status != TicketStatusWithdrawn || withdrawn.Outcome != TicketOutcomeWithdrawn || withdrawn.FinishedAt == nil {
		t.Fatalf("unexpected withdrawn ticket: %+v", withdrawn)
	}

	now := time.Now()
	responseDeadline := now.Add(30 * time.Minute)
	resolutionDeadline := now.Add(2 * time.Hour)
	if err := fixture.db.Model(&Ticket{}).Where("id = ?", fixture.activeTicket.ID).Updates(map[string]any{
		"sla_response_deadline":   responseDeadline,
		"sla_resolution_deadline": resolutionDeadline,
	}).Error; err != nil {
		t.Fatalf("seed sla deadlines: %v", err)
	}

	slaPausePath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/sla/pause"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, slaPausePath, []byte(`{}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("sla pause status=%d body=%s", rec.Code, rec.Body.String())
	}

	pausedAt := time.Now().Add(-5 * time.Minute)
	if err := fixture.db.Model(&Ticket{}).Where("id = ?", fixture.activeTicket.ID).Update("sla_paused_at", pausedAt).Error; err != nil {
		t.Fatalf("backdate sla paused at: %v", err)
	}

	slaResumePath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/sla/resume"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPut, slaResumePath, []byte(`{}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("sla resume status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resumed Ticket
	if err := fixture.db.First(&resumed, fixture.activeTicket.ID).Error; err != nil {
		t.Fatalf("reload resumed ticket: %v", err)
	}
	if resumed.SLAPausedAt != nil {
		t.Fatalf("expected SLA to be resumed, got paused_at=%v", resumed.SLAPausedAt)
	}
	if resumed.SLAResponseDeadline == nil || resumed.SLAResolutionDeadline == nil {
		t.Fatalf("expected SLA deadlines to remain populated, got %+v", resumed)
	}
	if resumed.SLAResponseDeadline.Sub(responseDeadline) < 4*time.Minute || resumed.SLAResolutionDeadline.Sub(resolutionDeadline) < 4*time.Minute {
		t.Fatalf("expected SLA deadlines to be extended, got response=%v resolution=%v", resumed.SLAResponseDeadline, resumed.SLAResolutionDeadline)
	}
}

func TestTicketHandlerRecoveryActionsForSmartTickets(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	smartRetryTicket := Ticket{
		Code:           "TICK-HANDLER-RETRY-AI",
		Title:          "智能重试测试",
		ServiceID:      fixture.activeTicket.ServiceID,
		EngineType:     "smart",
		Status:         TicketStatusDecisioning,
		PriorityID:     fixture.activeTicket.PriorityID,
		RequesterID:    fixture.activeTicket.RequesterID,
		Source:         TicketSourceAgent,
		AIFailureCount: 2,
	}
	if err := fixture.db.Create(&smartRetryTicket).Error; err != nil {
		t.Fatalf("create smart retry ticket: %v", err)
	}

	retryPath := "/tickets/" + strconv.FormatUint(uint64(smartRetryTicket.ID), 10) + "/retry-ai"
	rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, retryPath, []byte(`{"reason":"重新启用智能决策"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("retry ai status=%d body=%s", rec.Code, rec.Body.String())
	}

	var retried Ticket
	if err := fixture.db.First(&retried, smartRetryTicket.ID).Error; err != nil {
		t.Fatalf("reload retried ticket: %v", err)
	}
	if retried.AIFailureCount != 0 || retried.Status != TicketStatusDecisioning {
		t.Fatalf("unexpected retried ticket: %+v", retried)
	}
	var retryTimeline TicketTimeline
	if err := fixture.db.Where("ticket_id = ? AND event_type = ?", smartRetryTicket.ID, "ai_retry").First(&retryTimeline).Error; err != nil {
		t.Fatalf("load ai_retry timeline: %v", err)
	}
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, retryPath, []byte(`{"reason":"重复重试"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate retry to return 409, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	invalidRetryTicket := Ticket{
		Code:           "TICK-HANDLER-RETRY-AI-BADJSON",
		Title:          "智能重试坏请求",
		ServiceID:      fixture.activeTicket.ServiceID,
		EngineType:     "smart",
		Status:         TicketStatusDecisioning,
		PriorityID:     fixture.activeTicket.PriorityID,
		RequesterID:    fixture.activeTicket.RequesterID,
		Source:         TicketSourceAgent,
		AIFailureCount: 1,
	}
	if err := fixture.db.Create(&invalidRetryTicket).Error; err != nil {
		t.Fatalf("create invalid retry ticket: %v", err)
	}
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/"+strconv.FormatUint(uint64(invalidRetryTicket.ID), 10)+"/retry-ai", []byte(`{"reason":`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid retry payload to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	var invalidReloaded Ticket
	if err := fixture.db.First(&invalidReloaded, invalidRetryTicket.ID).Error; err != nil {
		t.Fatalf("reload invalid retry ticket: %v", err)
	}
	if invalidReloaded.AIFailureCount != 1 || invalidReloaded.Status != TicketStatusDecisioning {
		t.Fatalf("invalid retry payload should not mutate ticket, got %+v", invalidReloaded)
	}
	var invalidRetryTimelines int64
	if err := fixture.db.Model(&TicketTimeline{}).Where("ticket_id = ? AND event_type = ?", invalidRetryTicket.ID, "ai_retry").Count(&invalidRetryTimelines).Error; err != nil {
		t.Fatalf("count invalid retry timelines: %v", err)
	}
	if invalidRetryTimelines != 0 {
		t.Fatalf("invalid retry payload should not write timeline, got %d", invalidRetryTimelines)
	}

	manualRetryTicket := Ticket{
		Code:        "TICK-HANDLER-RETRY-AI-MANUAL",
		Title:       "手工工单错误重试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  "manual",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
	}
	if err := fixture.db.Create(&manualRetryTicket).Error; err != nil {
		t.Fatalf("create manual retry ticket: %v", err)
	}
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/"+strconv.FormatUint(uint64(manualRetryTicket.ID), 10)+"/retry-ai", []byte(`{"reason":"不应重试 AI"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected manual retry to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	smartRecoverTicket := Ticket{
		Code:        "TICK-HANDLER-RECOVER",
		Title:       "智能恢复测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceAgent,
	}
	if err := fixture.db.Create(&smartRecoverTicket).Error; err != nil {
		t.Fatalf("create smart recover ticket: %v", err)
	}

	recoverPath := "/tickets/" + strconv.FormatUint(uint64(smartRecoverTicket.ID), 10) + "/recover"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, recoverPath, []byte(`{"action":"handoff_human","reason":"需要人工接手"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("recover handoff status=%d body=%s", rec.Code, rec.Body.String())
	}

	var recovered Ticket
	if err := fixture.db.First(&recovered, smartRecoverTicket.ID).Error; err != nil {
		t.Fatalf("reload recovered ticket: %v", err)
	}
	if recovered.Status != TicketStatusWaitingHuman || recovered.CurrentActivityID == nil {
		t.Fatalf("unexpected recovered ticket: %+v", recovered)
	}
	var recoverTimeline TicketTimeline
	if err := fixture.db.Where("ticket_id = ? AND event_type = ?", smartRecoverTicket.ID, "recovery_handoff_human").First(&recoverTimeline).Error; err != nil {
		t.Fatalf("load recovery_handoff_human timeline: %v", err)
	}
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, recoverPath, []byte(`{"action":"handoff_human","reason":"重复转人工"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duplicate handoff to return 409, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	manualRecoverTicket := Ticket{
		Code:        "TICK-HANDLER-RECOVER-MANUAL",
		Title:       "手工恢复测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  "manual",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
	}
	if err := fixture.db.Create(&manualRecoverTicket).Error; err != nil {
		t.Fatalf("create manual recover ticket: %v", err)
	}
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/"+strconv.FormatUint(uint64(manualRecoverTicket.ID), 10)+"/recover", []byte(`{"action":"retry","reason":"不应重试"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected manual recover retry to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, recoverPath, []byte(`{"action":`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid recover payload to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, recoverPath, []byte(`{"action":"unknown"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid recover action to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTicketHandlerProgressAndSignalGuards(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	progressPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/progress"
	rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/not-a-number/progress", []byte(`{"activityId":1,"outcome":"approved"}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid progress id to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, progressPath, []byte(`{"activityId":`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid progress payload to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/999999/progress", []byte(`{"activityId":1,"outcome":"approved"}`), 20, model.RoleUser)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing ticket progress to return 404, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, progressPath, []byte(`{"activityId":999999,"outcome":"approved","opinion":"missing activity"}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing activity progress to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	terminalProgressPath := "/tickets/" + strconv.FormatUint(uint64(fixture.historyTicket.ID), 10) + "/progress"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, terminalProgressPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(fixture.activeActID), 10)+`,"outcome":"approved","opinion":"terminal"}`), 30, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected terminal progress to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, progressPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(fixture.activeActID), 10)+`,"outcome":"invalid","opinion":"bad"}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid outcome to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	smartProgressTicket := Ticket{
		Code:        "TICK-HANDLER-PROGRESS-SMART",
		Title:       "智能进度测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceAgent,
	}
	if err := fixture.db.Create(&smartProgressTicket).Error; err != nil {
		t.Fatalf("create smart progress ticket: %v", err)
	}
	smartActivity := TicketActivity{
		TicketID:     smartProgressTicket.ID,
		Name:         "智能审批",
		ActivityType: engine.NodeApprove,
		Status:       engine.ActivityPending,
	}
	if err := fixture.db.Create(&smartActivity).Error; err != nil {
		t.Fatalf("create smart activity: %v", err)
	}
	assigneeID := uint(20)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   smartProgressTicket.ID,
		ActivityID: smartActivity.ID,
		UserID:     &assigneeID,
		AssigneeID: &assigneeID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create smart progress assignment: %v", err)
	}

	progressPath = "/tickets/" + strconv.FormatUint(uint64(smartProgressTicket.ID), 10) + "/progress"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, progressPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(smartActivity.ID), 10)+`,"outcome":"approved","opinion":"越权处理"}`), 30, model.RoleUser)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected unauthorized progress to return 403, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	var guardedAssignment TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", smartProgressTicket.ID, smartActivity.ID, assigneeID).First(&guardedAssignment).Error; err != nil {
		t.Fatalf("reload guarded assignment: %v", err)
	}
	if guardedAssignment.Status != AssignmentPending {
		t.Fatalf("unauthorized progress mutated assignment: %+v", guardedAssignment)
	}

	signalPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/signal"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/not-a-number/signal", []byte(`{"activityId":1,"outcome":"done"}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid signal id to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, signalPath, []byte(`{"activityId":`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid signal payload to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, "/tickets/999999/signal", []byte(`{"activityId":1,"outcome":"done"}`), 20, model.RoleUser)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing ticket signal to return 404, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, signalPath, []byte(`{"activityId":999999,"outcome":"done","data":{"ok":true}}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing activity signal to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	otherWaitTicket := Ticket{
		Code:        "TICK-HANDLER-SIGNAL-OTHER",
		Title:       "跨工单 signal",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&otherWaitTicket).Error; err != nil {
		t.Fatalf("create cross-ticket wait ticket: %v", err)
	}
	otherWaitActivity := TicketActivity{
		TicketID:     otherWaitTicket.ID,
		Name:         "另一张单子的 wait",
		ActivityType: engine.NodeWait,
		Status:       engine.ActivityPending,
	}
	if err := fixture.db.Create(&otherWaitActivity).Error; err != nil {
		t.Fatalf("create cross-ticket wait activity: %v", err)
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, signalPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(otherWaitActivity.ID), 10)+`,"outcome":"done","data":{"ok":true}}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected cross-ticket signal to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, signalPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(fixture.activeActID), 10)+`,"outcome":"done","data":{"ok":true}}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected non-wait signal to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	waitTicket := Ticket{
		Code:        "TICK-HANDLER-WAIT",
		Title:       "等待节点测试",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusCompleted,
		Outcome:     TicketOutcomeFulfilled,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		Source:      TicketSourceCatalog,
		FinishedAt:  ptrTime(time.Now()),
	}
	if err := fixture.db.Create(&waitTicket).Error; err != nil {
		t.Fatalf("create wait ticket: %v", err)
	}
	waitActivity := TicketActivity{
		TicketID:     waitTicket.ID,
		Name:         "等待外部信号",
		ActivityType: engine.NodeWait,
		Status:       engine.ActivityPending,
	}
	if err := fixture.db.Create(&waitActivity).Error; err != nil {
		t.Fatalf("create wait activity: %v", err)
	}

	terminalSignalPath := "/tickets/" + strconv.FormatUint(uint64(waitTicket.ID), 10) + "/signal"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, terminalSignalPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(waitActivity.ID), 10)+`,"outcome":"done","data":{"ok":true}}`), 20, model.RoleUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected terminal signal to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTicketHandlerOverrideActionsForceManualIntervention(t *testing.T) {
	fixture := newTicketHandlerFixture(t)

	jumpPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/override-jump"
	rec := performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, jumpPath, []byte(`{"activityType":"approve","assigneeId":30,"reason":"需要主管审批"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("override jump status=%d body=%s", rec.Code, rec.Body.String())
	}

	var jumped Ticket
	if err := fixture.db.First(&jumped, fixture.activeTicket.ID).Error; err != nil {
		t.Fatalf("reload jumped ticket: %v", err)
	}
	if jumped.CurrentActivityID == nil || *jumped.CurrentActivityID == fixture.activeActID || jumped.AssigneeID == nil || *jumped.AssigneeID != 30 {
		t.Fatalf("unexpected jumped ticket: %+v", jumped)
	}
	if jumped.Status != TicketStatusWaitingHuman || jumped.Outcome != "" {
		t.Fatalf("override jump should reset status/outcome, got %+v", jumped)
	}

	var cancelled TicketActivity
	if err := fixture.db.First(&cancelled, fixture.activeActID).Error; err != nil {
		t.Fatalf("load cancelled activity: %v", err)
	}
	if cancelled.Status != engine.ActivityCancelled || cancelled.OverriddenBy == nil || *cancelled.OverriddenBy != 99 {
		t.Fatalf("unexpected cancelled activity: %+v", cancelled)
	}

	var jumpedActivity TicketActivity
	if err := fixture.db.First(&jumpedActivity, *jumped.CurrentActivityID).Error; err != nil {
		t.Fatalf("load jumped activity: %v", err)
	}
	if jumpedActivity.ActivityType != engine.NodeApprove || jumpedActivity.OverriddenBy == nil || *jumpedActivity.OverriddenBy != 99 {
		t.Fatalf("unexpected jumped activity: %+v", jumpedActivity)
	}

	var jumpedAssignment TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ?", jumped.ID, jumpedActivity.ID).First(&jumpedAssignment).Error; err != nil {
		t.Fatalf("load jumped assignment: %v", err)
	}
	if jumpedAssignment.AssigneeID == nil || *jumpedAssignment.AssigneeID != 30 || jumpedAssignment.Status != AssignmentPending {
		t.Fatalf("unexpected jumped assignment: %+v", jumpedAssignment)
	}

	unassignedTicket := Ticket{
		Code:        "TICK-HANDLER-OVERRIDE-CLEAR",
		Title:       "override clear assignee",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		AssigneeID:  fixture.activeTicket.AssigneeID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&unassignedTicket).Error; err != nil {
		t.Fatalf("create unassigned override ticket: %v", err)
	}
	unassignedActivity := TicketActivity{TicketID: unassignedTicket.ID, Name: "待处理", ActivityType: engine.NodeProcess, Status: engine.ActivityPending}
	if err := fixture.db.Create(&unassignedActivity).Error; err != nil {
		t.Fatalf("create unassigned override activity: %v", err)
	}
	if err := fixture.db.Model(&unassignedTicket).Update("current_activity_id", unassignedActivity.ID).Error; err != nil {
		t.Fatalf("bind unassigned override activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   unassignedTicket.ID,
		ActivityID: unassignedActivity.ID,
		UserID:     fixture.activeTicket.AssigneeID,
		AssigneeID: fixture.activeTicket.AssigneeID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create unassigned override assignment: %v", err)
	}

	unassignedJumpPath := "/tickets/" + strconv.FormatUint(uint64(unassignedTicket.ID), 10) + "/override-jump"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, unassignedJumpPath, []byte(`{"activityType":"process","reason":"转人工重新认领"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("override jump without assignee status=%d body=%s", rec.Code, rec.Body.String())
	}

	var cleared Ticket
	if err := fixture.db.First(&cleared, unassignedTicket.ID).Error; err != nil {
		t.Fatalf("reload cleared override ticket: %v", err)
	}
	if cleared.AssigneeID != nil {
		t.Fatalf("expected override jump without assignee to clear ticket assignee, got %+v", cleared)
	}
	if cleared.CurrentActivityID == nil || *cleared.CurrentActivityID == unassignedActivity.ID {
		t.Fatalf("expected new current activity for cleared override ticket, got %+v", cleared)
	}

	var clearedOldAssignment TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ?", unassignedTicket.ID, unassignedActivity.ID).First(&clearedOldAssignment).Error; err != nil {
		t.Fatalf("load cleared old assignment: %v", err)
	}
	if clearedOldAssignment.Status != AssignmentCancelled || clearedOldAssignment.IsCurrent {
		t.Fatalf("expected old assignment cancelled and not current, got %+v", clearedOldAssignment)
	}

	var newAssignmentCount int64
	if err := fixture.db.Model(&TicketAssignment{}).Where("ticket_id = ? AND activity_id = ?", unassignedTicket.ID, *cleared.CurrentActivityID).Count(&newAssignmentCount).Error; err != nil {
		t.Fatalf("count new override assignments: %v", err)
	}
	if newAssignmentCount != 0 {
		t.Fatalf("expected no assignment for override jump without assignee, got %d", newAssignmentCount)
	}

	explicitZeroTicket := Ticket{
		Code:        "TICK-HANDLER-OVERRIDE-ZERO",
		Title:       "override zero assignee",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		AssigneeID:  fixture.activeTicket.AssigneeID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&explicitZeroTicket).Error; err != nil {
		t.Fatalf("create explicit zero override ticket: %v", err)
	}
	explicitZeroActivity := TicketActivity{TicketID: explicitZeroTicket.ID, Name: "待处理", ActivityType: engine.NodeProcess, Status: engine.ActivityPending}
	if err := fixture.db.Create(&explicitZeroActivity).Error; err != nil {
		t.Fatalf("create explicit zero override activity: %v", err)
	}
	if err := fixture.db.Model(&explicitZeroTicket).Update("current_activity_id", explicitZeroActivity.ID).Error; err != nil {
		t.Fatalf("bind explicit zero override activity: %v", err)
	}
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   explicitZeroTicket.ID,
		ActivityID: explicitZeroActivity.ID,
		UserID:     fixture.activeTicket.AssigneeID,
		AssigneeID: fixture.activeTicket.AssigneeID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create explicit zero override assignment: %v", err)
	}

	explicitZeroJumpPath := "/tickets/" + strconv.FormatUint(uint64(explicitZeroTicket.ID), 10) + "/override-jump"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, explicitZeroJumpPath, []byte(`{"activityType":"action","assigneeId":0,"reason":"转自动动作"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("override jump with explicit zero assignee status=%d body=%s", rec.Code, rec.Body.String())
	}

	var explicitZeroReloaded Ticket
	if err := fixture.db.First(&explicitZeroReloaded, explicitZeroTicket.ID).Error; err != nil {
		t.Fatalf("reload explicit zero override ticket: %v", err)
	}
	if explicitZeroReloaded.AssigneeID != nil || explicitZeroReloaded.Status != TicketStatusExecutingAction {
		t.Fatalf("expected explicit zero override to clear assignee and enter executing_action, got %+v", explicitZeroReloaded)
	}
	var explicitZeroNewAssignmentCount int64
	if err := fixture.db.Model(&TicketAssignment{}).Where("ticket_id = ? AND activity_id = ?", explicitZeroTicket.ID, *explicitZeroReloaded.CurrentActivityID).Count(&explicitZeroNewAssignmentCount).Error; err != nil {
		t.Fatalf("count explicit zero new assignments: %v", err)
	}
	if explicitZeroNewAssignmentCount != 0 {
		t.Fatalf("expected no assignment for explicit zero override jump, got %d", explicitZeroNewAssignmentCount)
	}

	claimedOtherTicket := Ticket{
		Code:        "TICK-HANDLER-OVERRIDE-CLAIMED-OTHER",
		Title:       "override claimed by other",
		ServiceID:   fixture.activeTicket.ServiceID,
		EngineType:  fixture.activeTicket.EngineType,
		Status:      TicketStatusWaitingHuman,
		PriorityID:  fixture.activeTicket.PriorityID,
		RequesterID: fixture.activeTicket.RequesterID,
		AssigneeID:  fixture.activeTicket.AssigneeID,
		Source:      TicketSourceCatalog,
	}
	if err := fixture.db.Create(&claimedOtherTicket).Error; err != nil {
		t.Fatalf("create claimed_other override ticket: %v", err)
	}
	claimedOtherActivity := TicketActivity{TicketID: claimedOtherTicket.ID, Name: "待处理", ActivityType: engine.NodeProcess, Status: engine.ActivityInProgress}
	if err := fixture.db.Create(&claimedOtherActivity).Error; err != nil {
		t.Fatalf("create claimed_other activity: %v", err)
	}
	if err := fixture.db.Model(&claimedOtherTicket).Update("current_activity_id", claimedOtherActivity.ID).Error; err != nil {
		t.Fatalf("bind claimed_other activity: %v", err)
	}
	claimedAt := time.Now().Add(-time.Minute)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   claimedOtherTicket.ID,
		ActivityID: claimedOtherActivity.ID,
		UserID:     fixture.activeTicket.AssigneeID,
		AssigneeID: fixture.activeTicket.AssigneeID,
		Status:     AssignmentInProgress,
		IsCurrent:  true,
		ClaimedAt:  &claimedAt,
	}).Error; err != nil {
		t.Fatalf("create claimed_other primary assignment: %v", err)
	}
	claimedOtherUserID := uint(31)
	if err := fixture.db.Create(&TicketAssignment{
		TicketID:   claimedOtherTicket.ID,
		ActivityID: claimedOtherActivity.ID,
		UserID:     &claimedOtherUserID,
		AssigneeID: &claimedOtherUserID,
		Status:     AssignmentClaimedByOther,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create claimed_other companion assignment: %v", err)
	}

	claimedOtherJumpPath := "/tickets/" + strconv.FormatUint(uint64(claimedOtherTicket.ID), 10) + "/override-jump"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, claimedOtherJumpPath, []byte(`{"activityType":"process","reason":"切换流程"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("override jump with claimed_by_other status=%d body=%s", rec.Code, rec.Body.String())
	}

	var claimedOtherCompanion TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ? AND assignee_id = ?", claimedOtherTicket.ID, claimedOtherActivity.ID, claimedOtherUserID).First(&claimedOtherCompanion).Error; err != nil {
		t.Fatalf("reload claimed_other companion assignment: %v", err)
	}
	if claimedOtherCompanion.Status != AssignmentCancelled || claimedOtherCompanion.IsCurrent {
		t.Fatalf("expected claimed_by_other companion assignment cancelled, got %+v", claimedOtherCompanion)
	}

	invalidTypeJumpPath := "/tickets/" + strconv.FormatUint(uint64(fixture.activeTicket.ID), 10) + "/override-jump"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, invalidTypeJumpPath, []byte(`{"activityType":"bogus_node","reason":"非法跳转"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid override jump activity type to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	reassignPath := "/tickets/" + strconv.FormatUint(uint64(jumped.ID), 10) + "/override-reassign"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, reassignPath, []byte(`{"activityId":`+strconv.FormatUint(uint64(jumpedActivity.ID), 10)+`,"newAssigneeId":20,"reason":"主管回退给执行人"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("override reassign status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, reassignPath, []byte(`{"activityId":999999,"newAssigneeId":20,"reason":"缺失待办不应改派"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("override reassign without current assignment status=%d body=%s", rec.Code, rec.Body.String())
	}

	var reassigned Ticket
	if err := fixture.db.First(&reassigned, jumped.ID).Error; err != nil {
		t.Fatalf("reload reassigned ticket: %v", err)
	}
	if reassigned.AssigneeID == nil || *reassigned.AssigneeID != 20 || reassigned.Status != TicketStatusWaitingHuman {
		t.Fatalf("unexpected reassigned ticket: %+v", reassigned)
	}

	var reassignedAssignment TicketAssignment
	if err := fixture.db.Where("ticket_id = ? AND activity_id = ?", jumped.ID, jumpedActivity.ID).First(&reassignedAssignment).Error; err != nil {
		t.Fatalf("load reassigned assignment: %v", err)
	}
	if reassignedAssignment.AssigneeID == nil || *reassignedAssignment.AssigneeID != 20 {
		t.Fatalf("unexpected reassigned assignment: %+v", reassignedAssignment)
	}

	var overrideJumpTimeline TicketTimeline
	if err := fixture.db.Where("ticket_id = ? AND event_type = ?", jumped.ID, "override_jump").First(&overrideJumpTimeline).Error; err != nil {
		t.Fatalf("load override_jump timeline: %v", err)
	}
	var overrideReassignTimeline TicketTimeline
	if err := fixture.db.Where("ticket_id = ? AND event_type = ?", jumped.ID, "override_reassign").First(&overrideReassignTimeline).Error; err != nil {
		t.Fatalf("load override_reassign timeline: %v", err)
	}

	terminalJumpPath := "/tickets/" + strconv.FormatUint(uint64(fixture.historyTicket.ID), 10) + "/override-jump"
	rec = performTicketHandlerJSONRequest(t, fixture.router, http.MethodPost, terminalJumpPath, []byte(`{"activityType":"process","reason":"终态校验"}`), 99, model.RoleAdmin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected terminal override jump to return 400, got status=%d body=%s", rec.Code, rec.Body.String())
	}
}
