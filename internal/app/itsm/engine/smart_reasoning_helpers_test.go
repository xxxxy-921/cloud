package engine

import (
	"strings"
	"testing"
	"time"
)

func TestDecisionExplanationSnapshotAndAuditReasoning(t *testing.T) {
	plan := &DecisionPlan{
		Reasoning: "基于协作规范优先转网络管理员处理",
		Activities: []DecisionActivity{
			{Type: NodeAction, Instructions: "先跑预检"},
			{Type: NodeProcess, ParticipantType: "position_department", DepartmentCode: "it", PositionCode: "network_admin"},
		},
	}
	activityID := uint(12)
	snapshot := buildDecisionExplanationSnapshot(plan, "activity_completed", "网络分支继续处理", "等待人工处理", &activityID)
	if snapshot["basis"] != plan.Reasoning || snapshot["trigger"] != "activity_completed" || snapshot["decision"] != "网络分支继续处理" {
		t.Fatalf("unexpected snapshot core fields: %+v", snapshot)
	}
	if snapshot["humanOverride"] != "可通过转人工处理到 处理 节点" {
		t.Fatalf("unexpected human override: %+v", snapshot["humanOverride"])
	}
	if snapshot["activityId"] != uint(12) {
		t.Fatalf("unexpected activity id in snapshot: %+v", snapshot["activityId"])
	}

	details := decisionExplanationDetails(snapshot)
	explanation, ok := details["decision_explanation"].(map[string]any)
	if !ok || explanation["trigger"] != "activity_completed" {
		t.Fatalf("expected decision_explanation wrapper, got %+v", details)
	}

	engine := &SmartEngine{configProvider: fallbackOnlyConfigProvider{fallbackID: 1}}
	if got := engine.auditReasoning("完整推理"); got != "完整推理" {
		t.Fatalf("expected full audit reasoning by default, got %q", got)
	}
	engine.configProvider = auditLevelConfigProvider{level: "off"}
	if got := engine.auditReasoning("完整推理"); got != "" {
		t.Fatalf("expected audit off to suppress reasoning, got %q", got)
	}
	engine.configProvider = auditLevelConfigProvider{level: "summary"}
	longReasoning := strings.Repeat("证据", 200)
	got := engine.auditReasoning(longReasoning)
	if len(got) >= len(longReasoning) || !strings.HasSuffix(got, "...") {
		t.Fatalf("expected summary audit reasoning to truncate, got len=%d", len(got))
	}
}

func TestDecisionExplanationDetailsContracts(t *testing.T) {
	if got := decisionExplanationDetails(nil); got != nil {
		t.Fatalf("nil snapshot should yield nil details, got %+v", got)
	}
	if got := decisionExplanationDetails(map[string]any{}); got != nil {
		t.Fatalf("empty snapshot should yield nil details, got %+v", got)
	}
	snapshot := map[string]any{"trigger": "activity_completed", "decision": "继续推进"}
	got := decisionExplanationDetails(snapshot)
	if got == nil {
		t.Fatal("non-empty snapshot should be wrapped")
	}
	explanation, ok := got["decision_explanation"].(map[string]any)
	if !ok || explanation["trigger"] != "activity_completed" || explanation["decision"] != "继续推进" {
		t.Fatalf("unexpected decision explanation wrapper: %+v", got)
	}
}

func TestCompletionLabelsAndHumanDecisionExtraction(t *testing.T) {
	if got := completionDecisionLabel(TicketStatusRejected, ""); got != "已驳回" {
		t.Fatalf("unexpected rejected label: %q", got)
	}
	if got := completionDecisionLabel(TicketStatusCompleted, TicketOutcomeApproved); got != "已通过" {
		t.Fatalf("unexpected approved completion label: %q", got)
	}
	if got := completionDecisionLabel(TicketStatusCompleted, TicketOutcomeFulfilled); got != "已履约" {
		t.Fatalf("unexpected fulfilled completion label: %q", got)
	}
	if got := completionDecisionLabel(TicketStatusCompleted, "other"); got != "已完成" {
		t.Fatalf("unexpected generic completion label: %q", got)
	}
	if got := completionDecisionLabel(TicketStatusExecutingAction, ""); got != TicketStatusExecutingAction {
		t.Fatalf("unexpected passthrough label: %q", got)
	}

	activities := []DecisionActivity{
		{Type: NodeAction},
		{Type: NodeProcess, ParticipantType: "position_department", DepartmentCode: "it", PositionCode: "ops_admin"},
	}
	if got := firstHumanDecisionActivity(activities); got == nil || got.Type != NodeProcess {
		t.Fatalf("expected first human activity to be process, got %+v", got)
	}
	if got := firstHumanDecisionActivity([]DecisionActivity{{Type: NodeAction}}); got != nil {
		t.Fatalf("expected no human activity, got %+v", got)
	}

	msg := humanProgressMessage("网络管理员处理", "approved", "已完成网络排障")
	if !strings.Contains(msg, "网络管理员处理") || !strings.Contains(msg, "approved") || !strings.Contains(msg, "已完成网络排障") {
		t.Fatalf("unexpected human progress message: %q", msg)
	}
}

func TestAppendDecisionReasoningContracts(t *testing.T) {
	if got := appendDecisionReasoning("", "  新证据 "); got != "新证据" {
		t.Fatalf("empty existing append = %q, want 新证据", got)
	}
	if got := appendDecisionReasoning("已有结论", ""); got != "已有结论" {
		t.Fatalf("blank addition should keep existing, got %q", got)
	}
	if got := appendDecisionReasoning("已有结论", "已有结论"); got != "已有结论" {
		t.Fatalf("duplicate addition should not repeat, got %q", got)
	}
	if got := appendDecisionReasoning("已有结论", "补充依据"); got != "已有结论\n补充依据" {
		t.Fatalf("distinct reasoning should append on new line, got %q", got)
	}
}

type auditLevelConfigProvider struct {
	level string
}

func (m auditLevelConfigProvider) FallbackAssigneeID() uint                  { return 0 }
func (m auditLevelConfigProvider) DecisionMode() string                      { return "ai_only" }
func (m auditLevelConfigProvider) DecisionAgentID() uint                     { return 0 }
func (m auditLevelConfigProvider) AuditLevel() string                        { return m.level }
func (m auditLevelConfigProvider) SLACriticalThresholdSeconds() int          { return 1800 }
func (m auditLevelConfigProvider) SLAWarningThresholdSeconds() int           { return 3600 }
func (m auditLevelConfigProvider) SimilarHistoryLimit() int                  { return 5 }
func (m auditLevelConfigProvider) ParallelConvergenceTimeout() time.Duration { return time.Hour }
