package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"metis/internal/scheduler"
)

func TestHandleWaitTimer_NotReady(t *testing.T) {
	// When execute_after is in the future, handler should return ErrNotReady
	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	payload, _ := json.Marshal(WaitTimerPayload{
		TicketID:     1,
		ActivityID:   1,
		ExecuteAfter: future,
	})

	// We can call the handler directly — it will return ErrNotReady before touching DB
	handler := HandleWaitTimer(nil, nil)
	err := handler(context.Background(), payload)
	if !errors.Is(err, scheduler.ErrNotReady) {
		t.Errorf("expected ErrNotReady for future timer, got: %v", err)
	}
}

func TestHandleWaitTimer_InvalidPayload(t *testing.T) {
	handler := HandleWaitTimer(nil, nil)
	err := handler(context.Background(), json.RawMessage(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid payload")
	}
}

func TestHandleWaitTimer_InvalidTime(t *testing.T) {
	payload, _ := json.Marshal(WaitTimerPayload{
		TicketID:     1,
		ActivityID:   1,
		ExecuteAfter: "not-a-time",
	})
	handler := HandleWaitTimer(nil, nil)
	err := handler(context.Background(), payload)
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}

func TestHandleBoundaryTimer_NotReady(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	payload, _ := json.Marshal(BoundaryTimerPayload{
		TicketID:        1,
		BoundaryTokenID: 1,
		BoundaryNodeID:  "bt1",
		HostTokenID:     2,
		ExecuteAfter:    future,
	})

	handler := HandleBoundaryTimer(nil, nil)
	err := handler(context.Background(), payload)
	if !errors.Is(err, scheduler.ErrNotReady) {
		t.Errorf("expected ErrNotReady for future boundary timer, got: %v", err)
	}
}

func TestHandleBoundaryTimer_InvalidPayload(t *testing.T) {
	handler := HandleBoundaryTimer(nil, nil)
	err := handler(context.Background(), json.RawMessage(`{bad}`))
	if err == nil {
		t.Error("expected error for invalid payload")
	}
}

func TestHandleBoundaryTimer_InterruptsHostAndContinuesBoundaryPath(t *testing.T) {
	f := newClassicMatrixFixture(t)
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"review","type":"process","data":{"label":"人工审核","participants":[{"type":"user","value":"7"}]}},
			{"id":"timeout_boundary","type":"b_timer","data":{"label":"超时转交","attached_to":"review","duration":"1m"}},
			{"id":"manual","type":"process","data":{"label":"超时接管","participants":[{"type":"user","value":"7"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"review","data":{}},
			{"id":"e2","source":"review","target":"end","data":{"outcome":"completed"}},
			{"id":"e3","source":"timeout_boundary","target":"manual","data":{}},
			{"id":"e4","source":"manual","target":"end","data":{"outcome":"completed"}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(f.submitter.tasks) != 1 || f.submitter.tasks[0].name != "itsm-boundary-timer" {
		t.Fatalf("unexpected submitted tasks: %+v", f.submitter.tasks)
	}

	reviewActivity := f.firstActivity(t, ticket.ID, NodeProcess)
	var hostToken executionTokenModel
	if err := f.db.First(&hostToken, *reviewActivity.TokenID).Error; err != nil {
		t.Fatalf("load host token: %v", err)
	}
	var boundaryToken executionTokenModel
	if err := f.db.Where("parent_token_id = ? AND node_id = ?", hostToken.ID, "timeout_boundary").First(&boundaryToken).Error; err != nil {
		t.Fatalf("load boundary token: %v", err)
	}
	if boundaryToken.Status != TokenSuspended {
		t.Fatalf("boundary token status = %s, want suspended", boundaryToken.Status)
	}

	var payload BoundaryTimerPayload
	if err := json.Unmarshal(f.submitter.tasks[0].payload, &payload); err != nil {
		t.Fatalf("unmarshal submitted payload: %v", err)
	}
	payload.ExecuteAfter = time.Now().Add(-time.Minute).Format(time.RFC3339)
	raw, _ := json.Marshal(payload)

	handler := HandleBoundaryTimer(f.db, f.engine)
	if err := handler(context.Background(), raw); err != nil {
		t.Fatalf("HandleBoundaryTimer: %v", err)
	}

	var cancelledActivity activityModel
	if err := f.db.First(&cancelledActivity, reviewActivity.ID).Error; err != nil {
		t.Fatalf("reload host activity: %v", err)
	}
	if cancelledActivity.Status != ActivityCancelled {
		t.Fatalf("host activity status = %s, want cancelled", cancelledActivity.Status)
	}

	var refreshedHostToken executionTokenModel
	if err := f.db.First(&refreshedHostToken, hostToken.ID).Error; err != nil {
		t.Fatalf("reload host token: %v", err)
	}
	if refreshedHostToken.Status != TokenCancelled {
		t.Fatalf("host token status = %s, want cancelled", refreshedHostToken.Status)
	}

	manualActivity := f.firstActivity(t, ticket.ID, NodeProcess)
	if manualActivity.ID == reviewActivity.ID || manualActivity.NodeID != "manual" || manualActivity.Status != ActivityPending {
		t.Fatalf("unexpected boundary target activity: %+v", manualActivity)
	}
	if got := f.ticketStatus(t, ticket.ID); got != TicketStatusWaitingHuman {
		t.Fatalf("ticket status = %s, want %s", got, TicketStatusWaitingHuman)
	}

	var timelineCount int64
	if err := f.db.Model(&timelineModel{}).Where("ticket_id = ? AND event_type = ?", ticket.ID, "boundary_timer_fired").Count(&timelineCount).Error; err != nil {
		t.Fatalf("count boundary timer timeline: %v", err)
	}
	if timelineCount != 1 {
		t.Fatalf("boundary_timer_fired count = %d, want 1", timelineCount)
	}

	var assignment assignmentModel
	if err := f.db.Where("activity_id = ?", manualActivity.ID).First(&assignment).Error; err != nil {
		t.Fatalf("load manual assignment: %v", err)
	}
	if assignment.UserID == nil || *assignment.UserID != 7 || assignment.Status != "pending" {
		t.Fatalf("unexpected manual assignment: %+v", assignment)
	}
}

func TestHandleBoundaryTimer_SkipsWhenBoundaryAlreadyResolved(t *testing.T) {
	f := newClassicMatrixFixture(t)
	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"review","type":"process","data":{"label":"人工审核","participants":[{"type":"user","value":"7"}]}},
			{"id":"timeout_boundary","type":"b_timer","data":{"label":"超时转交","attached_to":"review","duration":"1m"}},
			{"id":"manual","type":"process","data":{"label":"超时接管","participants":[{"type":"user","value":"7"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"review","data":{}},
			{"id":"e2","source":"review","target":"end","data":{"outcome":"completed"}},
			{"id":"e3","source":"timeout_boundary","target":"manual","data":{}},
			{"id":"e4","source":"manual","target":"end","data":{"outcome":"completed"}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}
	reviewActivity := f.firstActivity(t, ticket.ID, NodeProcess)
	var boundaryToken executionTokenModel
	if err := f.db.Where("node_id = ?", "timeout_boundary").First(&boundaryToken).Error; err != nil {
		t.Fatalf("load boundary token: %v", err)
	}
	if err := f.db.Model(&executionTokenModel{}).Where("id = ?", boundaryToken.ID).Update("status", TokenCancelled).Error; err != nil {
		t.Fatalf("cancel boundary token: %v", err)
	}

	payload := BoundaryTimerPayload{
		TicketID:        ticket.ID,
		BoundaryTokenID: boundaryToken.ID,
		BoundaryNodeID:  "timeout_boundary",
		HostTokenID:     *reviewActivity.TokenID,
		ExecuteAfter:    time.Now().Add(-time.Minute).Format(time.RFC3339),
	}
	raw, _ := json.Marshal(payload)

	handler := HandleBoundaryTimer(f.db, f.engine)
	if err := handler(context.Background(), raw); err != nil {
		t.Fatalf("HandleBoundaryTimer already resolved: %v", err)
	}

	var reloaded activityModel
	if err := f.db.First(&reloaded, reviewActivity.ID).Error; err != nil {
		t.Fatalf("reload host activity: %v", err)
	}
	if reloaded.Status != ActivityPending {
		t.Fatalf("host activity status = %s, want pending", reloaded.Status)
	}

	var timelineCount int64
	if err := f.db.Model(&timelineModel{}).Where("ticket_id = ? AND event_type = ?", ticket.ID, "boundary_timer_fired").Count(&timelineCount).Error; err != nil {
		t.Fatalf("count boundary timer timeline: %v", err)
	}
	if timelineCount != 0 {
		t.Fatalf("boundary_timer_fired count = %d, want 0", timelineCount)
	}
}
