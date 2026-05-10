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

func TestValidateWorkflowAllowsProcessOutcomesToShareEndNode(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"process","type":"process","data":{"label":"处理","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"完成"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"process","data":{}},
			{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}},
			{"id":"e3","source":"process","target":"end","data":{"outcome":"rejected"}}
		]
	}`)

	var blocking []ValidationError
	for _, err := range ValidateWorkflow(workflowJSON) {
		if !err.IsWarning() {
			blocking = append(blocking, err)
		}
	}
	if len(blocking) > 0 {
		t.Fatalf("expected shared end node to validate, got %+v", blocking)
	}
}

func TestValidateWorkflowRejectsProcessOutcomesSharingNonEndNode(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"process","type":"process","data":{"label":"处理","participants":[{"type":"requester"}]}},
			{"id":"next","type":"process","data":{"label":"继续处理","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"完成"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"process","data":{}},
			{"id":"e2","source":"process","target":"next","data":{"outcome":"approved"}},
			{"id":"e3","source":"process","target":"next","data":{"outcome":"rejected"}},
			{"id":"e4","source":"next","target":"end","data":{"outcome":"approved"}},
			{"id":"e5","source":"next","target":"end","data":{"outcome":"rejected"}}
		]
	}`)

	var found bool
	for _, err := range ValidateWorkflow(workflowJSON) {
		if !err.IsWarning() && strings.Contains(err.Message, "共同指向非结束节点") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected shared non-end target to be rejected")
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

func TestValidateFormSchemaReferences(t *testing.T) {
	// Workflow: start -> form (request_kind, urgency) -> exclusive gateway -> two branches
	makeWorkflow := func(condField string) json.RawMessage {
		return json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"form1","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"request_kind","type":"select","label":"类型"},{"key":"urgency","type":"select","label":"紧急程度"}]}}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"p2","type":"process","data":{"label":"处理B","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"form1","data":{}},
				{"id":"e2","source":"form1","target":"gw","data":{"outcome":"submitted"}},
				{"id":"e3","source":"gw","target":"p1","data":{"condition":{"field":"` + condField + `","operator":"equals","value":"high"}}},
				{"id":"e4","source":"gw","target":"p2","data":{"default":true}},
				{"id":"e5","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p1","target":"end","data":{"outcome":"rejected"}},
				{"id":"e6","source":"p2","target":"end","data":{"outcome":"approved"}},
				{"id":"e6r","source":"p2","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
	}

	t.Run("field exists in formSchema", func(t *testing.T) {
		errs := ValidateWorkflow(makeWorkflow("form.urgency"))
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "formSchema") {
				t.Fatalf("unexpected formSchema warning: %s", e.Message)
			}
		}
	})

	t.Run("field missing from formSchema", func(t *testing.T) {
		errs := ValidateWorkflow(makeWorkflow("form.nonexistent"))
		var found bool
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "nonexistent") {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected warning about missing formSchema field")
		}
	})

	t.Run("non-form field skipped", func(t *testing.T) {
		errs := ValidateWorkflow(makeWorkflow("ticket.status"))
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "formSchema") {
				t.Fatalf("unexpected formSchema warning for non-form field: %s", e.Message)
			}
		}
	})

	t.Run("no upstream form node skipped", func(t *testing.T) {
		wf := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"p1","type":"process","data":{"label":"A","participants":[{"type":"requester"}]}},
				{"id":"p2","type":"process","data":{"label":"B","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"p1","data":{"condition":{"field":"form.request_kind","operator":"equals","value":"vpn"}}},
				{"id":"e3","source":"gw","target":"p2","data":{"default":true}},
				{"id":"e4","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e4r","source":"p1","target":"end","data":{"outcome":"rejected"}},
				{"id":"e5","source":"p2","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p2","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
		errs := ValidateWorkflow(wf)
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "formSchema") {
				t.Fatalf("unexpected formSchema warning when no upstream form: %s", e.Message)
			}
		}
	})

	t.Run("falls back to generated intake form schema", func(t *testing.T) {
		wf := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"form1","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"request_kind","type":"select","label":"类型"}]}}},
				{"id":"p1","type":"process","data":{"label":"A","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"form1","data":{"condition":{"field":"form.request_kind","operator":"equals","value":"vpn"}}},
				{"id":"e3","source":"gw","target":"p1","data":{"default":true}},
				{"id":"e4","source":"form1","target":"end","data":{}},
				{"id":"e5","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p1","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
		errs := ValidateWorkflow(wf)
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "formSchema") {
				t.Fatalf("unexpected formSchema warning when intake schema has field: %s", e.Message)
			}
		}
	})

	t.Run("warns when generated intake form schema misses field", func(t *testing.T) {
		wf := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"form1","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"request_kind","type":"select","label":"类型"}]}}},
				{"id":"p1","type":"process","data":{"label":"A","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"form1","data":{"condition":{"field":"form.urgency","operator":"equals","value":"high"}}},
				{"id":"e3","source":"gw","target":"p1","data":{"default":true}},
				{"id":"e4","source":"form1","target":"end","data":{}},
				{"id":"e5","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p1","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
		errs := ValidateWorkflow(wf)
		var found bool
		for _, e := range errs {
			if e.IsWarning() && strings.Contains(e.Message, "form.urgency") && strings.Contains(e.Message, "申请确认表单") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected missing intake schema field warning, got %+v", errs)
		}
	})
}

// ---------------------------------------------------------------------------
// Topology validation tests (task 3.7)
// ---------------------------------------------------------------------------

func TestValidateWorkflowNoCycle(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form1","type":"form","data":{"label":"表单","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form1","data":{}},
			{"id":"e2","source":"form1","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	for _, e := range errs {
		if strings.Contains(e.Message, "环路") {
			t.Fatalf("expected no cycle error in linear workflow, got: %s", e.Message)
		}
	}
}

func TestValidateWorkflowDirectCycle(t *testing.T) {
	// A→B→A direct cycle. Both nodes are form nodes so they are valid types.
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"A","type":"form","data":{"label":"节点A","participants":[{"type":"requester"}]}},
			{"id":"B","type":"form","data":{"label":"节点B","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"A","data":{}},
			{"id":"e2","source":"A","target":"B","data":{}},
			{"id":"e3","source":"B","target":"A","data":{}},
			{"id":"e4","source":"B","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Message, "环路") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected cycle detection error containing '环路', got errors: %+v", errs)
	}
}

func TestValidateWorkflowIndirectCycle(t *testing.T) {
	// A→B→C→A indirect cycle.
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"A","type":"form","data":{"label":"节点A","participants":[{"type":"requester"}]}},
			{"id":"B","type":"form","data":{"label":"节点B","participants":[{"type":"requester"}]}},
			{"id":"C","type":"form","data":{"label":"节点C","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"A","data":{}},
			{"id":"e2","source":"A","target":"B","data":{}},
			{"id":"e3","source":"B","target":"C","data":{}},
			{"id":"e4","source":"C","target":"A","data":{}},
			{"id":"e5","source":"C","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Message, "环路") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected cycle detection error containing '环路', got errors: %+v", errs)
	}
}

func TestValidateWorkflowDeadEnd(t *testing.T) {
	// form1 connects to form2, but form2 has no edge to end — form2 is a dead-end.
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form1","type":"form","data":{"label":"表单1","participants":[{"type":"requester"}]}},
			{"id":"form2","type":"form","data":{"label":"表单2","participants":[{"type":"requester"}]}},
			{"id":"form3","type":"form","data":{"label":"表单3","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form1","data":{}},
			{"id":"e2","source":"form1","target":"form2","data":{}},
			{"id":"e3","source":"form1","target":"form3","data":{}},
			{"id":"e4","source":"form3","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Message, "无法到达终点") && e.NodeID == "form2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected dead-end error for form2 containing '无法到达终点', got errors: %+v", errs)
	}
}

func TestValidateWorkflowAllowsMultipleTerminalBranches(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gw","type":"exclusive","data":{"label":"分支"}},
			{"id":"form_ok","type":"form","data":{"label":"正常补充","participants":[{"type":"requester"}]}},
			{"id":"form_reject","type":"form","data":{"label":"驳回确认","participants":[{"type":"requester"}]}},
			{"id":"end_ok","type":"end","data":{"label":"正常结束"}},
			{"id":"end_reject","type":"end","data":{"label":"驳回结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"gw","data":{}},
			{"id":"e2","source":"gw","target":"form_ok","data":{"condition":{"field":"ticket.status","operator":"equals","value":"approved"}}},
			{"id":"e3","source":"gw","target":"form_reject","data":{"default":true}},
			{"id":"e4","source":"form_ok","target":"end_ok","data":{}},
			{"id":"e5","source":"form_reject","target":"end_reject","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	for _, e := range errs {
		if strings.Contains(e.Message, "无法到达终点") {
			t.Fatalf("expected all branches to reach one terminal node, got dead-end error: %+v", errs)
		}
	}
}

func TestValidateWorkflowDoesNotRequireEndToReachAnotherEnd(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gw","type":"exclusive","data":{"label":"分支"}},
			{"id":"end_ok","type":"end","data":{"label":"正常结束"}},
			{"id":"end_reject","type":"end","data":{"label":"驳回结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"gw","data":{}},
			{"id":"e2","source":"gw","target":"end_ok","data":{"condition":{"field":"ticket.status","operator":"equals","value":"approved"}}},
			{"id":"e3","source":"gw","target":"end_reject","data":{"default":true}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	for _, e := range errs {
		if e.NodeID == "end_reject" && strings.Contains(e.Message, "无法到达终点") {
			t.Fatalf("end node must not be required to reach another end node, got %+v", errs)
		}
	}
}

func TestValidateWorkflowInvalidParticipantType(t *testing.T) {
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form1","type":"form","data":{"label":"表单","participants":[{"type":"invalid_type"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form1","data":{}},
			{"id":"e2","source":"form1","target":"end","data":{}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)
	var found bool
	for _, e := range errs {
		if strings.Contains(e.Message, "非法的参与者类型") && !e.IsWarning() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected blocking error containing '非法的参与者类型', got errors: %+v", errs)
	}
}

func TestValidateWorkflowValidParticipantTypes(t *testing.T) {
	validTypes := []string{
		"user", "position", "department",
		"position_department", "requester", "requester_manager",
	}
	for _, pt := range validTypes {
		t.Run(pt, func(t *testing.T) {
			var participantJSON string
			switch pt {
			case "user":
				participantJSON = `{"type":"user","value":"admin"}`
			case "position":
				participantJSON = `{"type":"position","value":"manager"}`
			case "department":
				participantJSON = `{"type":"department","value":"it"}`
			case "position_department":
				participantJSON = `{"type":"position_department","position_code":"admin","department_code":"it"}`
			case "requester":
				participantJSON = `{"type":"requester"}`
			case "requester_manager":
				participantJSON = `{"type":"requester_manager"}`
			}

			workflowJSON := json.RawMessage(`{
				"nodes": [
					{"id":"start","type":"start","data":{"label":"开始"}},
					{"id":"form1","type":"form","data":{"label":"表单","participants":[` + participantJSON + `]}},
					{"id":"end","type":"end","data":{"label":"结束"}}
				],
				"edges": [
					{"id":"e1","source":"start","target":"form1","data":{}},
					{"id":"e2","source":"form1","target":"end","data":{}}
				]
			}`)

			errs := ValidateWorkflow(workflowJSON)
			for _, e := range errs {
				if strings.Contains(e.Message, "非法的参与者类型") {
					t.Fatalf("participant type %q should be valid, got error: %s", pt, e.Message)
				}
			}
		})
	}
}

func TestValidateWorkflowBlockingVsWarning(t *testing.T) {
	// Workflow with both a topology issue (dead-end) and a formSchema reference issue.
	// form2 is a dead-end (topology → blocking), and the gateway condition references
	// a nonexistent form field (formSchema → warning).
	workflowJSON := json.RawMessage(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"form1","type":"form","data":{"label":"申请表","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"urgency","type":"select","label":"紧急程度"}]}}},
			{"id":"gw","type":"exclusive","data":{"label":"分支"}},
			{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
			{"id":"p2","type":"process","data":{"label":"处理B","participants":[{"type":"requester"}]}},
			{"id":"form2","type":"form","data":{"label":"死胡同","participants":[{"type":"requester"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"form1","data":{}},
			{"id":"e2","source":"form1","target":"gw","data":{"outcome":"submitted"}},
			{"id":"e3","source":"gw","target":"p1","data":{"condition":{"field":"form.nonexistent_field","operator":"equals","value":"high"}}},
			{"id":"e4","source":"gw","target":"p2","data":{"default":true}},
			{"id":"e5","source":"p1","target":"end","data":{"outcome":"approved"}},
			{"id":"e5r","source":"p1","target":"end","data":{"outcome":"rejected"}},
			{"id":"e6","source":"p2","target":"end","data":{"outcome":"approved"}},
			{"id":"e6r","source":"p2","target":"end","data":{"outcome":"rejected"}},
			{"id":"e7","source":"gw","target":"form2","data":{"condition":{"field":"form.urgency","operator":"equals","value":"low"}}}
		]
	}`)

	errs := ValidateWorkflow(workflowJSON)

	var foundTopologyBlocking, foundFormSchemaWarning bool
	for _, e := range errs {
		if strings.Contains(e.Message, "无法到达终点") {
			if e.Level != "blocking" {
				t.Fatalf("expected dead-end error to be blocking, got level=%q: %s", e.Level, e.Message)
			}
			foundTopologyBlocking = true
		}
		if strings.Contains(e.Message, "formSchema") && strings.Contains(e.Message, "nonexistent_field") {
			if e.Level != "warning" {
				t.Fatalf("expected formSchema reference error to be warning, got level=%q: %s", e.Level, e.Message)
			}
			foundFormSchemaWarning = true
		}
	}

	if !foundTopologyBlocking {
		t.Fatalf("expected a blocking topology error (dead-end), got errors: %+v", errs)
	}
	if !foundFormSchemaWarning {
		t.Fatalf("expected a warning-level formSchema reference error, got errors: %+v", errs)
	}
}

func TestValidateWorkflowRejectsInvalidGatewayConditionContracts(t *testing.T) {
	t.Run("compound condition requires supported logic and children", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"p2","type":"process","data":{"label":"处理B","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"p1","data":{"condition":{"logic":"xor","conditions":[]}}},
				{"id":"e3","source":"gw","target":"p2","data":{"default":true}},
				{"id":"e4","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e4r","source":"p1","target":"end","data":{"outcome":"rejected"}},
				{"id":"e5","source":"p2","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p2","target":"end","data":{"outcome":"rejected"}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var badLogic, missingChildren bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, `logic 值 "xor" 不合法`) {
				badLogic = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "复合条件（logic=xor）缺少子条件") {
				missingChildren = true
			}
		}
		if !badLogic || !missingChildren {
			t.Fatalf("expected invalid compound condition errors, got %+v", errs)
		}
	})

	t.Run("leaf condition requires field and operator even inside nested logic", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"p2","type":"process","data":{"label":"处理B","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"p1","data":{"condition":{
					"logic":"and",
					"conditions":[
						{"field":"form.urgency","operator":"equals","value":"high"},
						{"field":"","operator":"","value":"vpn"}
					]
				}}},
				{"id":"e3","source":"gw","target":"p2","data":{"default":true}},
				{"id":"e4","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e4r","source":"p1","target":"end","data":{"outcome":"rejected"}},
				{"id":"e5","source":"p2","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p2","target":"end","data":{"outcome":"rejected"}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var missingField, missingOperator bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "边 e2 的条件缺少 field") {
				missingField = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "边 e2 的条件缺少 operator") {
				missingOperator = true
			}
		}
		if !missingField || !missingOperator {
			t.Fatalf("expected missing field/operator errors, got %+v", errs)
		}
	})
}

func TestValidateWorkflowRejectsStructuralGatewayAndTopologyViolations(t *testing.T) {
	t.Run("invalid node type and missing start are blocking", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"mystery","type":"quantum","data":{"label":"未知节点"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"mystery","target":"end","data":{}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var badType, missingStart bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, `类型 "quantum" 不合法`) {
				badType = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "工作流必须包含一个开始节点") {
				missingStart = true
			}
		}
		if !badType || !missingStart {
			t.Fatalf("expected invalid node type and missing start errors, got %+v", errs)
		}
	})

	t.Run("multiple starts and end with outgoing edge are rejected", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start1","type":"start","data":{"label":"开始1"}},
				{"id":"start2","type":"start","data":{"label":"开始2"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start1","target":"end","data":{}},
				{"id":"e2","source":"start2","target":"end","data":{}},
				{"id":"e3","source":"end","target":"start1","data":{}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var multiStart, endOutgoing bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "工作流只能包含一个开始节点") {
				multiStart = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "结束节点 end 不应有出边") {
				endOutgoing = true
			}
		}
		if !multiStart || !endOutgoing {
			t.Fatalf("expected multiple start and end-outgoing errors, got %+v", errs)
		}
	})

	t.Run("exclusive gateway requires enough conditioned branches", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"exclusive","data":{"label":"分支"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"p1","data":{}},
				{"id":"e3","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e3r","source":"p1","target":"end","data":{"outcome":"rejected"}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var needTwoEdges, missingCondition bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "排他网关节点 gw 至少需要两条出边") {
				needTwoEdges = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "出边 e2 缺少条件配置") {
				missingCondition = true
			}
		}
		if !needTwoEdges && !missingCondition {
			t.Fatalf("expected exclusive gateway contract errors, got %+v", errs)
		}
	})

	t.Run("parallel gateway requires direction and sufficient branches", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"pg","type":"parallel","data":{"label":"并行网关"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"pg","data":{}},
				{"id":"e2","source":"pg","target":"end","data":{}}
			]
		}`)

		errs := ValidateWorkflow(workflowJSON)
		var missingDirection bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "必须配置 gateway_direction") {
				missingDirection = true
				break
			}
		}
		if !missingDirection {
			t.Fatalf("expected missing gateway_direction error, got %+v", errs)
		}
	})
}

func TestValidateWorkflowRejectsBrokenTopologyAndProcessOutcomeContracts(t *testing.T) {
	t.Run("start node must have single outgoing edge and no incoming edge", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"mid","type":"process","data":{"label":"处理","participants":[{"type":"requester"}]}},
				{"id":"other","type":"process","data":{"label":"其他","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"mid","data":{}},
				{"id":"e2","source":"start","target":"other","data":{}},
				{"id":"e3","source":"mid","target":"start","data":{"outcome":"approved"}},
				{"id":"e4","source":"mid","target":"end","data":{"outcome":"rejected"}},
				{"id":"e5","source":"other","target":"end","data":{"outcome":"approved"}},
				{"id":"e6","source":"other","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var wrongOutDegree, hasIncoming bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "开始节点必须有且仅有一条出边") {
				wrongOutDegree = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "开始节点不应有入边") {
				hasIncoming = true
			}
		}
		if !wrongOutDegree || !hasIncoming {
			t.Fatalf("expected start topology errors, got %+v", errs)
		}
	})

	t.Run("edge references and isolated nodes are rejected", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"isolated","type":"process","data":{"label":"孤立","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"ghost","data":{}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var badTarget, isolated bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "引用了不存在的目标节点 ghost") {
				badTarget = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "节点 isolated 没有入边，无法到达") {
				isolated = true
			}
		}
		if !badTarget || !isolated {
			t.Fatalf("expected edge reference and isolated node errors, got %+v", errs)
		}
	})

	t.Run("process nodes require approved and rejected outcome edges", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"process","type":"process","data":{"label":"处理","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"process","data":{}},
				{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var missingRejected bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, `缺少 outcome="rejected" 的出边`) {
				missingRejected = true
				break
			}
		}
		if !missingRejected {
			t.Fatalf("expected missing rejected outcome error, got %+v", errs)
		}
	})

	t.Run("workflow without end node is rejected", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"process","type":"process","data":{"label":"处理","participants":[{"type":"requester"}]}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"process","data":{}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var missingEnd bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "工作流必须包含至少一个结束节点") {
				missingEnd = true
				break
			}
		}
		if !missingEnd {
			t.Fatalf("expected missing end error, got %+v", errs)
		}
	})
}

func TestValidateWorkflowRejectsParallelBoundaryAndSubprocessContracts(t *testing.T) {
	t.Run("inclusive fork requires conditions on non-default branches", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"gw","type":"inclusive","data":{"label":"包含分支","gateway_direction":"fork"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"p2","type":"process","data":{"label":"处理B","participants":[{"type":"requester"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"gw","data":{}},
				{"id":"e2","source":"gw","target":"p1","data":{}},
				{"id":"e3","source":"gw","target":"p2","data":{"default":true}},
				{"id":"e4","source":"p1","target":"end","data":{"outcome":"approved"}},
				{"id":"e4r","source":"p1","target":"end","data":{"outcome":"rejected"}},
				{"id":"e5","source":"p2","target":"end","data":{"outcome":"approved"}},
				{"id":"e5r","source":"p2","target":"end","data":{"outcome":"rejected"}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var found bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "包含网关 fork 节点 gw 的出边 e2 缺少条件配置") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected inclusive fork condition error, got %+v", errs)
		}
	})

	t.Run("parallel join requires at least two incoming edges and exactly one outgoing edge", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"fork","type":"parallel","data":{"label":"并行开始","gateway_direction":"fork"}},
				{"id":"p1","type":"process","data":{"label":"处理A","participants":[{"type":"requester"}]}},
				{"id":"join","type":"parallel","data":{"label":"并行汇聚","gateway_direction":"join"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"fork","data":{}},
				{"id":"e2","source":"fork","target":"p1","data":{}},
				{"id":"e3","source":"fork","target":"join","data":{}},
				{"id":"e4","source":"p1","target":"join","data":{"outcome":"approved"}},
				{"id":"e4r","source":"p1","target":"join","data":{"outcome":"rejected"}},
				{"id":"e5","source":"join","target":"end","data":{}},
				{"id":"e6","source":"join","target":"p1","data":{}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var badJoinOut bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "并行网关 join 节点 join 必须有且仅有一条出边") {
				badJoinOut = true
				break
			}
		}
		if !badJoinOut {
			t.Fatalf("expected parallel join outgoing edge error, got %+v", errs)
		}
	})

	t.Run("boundary nodes enforce host and edge contracts", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"action","type":"action","data":{"label":"动作","action_id":1}},
				{"id":"bt","type":"b_timer","data":{"label":"超时","attached_to":"action"}},
				{"id":"be","type":"b_error","data":{"label":"异常","attached_to":"start"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"action","data":{}},
				{"id":"e2","source":"action","target":"end","data":{"outcome":"success"}},
				{"id":"e3","source":"bt","target":"end","data":{}},
				{"id":"e4","source":"be","target":"end","data":{}},
				{"id":"e5","source":"start","target":"bt","data":{}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var timerHost, timerDuration, timerIncoming, errorHost bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "边界定时器 bt 只能附着在人工节点上") {
				timerHost = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "边界定时器 bt 必须配置 duration") {
				timerDuration = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "边界定时器 bt 不应有入边") {
				timerIncoming = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "边界错误事件 be 只能附着在 action 节点上") {
				errorHost = true
			}
		}
		if !timerHost || !timerDuration || !timerIncoming || !errorHost {
			t.Fatalf("expected boundary validation errors, got %+v", errs)
		}
	})

	t.Run("subprocess rejects nested subprocess and malformed definitions", func(t *testing.T) {
		workflowJSON := json.RawMessage(`{
			"nodes": [
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"sub","type":"subprocess","data":{"label":"子流程","subprocess_def":{
					"nodes":[
						{"id":"sub_start","type":"start","data":{"label":"子开始"}},
						{"id":"nested","type":"subprocess","data":{"label":"嵌套子流程","subprocess_def":{"bad":true}}},
						{"id":"sub_end","type":"end","data":{"label":"子结束"}}
					],
					"edges":[
						{"id":"se1","source":"sub_start","target":"nested","data":{}},
						{"id":"se2","source":"nested","target":"sub_end","data":{}}
					]
				}}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges": [
				{"id":"e1","source":"start","target":"sub","data":{}},
				{"id":"e2","source":"sub","target":"end","data":{}}
			]
		}`)
		errs := ValidateWorkflow(workflowJSON)
		var nestedBlocked, missingStart bool
		for _, err := range errs {
			if !err.IsWarning() && strings.Contains(err.Message, "当前版本不支持嵌套子流程") {
				nestedBlocked = true
			}
			if !err.IsWarning() && strings.Contains(err.Message, "工作流必须包含一个开始节点") {
				missingStart = true
			}
		}
		if !nestedBlocked || !missingStart {
			t.Fatalf("expected subprocess validation errors, got %+v", errs)
		}
	})
}
