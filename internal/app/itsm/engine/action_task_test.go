package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleActionExecuteForSmartTicketMarksActivityAndSubmitsDecisionTask(t *testing.T) {
	db := setupActionExecutorDB(t)
	if err := db.AutoMigrate(&activityModel{}); err != nil {
		t.Fatalf("migrate activity: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	ticket := ticketModel{ID: 1, Code: "TICK-ACTION-SMART", Status: TicketStatusExecutingAction, EngineType: "smart", RequesterID: 7, PriorityID: 3}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{ID: 2, TicketID: ticket.ID, ActivityType: NodeAction, Status: ActivityInProgress, NodeID: "action_1"}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	configJSON := fmt.Sprintf(`{"url":%q,"method":"POST","body":"{}","timeout":5,"retries":0}`, server.URL)
	if err := db.Create(&serviceActionModel{
		ID:         3,
		Name:       "Notify",
		Code:       "notify",
		ServiceID:  1,
		IsActive:   true,
		ActionType: "http",
		ConfigJSON: configJSON,
	}).Error; err != nil {
		t.Fatalf("create action: %v", err)
	}

	submitter := &regularRecordingSubmitter{}
	smartEngine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	handler := HandleActionExecute(db, nil, smartEngine)

	payload, _ := json.Marshal(ActionExecutePayload{TicketID: ticket.ID, ActivityID: activity.ID, ActionID: 3})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("HandleActionExecute smart: %v", err)
	}

	var reloaded activityModel
	if err := db.First(&reloaded, activity.ID).Error; err != nil {
		t.Fatalf("reload activity: %v", err)
	}
	if reloaded.Status != ActivityCompleted || reloaded.TransitionOutcome != "success" || reloaded.FinishedAt == nil {
		t.Fatalf("unexpected smart activity after action execution: %+v", reloaded)
	}
	if submitter.calls != 1 || submitter.lastName != "itsm-smart-progress" {
		t.Fatalf("expected one smart progress submission, got calls=%d name=%s", submitter.calls, submitter.lastName)
	}
}

func TestHandleActionExecuteForClassicFailureFallsBackToProgressWithoutBoundary(t *testing.T) {
	f := newClassicMatrixFixture(t)
	if err := f.db.AutoMigrate(&serviceActionModel{}, &actionExecutionModel{}); err != nil {
		t.Fatalf("migrate action execution tables: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"action","type":"action","data":{"label":"执行动作"}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"action","data":{}},
			{"id":"e2","source":"action","target":"end","data":{"outcome":"failed","default":true}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}
	actionActivity := f.firstActivity(t, ticket.ID, NodeAction)
	if err := f.db.Create(&serviceActionModel{
		ID:         9,
		Name:       "Failing action",
		Code:       "failing-action",
		ServiceID:  1,
		IsActive:   true,
		ActionType: "http",
		ConfigJSON: fmt.Sprintf(`{"url":%q,"method":"POST","body":"{}","timeout":5,"retries":0}`, server.URL),
	}).Error; err != nil {
		t.Fatalf("create action config: %v", err)
	}

	handler := HandleActionExecute(f.db, f.engine, nil)
	payload, _ := json.Marshal(ActionExecutePayload{TicketID: ticket.ID, ActivityID: actionActivity.ID, ActionID: 9})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("HandleActionExecute classic failure: %v", err)
	}

	status, outcome := f.ticketStatusOutcome(t, ticket.ID)
	if status != TicketStatusCompleted || outcome != TicketOutcomeFulfilled {
		t.Fatalf("unexpected ticket state after failed action fallback: status=%s outcome=%s", status, outcome)
	}

	var execRow actionExecutionModel
	if err := f.db.First(&execRow).Error; err != nil {
		t.Fatalf("load execution row: %v", err)
	}
	if execRow.Status != "failed" || execRow.FailureReason != "HTTP 500" {
		t.Fatalf("unexpected action execution row: %+v", execRow)
	}

	var refreshed activityModel
	if err := f.db.First(&refreshed, actionActivity.ID).Error; err != nil {
		t.Fatalf("reload action activity: %v", err)
	}
	if refreshed.Status != ActivityCompleted {
		t.Fatalf("expected action activity completed after fallback progress, got %+v", refreshed)
	}
}

func TestHandleActionExecuteTriggersBoundaryErrorPath(t *testing.T) {
	f := newClassicMatrixFixture(t)
	if err := f.db.AutoMigrate(&serviceActionModel{}, &actionExecutionModel{}); err != nil {
		t.Fatalf("migrate action execution tables: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"action","type":"action","data":{"label":"执行动作","action_id":42}},
			{"id":"boundary_error","type":"b_error","data":{"label":"失败转人工","attached_to":"action"}},
			{"id":"manual","type":"process","data":{"label":"人工补救","participants":[{"type":"user","value":"7"}]}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"action","data":{}},
			{"id":"e2","source":"action","target":"end","data":{"outcome":"success"}},
			{"id":"e3","source":"boundary_error","target":"manual","data":{}},
			{"id":"e4","source":"manual","target":"end","data":{"outcome":"completed"}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}
	actionActivity := f.firstActivity(t, ticket.ID, NodeAction)
	if err := f.db.Create(&serviceActionModel{
		ID:         42,
		Name:       "Failing action",
		Code:       "failing-boundary-action",
		ServiceID:  1,
		IsActive:   true,
		ActionType: "http",
		ConfigJSON: fmt.Sprintf(`{"url":%q,"method":"POST","body":"{}","timeout":5,"retries":0}`, server.URL),
	}).Error; err != nil {
		t.Fatalf("create action config: %v", err)
	}

	handler := HandleActionExecute(f.db, f.engine, nil)
	payload, _ := json.Marshal(ActionExecutePayload{TicketID: ticket.ID, ActivityID: actionActivity.ID, ActionID: 42})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("HandleActionExecute with boundary error: %v", err)
	}

	var cancelled activityModel
	if err := f.db.First(&cancelled, actionActivity.ID).Error; err != nil {
		t.Fatalf("reload action activity: %v", err)
	}
	if cancelled.Status != ActivityCancelled {
		t.Fatalf("action activity status = %s, want cancelled", cancelled.Status)
	}

	manual := f.firstActivity(t, ticket.ID, NodeProcess)
	if manual.NodeID != "manual" || manual.Status != ActivityPending {
		t.Fatalf("unexpected boundary recovery activity: %+v", manual)
	}
	if got := f.ticketStatus(t, ticket.ID); got != TicketStatusWaitingHuman {
		t.Fatalf("ticket status after boundary recovery = %q, want %s", got, TicketStatusWaitingHuman)
	}

	var count int64
	if err := f.db.Model(&timelineModel{}).Where("ticket_id = ? AND event_type = ?", ticket.ID, "boundary_error_fired").Count(&count).Error; err != nil {
		t.Fatalf("count boundary_error_fired: %v", err)
	}
	if count != 1 {
		t.Fatalf("boundary_error_fired timeline count = %d, want 1", count)
	}
}

func TestHandleActionExecuteForClassicSuccessProgressesWorkflow(t *testing.T) {
	f := newClassicMatrixFixture(t)
	if err := f.db.AutoMigrate(&serviceActionModel{}, &actionExecutionModel{}); err != nil {
		t.Fatalf("migrate action execution tables: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	workflow := json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"开始"}},
			{"id":"action","type":"action","data":{"label":"执行动作"}},
			{"id":"end","type":"end","data":{"label":"结束"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"action","data":{}},
			{"id":"e2","source":"action","target":"end","data":{"outcome":"success","default":true}}
		]
	}`)
	ticket := f.createTicket(t, workflow)
	if err := f.start(t, ticket, workflow); err != nil {
		t.Fatalf("start: %v", err)
	}
	actionActivity := f.firstActivity(t, ticket.ID, NodeAction)
	if err := f.db.Create(&serviceActionModel{
		ID:         88,
		Name:       "Success action",
		Code:       "success-action",
		ServiceID:  1,
		IsActive:   true,
		ActionType: "http",
		ConfigJSON: fmt.Sprintf(`{"url":%q,"method":"POST","body":"{}","timeout":5,"retries":0}`, server.URL),
	}).Error; err != nil {
		t.Fatalf("create action config: %v", err)
	}

	handler := HandleActionExecute(f.db, f.engine, nil)
	payload, _ := json.Marshal(ActionExecutePayload{TicketID: ticket.ID, ActivityID: actionActivity.ID, ActionID: 88})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("HandleActionExecute classic success: %v", err)
	}

	status, outcome := f.ticketStatusOutcome(t, ticket.ID)
	if status != TicketStatusCompleted || outcome != TicketOutcomeFulfilled {
		t.Fatalf("unexpected ticket state after successful action: status=%s outcome=%s", status, outcome)
	}

	var execRow actionExecutionModel
	if err := f.db.First(&execRow).Error; err != nil {
		t.Fatalf("load execution row: %v", err)
	}
	if execRow.Status != "success" || execRow.FailureReason != "" {
		t.Fatalf("unexpected action execution row: %+v", execRow)
	}

	var refreshed activityModel
	if err := f.db.First(&refreshed, actionActivity.ID).Error; err != nil {
		t.Fatalf("reload action activity: %v", err)
	}
	if refreshed.Status != ActivityCompleted || refreshed.TransitionOutcome != "success" {
		t.Fatalf("expected action activity completed with success, got %+v", refreshed)
	}
}
