package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appcore "metis/internal/app"
	"metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/testutil"
)

type recordingTicketCreator struct {
	req    AgentTicketRequest
	result *AgentTicketResult
	err    error
}

func (r *recordingTicketCreator) CreateFromAgent(_ context.Context, req AgentTicketRequest) (*AgentTicketResult, error) {
	r.req = req
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

type recordingMatcher struct {
	query    string
	matches  []ServiceMatch
	decision MatchDecision
	err      error
}

func (m *recordingMatcher) MatchServices(_ context.Context, query string) ([]ServiceMatch, MatchDecision, error) {
	m.query = query
	if m.err != nil {
		return nil, MatchDecision{}, m.err
	}
	return m.matches, m.decision, nil
}

type ticketListWithdrawStub struct {
	tickets        []TicketSummary
	listStatus     string
	withdrawCode   string
	withdrawReason string
	withdrawUserID uint
	listErr        error
	withdrawErr    error
}

func (s *ticketListWithdrawStub) MatchServices(context.Context, string) ([]ServiceMatch, MatchDecision, error) {
	return nil, MatchDecision{}, nil
}

func (s *ticketListWithdrawStub) LoadService(uint) (*ServiceDetail, error) { return nil, nil }
func (s *ticketListWithdrawStub) CreateTicket(uint, uint, string, map[string]any, uint) (*TicketResult, error) {
	return nil, nil
}
func (s *ticketListWithdrawStub) SubmitConfirmedDraft(uint, uint, uint, string, map[string]any, uint, int, string, string) (*TicketResult, error) {
	return nil, nil
}
func (s *ticketListWithdrawStub) ListMyTickets(userID uint, status string) ([]TicketSummary, error) {
	s.listStatus = status
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.tickets, nil
}
func (s *ticketListWithdrawStub) WithdrawTicket(userID uint, ticketCode string, reason string) error {
	s.withdrawUserID = userID
	s.withdrawCode = ticketCode
	s.withdrawReason = reason
	return s.withdrawErr
}
func (s *ticketListWithdrawStub) ValidateParticipants(uint, map[string]any) (*ParticipantValidation, error) {
	return &ParticipantValidation{OK: true}, nil
}

func TestServiceDeskSessionStateViewResetAndCommandPayloadContracts(t *testing.T) {
	store := newMemStateStore()
	store.states[11] = &ServiceDeskState{
		Stage:                   "awaiting_confirmation",
		LoadedServiceID:         7,
		ServiceVersionID:        12,
		RequestText:             "申请 VPN",
		DraftSummary:            "VPN 开通申请",
		DraftVersion:            3,
		ConfirmedDraftVersion:   2,
		PendingNextRequiredTool: "itsm.draft_confirm",
	}
	session := NewServiceDeskSession(nil, store)

	view, err := session.StateView(11)
	if err != nil {
		t.Fatalf("state view: %v", err)
	}
	if !view.OK || view.NextExpectedAction != "itsm.draft_confirm" {
		t.Fatalf("unexpected state view: %+v", view)
	}
	if view.Payload["next_required_tool"] != "itsm.draft_confirm" {
		t.Fatalf("expected pending next tool in payload, got %+v", view.Payload)
	}

	payload := CommandPayload(&ServiceDeskCommandResult{
		OK:                 true,
		NextExpectedAction: "itsm.ticket_create",
		State:              store.states[11],
		Payload:            map[string]any{"ok": false, "custom": "kept"},
		Message:            "ready",
		Surface:            map[string]any{"kind": "draft"},
		Warnings:           []DraftWarning{{Type: "warn", Field: "vpn_account", Message: "补充账号"}},
		MissingFields:      []FieldCollectionItem{{Key: "vpn_account", Label: "VPN账号"}},
	})
	if payload["ok"] != true || payload["custom"] != "kept" {
		t.Fatalf("expected command payload to preserve custom fields and overwrite ok, got %+v", payload)
	}
	if payload["message"] != "ready" || payload["nextExpectedAction"] != "itsm.ticket_create" || payload["next_expected_action"] != "itsm.ticket_create" {
		t.Fatalf("expected message and next action in payload, got %+v", payload)
	}
	if _, ok := payload["surface"].(map[string]any); !ok {
		t.Fatalf("expected surface payload, got %+v", payload["surface"])
	}
	if _, ok := payload["warnings"].([]DraftWarning); !ok {
		t.Fatalf("expected warnings slice, got %+v", payload["warnings"])
	}
	if _, ok := payload["missingRequiredFields"].([]FieldCollectionItem); !ok {
		t.Fatalf("expected missingRequiredFields, got %+v", payload["missingRequiredFields"])
	}
	if _, ok := payload["missing_required_fields"].([]FieldCollectionItem); !ok {
		t.Fatalf("expected missing_required_fields, got %+v", payload["missing_required_fields"])
	}

	reset, err := session.Reset(11)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if !reset.OK || reset.State.Stage != "idle" || reset.NextExpectedAction != "itsm.service_match" {
		t.Fatalf("unexpected reset result: %+v", reset)
	}
	if reset.Payload["message"] != "已就绪，请描述您的需求" {
		t.Fatalf("expected reset message, got %+v", reset.Payload)
	}
	if got := store.states[11]; got == nil || got.Stage != "idle" || got.LoadedServiceID != 0 {
		t.Fatalf("expected idle state persisted after reset, got %+v", got)
	}
}

func TestOperatorCreateTicketAndSubmitConfirmedDraftCarryDraftIdentity(t *testing.T) {
	creator := &recordingTicketCreator{
		result: &AgentTicketResult{TicketID: 88, TicketCode: "TICK-88", Status: "submitted"},
	}
	op := NewOperator(nil, nil, nil, nil, creator, nil)

	ticket, err := op.CreateTicket(7, 9, "VPN 开通", map[string]any{"vpn_account": "tester"}, 21)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if ticket.TicketID != 88 || creator.req.ServiceID != 9 || creator.req.SessionID != 21 || creator.req.DraftVersion != 0 {
		t.Fatalf("unexpected create ticket contract: ticket=%+v req=%+v", ticket, creator.req)
	}

	draftTicket, err := op.SubmitConfirmedDraft(7, 9, 12, "VPN 开通", map[string]any{"vpn_account": "tester"}, 21, 3, "fields-hash", "request-hash")
	if err != nil {
		t.Fatalf("submit confirmed draft: %v", err)
	}
	if draftTicket.TicketCode != "TICK-88" {
		t.Fatalf("unexpected draft ticket response: %+v", draftTicket)
	}
	if creator.req.ServiceVersionID != 12 || creator.req.DraftVersion != 3 || creator.req.FieldsHash != "fields-hash" || creator.req.RequestHash != "request-hash" {
		t.Fatalf("expected draft identity to pass through, got %+v", creator.req)
	}

	op.ticketCreator = nil
	if _, err := op.CreateTicket(7, 9, "VPN 开通", nil, 21); err == nil {
		t.Fatal("expected create ticket to reject missing ticket creator")
	}
	if _, err := op.SubmitConfirmedDraft(7, 9, 12, "VPN 开通", nil, 21, 3, "fields-hash", "request-hash"); err == nil {
		t.Fatal("expected submit confirmed draft to reject missing ticket creator")
	}
}

func TestOperatorMatchAndLoadServiceBuildsReusableRuntimeSnapshot(t *testing.T) {
	db := testutil.NewTestDB(t)
	sla := domain.SLATemplate{Name: "标准 SLA", Code: "sla-standard", ResponseMinutes: 30, ResolutionMinutes: 240, IsActive: true}
	if err := db.Create(&sla).Error; err != nil {
		t.Fatalf("create sla: %v", err)
	}
	rule := domain.EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  15,
		ActionType:   "notify",
		TargetConfig: domain.JSONField(`{"channelId":1,"receivers":[{"type":"user","userId":7}]}`),
		IsActive:     true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create escalation rule: %v", err)
	}
	service := domain.ServiceDefinition{
		Name:              "VPN 开通申请",
		Code:              "vpn-access-load",
		EngineType:        "smart",
		SLAID:             &sla.ID,
		CollaborationSpec: "访问目的决定处理岗位",
		IntakeFormSchema:  domain.JSONField(`{"fields":[{"key":"request_kind","label":"访问原因","type":"select","required":true,"options":[{"label":"线上支持","value":"online_support"}]},{"key":"vpn_account","label":"VPN账号","type":"text","required":true}]}`),
		WorkflowJSON:      domain.JSONField(`{"nodes":[{"id":"network","data":{"label":"网络管理员处理"}}],"edges":[{"target":"network","data":{"condition":{"field":"form.request_kind","value":["online_support","troubleshooting"]}}}]}`),
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	action := domain.ServiceAction{
		Name:       "同步开通 VPN",
		Code:       "vpn_sync",
		ActionType: "http",
		ConfigJSON: domain.JSONField(`{"url":"https://example.com/vpn","timeout":30}`),
		ServiceID:  service.ID,
		IsActive:   true,
	}
	if err := db.Create(&action).Error; err != nil {
		t.Fatalf("create action: %v", err)
	}

	matcher := &recordingMatcher{
		matches:  []ServiceMatch{{ID: service.ID, Name: service.Name, Score: 0.98}},
		decision: MatchDecision{Kind: MatchDecisionSelectService, SelectedServiceID: service.ID},
	}
	op := NewOperator(db, nil, nil, nil, nil, matcher)

	matches, decision, err := op.MatchServices(context.Background(), "vpn")
	if err != nil {
		t.Fatalf("match services: %v", err)
	}
	if matcher.query != "vpn" || len(matches) != 1 || decision.SelectedServiceID != service.ID {
		t.Fatalf("expected matcher delegation, got query=%q matches=%+v decision=%+v", matcher.query, matches, decision)
	}

	detail, err := op.LoadService(service.ID)
	if err != nil {
		t.Fatalf("load service: %v", err)
	}
	if detail.ServiceID != service.ID || detail.ServiceVersionID == 0 || detail.ServiceVersionHash == "" {
		t.Fatalf("expected runtime snapshot metadata, got %+v", detail)
	}
	if len(detail.FormFields) != 2 || detail.FormFields[0].Key != "request_kind" {
		t.Fatalf("expected form fields parsed from schema, got %+v", detail.FormFields)
	}
	if detail.FormSchema == nil {
		t.Fatalf("expected form schema payload")
	}
	if len(detail.Actions) != 1 || detail.Actions[0].Code != "vpn_sync" {
		t.Fatalf("expected active action surfaced in detail, got %+v", detail.Actions)
	}
	if detail.RoutingFieldHint == nil || detail.RoutingFieldHint.FieldKey != "request_kind" || detail.RoutingFieldHint.OptionRouteMap["online_support"] != "网络管理员处理" {
		t.Fatalf("expected routing hint from workflow, got %+v", detail.RoutingFieldHint)
	}
	if detail.FieldsHash == "" {
		t.Fatalf("expected computed fields hash")
	}

	var snapshots []domain.ServiceDefinitionVersion
	if err := db.Where("service_id = ?", service.ID).Find(&snapshots).Error; err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].Version != 1 {
		t.Fatalf("expected one snapshot version 1, got %+v", snapshots)
	}
	if len(snapshots[0].SLATemplateJSON) == 0 || len(snapshots[0].EscalationRulesJSON) == 0 || len(snapshots[0].ActionsJSON) == 0 {
		t.Fatalf("expected snapshot to contain sla, escalation and action json, got %+v", snapshots[0])
	}

	second, err := op.LoadService(service.ID)
	if err != nil {
		t.Fatalf("load service second time: %v", err)
	}
	if second.ServiceVersionID != detail.ServiceVersionID || second.ServiceVersionHash != detail.ServiceVersionHash {
		t.Fatalf("expected repeated load to reuse same snapshot, got first=%+v second=%+v", detail, second)
	}
	if err := db.Where("service_id = ?", service.ID).Find(&snapshots).Error; err != nil {
		t.Fatalf("list snapshots after second load: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected second load not to create duplicate snapshot, got %d", len(snapshots))
	}

	op.matcher = nil
	if _, _, err := op.MatchServices(context.Background(), "vpn"); err == nil {
		t.Fatal("expected missing matcher to return error")
	}
}

func TestOperatorListMyTicketsAndWithdrawContracts(t *testing.T) {
	db := testutil.NewTestDB(t)
	serviceA := domain.ServiceDefinition{Name: "VPN 开通", Code: "vpn-open", EngineType: "classic", IsActive: true}
	serviceB := domain.ServiceDefinition{Name: "权限申请", Code: "access-open", EngineType: "classic", IsActive: true}
	if err := db.Create(&serviceA).Error; err != nil {
		t.Fatalf("create serviceA: %v", err)
	}
	if err := db.Create(&serviceB).Error; err != nil {
		t.Fatalf("create serviceB: %v", err)
	}
	now := time.Now()
	pending := domain.Ticket{Code: "TICK-1", Title: "待处理 VPN", ServiceID: serviceA.ID, EngineType: "classic", Status: domain.TicketStatusSubmitted, PriorityID: 1, RequesterID: 7, Source: domain.TicketSourceCatalog}
	claimed := domain.Ticket{Code: "TICK-2", Title: "已认领工单", ServiceID: serviceB.ID, EngineType: "classic", Status: domain.TicketStatusWaitingHuman, PriorityID: 1, RequesterID: 7, Source: domain.TicketSourceCatalog}
	terminal := domain.Ticket{Code: "TICK-3", Title: "已完成工单", ServiceID: serviceA.ID, EngineType: "classic", Status: domain.TicketStatusCompleted, PriorityID: 1, RequesterID: 7, Source: domain.TicketSourceCatalog}
	otherUser := domain.Ticket{Code: "TICK-4", Title: "他人工单", ServiceID: serviceA.ID, EngineType: "classic", Status: domain.TicketStatusSubmitted, PriorityID: 1, RequesterID: 8, Source: domain.TicketSourceCatalog}
	for _, ticket := range []*domain.Ticket{&pending, &claimed, &terminal, &otherUser} {
		if err := db.Create(ticket).Error; err != nil {
			t.Fatalf("create ticket %s: %v", ticket.Code, err)
		}
	}
	for _, item := range []struct {
		ticket *domain.Ticket
		at     time.Time
	}{
		{ticket: &pending, at: now.Add(-time.Minute)},
		{ticket: &claimed, at: now},
		{ticket: &terminal, at: now.Add(time.Minute)},
		{ticket: &otherUser, at: now.Add(2 * time.Minute)},
	} {
		if err := db.Model(item.ticket).Update("created_at", item.at).Error; err != nil {
			t.Fatalf("set created_at for %s: %v", item.ticket.Code, err)
		}
	}
	claimedAt := now
	assignment := domain.TicketAssignment{TicketID: claimed.ID, ActivityID: 1, ParticipantType: "user", Status: domain.AssignmentInProgress, ClaimedAt: &claimedAt}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create claimed assignment: %v", err)
	}

	var withdrawnTicketID uint
	var withdrawnReason string
	var withdrawnUser uint
	op := NewOperator(db, nil, nil, func(ticketID uint, reason string, operatorID uint) error {
		withdrawnTicketID = ticketID
		withdrawnReason = reason
		withdrawnUser = operatorID
		return nil
	}, nil, nil)

	tickets, err := op.ListMyTickets(7, "")
	if err != nil {
		t.Fatalf("list my tickets: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected only non-terminal requester tickets, got %+v", tickets)
	}
	if tickets[0].TicketCode != "TICK-2" || tickets[0].ServiceName != "权限申请" || tickets[0].CanWithdraw {
		t.Fatalf("expected newest claimed ticket to disable withdraw, got %+v", tickets[0])
	}
	if tickets[1].TicketCode != "TICK-1" || tickets[1].ServiceName != "VPN 开通" || !tickets[1].CanWithdraw {
		t.Fatalf("expected pending ticket to allow withdraw, got %+v", tickets[1])
	}

	filtered, err := op.ListMyTickets(7, domain.TicketStatusSubmitted)
	if err != nil {
		t.Fatalf("list filtered tickets: %v", err)
	}
	if len(filtered) != 1 || filtered[0].TicketCode != "TICK-1" {
		t.Fatalf("expected status filter to keep submitted ticket only, got %+v", filtered)
	}

	if err := op.WithdrawTicket(7, "TICK-1", "用户撤回"); err != nil {
		t.Fatalf("withdraw ticket: %v", err)
	}
	if withdrawnTicketID != pending.ID || withdrawnReason != "用户撤回" || withdrawnUser != 7 {
		t.Fatalf("expected withdraw callback with resolved ticket id, got ticketID=%d reason=%q user=%d", withdrawnTicketID, withdrawnReason, withdrawnUser)
	}
	if err := op.WithdrawTicket(7, "UNKNOWN", "用户撤回"); err == nil {
		t.Fatal("expected unknown ticket code to return error")
	}
}

func TestTicketHandlersListAndWithdrawContracts(t *testing.T) {
	op := &ticketListWithdrawStub{
		tickets: []TicketSummary{{TicketID: 1, TicketCode: "TICK-1", Summary: "VPN", Status: "submitted"}},
	}
	ctx := context.WithValue(context.Background(), appcore.SessionIDKey, uint(9))

	payload, err := myTicketsHandler(op)(ctx, 7, []byte(`{"status":"submitted"}`))
	if err != nil {
		t.Fatalf("my tickets handler: %v", err)
	}
	var listed map[string]any
	if err := json.Unmarshal(payload, &listed); err != nil {
		t.Fatalf("unmarshal tickets payload: %v", err)
	}
	if listed["ok"] != true || op.listStatus != "submitted" {
		t.Fatalf("expected status filter to pass through, got payload=%+v status=%q", listed, op.listStatus)
	}

	withdrawPayload, err := ticketWithdrawHandler(op)(ctx, 7, []byte(`{"ticket_code":"TICK-1"}`))
	if err != nil {
		t.Fatalf("withdraw handler: %v", err)
	}
	var withdrawn map[string]any
	if err := json.Unmarshal(withdrawPayload, &withdrawn); err != nil {
		t.Fatalf("unmarshal withdraw payload: %v", err)
	}
	if op.withdrawUserID != 7 || op.withdrawCode != "TICK-1" || op.withdrawReason != "用户撤回" {
		t.Fatalf("expected default withdraw reason and code to pass through, got user=%d code=%q reason=%q", op.withdrawUserID, op.withdrawCode, op.withdrawReason)
	}
	if withdrawn["ticket_code"] != "TICK-1" || withdrawn["ok"] != true {
		t.Fatalf("unexpected withdraw response: %+v", withdrawn)
	}

	op.listErr = errors.New("db unavailable")
	if _, err := myTicketsHandler(op)(ctx, 7, []byte(`{}`)); err == nil {
		t.Fatal("expected list handler to wrap operator error")
	}

	op.withdrawErr = errors.New("already claimed")
	if _, err := ticketWithdrawHandler(op)(ctx, 7, []byte(`{"ticket_code":"TICK-2","reason":"撤回"}`)); err == nil {
		t.Fatal("expected withdraw handler to wrap operator error")
	}
	if _, err := ticketWithdrawHandler(op)(ctx, 7, []byte(`{"reason":"撤回"}`)); err == nil {
		t.Fatal("expected withdraw handler to reject empty ticket code")
	}
}

func TestNewRequestHandlerResetsConversationState(t *testing.T) {
	store := newMemStateStore()
	store.states[9] = &ServiceDeskState{
		Stage:                   "awaiting_confirmation",
		LoadedServiceID:         3,
		DraftVersion:            2,
		PendingNextRequiredTool: "itsm.draft_confirm",
	}
	ctx := context.WithValue(context.Background(), appcore.SessionIDKey, uint(9))

	payload, err := newRequestHandler(store)(ctx, 0, []byte(`{}`))
	if err != nil {
		t.Fatalf("new request handler: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal new request payload: %v", err)
	}
	if resp["ok"] != true || resp["message"] != "已就绪，请描述您的需求" {
		t.Fatalf("unexpected reset payload: %+v", resp)
	}
	state, ok := resp["state"].(map[string]any)
	if !ok || state["stage"] != "idle" {
		t.Fatalf("expected idle state in payload, got %+v", resp["state"])
	}
	if got := store.states[9]; got == nil || got.Stage != "idle" || got.LoadedServiceID != 0 || got.DraftVersion != 0 {
		t.Fatalf("expected handler to persist reset idle state, got %+v", got)
	}
}
