package engine

import (
	"strings"
	"testing"
)

func TestParseDecisionPlanContracts(t *testing.T) {
	t.Run("extracts fenced json and normalizes invalid execution mode", func(t *testing.T) {
		plan, err := parseDecisionPlan("```json\n{\n  \"next_step_type\": \"process\",\n  \"execution_mode\": \"fanout\",\n  \"confidence\": 0.91,\n  \"reasoning\": \"需要人工处理\",\n  \"activities\": [{\"type\": \"process\", \"participant_type\": \"requester\"}]\n}\n```")
		if err != nil {
			t.Fatalf("parse fenced plan: %v", err)
		}
		if plan.NextStepType != "process" {
			t.Fatalf("expected next_step_type process, got %q", plan.NextStepType)
		}
		if plan.ExecutionMode != "" {
			t.Fatalf("expected invalid execution_mode to normalize to empty single-mode, got %q", plan.ExecutionMode)
		}
		if len(plan.Activities) != 1 || plan.Activities[0].ParticipantType != "requester" {
			t.Fatalf("expected requester activity, got %+v", plan.Activities)
		}
	})

	t.Run("preserves supported execution mode", func(t *testing.T) {
		plan, err := parseDecisionPlan("{\"next_step_type\":\"process\",\"execution_mode\":\"parallel\",\"activities\":[{\"type\":\"process\"}],\"confidence\":0.8}")
		if err != nil {
			t.Fatalf("parse parallel plan: %v", err)
		}
		if plan.ExecutionMode != "parallel" {
			t.Fatalf("expected execution_mode parallel, got %q", plan.ExecutionMode)
		}
	})

	t.Run("rejects missing json payload", func(t *testing.T) {
		_, err := parseDecisionPlan("模型只返回了口头结论，没有结构化计划")
		if err == nil || !strings.Contains(err.Error(), "JSON 解析失败") {
			t.Fatalf("expected unmarshal error for non-object content, got %v", err)
		}
	})

	t.Run("repairs truncated json before decoding", func(t *testing.T) {
		plan, err := parseDecisionPlan("{\"next_step_type\":\"process\",")
		if err != nil {
			t.Fatalf("expected truncated json to be repaired, got %v", err)
		}
		if plan.NextStepType != "process" {
			t.Fatalf("expected repaired next_step_type process, got %q", plan.NextStepType)
		}
	})
}
