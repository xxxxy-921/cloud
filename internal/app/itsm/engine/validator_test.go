package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateWorkflowAllowsRequesterParticipantOnForm(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form","type":"form","data":{"label":"填写申请","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form","data":{}},
			{"id":"e2","source":"form","target":"end","data":{}}
		]
	}`)

	if errs := ValidateWorkflow(workflowJSON); len(errs) > 0 {
		t.Fatalf("expected requester participant to validate, got %+v", errs)
	}
}

func TestValidateWorkflowMissingFormParticipantSuggestsRequester(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form","type":"form","data":{"label":"填写临时访问申请"}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form","data":{}},
			{"id":"e2","source":"form","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
	got := errs[0].Message
	if !strings.Contains(got, `{"type":"requester"}`) {
		t.Fatalf("expected requester repair suggestion, got %q", got)
	}
	if !strings.Contains(got, "form（填写临时访问申请）") {
		t.Fatalf("expected node label in validation message, got %q", got)
	}
}
