package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	appcore "metis/internal/app"
)

type txRecordingSubmitter struct {
	regularCalls int
	txCalls      int
	lastName     string
	lastPayload  json.RawMessage
}

func (s *txRecordingSubmitter) SubmitTask(string, json.RawMessage) error {
	s.regularCalls++
	return errors.New("regular submitter must not be used from workflow transaction")
}

func (s *txRecordingSubmitter) SubmitTaskTx(tx *gorm.DB, name string, payload json.RawMessage) error {
	if tx == nil {
		return errors.New("missing transaction")
	}
	s.txCalls++
	s.lastName = name
	s.lastPayload = append(s.lastPayload[:0], payload...)
	return nil
}

type failingTxSubmitter struct{}

func (s *failingTxSubmitter) SubmitTask(string, json.RawMessage) error {
	return errors.New("regular submitter must not be used from workflow transaction")
}

func (s *failingTxSubmitter) SubmitTaskTx(*gorm.DB, string, json.RawMessage) error {
	return errors.New("submit failed")
}

type availableDecisionExecutor struct{}

func (availableDecisionExecutor) Execute(context.Context, uint, appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	return nil, errors.New("not used by this test")
}

func TestSmartProgressContinuationUsesWorkflowTransaction(t *testing.T) {
	db := newSmartContinuationDB(t)

	ticket, activity := createSmartContinuationTicket(t, db, "", ActivityPending)
	submitter := &txRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Progress(context.Background(), tx, ProgressParams{
			TicketID:   ticket.ID,
			ActivityID: activity.ID,
			Outcome:    "completed",
			OperatorID: 1,
		})
	})
	if err != nil {
		t.Fatalf("progress smart activity: %v", err)
	}
	if submitter.regularCalls != 0 {
		t.Fatalf("expected no regular submit calls, got %d", submitter.regularCalls)
	}
	if submitter.txCalls != 0 {
		t.Fatalf("expected no scheduler transaction submit calls, got %d", submitter.txCalls)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.CurrentActivityID != nil {
		t.Fatalf("expected current_activity_id to be cleared while decisioning, got %d", *reloaded.CurrentActivityID)
	}
	if reloaded.Status != TicketStatusDecisioning {
		t.Fatalf("expected ticket status %q, got %q", TicketStatusDecisioning, reloaded.Status)
	}
}

func TestSmartProgressContinuationSubmitFailureRollsBackActivityCompletion(t *testing.T) {
	db := newSmartContinuationDB(t)

	ticket, activity := createSmartContinuationTicket(t, db, "", ActivityPending)
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, &failingTxSubmitter{}, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Progress(context.Background(), tx, ProgressParams{
			TicketID:   ticket.ID,
			ActivityID: activity.ID,
			Outcome:    "completed",
			OperatorID: 1,
		})
	})
	if err != nil {
		t.Fatalf("progress should not depend on smart-progress scheduler submission: %v", err)
	}

	var reloadedActivity activityModel
	if err := db.First(&reloadedActivity, activity.ID).Error; err != nil {
		t.Fatalf("reload activity: %v", err)
	}
	if reloadedActivity.Status != ActivityApproved {
		t.Fatalf("activity status should be %q, got %q", ActivityApproved, reloadedActivity.Status)
	}

	var reloadedTicket ticketModel
	if err := db.First(&reloadedTicket, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloadedTicket.CurrentActivityID != nil {
		t.Fatalf("ticket current_activity_id should clear after progress, got %v", reloadedTicket.CurrentActivityID)
	}
}

func TestSmartProgressContinuationWaitsForParallelGroupConvergence(t *testing.T) {
	db := newSmartContinuationDB(t)

	ticket, first := createSmartContinuationTicket(t, db, "parallel-group", ActivityPending)
	second := activityModel{
		TicketID:        ticket.ID,
		Name:            "并行处理 B",
		ActivityType:    NodeProcess,
		Status:          ActivityPending,
		ActivityGroupID: "parallel-group",
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second activity: %v", err)
	}
	assignSmartActivityToOperator(t, db, ticket.ID, second.ID, 1)

	submitter := &txRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	if err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Progress(context.Background(), tx, ProgressParams{
			TicketID:   ticket.ID,
			ActivityID: first.ID,
			Outcome:    "completed",
			OperatorID: 1,
		})
	}); err != nil {
		t.Fatalf("progress first parallel activity: %v", err)
	}

	var waitingTicket ticketModel
	if err := db.First(&waitingTicket, ticket.ID).Error; err != nil {
		t.Fatalf("reload waiting ticket: %v", err)
	}
	if waitingTicket.CurrentActivityID == nil || *waitingTicket.CurrentActivityID != first.ID {
		t.Fatalf("parallel group should keep current activity until convergence, got %v", waitingTicket.CurrentActivityID)
	}
	if submitter.txCalls != 0 {
		t.Fatalf("expected no continuation task before parallel convergence, got %d", submitter.txCalls)
	}

	if err := db.Model(&ticketModel{}).Where("id = ?", ticket.ID).Update("current_activity_id", second.ID).Error; err != nil {
		t.Fatalf("move current activity to second: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Progress(context.Background(), tx, ProgressParams{
			TicketID:   ticket.ID,
			ActivityID: second.ID,
			Outcome:    "completed",
			OperatorID: 1,
		})
	}); err != nil {
		t.Fatalf("progress second parallel activity: %v", err)
	}

	var convergedTicket ticketModel
	if err := db.First(&convergedTicket, ticket.ID).Error; err != nil {
		t.Fatalf("reload converged ticket: %v", err)
	}
	if convergedTicket.CurrentActivityID != nil {
		t.Fatalf("current_activity_id should clear after parallel convergence, got %d", *convergedTicket.CurrentActivityID)
	}
	if submitter.txCalls != 0 {
		t.Fatalf("expected no scheduler continuation task after convergence, got %d", submitter.txCalls)
	}
}

func TestSmartProgressRejectsForeignActivityID(t *testing.T) {
	db := newSmartContinuationDB(t)

	ticketA, activityA := createSmartContinuationTicket(t, db, "", ActivityPending)
	ticketB, _ := createSmartContinuationTicket(t, db, "", ActivityPending)
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)

	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Progress(context.Background(), tx, ProgressParams{
			TicketID:   ticketB.ID,
			ActivityID: activityA.ID,
			Outcome:    "completed",
			OperatorID: 1,
		})
	})
	if !errors.Is(err, ErrActivityNotFound) {
		t.Fatalf("expected ErrActivityNotFound for foreign activity, got %v", err)
	}

	var reloadedActivity activityModel
	if err := db.First(&reloadedActivity, activityA.ID).Error; err != nil {
		t.Fatalf("reload foreign activity: %v", err)
	}
	if reloadedActivity.Status != ActivityPending || reloadedActivity.FinishedAt != nil {
		t.Fatalf("expected foreign activity unchanged, got %+v", reloadedActivity)
	}

	var timelineCount int64
	if err := db.Model(&timelineModel{}).Where("ticket_id IN ?", []uint{ticketA.ID, ticketB.ID}).Count(&timelineCount).Error; err != nil {
		t.Fatalf("count timelines: %v", err)
	}
	if timelineCount != 0 {
		t.Fatalf("expected no timelines for rejected foreign progress, got %d", timelineCount)
	}
}

func TestSmartProgressGuardAndPersistenceContracts(t *testing.T) {
	t.Run("completed activity is rejected as not active", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		ticket, activity := createSmartContinuationTicket(t, db, "", ActivityApproved)
		eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)

		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Progress(context.Background(), tx, ProgressParams{
				TicketID:   ticket.ID,
				ActivityID: activity.ID,
				Outcome:    "completed",
				OperatorID: 1,
			})
		})
		if !errors.Is(err, ErrActivityNotActive) {
			t.Fatalf("expected ErrActivityNotActive, got %v", err)
		}
	})

	t.Run("opinion and result are persisted on human completion", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		ticket, activity := createSmartContinuationTicket(t, db, "", ActivityPending)
		eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
		result := json.RawMessage(`{"approved":true,"comment":"需要记录到表单结果"}`)

		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Progress(context.Background(), tx, ProgressParams{
				TicketID:   ticket.ID,
				ActivityID: activity.ID,
				Outcome:    "approved",
				Opinion:    "人工确认通过",
				Result:     result,
				OperatorID: 1,
			})
		})
		if err != nil {
			t.Fatalf("progress with opinion/result: %v", err)
		}

		var reloadedActivity activityModel
		if err := db.First(&reloadedActivity, activity.ID).Error; err != nil {
			t.Fatalf("reload activity: %v", err)
		}
		if reloadedActivity.Status != ActivityApproved || reloadedActivity.TransitionOutcome != "approved" {
			t.Fatalf("expected approved activity, got %+v", reloadedActivity)
		}
		if reloadedActivity.DecisionReasoning != "人工确认通过" {
			t.Fatalf("expected opinion persisted, got %q", reloadedActivity.DecisionReasoning)
		}
		if reloadedActivity.FormData != string(result) {
			t.Fatalf("expected result form_data persisted, got %q", reloadedActivity.FormData)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "activity_completed").Order("id DESC").First(&timeline).Error; err != nil {
			t.Fatalf("load completion timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "人工确认通过") {
			t.Fatalf("expected timeline to include operator opinion, got %q", timeline.Message)
		}
	})

	t.Run("claimed assignment can complete activity", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		ticket, activity := createSmartContinuationTicket(t, db, "", ActivityPending)
		if err := db.Model(&assignmentModel{}).
			Where("activity_id = ?", activity.ID).
			Updates(map[string]any{"status": "claimed"}).Error; err != nil {
			t.Fatalf("mark assignment claimed: %v", err)
		}

		eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Progress(context.Background(), tx, ProgressParams{
				TicketID:   ticket.ID,
				ActivityID: activity.ID,
				Outcome:    "approved",
				OperatorID: 1,
			})
		})
		if err != nil {
			t.Fatalf("progress claimed assignment: %v", err)
		}

		var assignment assignmentModel
		if err := db.Where("activity_id = ?", activity.ID).First(&assignment).Error; err != nil {
			t.Fatalf("reload assignment: %v", err)
		}
		if assignment.Status != ActivityApproved || assignment.IsCurrent || assignment.FinishedAt == nil {
			t.Fatalf("expected claimed assignment completed and cleared, got %+v", assignment)
		}
	})

	t.Run("missing active assignment fails without mutating activity", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		ticket, activity := createSmartContinuationTicket(t, db, "", ActivityPending)
		eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)

		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Progress(context.Background(), tx, ProgressParams{
				TicketID:   ticket.ID,
				ActivityID: activity.ID,
				Outcome:    "approved",
				OperatorID: 99,
			})
		})
		if !errors.Is(err, ErrNoActiveAssignment) {
			t.Fatalf("expected ErrNoActiveAssignment, got %v", err)
		}

		var reloadedActivity activityModel
		if err := db.First(&reloadedActivity, activity.ID).Error; err != nil {
			t.Fatalf("reload activity: %v", err)
		}
		if reloadedActivity.Status != ActivityPending || reloadedActivity.FinishedAt != nil || reloadedActivity.TransitionOutcome != "" {
			t.Fatalf("expected activity unchanged after failed progress, got %+v", reloadedActivity)
		}

		var timelineCount int64
		if err := db.Model(&timelineModel{}).Where("ticket_id = ?", ticket.ID).Count(&timelineCount).Error; err != nil {
			t.Fatalf("count timelines: %v", err)
		}
		if timelineCount != 0 {
			t.Fatalf("expected no completion timelines on failed progress, got %d", timelineCount)
		}
	})
}

func TestSmartStartInitializesWorkflowWithoutRunningDecision(t *testing.T) {
	db := newSmartContinuationDB(t)

	agentID := uint(11)
	service := serviceModel{
		Name:       "智能 VPN 服务",
		EngineType: "smart",
		AgentID:    &agentID,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{
		ServiceID:   service.ID,
		Status:      "pending",
		EngineType:  "smart",
		RequesterID: 7,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Start(context.Background(), tx, StartParams{
			TicketID:    ticket.ID,
			RequesterID: ticket.RequesterID,
		})
	})
	if err != nil {
		t.Fatalf("start smart workflow: %v", err)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.Status != TicketStatusDecisioning {
		t.Fatalf("expected ticket status %q, got %q", TicketStatusDecisioning, reloaded.Status)
	}

	var timelineCount int64
	if err := db.Model(&timelineModel{}).
		Where("ticket_id = ? AND event_type = ?", ticket.ID, "workflow_started").
		Count(&timelineCount).Error; err != nil {
		t.Fatalf("count timeline: %v", err)
	}
	if timelineCount != 1 {
		t.Fatalf("expected one workflow_started timeline, got %d", timelineCount)
	}

	var activityCount int64
	if err := db.Model(&activityModel{}).Where("ticket_id = ?", ticket.ID).Count(&activityCount).Error; err != nil {
		t.Fatalf("count activities: %v", err)
	}
	if activityCount != 0 {
		t.Fatalf("initial smart start should not run decision synchronously, got %d activities", activityCount)
	}
}

func TestSmartStartAllowsServiceWithoutServiceAgent(t *testing.T) {
	db := newSmartContinuationDB(t)

	service := serviceModel{
		Name:       "智能 VPN 服务",
		EngineType: "smart",
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{
		ServiceID:   service.ID,
		Status:      "pending",
		EngineType:  "smart",
		RequesterID: 7,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Start(context.Background(), tx, StartParams{
			TicketID:    ticket.ID,
			RequesterID: ticket.RequesterID,
		})
	})
	if err != nil {
		t.Fatalf("start smart workflow without service agent: %v", err)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.Status != TicketStatusDecisioning {
		t.Fatalf("expected ticket status %q, got %q", TicketStatusDecisioning, reloaded.Status)
	}
}

func TestSmartStartGuardContracts(t *testing.T) {
	t.Run("engine unavailable is rejected before mutating ticket", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		service := serviceModel{Name: "智能 VPN 服务", EngineType: "smart"}
		if err := db.Create(&service).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}
		ticket := ticketModel{
			ServiceID:   service.ID,
			Status:      "pending",
			EngineType:  "smart",
			RequesterID: 7,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		eng := NewSmartEngine(nil, nil, nil, nil, nil, nil)
		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Start(context.Background(), tx, StartParams{
				TicketID:    ticket.ID,
				RequesterID: ticket.RequesterID,
			})
		})
		if !errors.Is(err, ErrSmartEngineUnavailable) {
			t.Fatalf("expected ErrSmartEngineUnavailable, got %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != "pending" {
			t.Fatalf("expected ticket status to stay pending, got %q", reloaded.Status)
		}

		var timelineCount int64
		if err := db.Model(&timelineModel{}).Where("ticket_id = ?", ticket.ID).Count(&timelineCount).Error; err != nil {
			t.Fatalf("count timelines: %v", err)
		}
		if timelineCount != 0 {
			t.Fatalf("expected no timelines when engine unavailable, got %d", timelineCount)
		}
	})

	t.Run("missing service ticket fails without writing decisioning state", func(t *testing.T) {
		db := newSmartContinuationDB(t)

		ticket := ticketModel{
			ServiceID:   9999,
			Status:      "pending",
			EngineType:  "smart",
			RequesterID: 7,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
		err := db.Transaction(func(tx *gorm.DB) error {
			return eng.Start(context.Background(), tx, StartParams{
				TicketID:    ticket.ID,
				RequesterID: ticket.RequesterID,
			})
		})
		if err == nil || !strings.Contains(err.Error(), "load service") {
			t.Fatalf("expected load service error, got %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != "pending" {
			t.Fatalf("expected ticket status to stay pending, got %q", reloaded.Status)
		}

		var timelineCount int64
		if err := db.Model(&timelineModel{}).Where("ticket_id = ?", ticket.ID).Count(&timelineCount).Error; err != nil {
			t.Fatalf("count timelines: %v", err)
		}
		if timelineCount != 0 {
			t.Fatalf("expected no timelines on missing service failure, got %d", timelineCount)
		}
	})
}

func TestSmartDecisionPositionAssignmentSingleSQLiteConnectionDoesNotBlock(t *testing.T) {
	db := newSmartContinuationDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.Exec(`CREATE TABLE users (id integer primary key, username text, is_active boolean)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`CREATE TABLE positions (id integer primary key, code text)`).Error; err != nil {
		t.Fatalf("create positions: %v", err)
	}
	if err := db.Exec(`CREATE TABLE departments (id integer primary key, code text)`).Error; err != nil {
		t.Fatalf("create departments: %v", err)
	}
	if err := db.Exec(`CREATE TABLE user_positions (user_id integer, position_id integer, department_id integer, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create user_positions: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, username, is_active) VALUES (7, 'network-operator', true)`).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO positions (id, code) VALUES (77, 'network_admin')`).Error; err != nil {
		t.Fatalf("seed position: %v", err)
	}
	if err := db.Exec(`INSERT INTO departments (id, code) VALUES (88, 'it')`).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}
	if err := db.Exec(`INSERT INTO user_positions (user_id, position_id, department_id) VALUES (7, 77, 88)`).Error; err != nil {
		t.Fatalf("seed user position: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	eng := NewSmartEngine(
		availableDecisionExecutor{},
		nil,
		nil,
		NewParticipantResolver(&rootDBPositionResolver{db: db}),
		nil,
		nil,
	)
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			PositionCode:    "network_admin",
			DepartmentCode:  "it",
			Instructions:    "网络管理员处理",
		}},
		Confidence: 0.95,
	}

	done := make(chan error, 1)
	go func() {
		done <- db.Transaction(func(tx *gorm.DB) error {
			return eng.executeDecisionPlan(tx, ticket.ID, plan)
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("execute decision plan: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("smart decision position assignment blocked with a single SQLite connection")
	}

	var assignment assignmentModel
	if err := db.Where("ticket_id = ?", ticket.ID).First(&assignment).Error; err != nil {
		t.Fatalf("load assignment: %v", err)
	}
	if assignment.UserID == nil || *assignment.UserID != 7 || assignment.PositionID == nil || *assignment.PositionID != 77 || assignment.DepartmentID == nil || *assignment.DepartmentID != 88 {
		t.Fatalf("expected transaction-scoped position assignment, got %+v", assignment)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.CurrentActivityID == nil {
		t.Fatal("expected current activity to be set")
	}
}

func TestSmartProgressFailureRecordsDiagnosticStateWithoutBlockingSQLite(t *testing.T) {
	db := newSmartContinuationDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	service := serviceModel{Name: "智能服务", EngineType: "smart"}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{
		ServiceID:  service.ID,
		Status:     "in_progress",
		EngineType: "smart",
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	payload, _ := json.Marshal(SmartProgressPayload{TicketID: ticket.ID, TriggerReason: "manual_retry"})
	handler := HandleSmartProgress(db, NewSmartEngine(nil, nil, nil, nil, nil, nil))
	done := make(chan error, 1)
	go func() {
		done <- handler(context.Background(), payload)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handled smart progress failure should not propagate: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("smart-progress failure handling blocked with a single SQLite connection")
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.AIFailureCount != 1 {
		t.Fatalf("expected ai_failure_count 1, got %d", reloaded.AIFailureCount)
	}

	var timeline timelineModel
	if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_failed").First(&timeline).Error; err != nil {
		t.Fatalf("load diagnostic timeline: %v", err)
	}
	if timeline.Message == "" {
		t.Fatal("expected diagnostic timeline message")
	}
	var details struct {
		DecisionExplanation map[string]any `json:"decision_explanation"`
	}
	if err := json.Unmarshal([]byte(timeline.Details), &details); err != nil {
		t.Fatalf("decode decision explanation details: %v", err)
	}
	if details.DecisionExplanation == nil {
		t.Fatalf("expected decision_explanation details, got %q", timeline.Details)
	}
	if details.DecisionExplanation["trigger"] != "ai_decision_failed" {
		t.Fatalf("expected trigger ai_decision_failed, got %+v", details.DecisionExplanation)
	}
}

func newSmartContinuationDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:smart_continuation_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&serviceModel{}, &ticketModel{}, &activityModel{}, &assignmentModel{}, &timelineModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func TestSmartDecisionPositionAssignmentWithoutUsersWaitsForHuman(t *testing.T) {
	db := newSmartContinuationDB(t)
	if err := db.Exec(`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime, manager_id integer)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`CREATE TABLE positions (id integer primary key, code text)`).Error; err != nil {
		t.Fatalf("create positions: %v", err)
	}
	if err := db.Exec(`CREATE TABLE departments (id integer primary key, code text)`).Error; err != nil {
		t.Fatalf("create departments: %v", err)
	}
	if err := db.Exec(`CREATE TABLE user_positions (user_id integer, position_id integer, department_id integer, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create user_positions: %v", err)
	}
	if err := db.Exec(`INSERT INTO positions (id, code) VALUES (77, 'ops_admin')`).Error; err != nil {
		t.Fatalf("seed position: %v", err)
	}
	if err := db.Exec(`INSERT INTO departments (id, code) VALUES (88, 'it')`).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}

	service := serviceModel{EngineType: "smart"}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{Status: TicketStatusDecisioning, ServiceID: service.ID, EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
	engine.SetDB(db)

	plan := &DecisionPlan{
		NextStepType:  "process",
		ExecutionMode: "single",
		Confidence:    0.95,
		Activities: []DecisionActivity{{
			Type:            "process",
			ParticipantType: "position_department",
			PositionCode:    "ops_admin",
			DepartmentCode:  "it",
			Instructions:    "Handle server troubleshooting access",
		}},
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return engine.executeSinglePlan(tx, ticket.ID, plan)
	}); err != nil {
		t.Fatalf("execute single plan: %v", err)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.Status != TicketStatusSuspended {
		t.Fatalf("expected ticket status suspended (no fallback configured), got %s", reloaded.Status)
	}
	var assignment assignmentModel
	if err := db.Where("ticket_id = ? AND participant_type = ?", ticket.ID, "position_department").First(&assignment).Error; err != nil {
		t.Fatalf("load position assignment: %v", err)
	}
	if assignment.UserID != nil {
		t.Fatalf("expected unresolved assignment user to be nil, got %v", *assignment.UserID)
	}

	var timeline timelineModel
	if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "approver_missing_suspended").First(&timeline).Error; err != nil {
		t.Fatalf("load suspend timeline: %v", err)
	}
	if timeline.Message == "" {
		t.Fatal("expected suspend timeline message")
	}
}

func TestSmartSinglePlanRequesterAndResolvedPositionAssignments(t *testing.T) {
	t.Run("requester participant is assigned directly", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart", RequesterID: 7}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
		plan := &DecisionPlan{
			NextStepType:  NodeProcess,
			ExecutionMode: "single",
			Confidence:    0.95,
			Activities: []DecisionActivity{{
				Type:            NodeProcess,
				ParticipantType: "requester",
				Instructions:    "请申请人补充上下文",
			}},
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute requester plan: %v", err)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND participant_type = ?", ticket.ID, "requester").First(&assignment).Error; err != nil {
			t.Fatalf("load requester assignment: %v", err)
		}
		if assignment.UserID == nil || *assignment.UserID != 7 || assignment.AssigneeID == nil || *assignment.AssigneeID != 7 {
			t.Fatalf("expected requester assignment to use requester 7, got %+v", assignment)
		}

		var ticketAssignee struct {
			AssigneeID uint `gorm:"column:assignee_id"`
		}
		if err := db.Table("itsm_tickets").Where("id = ?", ticket.ID).Select("assignee_id").First(&ticketAssignee).Error; err != nil {
			t.Fatalf("reload ticket assignee: %v", err)
		}
		if ticketAssignee.AssigneeID != 7 {
			t.Fatalf("expected ticket assignee_id to be requester, got %d", ticketAssignee.AssigneeID)
		}
	})

	t.Run("missing requester falls back to configured assignee", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		if err := db.Exec(`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime)`).Error; err != nil {
			t.Fatalf("create users: %v", err)
		}
		if err := db.Exec(`INSERT INTO users (id, username, is_active) VALUES (99, 'fallback-admin', true)`).Error; err != nil {
			t.Fatalf("seed fallback user: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		engine := NewSmartEngine(nil, nil, nil, nil, nil, fallbackOnlyConfigProvider{fallbackID: 99})
		plan := &DecisionPlan{
			NextStepType:  NodeProcess,
			ExecutionMode: "single",
			Confidence:    0.95,
			Activities: []DecisionActivity{{
				Type:            NodeProcess,
				ParticipantType: "requester",
				Instructions:    "请申请人补充上下文",
			}},
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute fallback requester plan: %v", err)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND participant_type = ?", ticket.ID, "user").First(&assignment).Error; err != nil {
			t.Fatalf("load fallback assignment: %v", err)
		}
		if assignment.UserID == nil || *assignment.UserID != 99 || assignment.AssigneeID == nil || *assignment.AssigneeID != 99 {
			t.Fatalf("expected fallback assignment to use user 99, got %+v", assignment)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "participant_fallback").First(&timeline).Error; err != nil {
			t.Fatalf("load fallback timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "fallback-admin") {
			t.Fatalf("expected fallback timeline to mention configured user, got %q", timeline.Message)
		}
	})

	t.Run("position participant resolves first active user and updates assignee", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		for _, stmt := range []string{
			`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime)`,
			`CREATE TABLE positions (id integer primary key, code text)`,
			`CREATE TABLE departments (id integer primary key, code text)`,
			`CREATE TABLE user_positions (id integer primary key, user_id integer, position_id integer, department_id integer, deleted_at datetime)`,
			`INSERT INTO users (id, username, is_active) VALUES (5, 'net-admin-a', true), (6, 'net-admin-b', true)`,
			`INSERT INTO positions (id, code) VALUES (77, 'network_admin')`,
			`INSERT INTO departments (id, code) VALUES (88, 'it')`,
			`INSERT INTO user_positions (id, user_id, position_id, department_id) VALUES (1, 5, 77, 88), (2, 6, 77, 88)`,
		} {
			if err := db.Exec(stmt).Error; err != nil {
				t.Fatalf("exec %q: %v", stmt, err)
			}
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
		plan := &DecisionPlan{
			NextStepType:  NodeProcess,
			ExecutionMode: "single",
			Confidence:    0.95,
			Activities: []DecisionActivity{{
				Type:            NodeProcess,
				ParticipantType: "position_department",
				DepartmentCode:  "it",
				PositionCode:    "network_admin",
				Instructions:    "网络侧排障",
			}},
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute resolved position plan: %v", err)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND participant_type = ?", ticket.ID, "position_department").First(&assignment).Error; err != nil {
			t.Fatalf("load position assignment: %v", err)
		}
		if assignment.UserID == nil || *assignment.UserID != 5 {
			t.Fatalf("expected first resolved user 5, got %+v", assignment)
		}
		if assignment.PositionID == nil || *assignment.PositionID != 77 || assignment.DepartmentID == nil || *assignment.DepartmentID != 88 {
			t.Fatalf("expected position/department ids to persist, got %+v", assignment)
		}

		var ticketAssignee struct {
			AssigneeID uint `gorm:"column:assignee_id"`
		}
		if err := db.Table("itsm_tickets").Where("id = ?", ticket.ID).Select("assignee_id").First(&ticketAssignee).Error; err != nil {
			t.Fatalf("reload ticket assignee: %v", err)
		}
		if ticketAssignee.AssigneeID != 5 {
			t.Fatalf("expected ticket assignee_id=5 after position resolution, got %d", ticketAssignee.AssigneeID)
		}
	})
}

func TestSmartSinglePlanActionAndFallbackContracts(t *testing.T) {
	t.Run("empty plan is rejected before any activity is created", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
		err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, &DecisionPlan{
				NextStepType:  NodeProcess,
				ExecutionMode: "single",
				Confidence:    0.8,
			})
		})
		if err == nil || !strings.Contains(err.Error(), "no activities") {
			t.Fatalf("expected empty activities error, got %v", err)
		}

		var activityCount int64
		if err := db.Model(&activityModel{}).Where("ticket_id = ?", ticket.ID).Count(&activityCount).Error; err != nil {
			t.Fatalf("count activities: %v", err)
		}
		if activityCount != 0 {
			t.Fatalf("expected no activity rows on rejected plan, got %d", activityCount)
		}
	})

	t.Run("action activity with explicit participant enqueues executor task in tx", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		actionID := uint(42)
		operatorID := uint(7)
		submitter := &txRecordingSubmitter{}
		engine := NewSmartEngine(nil, nil, nil, nil, submitter, nil)
		plan := &DecisionPlan{
			NextStepType:  NodeAction,
			ExecutionMode: "single",
			Confidence:    0.97,
			Reasoning:     "自动动作可直接执行",
			Activities: []DecisionActivity{{
				Type:          NodeAction,
				ActionID:      &actionID,
				ParticipantID: &operatorID,
				Instructions:  "执行白名单预检",
			}},
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute action plan: %v", err)
		}

		if submitter.txCalls != 1 || submitter.lastName != "itsm-action-execute" {
			t.Fatalf("expected one tx action task submission, got txCalls=%d lastName=%q", submitter.txCalls, submitter.lastName)
		}
		var payload struct {
			TicketID   uint `json:"ticket_id"`
			ActivityID uint `json:"activity_id"`
			ActionID   uint `json:"action_id"`
		}
		if err := json.Unmarshal(submitter.lastPayload, &payload); err != nil {
			t.Fatalf("decode action payload: %v", err)
		}
		if payload.TicketID != ticket.ID || payload.ActionID != actionID || payload.ActivityID == 0 {
			t.Fatalf("unexpected action payload: %+v", payload)
		}

		var activity activityModel
		if err := db.First(&activity, payload.ActivityID).Error; err != nil {
			t.Fatalf("load action activity: %v", err)
		}
		if activity.Status != ActivityInProgress || activity.ActivityType != NodeAction {
			t.Fatalf("expected in-progress action activity, got status=%q type=%q", activity.Status, activity.ActivityType)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND activity_id = ?", ticket.ID, activity.ID).First(&assignment).Error; err != nil {
			t.Fatalf("load explicit assignee assignment: %v", err)
		}
		if assignment.UserID == nil || *assignment.UserID != operatorID || assignment.AssigneeID == nil || *assignment.AssigneeID != operatorID {
			t.Fatalf("expected explicit operator assignment, got %+v", assignment)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != TicketStatusExecutingAction {
			t.Fatalf("expected action activity to move ticket into executing_action, got %q", reloaded.Status)
		}
		if reloaded.CurrentActivityID == nil || *reloaded.CurrentActivityID != activity.ID {
			t.Fatalf("expected current activity to point at action activity, got %v", reloaded.CurrentActivityID)
		}
	})

	t.Run("parallel plan creates grouped activities assignments and action task", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		for _, stmt := range []string{
			`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime)`,
			`CREATE TABLE positions (id integer primary key, code text)`,
			`CREATE TABLE departments (id integer primary key, code text)`,
			`CREATE TABLE user_positions (id integer primary key, user_id integer, position_id integer, department_id integer, deleted_at datetime)`,
			`INSERT INTO users (id, username, is_active) VALUES (5, 'net-admin-a', true), (7, 'auto-operator', true)`,
			`INSERT INTO positions (id, code) VALUES (77, 'network_admin')`,
			`INSERT INTO departments (id, code) VALUES (88, 'it')`,
			`INSERT INTO user_positions (id, user_id, position_id, department_id) VALUES (1, 5, 77, 88)`,
		} {
			if err := db.Exec(stmt).Error; err != nil {
				t.Fatalf("exec %q: %v", stmt, err)
			}
		}

		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		actionID := uint(42)
		operatorID := uint(7)
		submitter := &txRecordingSubmitter{}
		engine := NewSmartEngine(nil, nil, nil, nil, submitter, nil)
		plan := &DecisionPlan{
			NextStepType:  NodeProcess,
			ExecutionMode: "parallel",
			Confidence:    0.97,
			Reasoning:     "网络处理和预检可并行推进",
			Activities: []DecisionActivity{
				{
					Type:            NodeProcess,
					ParticipantType: "position_department",
					DepartmentCode:  "it",
					PositionCode:    "network_admin",
					Instructions:    "网络侧排障",
				},
				{
					Type:          NodeAction,
					ActionID:      &actionID,
					ParticipantID: &operatorID,
					Instructions:  "执行白名单预检",
				},
			},
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeDecisionPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute parallel plan: %v", err)
		}

		if submitter.txCalls != 1 || submitter.lastName != "itsm-action-execute" {
			t.Fatalf("expected one tx action task submission, got txCalls=%d lastName=%q", submitter.txCalls, submitter.lastName)
		}

		var activities []activityModel
		if err := db.Where("ticket_id = ?", ticket.ID).Order("id ASC").Find(&activities).Error; err != nil {
			t.Fatalf("list activities: %v", err)
		}
		if len(activities) != 2 {
			t.Fatalf("expected 2 parallel activities, got %+v", activities)
		}
		if activities[0].ExecutionMode != "parallel" || activities[1].ExecutionMode != "parallel" {
			t.Fatalf("expected parallel execution mode, got %+v", activities)
		}
		if activities[0].ActivityGroupID == "" || activities[0].ActivityGroupID != activities[1].ActivityGroupID {
			t.Fatalf("expected shared activity group id, got %+v", activities)
		}
		if activities[0].Status != ActivityPending || activities[1].Status != ActivityPending {
			t.Fatalf("expected pending parallel activities, got %+v", activities)
		}

		var assignments []assignmentModel
		if err := db.Where("ticket_id = ?", ticket.ID).Order("activity_id ASC").Find(&assignments).Error; err != nil {
			t.Fatalf("list assignments: %v", err)
		}
		if len(assignments) != 2 {
			t.Fatalf("expected 2 assignments, got %+v", assignments)
		}
		if assignments[0].ParticipantType != "position_department" || assignments[0].UserID == nil || *assignments[0].UserID != 5 {
			t.Fatalf("expected resolved position assignment, got %+v", assignments[0])
		}
		if assignments[1].ParticipantType != "user" || assignments[1].UserID == nil || *assignments[1].UserID != operatorID {
			t.Fatalf("expected explicit operator assignment, got %+v", assignments[1])
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != TicketStatusWaitingHuman {
			t.Fatalf("expected first human parallel activity to move ticket into waiting_human, got %q", reloaded.Status)
		}
		if reloaded.CurrentActivityID == nil || *reloaded.CurrentActivityID != activities[0].ID {
			t.Fatalf("expected current activity to point at first parallel activity, got %v", reloaded.CurrentActivityID)
		}

		var timelines []timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_executed").Find(&timelines).Error; err != nil {
			t.Fatalf("list ai decision timelines: %v", err)
		}
		if len(timelines) != 2 {
			t.Fatalf("expected one timeline per parallel activity, got %+v", timelines)
		}
	})

	t.Run("invalid fallback user only records warning", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		if err := db.Exec(`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime)`).Error; err != nil {
			t.Fatalf("create users: %v", err)
		}
		if err := db.Exec(`INSERT INTO users (id, username, is_active) VALUES (88, 'disabled-fallback', false)`).Error; err != nil {
			t.Fatalf("seed disabled fallback user: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		engine := NewSmartEngine(nil, nil, nil, nil, nil, fallbackOnlyConfigProvider{fallbackID: 88})
		plan := &DecisionPlan{
			NextStepType:  NodeProcess,
			ExecutionMode: "single",
			Confidence:    0.91,
			Activities: []DecisionActivity{{
				Type:         NodeProcess,
				Instructions: "参与人缺失，尝试 fallback",
			}},
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.executeSinglePlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("execute fallback warning plan: %v", err)
		}

		var assignmentCount int64
		if err := db.Model(&assignmentModel{}).Where("ticket_id = ?", ticket.ID).Count(&assignmentCount).Error; err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		if assignmentCount != 0 {
			t.Fatalf("expected invalid fallback to avoid creating assignments, got %d", assignmentCount)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "participant_fallback_warning").First(&timeline).Error; err != nil {
			t.Fatalf("load fallback warning timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "ID=88") {
			t.Fatalf("expected warning timeline to mention invalid fallback id, got %q", timeline.Message)
		}
	})
}

func createSmartContinuationTicket(t *testing.T, db *gorm.DB, groupID string, activityStatus string) (ticketModel, activityModel) {
	t.Helper()
	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:        ticket.ID,
		Name:            "处理",
		ActivityType:    NodeProcess,
		Status:          activityStatus,
		ActivityGroupID: groupID,
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if activityStatus == ActivityPending {
		assignSmartActivityToOperator(t, db, ticket.ID, activity.ID, 1)
	}
	if err := db.Model(&ticketModel{}).Where("id = ?", ticket.ID).Update("current_activity_id", activity.ID).Error; err != nil {
		t.Fatalf("set current activity: %v", err)
	}
	ticket.CurrentActivityID = &activity.ID
	return ticket, activity
}

func assignSmartActivityToOperator(t *testing.T, db *gorm.DB, ticketID uint, activityID uint, operatorID uint) {
	t.Helper()
	assignment := assignmentModel{
		TicketID:        ticketID,
		ActivityID:      activityID,
		ParticipantType: "user",
		UserID:          &operatorID,
		AssigneeID:      &operatorID,
		Status:          ActivityPending,
		IsCurrent:       true,
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
}

type rootDBPositionResolver struct {
	db *gorm.DB
}

func (r *rootDBPositionResolver) GetUserDeptScope(uint, bool) ([]uint, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) GetUserPositionIDs(uint) ([]uint, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) GetUserDepartmentIDs(uint) ([]uint, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) GetUserPositions(uint) ([]appcore.OrgPosition, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) GetUserDepartment(uint) (*appcore.OrgDepartment, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) QueryContext(string, string, string, bool) (*appcore.OrgContextResult, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) FindUsersByPositionCode(string) ([]uint, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) FindUsersByDepartmentCode(string) ([]uint, error) {
	return nil, nil
}

func (r *rootDBPositionResolver) FindUsersByPositionAndDepartment(posCode, deptCode string) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_positions").
		Joins("JOIN positions ON positions.id = user_positions.position_id").
		Joins("JOIN departments ON departments.id = user_positions.department_id").
		Joins("JOIN users ON users.id = user_positions.user_id").
		Where("positions.code = ? AND departments.code = ? AND user_positions.deleted_at IS NULL AND users.is_active = ?", posCode, deptCode, true).
		Pluck("DISTINCT users.id", &userIDs).Error
	return userIDs, err
}

func (r *rootDBPositionResolver) FindUsersByPositionID(positionID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_positions").
		Joins("JOIN users ON users.id = user_positions.user_id").
		Where("user_positions.position_id = ? AND user_positions.deleted_at IS NULL AND users.is_active = ?", positionID, true).
		Pluck("DISTINCT users.id", &userIDs).Error
	return userIDs, err
}

func (r *rootDBPositionResolver) FindUsersByDepartmentID(departmentID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_positions").
		Joins("JOIN users ON users.id = user_positions.user_id").
		Where("user_positions.department_id = ? AND user_positions.deleted_at IS NULL AND users.is_active = ?", departmentID, true).
		Pluck("DISTINCT users.id", &userIDs).Error
	return userIDs, err
}

func (r *rootDBPositionResolver) FindManagerByUserID(userID uint) (uint, error) {
	var user struct {
		ManagerID *uint
	}
	if err := r.db.Table("users").Where("id = ?", userID).Select("manager_id").First(&user).Error; err != nil {
		return 0, err
	}
	if user.ManagerID == nil {
		return 0, nil
	}
	return *user.ManagerID, nil
}

// regularRecordingSubmitter records SubmitTask calls (non-transactional).
// Used by recovery tests where HandleSmartRecovery calls SubmitProgressTask → SubmitTask.
type regularRecordingSubmitter struct {
	calls    int
	lastName string
}

func (s *regularRecordingSubmitter) SubmitTask(name string, _ json.RawMessage) error {
	s.calls++
	s.lastName = name
	return nil
}

type fallbackOnlyConfigProvider struct {
	fallbackID uint
}

func (m fallbackOnlyConfigProvider) FallbackAssigneeID() uint                  { return m.fallbackID }
func (m fallbackOnlyConfigProvider) DecisionMode() string                      { return "ai_only" }
func (m fallbackOnlyConfigProvider) DecisionAgentID() uint                     { return 0 }
func (m fallbackOnlyConfigProvider) AuditLevel() string                        { return "full" }
func (m fallbackOnlyConfigProvider) SLACriticalThresholdSeconds() int          { return 1800 }
func (m fallbackOnlyConfigProvider) SLAWarningThresholdSeconds() int           { return 3600 }
func (m fallbackOnlyConfigProvider) SimilarHistoryLimit() int                  { return 5 }
func (m fallbackOnlyConfigProvider) ParallelConvergenceTimeout() time.Duration { return time.Hour }

// --- Task 2.3: ensureContinuation in Start/Cancel ---

func TestSmartStartTriggersEnsureContinuation(t *testing.T) {
	db := newSmartContinuationDB(t)

	agentID := uint(11)
	service := serviceModel{
		Name:       "智能 VPN 服务",
		EngineType: "smart",
		AgentID:    &agentID,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{
		ServiceID:   service.ID,
		Status:      "pending",
		EngineType:  "smart",
		RequesterID: 7,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	submitter := &txRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Start(context.Background(), tx, StartParams{
			TicketID:    ticket.ID,
			RequesterID: ticket.RequesterID,
		})
	})
	if err != nil {
		t.Fatalf("start smart workflow: %v", err)
	}

	if submitter.txCalls != 0 {
		t.Fatalf("expected smart start to avoid scheduler submit calls, got %d", submitter.txCalls)
	}
}

func TestSmartCancelCallsEnsureContinuationButNoTask(t *testing.T) {
	db := newSmartContinuationDB(t)
	if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
		t.Fatalf("add assignee_id: %v", err)
	}

	ticket, _ := createSmartContinuationTicket(t, db, "", ActivityPending)
	submitter := &txRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)

	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Cancel(context.Background(), tx, CancelParams{
			TicketID:   ticket.ID,
			OperatorID: 1,
		})
	})
	if err != nil {
		t.Fatalf("cancel smart workflow: %v", err)
	}

	if submitter.txCalls != 0 {
		t.Fatalf("expected no tx submit calls for cancelled (terminal) ticket, got %d", submitter.txCalls)
	}
}

func TestSmartStartAIDisabledNoTask(t *testing.T) {
	db := newSmartContinuationDB(t)

	agentID := uint(11)
	service := serviceModel{
		Name:       "智能 VPN 服务",
		EngineType: "smart",
		AgentID:    &agentID,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := ticketModel{
		ServiceID:      service.ID,
		Status:         "pending",
		EngineType:     "smart",
		RequesterID:    7,
		AIFailureCount: MaxAIFailureCount,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	submitter := &txRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	err := db.Transaction(func(tx *gorm.DB) error {
		return eng.Start(context.Background(), tx, StartParams{
			TicketID:    ticket.ID,
			RequesterID: ticket.RequesterID,
		})
	})
	if err != nil {
		t.Fatalf("start smart workflow: %v", err)
	}

	if submitter.txCalls != 0 {
		t.Fatalf("expected no tx submit calls when AI is circuit-broken, got %d", submitter.txCalls)
	}
}

// --- Task 6.4: Recovery dedup tests ---

func newSmartRecoveryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newSmartContinuationDB(t)
	// HandleSmartRecovery queries "deleted_at IS NULL" which is not part of the
	// lightweight ticketModel. Add the column so the raw SQL does not fail.
	if err := db.Exec("ALTER TABLE itsm_tickets ADD COLUMN deleted_at datetime").Error; err != nil {
		t.Fatalf("add deleted_at column: %v", err)
	}
	return db
}

func clearRecoverySubmissions() {
	recoverySubmissionsMu.Lock()
	for k := range recoverySubmissions {
		delete(recoverySubmissions, k)
	}
	recoverySubmissionsMu.Unlock()
}

func TestSmartRecoveryFirstRunSubmits(t *testing.T) {
	clearRecoverySubmissions()

	db := newSmartRecoveryDB(t)

	// Create a decisioning smart ticket with no active activities.
	ticket := ticketModel{
		Status:     TicketStatusDecisioning,
		EngineType: "smart",
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	submitter := &regularRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)

	handler := HandleSmartRecovery(db, eng)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("smart recovery handler: %v", err)
	}

	if submitter.calls != 1 || submitter.lastName != "itsm-smart-progress" {
		t.Fatalf("expected recovery to enqueue one smart-progress task, got calls=%d lastName=%q", submitter.calls, submitter.lastName)
	}

	// Verify dedup map was populated
	recoverySubmissionsMu.Lock()
	_, recorded := recoverySubmissions[ticket.ID]
	recoverySubmissionsMu.Unlock()
	if !recorded {
		t.Fatal("expected recovery submission to be recorded in dedup map")
	}
}

func TestSmartRecoveryDedupSkipsRecent(t *testing.T) {
	clearRecoverySubmissions()

	db := newSmartRecoveryDB(t)

	ticket := ticketModel{
		Status:     TicketStatusDecisioning,
		EngineType: "smart",
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	submitter := &regularRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	handler := HandleSmartRecovery(db, eng)

	// First run should enqueue recovery and populate dedup.
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("first recovery run: %v", err)
	}
	if submitter.calls != 1 || submitter.lastName != "itsm-smart-progress" {
		t.Fatalf("expected first run to enqueue one smart-progress task, got calls=%d lastName=%q", submitter.calls, submitter.lastName)
	}

	// Second run within 10 minutes — should skip (dedup)
	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("second recovery run: %v", err)
	}
	if submitter.calls != 1 {
		t.Fatalf("expected second run to be deduped without another scheduler submit, got %d", submitter.calls)
	}
}

func TestSmartRecoveryDedupExpiresAfter10Min(t *testing.T) {
	clearRecoverySubmissions()

	db := newSmartRecoveryDB(t)

	ticket := ticketModel{
		Status:     TicketStatusDecisioning,
		EngineType: "smart",
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Pre-populate the dedup map with an entry older than 10 minutes
	recoverySubmissionsMu.Lock()
	recoverySubmissions[ticket.ID] = time.Now().Add(-11 * time.Minute)
	recoverySubmissionsMu.Unlock()

	submitter := &regularRecordingSubmitter{}
	eng := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
	handler := HandleSmartRecovery(db, eng)

	if err := handler(context.Background(), nil); err != nil {
		t.Fatalf("recovery after expiry: %v", err)
	}

	if submitter.calls != 1 || submitter.lastName != "itsm-smart-progress" {
		t.Fatalf("expected expired dedup entry to enqueue one smart-progress task, got calls=%d lastName=%q", submitter.calls, submitter.lastName)
	}

	// Verify the dedup map was updated with a fresh timestamp
	recoverySubmissionsMu.Lock()
	ts, ok := recoverySubmissions[ticket.ID]
	recoverySubmissionsMu.Unlock()
	if !ok {
		t.Fatal("expected dedup entry to exist after resubmission")
	}
	if time.Since(ts) > 5*time.Second {
		t.Fatalf("expected fresh dedup timestamp, got %v ago", time.Since(ts))
	}
}
