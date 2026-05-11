package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestClassicEngineCancelCancelsActiveWorkflowState(t *testing.T) {
	f := setupDispatchTest(t)
	if err := f.db.Model(&ticketModel{}).Where("id = ?", f.ticket.ID).Update("assignee_id", 100).Error; err != nil {
		t.Fatalf("seed assignee_id: %v", err)
	}

	err := f.db.Transaction(func(tx *gorm.DB) error {
		return f.engine.Cancel(context.Background(), tx, CancelParams{
			TicketID:   f.ticket.ID,
			Reason:     "用户撤单",
			OperatorID: 42,
		})
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	var token executionTokenModel
	if err := f.db.First(&token, f.token.ID).Error; err != nil {
		t.Fatalf("load token: %v", err)
	}
	if token.Status != TokenCancelled {
		t.Fatalf("token status = %s, want cancelled", token.Status)
	}

	var activity activityModel
	if err := f.db.First(&activity, f.activity.ID).Error; err != nil {
		t.Fatalf("load activity: %v", err)
	}
	if activity.Status != ActivityCancelled || activity.FinishedAt == nil {
		t.Fatalf("unexpected cancelled activity: %+v", activity)
	}

	var assignment assignmentModel
	if err := f.db.First(&assignment, f.assignment.ID).Error; err != nil {
		t.Fatalf("load assignment: %v", err)
	}
	if assignment.Status != ActivityCancelled {
		t.Fatalf("assignment status = %s, want cancelled", assignment.Status)
	}

	var ticket ticketModel
	if err := f.db.First(&ticket, f.ticket.ID).Error; err != nil {
		t.Fatalf("load ticket: %v", err)
	}
	if ticket.Status != TicketStatusCancelled || ticket.Outcome != TicketOutcomeCancelled || ticket.FinishedAt == nil {
		t.Fatalf("unexpected cancelled ticket: %+v", ticket)
	}
	if ticket.CurrentActivityID != nil {
		t.Fatalf("expected current_activity_id to clear, got %v", *ticket.CurrentActivityID)
	}
	var ticketAssignee struct {
		AssigneeID *uint `gorm:"column:assignee_id"`
	}
	if err := f.db.Table("itsm_tickets").Select("assignee_id").Where("id = ?", f.ticket.ID).First(&ticketAssignee).Error; err != nil {
		t.Fatalf("load ticket assignee_id: %v", err)
	}
	if ticketAssignee.AssigneeID != nil {
		t.Fatalf("expected assignee_id to clear, got %v", *ticketAssignee.AssigneeID)
	}

	var timeline timelineModel
	if err := f.db.Where("ticket_id = ? AND event_type = ?", f.ticket.ID, "ticket_cancelled").First(&timeline).Error; err != nil {
		t.Fatalf("load cancel timeline: %v", err)
	}
	if timeline.Message != "工单已取消: 用户撤单" {
		t.Fatalf("timeline message = %q, want cancel reason", timeline.Message)
	}
}

func TestSmartEngineCancelClearsCurrentActivityAndAssignee(t *testing.T) {
	db := newSmartContinuationDB(t)
	if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
		t.Fatalf("add assignee_id: %v", err)
	}

	ticket := ticketModel{Status: TicketStatusWaitingHuman, EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:     ticket.ID,
		Name:         "人工处理",
		ActivityType: NodeProcess,
		Status:       ActivityPending,
		NodeID:       "human-1",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if err := db.Model(&ticketModel{}).Where("id = ?", ticket.ID).Updates(map[string]any{
		"current_activity_id": activity.ID,
		"assignee_id":         200,
	}).Error; err != nil {
		t.Fatalf("seed ticket state: %v", err)
	}
	assigneeID := uint(200)
	assignment := assignmentModel{
		TicketID:        ticket.ID,
		ActivityID:      activity.ID,
		ParticipantType: "user",
		UserID:          &assigneeID,
		AssigneeID:      &assigneeID,
		Status:          ActivityPending,
		IsCurrent:       true,
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
	if err := db.Transaction(func(tx *gorm.DB) error {
		return engine.Cancel(context.Background(), tx, CancelParams{
			TicketID:   ticket.ID,
			Reason:     "用户撤单",
			OperatorID: 9,
		})
	}); err != nil {
		t.Fatalf("cancel smart ticket: %v", err)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.Status != TicketStatusCancelled || reloaded.Outcome != TicketOutcomeCancelled || reloaded.FinishedAt == nil {
		t.Fatalf("unexpected cancelled ticket: %+v", reloaded)
	}
	if reloaded.CurrentActivityID != nil {
		t.Fatalf("expected current_activity_id to clear, got %v", *reloaded.CurrentActivityID)
	}
	var ticketAssignee struct {
		AssigneeID *uint `gorm:"column:assignee_id"`
	}
	if err := db.Table("itsm_tickets").Select("assignee_id").Where("id = ?", ticket.ID).First(&ticketAssignee).Error; err != nil {
		t.Fatalf("load ticket assignee_id: %v", err)
	}
	if ticketAssignee.AssigneeID != nil {
		t.Fatalf("expected assignee_id to clear, got %v", *ticketAssignee.AssigneeID)
	}

	var refreshedActivity activityModel
	if err := db.First(&refreshedActivity, activity.ID).Error; err != nil {
		t.Fatalf("reload activity: %v", err)
	}
	if refreshedActivity.Status != ActivityCancelled || refreshedActivity.FinishedAt == nil {
		t.Fatalf("unexpected cancelled activity: %+v", refreshedActivity)
	}

	var refreshedAssignment assignmentModel
	if err := db.First(&refreshedAssignment, assignment.ID).Error; err != nil {
		t.Fatalf("reload assignment: %v", err)
	}
	if refreshedAssignment.Status != ActivityCancelled {
		t.Fatalf("assignment status = %s, want cancelled", refreshedAssignment.Status)
	}

	var timeline timelineModel
	if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ticket_cancelled").First(&timeline).Error; err != nil {
		t.Fatalf("load cancel timeline: %v", err)
	}
	if timeline.Message != "工单已取消: 用户撤单" {
		t.Fatalf("timeline message = %q, want cancel reason", timeline.Message)
	}
}

func TestClassicEngineCancelSupportsWithdrawEvent(t *testing.T) {
	f := setupDispatchTest(t)

	err := f.db.Transaction(func(tx *gorm.DB) error {
		return f.engine.Cancel(context.Background(), tx, CancelParams{
			TicketID:   f.ticket.ID,
			OperatorID: 7,
			EventType:  "withdrawn",
			Message:    "申请人撤回工单",
		})
	})
	if err != nil {
		t.Fatalf("Cancel withdraw: %v", err)
	}

	var ticket ticketModel
	if err := f.db.First(&ticket, f.ticket.ID).Error; err != nil {
		t.Fatalf("load ticket: %v", err)
	}
	if ticket.Status != TicketStatusWithdrawn || ticket.Outcome != TicketOutcomeWithdrawn {
		t.Fatalf("unexpected withdrawn ticket: %+v", ticket)
	}

	var timeline timelineModel
	if err := f.db.Where("ticket_id = ? AND event_type = ?", f.ticket.ID, "withdrawn").First(&timeline).Error; err != nil {
		t.Fatalf("load withdraw timeline: %v", err)
	}
	if timeline.Message != "申请人撤回工单" {
		t.Fatalf("withdraw timeline message = %q", timeline.Message)
	}
}

func TestHandleWaitTimerProgressesExpiredWaitNode(t *testing.T) {
	f := newClassicMatrixFixture(t)
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"wait","type":"wait","data":{"label":"等待外部回调","wait_mode":"timer","duration":"1m"}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"wait","data":{}},
			{"id":"e2","source":"wait","target":"end","data":{"outcome":"timeout","default":true}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}

	waitActivity := f.firstActivity(t, ticket.ID, NodeWait)
	handler := HandleWaitTimer(f.db, f.engine)
	payload, _ := json.Marshal(WaitTimerPayload{
		TicketID:     ticket.ID,
		ActivityID:   waitActivity.ID,
		ExecuteAfter: time.Now().Add(-time.Minute).Format(time.RFC3339),
	})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("HandleWaitTimer: %v", err)
	}

	status, outcome := f.ticketStatusOutcome(t, ticket.ID)
	if status != TicketStatusCompleted || outcome != TicketOutcomeFulfilled {
		t.Fatalf("unexpected ticket terminal state: status=%s outcome=%s", status, outcome)
	}

	var refreshed activityModel
	if err := f.db.First(&refreshed, waitActivity.ID).Error; err != nil {
		t.Fatalf("reload wait activity: %v", err)
	}
	if refreshed.Status != ActivityApproved {
		t.Fatalf("wait activity status = %s, want approved", refreshed.Status)
	}
}
