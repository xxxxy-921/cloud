package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// ActionExecutePayload is the async task payload for itsm-action-execute.
type ActionExecutePayload struct {
	TicketID   uint `json:"ticket_id"`
	ActivityID uint `json:"activity_id"`
	ActionID   uint `json:"action_id"`
}

// WaitTimerPayload is the async task payload for itsm-wait-timer.
type WaitTimerPayload struct {
	TicketID     uint   `json:"ticket_id"`
	ActivityID   uint   `json:"activity_id"`
	ExecuteAfter string `json:"execute_after"` // RFC3339
}

// HandleActionExecute is the scheduler task handler for itsm-action-execute.
// It executes the HTTP webhook and then calls Progress on the engine.
func HandleActionExecute(db *gorm.DB, classicEngine *ClassicEngine) func(ctx context.Context, payload json.RawMessage) error {
	executor := NewActionExecutor(db)

	return func(ctx context.Context, payload json.RawMessage) error {
		var p ActionExecutePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("invalid payload: %w", err)
		}

		slog.Info("executing action", "ticketID", p.TicketID, "activityID", p.ActivityID, "actionID", p.ActionID)

		err := executor.Execute(ctx, p.TicketID, p.ActivityID, p.ActionID)

		outcome := "success"
		if err != nil {
			outcome = "failed"
			slog.Error("action execution failed", "error", err, "ticketID", p.TicketID, "actionID", p.ActionID)
		}

		// Progress the workflow with the outcome
		if progressErr := db.Transaction(func(tx *gorm.DB) error {
			return classicEngine.Progress(ctx, tx, ProgressParams{
				TicketID:   p.TicketID,
				ActivityID: p.ActivityID,
				Outcome:    outcome,
				OperatorID: 0, // system
			})
		}); progressErr != nil {
			slog.Error("failed to progress after action", "error", progressErr, "ticketID", p.TicketID)
			return progressErr
		}

		return nil
	}
}

// HandleWaitTimer is the scheduler task handler for itsm-wait-timer.
// It checks if the execute_after time has been reached and triggers Progress.
func HandleWaitTimer(db *gorm.DB, classicEngine *ClassicEngine) func(ctx context.Context, payload json.RawMessage) error {
	return func(ctx context.Context, payload json.RawMessage) error {
		var p WaitTimerPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("invalid payload: %w", err)
		}

		executeAfter, err := time.Parse(time.RFC3339, p.ExecuteAfter)
		if err != nil {
			return fmt.Errorf("invalid execute_after time: %w", err)
		}

		// Not yet time — skip without error (will be retried on next poll)
		if time.Now().Before(executeAfter) {
			return nil
		}

		slog.Info("wait timer expired", "ticketID", p.TicketID, "activityID", p.ActivityID)

		// Verify the activity is still active
		var activity activityModel
		if err := db.First(&activity, p.ActivityID).Error; err != nil {
			return nil // activity gone, skip
		}
		if activity.Status != ActivityPending && activity.Status != ActivityInProgress {
			return nil // already handled
		}

		return db.Transaction(func(tx *gorm.DB) error {
			return classicEngine.Progress(ctx, tx, ProgressParams{
				TicketID:   p.TicketID,
				ActivityID: p.ActivityID,
				Outcome:    "timeout",
				OperatorID: 0, // system
			})
		})
	}
}
