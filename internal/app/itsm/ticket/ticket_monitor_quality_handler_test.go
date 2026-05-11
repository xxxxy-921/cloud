package ticket

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
	"metis/internal/model"

	"github.com/gin-gonic/gin"
)

func TestTicketHandlerMonitorReturnsRiskSummary(t *testing.T) {
	db := newTestDB(t)
	service, priority, requester := seedTicketMonitorBase(t, db)
	svc := newTicketMonitorServiceForTest(db)
	handler := &TicketHandler{svc: svc}

	now := time.Now()
	aiTicket := createMonitorTicket(t, db, service, priority, requester, func(ticket *Ticket) {
		ticket.Code = "TICK-HANDLER-MONITOR-AI"
		ticket.AIFailureCount = 3
	})
	humanTicket := createMonitorTicket(t, db, service, priority, requester, func(ticket *Ticket) {
		ticket.Code = "TICK-HANDLER-MONITOR-HUMAN"
		ticket.EngineType = "classic"
		ticket.Source = TicketSourceCatalog
	})
	createCurrentActivity(t, db, humanTicket, "process", now.Add(-15*time.Minute))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", requester.ID)
		c.Set("userRole", model.RoleUser)
		c.Next()
	})
	router.GET("/tickets/monitor", handler.Monitor)

	req := httptest.NewRequest(http.MethodGet, "/tickets/monitor?page=1&pageSize=20&riskLevel=blocked", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("monitor status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"activeTotal":2`) || !strings.Contains(rec.Body.String(), `"stuckTotal":2`) {
		t.Fatalf("unexpected monitor summary: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), aiTicket.Code) || !strings.Contains(rec.Body.String(), humanTicket.Code) {
		t.Fatalf("expected monitor response to include seeded tickets: %s", rec.Body.String())
	}
}

func TestTicketHandlerDecisionQualityReturnsAggregatedMetrics(t *testing.T) {
	db := newDecisionQualityDB(t)
	now := time.Now()

	service := ServiceDefinition{Name: "VPN 开通", EngineType: "smart", IsActive: true}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS departments (id INTEGER PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatalf("create departments table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS user_positions (id INTEGER PRIMARY KEY, user_id INTEGER, department_id INTEGER)`).Error; err != nil {
		t.Fatalf("create user_positions table: %v", err)
	}
	if err := db.Exec(`INSERT INTO departments (id, name) VALUES (10, '网络部'), (20, '安全部')`).Error; err != nil {
		t.Fatalf("seed departments: %v", err)
	}
	if err := db.Exec(`INSERT INTO user_positions (user_id, department_id) VALUES (1, 10), (2, 20)`).Error; err != nil {
		t.Fatalf("seed user_positions: %v", err)
	}

	ticket1 := Ticket{Code: "TICK-HANDLER-DQ-1", Title: "VPN 申请", ServiceID: service.ID, EngineType: "smart", Status: TicketStatusCompleted, Outcome: TicketOutcomeApproved, PriorityID: 1, RequesterID: 1}
	ticket2 := Ticket{Code: "TICK-HANDLER-DQ-2", Title: "账号放行", ServiceID: service.ID, EngineType: "smart", Status: TicketStatusFailed, Outcome: TicketOutcomeFailed, PriorityID: 1, RequesterID: 2}
	if err := db.Create(&ticket1).Error; err != nil {
		t.Fatalf("create ticket1: %v", err)
	}
	if err := db.Create(&ticket2).Error; err != nil {
		t.Fatalf("create ticket2: %v", err)
	}
	activity1 := TicketActivity{TicketID: ticket1.ID, ActivityType: "process", TransitionOutcome: TicketOutcomeApproved, Status: "approved", FinishedAt: &now}
	activity2 := TicketActivity{TicketID: ticket2.ID, ActivityType: "process", TransitionOutcome: TicketOutcomeRejected, Status: "rejected", FinishedAt: &now}
	if err := db.Create(&activity1).Error; err != nil {
		t.Fatalf("create activity1: %v", err)
	}
	if err := db.Create(&activity2).Error; err != nil {
		t.Fatalf("create activity2: %v", err)
	}
	for _, tl := range []TicketTimeline{
		{TicketID: ticket1.ID, EventType: "activity_completed", BaseModel: model.BaseModel{CreatedAt: now.Add(-40 * time.Minute)}},
		{TicketID: ticket1.ID, EventType: "ai_decision_executed", BaseModel: model.BaseModel{CreatedAt: now.Add(-35 * time.Minute)}},
		{TicketID: ticket1.ID, EventType: "recovery_retry", BaseModel: model.BaseModel{CreatedAt: now.Add(-20 * time.Minute)}},
		{TicketID: ticket1.ID, EventType: "workflow_completed", BaseModel: model.BaseModel{CreatedAt: now.Add(-10 * time.Minute)}},
		{TicketID: ticket2.ID, EventType: "activity_completed", BaseModel: model.BaseModel{CreatedAt: now.Add(-30 * time.Minute)}},
		{TicketID: ticket2.ID, EventType: "ai_decision_failed", BaseModel: model.BaseModel{CreatedAt: now.Add(-29 * time.Minute)}},
		{TicketID: ticket2.ID, EventType: "recovery_handoff_human", BaseModel: model.BaseModel{CreatedAt: now.Add(-15 * time.Minute)}},
	} {
		row := tl
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create timeline row %s: %v", row.EventType, err)
		}
	}

	handler := &TicketHandler{svc: &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/tickets/decision-quality", handler.DecisionQuality)

	req := httptest.NewRequest(http.MethodGet, "/tickets/decision-quality?dimension=service&serviceId="+strconv.FormatUint(uint64(service.ID), 10), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("decision quality status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), DecisionQualityMetricVersion) || !strings.Contains(rec.Body.String(), `"decisionCount":2`) {
		t.Fatalf("unexpected decision quality response: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"approvalRate":0.5`) || !strings.Contains(rec.Body.String(), `"retryRate":0.5`) {
		t.Fatalf("expected aggregated rates in response: %s", rec.Body.String())
	}
}
