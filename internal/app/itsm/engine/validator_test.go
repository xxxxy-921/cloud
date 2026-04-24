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

func TestValidateWorkflowClassicNodeMatrix(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"action","type":"action","data":{"label":"动作","action_id":1}},
			{"id":"notify","type":"notify","data":{"label":"通知","channel_id":2}},
			{"id":"wait","type":"wait","data":{"label":"等待","wait_mode":"timer","duration":"1h","participants":[{"type":"requester"}]}},
			{"id":"script","type":"script","data":{"label":"脚本","assignments":[{"variable":"x","expression":"1 + 1"}]}},
			{"id":"sub","type":"subprocess","data":{"label":"子流程","subprocess_def":{"nodes":[{"id":"sub_start","type":"start","data":{"label":"子开始"}},{"id":"sub_end","type":"end","data":{"label":"子结束"}}],"edges":[{"id":"se1","source":"sub_start","target":"sub_end","data":{}}]}}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"action","data":{}},
			{"id":"e2","source":"action","target":"notify","data":{"outcome":"success"}},
			{"id":"e3","source":"notify","target":"wait","data":{}},
			{"id":"e4","source":"wait","target":"script","data":{"default":true}},
			{"id":"e5","source":"script","target":"sub","data":{}},
			{"id":"e6","source":"sub","target":"end","data":{}}
		]
	}`)

	var blocking []ValidationError
	for _, err := range ValidateWorkflow(workflowJSON) {
		if !err.IsWarning() {
			blocking = append(blocking, err)
		}
	}
	if len(blocking) > 0 {
		t.Fatalf("expected workflow matrix to validate, got %+v", blocking)
	}
}

func TestValidateWorkflowRejectsNonRunnableClassicNodeConfig(t *testing.T) {
	tests := []struct {
		name    string
		node    string
		edge    string
		wantMsg string
	}{
		{
			name:    "action missing action_id",
			node:    `{"id":"node","type":"action","data":{"label":"动作"}}`,
			edge:    `{"id":"e2","source":"node","target":"end","data":{}}`,
			wantMsg: "action_id",
		},
		{
			name:    "wait missing wait_mode",
			node:    `{"id":"node","type":"wait","data":{"label":"等待","participants":[{"type":"requester"}]}}`,
			edge:    `{"id":"e2","source":"node","target":"end","data":{}}`,
			wantMsg: "wait_mode",
		},
		{
			name:    "script missing assignments",
			node:    `{"id":"node","type":"script","data":{"label":"脚本"}}`,
			edge:    `{"id":"e2","source":"node","target":"end","data":{}}`,
			wantMsg: "assignments",
		},
		{
			name:    "subprocess missing subprocess_def",
			node:    `{"id":"node","type":"subprocess","data":{"label":"子流程"}}`,
			edge:    `{"id":"e2","source":"node","target":"end","data":{}}`,
			wantMsg: "subprocess_def",
		},
		{
			name:    "timer event remains non runnable",
			node:    `{"id":"node","type":"timer","data":{"label":"定时事件"}}`,
			edge:    `{"id":"e2","source":"node","target":"end","data":{}}`,
			wantMsg: "尚未实现",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowJSON := json.RawMessage(`{
				"nodes": [
					{"id":"start","type":"start","data":{"label":"开始"}},
					` + tt.node + `,
					{"id":"end","type":"end","data":{"label":"结束"}}
				],
				"edges": [
					{"id":"e1","source":"start","target":"node","data":{}},
					` + tt.edge + `
				]
			}`)

			errs := ValidateWorkflow(workflowJSON)
			if len(errs) == 0 {
				t.Fatal("expected validation error")
			}
			var found bool
			for _, err := range errs {
				if !err.IsWarning() && strings.Contains(err.Message, tt.wantMsg) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected blocking message containing %q, got %+v", tt.wantMsg, errs)
			}
		})
	}
}
