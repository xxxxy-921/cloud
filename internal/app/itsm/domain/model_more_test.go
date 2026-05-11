package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestJSONFieldHelpersAndDomainUtilities(t *testing.T) {
	var field JSONField
	if value, err := field.Value(); err != nil || value != nil {
		t.Fatalf("empty Value=%v err=%v", value, err)
	}
	if err := field.UnmarshalJSON([]byte(`{"ok":true}`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if value, err := field.Value(); err != nil || value.(string) != `{"ok":true}` {
		t.Fatalf("Value=%v err=%v", value, err)
	}
	if err := field.Scan(123); err == nil {
		t.Fatal("expected unsupported Scan type to fail")
	}
	if err := field.Scan(nil); err != nil {
		t.Fatalf("Scan nil: %v", err)
	}

	c, _ := gin.CreateTestContext(nil)
	c.Params = gin.Params{{Key: "id", Value: "12"}, {Key: "serviceId", Value: "34"}}
	if id, err := ParseID(c); err != nil || id != 12 {
		t.Fatalf("ParseID=%d err=%v", id, err)
	}
	if id, err := ParseParamID(c, "serviceId"); err != nil || id != 34 {
		t.Fatalf("ParseParamID=%d err=%v", id, err)
	}
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	if _, err := ParseID(c); err == nil {
		t.Fatal("expected ParseID bad input to fail")
	}

	if !IsSQLiteUniqueError(errors.New("UNIQUE constraint failed: table.column")) {
		t.Fatal("expected sqlite unique error to be recognized")
	}
	if IsSQLiteUniqueError(errors.New("other error")) || IsSQLiteUniqueError(nil) {
		t.Fatal("expected non unique error to be false")
	}
}

func TestDomainTableNamesAndActionConfigNormalization(t *testing.T) {
	if (ServiceCatalog{}).TableName() != "itsm_service_catalogs" ||
		(ServiceDefinition{}).TableName() != "itsm_service_definitions" ||
		(ServiceAction{}).TableName() != "itsm_service_actions" ||
		(ServiceKnowledgeDocument{}).TableName() != "itsm_service_knowledge_documents" ||
		(ServiceDefinitionVersion{}).TableName() != "itsm_service_definition_versions" ||
		(Priority{}).TableName() != "itsm_priorities" ||
		(SLATemplate{}).TableName() != "itsm_sla_templates" ||
		(EscalationRule{}).TableName() != "itsm_escalation_rules" ||
		(Ticket{}).TableName() != "itsm_tickets" ||
		(TicketActivity{}).TableName() != "itsm_ticket_activities" ||
		(TicketAssignment{}).TableName() != "itsm_ticket_assignments" ||
		(TicketTimeline{}).TableName() != "itsm_ticket_timelines" ||
		(TicketActionExecution{}).TableName() != "itsm_ticket_action_executions" ||
		(TicketLink{}).TableName() != "itsm_ticket_links" ||
		(PostMortem{}).TableName() != "itsm_post_mortems" ||
		(ExecutionToken{}).TableName() != "itsm_execution_tokens" ||
		(ProcessVariable{}).TableName() != "itsm_process_variables" {
		t.Fatal("unexpected table name mapping")
	}

	normalized, err := NormalizeServiceActionConfig("http", JSONField(`{"url":"https://example.com/hook","headers":{"X-Test":"1"}}`))
	if err != nil {
		t.Fatalf("NormalizeServiceActionConfig: %v", err)
	}
	var cfg ServiceActionHTTPConfig
	if err := json.Unmarshal(normalized, &cfg); err != nil {
		t.Fatalf("decode normalized config: %v", err)
	}
	if cfg.Method != "POST" || cfg.Timeout != 30 || cfg.Retries != 3 {
		t.Fatalf("unexpected normalized cfg: %+v", cfg)
	}

	cases := []JSONField{
		JSONField(`{"url":"ftp://example.com","method":"POST"}`),
		JSONField(`{"url":"https://example.com","method":"TRACE"}`),
		JSONField(`{"url":"https://example.com","timeout":121}`),
		JSONField(`{"url":"https://example.com","retries":6}`),
		JSONField(`{"url":"https://example.com","headers":{"":"1"}}`),
	}
	for _, raw := range cases {
		if _, err := NormalizeServiceActionConfig("http", raw); err == nil {
			t.Fatalf("expected config to be rejected: %s", raw)
		}
	}
	if _, err := NormalizeServiceActionConfig("script", JSONField(`{"url":"https://example.com"}`)); err == nil {
		t.Fatal("expected unsupported action type to fail")
	}
}

func TestTicketStatusAndVariableResponses(t *testing.T) {
	if statuses := TerminalTicketStatuses(); len(statuses) != 5 {
		t.Fatalf("unexpected terminal statuses: %+v", statuses)
	}
	if !IsActiveTicketStatus(TicketStatusDecisioning) || IsActiveTicketStatus(TicketStatusRejected) {
		t.Fatal("unexpected active ticket status result")
	}

	labelCases := map[string]string{
		TicketStatusSubmitted:           "已提交",
		TicketStatusWaitingHuman:        "待人工处理",
		TicketStatusApprovedDecisioning: "已同意，决策中",
		TicketStatusRejectedDecisioning: "已驳回，决策中",
		TicketStatusDecisioning:         "AI 决策中",
		TicketStatusExecutingAction:     "自动执行中",
		TicketStatusRejected:            "已驳回",
		TicketStatusWithdrawn:           "已撤回",
		TicketStatusCancelled:           "已取消",
		TicketStatusFailed:              "失败",
	}
	for status, want := range labelCases {
		if got := TicketStatusLabel(status, TicketOutcomeApproved); got != want {
			t.Fatalf("TicketStatusLabel(%s)=%q, want %q", status, got, want)
		}
	}
	if TicketStatusLabel(TicketStatusCompleted, TicketOutcomeFulfilled) != "已履约" {
		t.Fatal("expected fulfilled completed label")
	}
	if TicketStatusTone(TicketStatusCompleted, "") != "success" ||
		TicketStatusTone(TicketStatusRejected, "") != "destructive" ||
		TicketStatusTone(TicketStatusWithdrawn, "") != "secondary" ||
		TicketStatusTone(TicketStatusWaitingHuman, "") != "warning" ||
		TicketStatusTone("other", "") != "secondary" {
		t.Fatal("unexpected ticket status tone mapping")
	}

	now := time.Unix(1710000000, 0)
	parentID := uint(9)
	timeline := (&TicketTimeline{
		TicketID:   1,
		ActivityID: &parentID,
		OperatorID: 7,
		EventType:  "comment",
		Message:    "已处理",
		Details:    JSONField(`{"step":"process"}`),
		Reasoning:  "manual",
	})
	timeline.CreatedAt = now
	if resp := timeline.ToResponse(); resp.Content != "已处理" || resp.Metadata == nil || resp.Reasoning != "manual" {
		t.Fatalf("unexpected timeline response: %+v", resp)
	}

	token := &ExecutionToken{TicketID: 1, ParentTokenID: &parentID, NodeID: "n1", Status: "active", TokenType: "main", ScopeID: "root"}
	token.CreatedAt = now
	if resp := token.ToResponse(); resp.ScopeID != "root" || resp.ParentTokenID == nil || *resp.ParentTokenID != parentID {
		t.Fatalf("unexpected token response: %+v", resp)
	}

	variableCases := []struct {
		raw       string
		valueType string
		want      any
	}{
		{raw: "12.5", valueType: ValueTypeNumber, want: 12.5},
		{raw: "true", valueType: ValueTypeBoolean, want: true},
		{raw: `{"env":"prod"}`, valueType: ValueTypeJSON, want: map[string]any{"env": "prod"}},
		{raw: "2026-05-10", valueType: ValueTypeDate, want: "2026-05-10"},
		{raw: "vpn", valueType: ValueTypeString, want: "vpn"},
	}
	for _, tc := range variableCases {
		variable := &ProcessVariable{TicketID: 1, ScopeID: "root", Key: "k", Value: tc.raw, ValueType: tc.valueType, Source: "test"}
		resp := variable.ToResponse()
		switch want := tc.want.(type) {
		case map[string]any:
			got, ok := resp.Value.(map[string]any)
			if !ok || got["env"] != want["env"] {
				t.Fatalf("unexpected json variable response: %+v", resp)
			}
		default:
			if resp.Value != want {
				t.Fatalf("unexpected variable response: %+v", resp)
			}
		}
	}

	action := &ServiceAction{Name: "Webhook", Code: "notify", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com"}`), ServiceID: 1, IsActive: true}
	if action.ToResponse().Code != "notify" {
		t.Fatalf("unexpected action response: %+v", action.ToResponse())
	}
}
