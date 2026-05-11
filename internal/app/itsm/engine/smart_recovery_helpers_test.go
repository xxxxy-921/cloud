package engine

import "testing"

func TestRejectedRecoveryHelpersClassifyPlansAndSpecs(t *testing.T) {
	formPlan := &DecisionPlan{
		NextStepType: NodeForm,
		Activities: []DecisionActivity{
			{Type: NodeProcess, ParticipantType: "position_department", DepartmentCode: "it", PositionCode: "network_admin"},
		},
	}
	if !rejectedRecoveryCreatesForm(formPlan) {
		t.Fatal("expected form next step to be treated as rejected recovery form creation")
	}

	requesterPlan := &DecisionPlan{
		NextStepType: NodeProcess,
		Activities: []DecisionActivity{
			{Type: NodeProcess, ParticipantType: "requester", Instructions: "请申请人补充环境信息"},
		},
	}
	if !rejectedRecoveryCreatesRequesterHumanWork(requesterPlan) {
		t.Fatal("expected requester human work to be treated as rejected recovery requester work")
	}

	otherPlan := &DecisionPlan{
		NextStepType: NodeProcess,
		Activities: []DecisionActivity{
			{Type: NodeProcess, ParticipantType: "position_department", DepartmentCode: "it", PositionCode: "ops_admin"},
		},
	}
	if rejectedRecoveryCreatesForm(otherPlan) || rejectedRecoveryCreatesRequesterHumanWork(otherPlan) {
		t.Fatalf("did not expect ordinary process plan to be treated as requester/form recovery: %+v", otherPlan)
	}

	if !collaborationSpecAllowsRejectedFormRecovery(&serviceModel{CollaborationSpec: "处理人驳回后，流程退回申请人补充信息，申请人修改后提交。"}) {
		t.Fatal("expected explicit supplement/return cues to allow rejected form recovery")
	}
	if collaborationSpecAllowsRejectedFormRecovery(&serviceModel{CollaborationSpec: "处理完成后直接结束流程。"}) {
		t.Fatal("did not expect plain completion spec to allow rejected form recovery")
	}
}

func TestHasExplicitRecoveryIntentRecognizesRecoveryLanguage(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{text: "基于驳回原因升级给安全管理员继续处理", want: true},
		{text: "补充新的访问依据后重新提交", want: true},
		{text: "重新执行原审批", want: false},
		{text: "安排网络管理员处理", want: false},
	}

	for _, tc := range cases {
		if got := hasExplicitRecoveryIntent(tc.text); got != tc.want {
			t.Fatalf("hasExplicitRecoveryIntent(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}
