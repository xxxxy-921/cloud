package engine

import "testing"

func TestDecisionActivityTargetsPositionMatchesDirectAndResolvedParticipants(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"request_kind":"network_access_issue"}`)
	_ = ticket

	if !decisionActivityTargetsPosition(db, DecisionActivity{
		ParticipantType: "position_department",
		PositionCode:    "network_admin",
		DepartmentCode:  "it",
	}, "network_admin") {
		t.Fatal("expected direct position_department participant to match target position")
	}

	if decisionActivityTargetsPosition(db, DecisionActivity{
		ParticipantType: "position_department",
		PositionCode:    "ops_admin",
		DepartmentCode:  "it",
	}, "network_admin") {
		t.Fatal("did not expect mismatched direct position_department participant to match target position")
	}

	if !decisionActivityTargetsPosition(db, DecisionActivity{
		ParticipantID: uintPtrIf(1),
	}, "network_admin") {
		t.Fatal("expected participant user 1 to map to network_admin via user_positions")
	}

	if decisionActivityTargetsPosition(db, DecisionActivity{
		ParticipantID: uintPtrIf(2),
	}, "network_admin") {
		t.Fatal("did not expect security_admin user to match network_admin target")
	}

	if decisionActivityTargetsPosition(db, DecisionActivity{}, "") {
		t.Fatal("did not expect empty expected position to ever match")
	}
}

func TestFindOutcomeEdgeTargetInfoAndDescriptions(t *testing.T) {
	workflow := `{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"approve","type":"process","data":{"label":"人工审批"}},
			{"id":"supplement","type":"form","data":{"label":"补充信息"}},
			{"id":"end_ok","type":"end","data":{"label":"完成"}},
			{"id":"end_fail","type":"end","data":{"label":"拒绝结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"approve","data":{}},
			{"id":"e2","source":"approve","target":"end_ok","data":{"outcome":"approved"}},
			{"id":"e3","source":"approve","target":"supplement","data":{"outcome":"rejected"}},
			{"id":"e4","source":"supplement","target":"end_fail","data":{"outcome":"submitted"}}
		]
	}`

	targetID, label, nodeType := findOutcomeEdgeTargetInfo(workflow, "approve", "approved")
	if targetID != "end_ok" || label != "完成" || nodeType != "end" {
		t.Fatalf("approved target info = %q/%q/%q, want end_ok/完成/end", targetID, label, nodeType)
	}

	targetID, label, nodeType = findOutcomeEdgeTargetInfo(workflow, "approve", "rejected")
	if targetID != "supplement" || label != "补充信息" || nodeType != "form" {
		t.Fatalf("rejected target info = %q/%q/%q, want supplement/补充信息/form", targetID, label, nodeType)
	}

	if got := findApprovedEdgeTarget(workflow, "approve"); got != "end_ok（完成，类型: end）" {
		t.Fatalf("findApprovedEdgeTarget = %q", got)
	}
	if got := findRejectedEdgeTarget(workflow, "approve"); got != "supplement（补充信息，类型: form）" {
		t.Fatalf("findRejectedEdgeTarget = %q", got)
	}
	targetID, label, nodeType = findOutcomeEdgeTargetInfo("", "approve", "approved")
	if targetID != "" || label != "" || nodeType != "" {
		t.Fatalf("expected empty workflow to return zero target info, got %q/%q/%q", targetID, label, nodeType)
	}
}
