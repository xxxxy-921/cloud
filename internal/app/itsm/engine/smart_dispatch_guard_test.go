package engine

import (
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestSmartEngineDispatchGuardsAndProgressSubmission(t *testing.T) {
	t.Run("CanDispatchDecisionTask rejects missing executor and scheduler", func(t *testing.T) {
		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)
		if err := engine.CanDispatchDecisionTask(); !errors.Is(err, ErrSmartEngineUnavailable) {
			t.Fatalf("expected ErrSmartEngineUnavailable, got %v", err)
		}

		engine = NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
		if err := engine.CanDispatchDecisionTask(); !errors.Is(err, ErrSmartTaskSchedulerUnavailable) {
			t.Fatalf("expected ErrSmartTaskSchedulerUnavailable, got %v", err)
		}

		engine = NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, &recordingSubmitter{}, nil)
		if err := engine.CanDispatchDecisionTask(); err != nil {
			t.Fatalf("expected scheduler-ready engine to dispatch, got %v", err)
		}
	})

	t.Run("SubmitProgressTask uses regular scheduler path", func(t *testing.T) {
		submitter := &recordingSubmitter{}
		engine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
		payload := json.RawMessage(`{"ticket_id":42,"trigger_reason":"manual_retry"}`)
		if err := engine.SubmitProgressTask(payload); err != nil {
			t.Fatalf("submit progress task: %v", err)
		}
		if len(submitter.tasks) != 1 {
			t.Fatalf("expected one scheduler task, got %d", len(submitter.tasks))
		}
		if submitter.tasks[0].name != "itsm-smart-progress" {
			t.Fatalf("expected itsm-smart-progress task, got %q", submitter.tasks[0].name)
		}

		var decoded SmartProgressPayload
		if err := json.Unmarshal(submitter.tasks[0].payload, &decoded); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if decoded.TicketID != 42 || decoded.TriggerReason != "manual_retry" {
			t.Fatalf("unexpected payload: %+v", decoded)
		}
	})

	t.Run("SubmitProgressTask rejects missing scheduler even when executor exists", func(t *testing.T) {
		engine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, nil, nil)
		if err := engine.SubmitProgressTask(json.RawMessage(`{"ticket_id":1}`)); !errors.Is(err, ErrSmartTaskSchedulerUnavailable) {
			t.Fatalf("expected ErrSmartTaskSchedulerUnavailable, got %v", err)
		}
	})

	t.Run("SubmitProgressTaskTx uses transactional scheduler path and preserves payload fields", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		submitter := &txRecordingSubmitter{}
		engine := NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, submitter, nil)
		completedActivityID := uint(77)

		err := db.Transaction(func(tx *gorm.DB) error {
			return engine.SubmitDecisionTaskTx(tx, 41, &completedActivityID, TriggerReasonManualRetry)
		})
		if err != nil {
			t.Fatalf("submit decision task tx: %v", err)
		}
		if submitter.regularCalls != 0 {
			t.Fatalf("expected no regular scheduler calls, got %d", submitter.regularCalls)
		}
		if submitter.txCalls != 1 {
			t.Fatalf("expected one transactional scheduler call, got %d", submitter.txCalls)
		}
		if submitter.lastName != "itsm-smart-progress" {
			t.Fatalf("expected itsm-smart-progress task, got %q", submitter.lastName)
		}

		var decoded SmartProgressPayload
		if err := json.Unmarshal(submitter.lastPayload, &decoded); err != nil {
			t.Fatalf("unmarshal tx payload: %v", err)
		}
		if decoded.TicketID != 41 {
			t.Fatalf("expected ticket_id=41, got %d", decoded.TicketID)
		}
		if decoded.CompletedActivityID == nil || *decoded.CompletedActivityID != completedActivityID {
			t.Fatalf("expected completed_activity_id=%d, got %+v", completedActivityID, decoded.CompletedActivityID)
		}
		if decoded.TriggerReason != TriggerReasonManualRetry {
			t.Fatalf("expected trigger reason %q, got %q", TriggerReasonManualRetry, decoded.TriggerReason)
		}
	})
}
