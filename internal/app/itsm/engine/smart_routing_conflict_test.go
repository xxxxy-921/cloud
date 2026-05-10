package engine

import (
	"strings"
	"testing"
)

func TestDetectTicketRoutingConflictsFindsCrossBranchFormSelections(t *testing.T) {
	workflow := `{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gateway","type":"exclusive","data":{"label":"分流"}},
			{"id":"network","type":"process","data":{"label":"网络管理员处理"}},
			{"id":"security","type":"process","data":{"label":"安全管理员处理"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"gateway","data":{}},
			{"id":"e2","source":"gateway","target":"network","data":{"condition":{"field":"form.request_kind","op":"eq","value":"network_access_issue"}}},
			{"id":"e3","source":"gateway","target":"security","data":{"condition":{"field":"form.request_kind","op":"eq","value":"security_compliance"}}}
		]
	}`

	t.Run("array value conflicts across branches", func(t *testing.T) {
		db, ticket := setupStructuredRoutingValidationDB(t, `{"request_kind":["network_access_issue","security_compliance"]}`)
		conflicts, err := detectTicketRoutingConflicts(db, ticket.ID, workflow)
		if err != nil {
			t.Fatalf("detectTicketRoutingConflicts: %v", err)
		}
		if len(conflicts) != 1 || !strings.Contains(conflicts[0], "form.request_kind") {
			t.Fatalf("expected request_kind conflict, got %+v", conflicts)
		}
	})

	t.Run("comma separated string also conflicts", func(t *testing.T) {
		db, ticket := setupStructuredRoutingValidationDB(t, `{"request_kind":"network_access_issue, security_compliance"}`)
		conflicts, err := detectTicketRoutingConflicts(db, ticket.ID, workflow)
		if err != nil {
			t.Fatalf("detectTicketRoutingConflicts: %v", err)
		}
		if len(conflicts) != 1 || !strings.Contains(conflicts[0], "命中 2 条分支") {
			t.Fatalf("expected two-branch conflict, got %+v", conflicts)
		}
	})
}

func TestValidateRoutingConflictDecisionRejectsHighConfidenceSingleRoute(t *testing.T) {
	workflow := `{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"gateway","type":"exclusive","data":{"label":"分流"}},
			{"id":"network","type":"process","data":{"label":"网络管理员处理"}},
			{"id":"security","type":"process","data":{"label":"安全管理员处理"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"gateway","data":{}},
			{"id":"e2","source":"gateway","target":"network","data":{"condition":{"field":"form.request_kind","op":"eq","value":["network_access_issue","troubleshooting"]}}},
			{"id":"e3","source":"gateway","target":"security","data":{"condition":{"field":"form.request_kind","op":"eq","value":"security_compliance"}}}
		]
	}`
	db, ticket := setupStructuredRoutingValidationDB(t, `{"request_kind":["troubleshooting","security_compliance"]}`)

	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "network_admin",
		}},
		Confidence: 0.95,
	}

	err := (&SmartEngine{}).validateRoutingConflictDecision(db, ticket.ID, plan, &serviceModel{WorkflowJSON: workflow})
	if err == nil || !strings.Contains(err.Error(), "表单路由字段存在跨分支冲突") {
		t.Fatalf("expected routing conflict validation error, got %v", err)
	}

	lowConfidence := *plan
	lowConfidence.Confidence = 0.5
	if err := (&SmartEngine{}).validateRoutingConflictDecision(db, ticket.ID, &lowConfidence, &serviceModel{WorkflowJSON: workflow}); err != nil {
		t.Fatalf("expected low confidence plan to skip conflict rejection, got %v", err)
	}

	requesterOnly := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "requester",
		}},
		Confidence: 0.95,
	}
	if err := (&SmartEngine{}).validateRoutingConflictDecision(db, ticket.ID, requesterOnly, &serviceModel{WorkflowJSON: workflow}); err != nil {
		t.Fatalf("expected requester fallback plan to skip single-route conflict rejection, got %v", err)
	}
}
