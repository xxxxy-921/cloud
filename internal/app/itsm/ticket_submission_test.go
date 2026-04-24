package itsm

import (
	"context"
	"encoding/json"
	"testing"

	appcore "metis/internal/app"
	"metis/internal/app/ai"
	"metis/internal/app/itsm/engine"
	"metis/internal/app/itsm/tools"
	"metis/internal/database"
	"metis/internal/model"

	"gorm.io/gorm"
)

type submissionTestDecisionExecutor struct{}

func (submissionTestDecisionExecutor) Execute(context.Context, uint, appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	return nil, nil
}

func TestCreateFromAgent_IdempotentConfirmedDraftStartsSmartProgressTask(t *testing.T) {
	db := newTestDB(t)
	ticketSvc := newSubmissionTicketService(t, db)
	service := seedSmartSubmissionService(t, db)

	req := tools.AgentTicketRequest{
		UserID:       7,
		ServiceID:    service.ID,
		Summary:      "VPN 开通申请",
		FormData:     map[string]any{"vpn_account": "admin@dev.com", "request_kind": "线上支持"},
		SessionID:    99,
		DraftVersion: 3,
		FieldsHash:   "fields-v1",
		RequestHash:  "request-v1",
	}

	first, err := ticketSvc.CreateFromAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("first create from agent: %v", err)
	}
	second, err := ticketSvc.CreateFromAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("second create from agent: %v", err)
	}
	if second.TicketID != first.TicketID || second.TicketCode != first.TicketCode {
		t.Fatalf("expected idempotent ticket result, first=%+v second=%+v", first, second)
	}

	var ticketCount int64
	if err := db.Model(&Ticket{}).Count(&ticketCount).Error; err != nil {
		t.Fatalf("count tickets: %v", err)
	}
	if ticketCount != 1 {
		t.Fatalf("expected one ticket after duplicate submit, got %d", ticketCount)
	}

	var ticket Ticket
	if err := db.First(&ticket, first.TicketID).Error; err != nil {
		t.Fatalf("load ticket: %v", err)
	}
	if ticket.Source != TicketSourceAgent {
		t.Fatalf("expected source=agent, got %q", ticket.Source)
	}
	if ticket.AgentSessionID == nil || *ticket.AgentSessionID != req.SessionID {
		t.Fatalf("expected agent_session_id=%d, got %v", req.SessionID, ticket.AgentSessionID)
	}

	var submissions []ServiceDeskSubmission
	if err := db.Find(&submissions).Error; err != nil {
		t.Fatalf("list submissions: %v", err)
	}
	if len(submissions) != 1 || submissions[0].TicketID != first.TicketID || submissions[0].Status != "submitted" {
		t.Fatalf("unexpected submissions: %+v", submissions)
	}

	var draftTimeline TicketTimeline
	if err := db.Where("ticket_id = ? AND event_type = ?", first.TicketID, "draft_submitted").First(&draftTimeline).Error; err != nil {
		t.Fatalf("load draft_submitted timeline: %v", err)
	}

	var task model.TaskExecution
	if err := db.Where("task_name = ?", "itsm-smart-progress").First(&task).Error; err != nil {
		t.Fatalf("load smart progress task: %v", err)
	}
	var payload engine.SmartProgressPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.TicketID != first.TicketID || payload.TriggerReason != "initial_decision" || payload.CompletedActivityID != nil {
		t.Fatalf("unexpected smart progress payload: %+v", payload)
	}
}

func newSubmissionTicketService(t *testing.T, db *gorm.DB) *TicketService {
	t.Helper()
	wrapped := &database.DB{DB: db}
	resolver := engine.NewParticipantResolver(nil)
	return &TicketService{
		ticketRepo:    &TicketRepo{db: wrapped},
		timelineRepo:  &TimelineRepo{db: wrapped},
		serviceRepo:   &ServiceDefRepo{db: wrapped},
		slaRepo:       &SLATemplateRepo{db: wrapped},
		priorityRepo:  &PriorityRepo{db: wrapped},
		classicEngine: engine.NewClassicEngine(resolver, nil, nil),
		smartEngine:   engine.NewSmartEngine(submissionTestDecisionExecutor{}, nil, nil, resolver, &schedulerSubmitter{db: db}, nil),
	}
}

func seedSmartSubmissionService(t *testing.T, db *gorm.DB) ServiceDefinition {
	t.Helper()
	priority := Priority{Name: "P3", Code: "P3", Value: 3, Color: "#64748b", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}
	catalog := ServiceCatalog{Name: "账号与权限", Code: "account", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	agent := ai.Agent{Name: "流程决策智能体", Type: ai.AgentTypeAssistant, IsActive: true}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	service := ServiceDefinition{
		Name:              "VPN 开通申请",
		Code:              "vpn-access",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "收到申请后分配网络管理员处理。",
		AgentID:           &agent.ID,
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	return service
}
