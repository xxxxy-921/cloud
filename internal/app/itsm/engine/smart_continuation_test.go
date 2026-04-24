package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

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
	if submitter.txCalls != 1 {
		t.Fatalf("expected one transaction submit call, got %d", submitter.txCalls)
	}

	var reloaded ticketModel
	if err := db.First(&reloaded, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloaded.CurrentActivityID != nil {
		t.Fatalf("expected current_activity_id to be cleared while smart-progress is queued, got %d", *reloaded.CurrentActivityID)
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
	if err == nil {
		t.Fatal("expected progress to fail when smart-progress cannot be queued")
	}

	var reloadedActivity activityModel
	if err := db.First(&reloadedActivity, activity.ID).Error; err != nil {
		t.Fatalf("reload activity: %v", err)
	}
	if reloadedActivity.Status != ActivityPending {
		t.Fatalf("activity status should roll back to %q, got %q", ActivityPending, reloadedActivity.Status)
	}

	var reloadedTicket ticketModel
	if err := db.First(&reloadedTicket, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if reloadedTicket.CurrentActivityID == nil || *reloadedTicket.CurrentActivityID != activity.ID {
		t.Fatalf("ticket current_activity_id should remain %d after rollback, got %v", activity.ID, reloadedTicket.CurrentActivityID)
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
	if submitter.txCalls != 1 {
		t.Fatalf("expected one continuation task after convergence, got %d", submitter.txCalls)
	}
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
	if reloaded.Status != "in_progress" {
		t.Fatalf("expected ticket status in_progress, got %q", reloaded.Status)
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
	if err := db.Model(&ticketModel{}).Where("id = ?", ticket.ID).Update("current_activity_id", activity.ID).Error; err != nil {
		t.Fatalf("set current activity: %v", err)
	}
	ticket.CurrentActivityID = &activity.ID
	return ticket, activity
}
