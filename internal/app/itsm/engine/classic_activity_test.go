package engine

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestClassicMatrixAssignParticipantsWarningsAndAssignments(t *testing.T) {
	t.Run("missing participants records manual assignment warning", func(t *testing.T) {
		f := newClassicMatrixFixture(t)
		workflow := json.RawMessage(`{
			"nodes":[
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"process","type":"process","data":{"label":"人工处理"}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges":[
				{"id":"e1","source":"start","target":"process","data":{}},
				{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}}
			]
		}`)
		ticket := f.createTicket(t, workflow)

		if err := f.start(t, ticket, workflow); err != nil {
			t.Fatalf("start: %v", err)
		}

		var assignments int64
		if err := f.db.Model(&assignmentModel{}).Where("ticket_id = ?", ticket.ID).Count(&assignments).Error; err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		if assignments != 0 {
			t.Fatalf("assignment count = %d, want 0", assignments)
		}

		var timeline timelineModel
		if err := f.db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "warning").Order("id ASC").First(&timeline).Error; err != nil {
			t.Fatalf("load warning timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "节点未配置参与人") {
			t.Fatalf("warning timeline message = %q, want missing participant warning", timeline.Message)
		}
	})

	t.Run("resolution failures warn instead of creating phantom assignments", func(t *testing.T) {
		f := newClassicMatrixFixture(t)
		workflow := json.RawMessage(`{
			"nodes":[
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"process","type":"process","data":{"label":"人工处理","participants":[{"type":"user","value":"ghost-user"}]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges":[
				{"id":"e1","source":"start","target":"process","data":{}},
				{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}}
			]
		}`)
		ticket := f.createTicket(t, workflow)

		if err := f.start(t, ticket, workflow); err != nil {
			t.Fatalf("start: %v", err)
		}

		var assignments int64
		if err := f.db.Model(&assignmentModel{}).Where("ticket_id = ?", ticket.ID).Count(&assignments).Error; err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		if assignments != 0 {
			t.Fatalf("assignment count = %d, want 0", assignments)
		}

		var timeline timelineModel
		if err := f.db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "warning").Order("id ASC").First(&timeline).Error; err != nil {
			t.Fatalf("load warning timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "参与人解析失败") {
			t.Fatalf("warning timeline message = %q, want resolution failure warning", timeline.Message)
		}
	})

	t.Run("multiple participants keep ordering and current assignee", func(t *testing.T) {
		f := newClassicMatrixFixture(t)
		workflow := json.RawMessage(`{
			"nodes":[
				{"id":"start","type":"start","data":{"label":"开始"}},
				{"id":"process","type":"process","data":{"label":"人工处理","participants":[
					{"type":"user","value":"7"},
					{"type":"requester"}
				]}},
				{"id":"end","type":"end","data":{"label":"结束"}}
			],
			"edges":[
				{"id":"e1","source":"start","target":"process","data":{}},
				{"id":"e2","source":"process","target":"end","data":{"outcome":"approved"}}
			]
		}`)
		ticket := f.createTicket(t, workflow)

		if err := f.start(t, ticket, workflow); err != nil {
			t.Fatalf("start: %v", err)
		}

		var assignments []assignmentModel
		if err := f.db.Where("ticket_id = ?", ticket.ID).Order("sequence ASC, id ASC").Find(&assignments).Error; err != nil {
			t.Fatalf("load assignments: %v", err)
		}
		if len(assignments) != 2 {
			t.Fatalf("assignment count = %d, want 2", len(assignments))
		}
		if assignments[0].Sequence != 0 || !assignments[0].IsCurrent {
			t.Fatalf("first assignment = %+v, want sequence 0 current", assignments[0])
		}
		if assignments[1].Sequence != 1 || assignments[1].IsCurrent {
			t.Fatalf("second assignment = %+v, want sequence 1 non-current", assignments[1])
		}
		if assignments[0].AssigneeID == nil || *assignments[0].AssigneeID != 7 {
			t.Fatalf("first assignee = %+v, want user 7", assignments[0].AssigneeID)
		}

		var ticketRow struct {
			AssigneeID *uint
		}
		if err := f.db.Table("itsm_tickets").Where("id = ?", ticket.ID).Select("assignee_id").First(&ticketRow).Error; err != nil {
			t.Fatalf("reload ticket assignee: %v", err)
		}
		if ticketRow.AssigneeID == nil || *ticketRow.AssigneeID != 7 {
			t.Fatalf("ticket assignee = %+v, want 7", ticketRow.AssigneeID)
		}
	})
}

func TestResolveClassicCompletionStatusUsesLatestHumanOutcome(t *testing.T) {
	f := newClassicMatrixFixture(t)
	ticket := f.createTicket(t, json.RawMessage(`{"nodes":[],"edges":[]}`))

	status, outcome, err := resolveClassicCompletionStatus(f.db, ticket.ID)
	if err != nil {
		t.Fatalf("resolve completion status without history: %v", err)
	}
	if status != TicketStatusCompleted || outcome != TicketOutcomeFulfilled {
		t.Fatalf("no human history status/outcome = %s/%s, want %s/%s", status, outcome, TicketStatusCompleted, TicketOutcomeFulfilled)
	}

	now := time.Now()
	approvedAt := now.Add(-2 * time.Minute)
	rejectedAt := now.Add(-time.Minute)
	activities := []activityModel{
		{TicketID: ticket.ID, Name: "审批通过", ActivityType: NodeApprove, Status: ActivityApproved, TransitionOutcome: ActivityApproved, FinishedAt: &approvedAt},
		{TicketID: ticket.ID, Name: "表单驳回", ActivityType: NodeForm, Status: ActivityRejected, TransitionOutcome: ActivityRejected, FinishedAt: &rejectedAt},
	}
	for i := range activities {
		if err := f.db.Create(&activities[i]).Error; err != nil {
			t.Fatalf("create completed activity %d: %v", i, err)
		}
	}

	status, outcome, err = resolveClassicCompletionStatus(f.db, ticket.ID)
	if err != nil {
		t.Fatalf("resolve completion status with latest rejected activity: %v", err)
	}
	if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
		t.Fatalf("latest rejected status/outcome = %s/%s, want %s/%s", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
	}

	if err := f.db.Model(&activityModel{}).Where("id = ?", activities[1].ID).
		Updates(map[string]any{"status": ActivityCompleted, "transition_outcome": "completed", "finished_at": now}).Error; err != nil {
		t.Fatalf("rewrite latest activity: %v", err)
	}

	status, outcome, err = resolveClassicCompletionStatus(f.db, ticket.ID)
	if err != nil {
		t.Fatalf("resolve completion status with latest neutral activity: %v", err)
	}
	if status != TicketStatusCompleted || outcome != TicketOutcomeFulfilled {
		t.Fatalf("latest neutral status/outcome = %s/%s, want %s/%s", status, outcome, TicketStatusCompleted, TicketOutcomeFulfilled)
	}
}

func TestClassicEngineCompleteActivityPersistsOpinionAndResult(t *testing.T) {
	f := newClassicMatrixFixture(t)
	ticket := f.createTicket(t, json.RawMessage(`{"nodes":[],"edges":[]}`))

	now := time.Now()
	activity := activityModel{
		TicketID:      ticket.ID,
		Name:          "人工处理",
		ActivityType:  NodeProcess,
		Status:        ActivityPending,
		NodeID:        "process_1",
		ExecutionMode: "single",
		StartedAt:     &now,
	}
	if err := f.db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}

	result := json.RawMessage(`{"approved":true,"remark":"ok"}`)
	completed, err := f.engine.completeActivity(f.db, &activity, ProgressParams{
		ActivityID: activity.ID,
		Outcome:    ActivityApproved,
		Opinion:    "已审核通过",
		Result:     result,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("completeActivity: %v", err)
	}
	if !completed {
		t.Fatal("expected completeActivity to report completed")
	}

	var reloaded activityModel
	if err := f.db.First(&reloaded, activity.ID).Error; err != nil {
		t.Fatalf("reload activity: %v", err)
	}
	if reloaded.Status != ActivityCompleted || reloaded.TransitionOutcome != ActivityApproved {
		t.Fatalf("unexpected activity status after completion: %+v", reloaded)
	}
	if reloaded.DecisionReasoning != "已审核通过" {
		t.Fatalf("decision reasoning = %q, want %q", reloaded.DecisionReasoning, "已审核通过")
	}
	if reloaded.FormData != string(result) {
		t.Fatalf("form data = %s, want %s", reloaded.FormData, string(result))
	}
	if reloaded.FinishedAt == nil || !reloaded.FinishedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("finished_at = %v, want %v", reloaded.FinishedAt, now.Add(time.Minute))
	}
}
