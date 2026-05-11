package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestHandleSmartProgressContracts(t *testing.T) {
	t.Run("invalid payload is rejected", func(t *testing.T) {
		handler := HandleSmartProgress(newSmartContinuationDB(t), NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, &recordingSubmitter{}, nil))
		err := handler(context.Background(), json.RawMessage(`{"ticket_id":`))
		if err == nil || !strings.Contains(err.Error(), "invalid payload") {
			t.Fatalf("expected invalid payload error, got %v", err)
		}
	})

	t.Run("ai disabled decision is treated as handled without retry", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		service := serviceModel{Name: "智能服务", EngineType: "smart"}
		if err := db.Create(&service).Error; err != nil {
			t.Fatalf("create service: %v", err)
		}
		ticket := ticketModel{
			ServiceID:       service.ID,
			Status:          TicketStatusDecisioning,
			EngineType:      "smart",
			AIFailureCount:  MaxAIFailureCount,
			RequesterID:     1,
			CurrentActivityID: nil,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		payload, _ := json.Marshal(SmartProgressPayload{TicketID: ticket.ID, TriggerReason: TriggerReasonManualRetry})
		handler := HandleSmartProgress(db, NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, &recordingSubmitter{}, decisionAgentConfigProvider{agentID: 1}))
		if err := handler(context.Background(), payload); err != nil {
			t.Fatalf("expected handled ai disabled error to be swallowed, got %v", err)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_disabled").First(&timeline).Error; err != nil {
			t.Fatalf("expected ai_disabled timeline: %v", err)
		}
	})

	t.Run("non handled runtime errors are propagated", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		payload, _ := json.Marshal(SmartProgressPayload{TicketID: 999, TriggerReason: TriggerReasonInitialDecision})
		handler := HandleSmartProgress(db, NewSmartEngine(availableDecisionExecutor{}, nil, nil, nil, &recordingSubmitter{}, nil))
		err := handler(context.Background(), payload)
		if err == nil || !strings.Contains(err.Error(), "load service") {
			t.Fatalf("expected load service error to propagate, got %v", err)
		}
	})
}

func TestDecisionTriggerReasonHelpers(t *testing.T) {
	if got := decisionTriggerReason(nil); got != TriggerReasonInitialDecision {
		t.Fatalf("decisionTriggerReason(nil) = %q, want %q", got, TriggerReasonInitialDecision)
	}
	completedActivityID := uint(42)
	if got := decisionTriggerReason(&completedActivityID); got != TriggerReasonActivityDone {
		t.Fatalf("decisionTriggerReason(activity) = %q, want %q", got, TriggerReasonActivityDone)
	}
	if got := normalizedTriggerReason(&completedActivityID, "", TriggerReasonRecovery); got != TriggerReasonRecovery {
		t.Fatalf("normalizedTriggerReason explicit = %q, want %q", got, TriggerReasonRecovery)
	}
	if got := normalizedTriggerReason(nil, "", "   "); got != TriggerReasonInitialDecision {
		t.Fatalf("normalizedTriggerReason blank fallback = %q, want %q", got, TriggerReasonInitialDecision)
	}
}

var _ = errors.Is
var _ *gorm.DB
