package engine

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSmartEngineExecuteDecisionPlanCompleteUsesWorkflowEndAndLatestHumanOutcome(t *testing.T) {
	t.Run("latest rejected human outcome keeps ticket rejected", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		service := serviceModel{
			Name:       "智能 VPN 服务",
			EngineType: "smart",
			WorkflowJSON: `{
				"nodes": [
					{"id": "start-1", "type": "start", "label": "开始"},
					{"id": "end-1", "type": "end", "label": "结束"}
				],
				"edges": []
			}`,
		}
		if err := db.Create(&service).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}

		ticket := ticketModel{
			ServiceID:  service.ID,
			Status:     TicketStatusDecisioning,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		approvedAt := time.Now().Add(-2 * time.Hour)
		rejectedAt := time.Now().Add(-time.Hour)
		if err := db.Create(&activityModel{
			TicketID:          ticket.ID,
			Name:              "总部初审",
			ActivityType:      NodeProcess,
			Status:            ActivityApproved,
			TransitionOutcome: ActivityApproved,
			StartedAt:         &approvedAt,
			FinishedAt:        &approvedAt,
		}).Error; err != nil {
			t.Fatalf("seed approved activity: %v", err)
		}
		if err := db.Create(&activityModel{
			TicketID:          ticket.ID,
			Name:              "申请人补充",
			ActivityType:      NodeForm,
			Status:            ActivityRejected,
			TransitionOutcome: ActivityRejected,
			StartedAt:         &rejectedAt,
			FinishedAt:        &rejectedAt,
		}).Error; err != nil {
			t.Fatalf("seed rejected activity: %v", err)
		}

		plan := &DecisionPlan{
			NextStepType: "complete",
			Reasoning:    "最新人工结论为驳回，流程直接收口",
			Confidence:   0.99,
		}
		engine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.ExecuteDecisionPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute completion plan: %v", err)
		}

		var completed activityModel
		if err := db.Where("ticket_id = ? AND activity_type = ?", ticket.ID, "complete").
			Order("id DESC").
			First(&completed).Error; err != nil {
			t.Fatalf("load completed activity: %v", err)
		}
		if completed.NodeID != "end-1" {
			t.Fatalf("expected end node id end-1, got %q", completed.NodeID)
		}
		if completed.Status != ActivityCompleted || completed.TransitionOutcome != "completed" {
			t.Fatalf("expected completed end activity, got status=%q outcome=%q", completed.Status, completed.TransitionOutcome)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != TicketStatusRejected || reloaded.Outcome != TicketOutcomeRejected {
			t.Fatalf("expected rejected ticket outcome, got status=%q outcome=%q", reloaded.Status, reloaded.Outcome)
		}
		if reloaded.CurrentActivityID == nil || *reloaded.CurrentActivityID != completed.ID {
			t.Fatalf("expected current activity to point at complete node, got %v", reloaded.CurrentActivityID)
		}
		if reloaded.FinishedAt == nil || reloaded.FinishedAt.IsZero() {
			t.Fatal("expected ticket finished_at to be set")
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "workflow_completed").
			Order("id DESC").
			First(&timeline).Error; err != nil {
			t.Fatalf("load completion timeline: %v", err)
		}
		if timeline.ActivityID == nil || *timeline.ActivityID != completed.ID {
			t.Fatalf("expected completion timeline to point at completed activity, got %v", timeline.ActivityID)
		}
	})

	t.Run("without prior human result ticket falls back to fulfilled", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		service := serviceModel{
			Name:       "智能 VPN 服务",
			EngineType: "smart",
		}
		if err := db.Create(&service).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}

		ticket := ticketModel{
			ServiceID:  service.ID,
			Status:     TicketStatusDecisioning,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		plan := &DecisionPlan{
			NextStepType: "complete",
			Reasoning:    "没有人工驳回记录，按 fulfilled 收口",
			Confidence:   0.92,
		}
		engine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.ExecuteDecisionPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute completion plan: %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != TicketStatusCompleted || reloaded.Outcome != TicketOutcomeFulfilled {
			t.Fatalf("expected fulfilled completion, got status=%q outcome=%q", reloaded.Status, reloaded.Outcome)
		}
	})
}
