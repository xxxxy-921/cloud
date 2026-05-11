package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	appcore "metis/internal/app"
	"gorm.io/gorm"
)

type scriptedRunDecisionExecutor struct {
	content string
	err     error
	calls   int
}

func (e *scriptedRunDecisionExecutor) Execute(_ context.Context, _ uint, _ appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	e.calls++
	if e.err != nil {
		return nil, e.err
	}
	return &appcore.AIDecisionResponse{Content: e.content}, nil
}

type decisionAgentConfigProvider struct {
	agentID uint
}

func (m decisionAgentConfigProvider) FallbackAssigneeID() uint                  { return 0 }
func (m decisionAgentConfigProvider) DecisionMode() string                      { return "ai_only" }
func (m decisionAgentConfigProvider) DecisionAgentID() uint                     { return m.agentID }
func (m decisionAgentConfigProvider) AuditLevel() string                        { return "full" }
func (m decisionAgentConfigProvider) SLACriticalThresholdSeconds() int          { return 1800 }
func (m decisionAgentConfigProvider) SLAWarningThresholdSeconds() int           { return 3600 }
func (m decisionAgentConfigProvider) SimilarHistoryLimit() int                  { return 5 }
func (m decisionAgentConfigProvider) ParallelConvergenceTimeout() time.Duration { return 72 * time.Hour }

func newSmartRunCycleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newSmartContinuationDB(t)
	for _, stmt := range []string{
		`ALTER TABLE itsm_tickets ADD COLUMN description text`,
		`ALTER TABLE itsm_tickets ADD COLUMN source text`,
		`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
	return db
}

func TestSmartEngineRunDecisionCycleContracts(t *testing.T) {
	t.Run("terminal tickets are skipped before executor runs", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:     TicketStatusCompleted,
			Outcome:    TicketOutcomeFulfilled,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		executor := &scriptedRunDecisionExecutor{content: `{"next_step_type":"complete","confidence":0.99}`}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 7})
		if err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision); err != nil {
			t.Fatalf("runDecisionCycle terminal: %v", err)
		}
		if executor.calls != 0 {
			t.Fatalf("expected executor to be skipped for terminal ticket, got %d calls", executor.calls)
		}

		var timelineCount int64
		if err := db.Table("itsm_ticket_timelines").Where("ticket_id = ?", ticket.ID).Count(&timelineCount).Error; err != nil {
			t.Fatalf("count timelines: %v", err)
		}
		if timelineCount != 0 {
			t.Fatalf("expected terminal skip to avoid timeline writes, got %d", timelineCount)
		}
	})

	t.Run("active activities skip executor until workflow converges", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:     TicketStatusDecisioning,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		activity := activityModel{
			TicketID:     ticket.ID,
			Name:         "等待人工审批",
			ActivityType: NodeApprove,
			Status:       ActivityPending,
		}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}

		executor := &scriptedRunDecisionExecutor{content: `{"next_step_type":"complete","confidence":0.99}`}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 8})
		if err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision); err != nil {
			t.Fatalf("runDecisionCycle active activity: %v", err)
		}
		if executor.calls != 0 {
			t.Fatalf("expected executor to be skipped while active activity exists, got %d calls", executor.calls)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.AIFailureCount != 0 {
			t.Fatalf("expected ai_failure_count unchanged, got %d", reloaded.AIFailureCount)
		}
	})

	t.Run("ai disabled tickets are skipped before executor runs", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:         TicketStatusDecisioning,
			EngineType:     "smart",
			AIFailureCount: MaxAIFailureCount,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		executor := &scriptedRunDecisionExecutor{content: `{"next_step_type":"complete","confidence":0.99}`}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 7})
		err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision)
		if !errors.Is(err, ErrAIDisabled) {
			t.Fatalf("expected ErrAIDisabled, got %v", err)
		}
		if executor.calls != 0 {
			t.Fatalf("expected executor to be skipped, got %d calls", executor.calls)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_disabled").First(&timeline).Error; err != nil {
			t.Fatalf("load ai_disabled timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "AI 决策已停用") {
			t.Fatalf("unexpected ai_disabled message: %q", timeline.Message)
		}
	})

	t.Run("executor failure increments count and records disable on threshold", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:         TicketStatusDecisioning,
			EngineType:     "smart",
			AIFailureCount: MaxAIFailureCount - 1,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		executor := &scriptedRunDecisionExecutor{err: errors.New("gateway down")}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 9})
		err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision)
		if !errors.Is(err, ErrAIDecisionFailed) {
			t.Fatalf("expected ErrAIDecisionFailed, got %v", err)
		}
		if executor.calls != 1 {
			t.Fatalf("expected one executor call, got %d", executor.calls)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.AIFailureCount != MaxAIFailureCount {
			t.Fatalf("expected ai_failure_count=%d, got %d", MaxAIFailureCount, reloaded.AIFailureCount)
		}

		var failed timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_failed").First(&failed).Error; err != nil {
			t.Fatalf("load ai_decision_failed timeline: %v", err)
		}
		if !strings.Contains(failed.Message, "gateway down") {
			t.Fatalf("expected failure reason in timeline, got %q", failed.Message)
		}
		var disabled timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_disabled").First(&disabled).Error; err != nil {
			t.Fatalf("load ai_disabled threshold timeline: %v", err)
		}
	})

	t.Run("low confidence plan resets failures and routes to manual requester handling", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:         TicketStatusDecisioning,
			EngineType:     "smart",
			RequesterID:    44,
			AIFailureCount: 2,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		planJSON, _ := json.Marshal(map[string]any{
			"next_step_type": "process",
			"execution_mode": "single",
			"confidence":     0.42,
			"reasoning":      "当前信息不足，转申请人补充后再继续判断",
			"activities": []map[string]any{
				{
					"type":             "process",
					"participant_type": "requester",
					"instructions":     "请补充缺失上下文",
				},
			},
		})
		executor := &scriptedRunDecisionExecutor{content: string(planJSON)}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 12})
		err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision)
		if err != nil {
			t.Fatalf("runDecisionCycle low confidence: %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.AIFailureCount != 0 {
			t.Fatalf("expected ai_failure_count reset to 0, got %d", reloaded.AIFailureCount)
		}
		if reloaded.Status != TicketStatusWaitingHuman || reloaded.CurrentActivityID == nil {
			t.Fatalf("expected waiting_human with current activity, got status=%q current=%v", reloaded.Status, reloaded.CurrentActivityID)
		}

		var activity activityModel
		if err := db.First(&activity, *reloaded.CurrentActivityID).Error; err != nil {
			t.Fatalf("load pending manual activity: %v", err)
		}
		if activity.Status != ActivityPending || activity.ActivityType != NodeProcess {
			t.Fatalf("expected pending process activity, got status=%q type=%q", activity.Status, activity.ActivityType)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND activity_id = ?", ticket.ID, activity.ID).First(&assignment).Error; err != nil {
			t.Fatalf("load requester assignment: %v", err)
		}
		if assignment.ParticipantType != "requester" || assignment.UserID == nil || *assignment.UserID != 44 {
			t.Fatalf("expected requester assignment for user 44, got %+v", assignment)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_pending").First(&timeline).Error; err != nil {
			t.Fatalf("load ai_decision_pending timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "42%") {
			t.Fatalf("expected low confidence timeline to mention confidence, got %q", timeline.Message)
		}
	})

	t.Run("invalid plan is folded into decision failure contract", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		ticket := ticketModel{
			Status:         TicketStatusDecisioning,
			EngineType:     "smart",
			AIFailureCount: 1,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		planJSON, _ := json.Marshal(map[string]any{
			"next_step_type": "complete",
			"confidence":     0.99,
			"activities": []map[string]any{
				{"type": "process", "participant_type": "requester", "instructions": "这条 complete plan 不应带活动"},
			},
		})
		executor := &scriptedRunDecisionExecutor{content: string(planJSON)}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 13})

		err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision)
		if !errors.Is(err, ErrAIDecisionFailed) {
			t.Fatalf("expected ErrAIDecisionFailed, got %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.AIFailureCount != 2 {
			t.Fatalf("expected ai_failure_count incremented to 2, got %d", reloaded.AIFailureCount)
		}
		if reloaded.Status != TicketStatusDecisioning {
			t.Fatalf("expected ticket to stay decisioning after invalid plan, got %q", reloaded.Status)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_failed").Order("id DESC").First(&timeline).Error; err != nil {
			t.Fatalf("load ai_decision_failed timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "校验失败") {
			t.Fatalf("expected validation failure in timeline, got %q", timeline.Message)
		}
	})

	t.Run("complete plan resets failures and closes ticket", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
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
			ServiceID:      service.ID,
			Status:         TicketStatusDecisioning,
			EngineType:     "smart",
			AIFailureCount: 2,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		planJSON, _ := json.Marshal(map[string]any{
			"next_step_type": "complete",
			"confidence":     0.99,
			"reasoning":      "信息已齐备，直接完成",
		})
		executor := &scriptedRunDecisionExecutor{content: string(planJSON)}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 14})

		if err := engine.runDecisionCycle(context.Background(), db, ticket.ID, nil, &service, TriggerReasonInitialDecision); err != nil {
			t.Fatalf("runDecisionCycle complete plan: %v", err)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.AIFailureCount != 0 {
			t.Fatalf("expected ai_failure_count reset to 0, got %d", reloaded.AIFailureCount)
		}
		if reloaded.Status != TicketStatusCompleted || reloaded.Outcome != TicketOutcomeFulfilled {
			t.Fatalf("expected completed/fulfilled ticket, got status=%q outcome=%q", reloaded.Status, reloaded.Outcome)
		}
		if reloaded.CurrentActivityID == nil {
			t.Fatal("expected current_activity_id to point at generated complete activity")
		}

		var activity activityModel
		if err := db.First(&activity, *reloaded.CurrentActivityID).Error; err != nil {
			t.Fatalf("load completion activity: %v", err)
		}
		if activity.ActivityType != "complete" || activity.Status != ActivityCompleted || activity.NodeID != "end-1" {
			t.Fatalf("expected completed end activity, got %+v", activity)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "workflow_completed").Order("id DESC").First(&timeline).Error; err != nil {
			t.Fatalf("load workflow_completed timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "智能流程已完结") {
			t.Fatalf("unexpected workflow_completed timeline: %q", timeline.Message)
		}
	})
}
