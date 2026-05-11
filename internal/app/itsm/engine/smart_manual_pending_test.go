package engine

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestSmartEnginePendManualHandlingPlanContracts(t *testing.T) {
	t.Run("low-confidence plan without activities still creates manual pending gate", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		plan := &DecisionPlan{
			NextStepType: "process",
			Reasoning:    "当前证据不足，需要人工接管",
			Confidence:   0.41,
		}
		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.pendManualHandlingPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("pend manual handling plan: %v", err)
		}

		var activity activityModel
		if err := db.Where("ticket_id = ?", ticket.ID).First(&activity).Error; err != nil {
			t.Fatalf("load pending activity: %v", err)
		}
		if activity.ActivityType != NodeProcess || activity.Status != ActivityPending {
			t.Fatalf("expected pending process activity, got type=%q status=%q", activity.ActivityType, activity.Status)
		}
		if activity.Name != "AI 低置信待处置" {
			t.Fatalf("expected generic pending activity name, got %q", activity.Name)
		}
		if activity.AIConfidence != plan.Confidence || !strings.Contains(activity.AIReasoning, "人工接管") {
			t.Fatalf("expected plan reasoning/confidence to persist, got confidence=%v reasoning=%q", activity.AIConfidence, activity.AIReasoning)
		}

		var reloaded ticketModel
		if err := db.First(&reloaded, ticket.ID).Error; err != nil {
			t.Fatalf("reload ticket: %v", err)
		}
		if reloaded.Status != TicketStatusWaitingHuman {
			t.Fatalf("expected waiting_human status, got %q", reloaded.Status)
		}
		if reloaded.CurrentActivityID == nil || *reloaded.CurrentActivityID != activity.ID {
			t.Fatalf("expected current activity to point at manual gate, got %v", reloaded.CurrentActivityID)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "ai_decision_pending").First(&timeline).Error; err != nil {
			t.Fatalf("load pending timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "41%") {
			t.Fatalf("expected pending timeline to mention confidence, got %q", timeline.Message)
		}

		var assignmentCount int64
		if err := db.Model(&assignmentModel{}).Where("ticket_id = ?", ticket.ID).Count(&assignmentCount).Error; err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		if assignmentCount != 0 {
			t.Fatalf("expected no assignment when low-confidence plan has no participant, got %d", assignmentCount)
		}
	})

	t.Run("low-confidence requester plan preserves requester assignment", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart", RequesterID: 23}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		plan := &DecisionPlan{
			NextStepType: "form",
			Reasoning:    "需要申请人补充字段后再做下一轮判断",
			Confidence:   0.38,
			Activities: []DecisionActivity{{
				Type:            NodeForm,
				NodeID:          "fill-form-1",
				ParticipantType: "requester",
				Instructions:    "补充访问目的和影响范围",
			}},
		}
		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.pendManualHandlingPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("pend requester manual plan: %v", err)
		}

		var activity activityModel
		if err := db.Where("ticket_id = ?", ticket.ID).First(&activity).Error; err != nil {
			t.Fatalf("load pending activity: %v", err)
		}
		if activity.Name != "AI 低置信待处置：表单填写" || activity.NodeID != "fill-form-1" {
			t.Fatalf("expected requester pending activity name/node to persist, got name=%q node=%q", activity.Name, activity.NodeID)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND activity_id = ?", ticket.ID, activity.ID).First(&assignment).Error; err != nil {
			t.Fatalf("load requester assignment: %v", err)
		}
		if assignment.ParticipantType != "requester" || assignment.UserID == nil || *assignment.UserID != 23 {
			t.Fatalf("expected requester assignment for user 23, got %+v", assignment)
		}

		var ticketAssignee struct {
			AssigneeID uint `gorm:"column:assignee_id"`
		}
		if err := db.Table("itsm_tickets").Where("id = ?", ticket.ID).Select("assignee_id").First(&ticketAssignee).Error; err != nil {
			t.Fatalf("reload assignee_id: %v", err)
		}
		if ticketAssignee.AssigneeID != 23 {
			t.Fatalf("expected assignee_id to follow requester, got %d", ticketAssignee.AssigneeID)
		}
	})

	t.Run("low-confidence explicit participant plan assigns target user directly", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		assigneeID := uint(88)
		plan := &DecisionPlan{
			NextStepType: "process",
			Reasoning:    "低置信转指定人工处理",
			Confidence:   0.35,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				NodeID:        "manual-review",
				ParticipantID: &assigneeID,
				Instructions:  "请指定人工复核",
			}},
		}
		engine := NewSmartEngine(nil, nil, nil, nil, nil, nil)

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.pendManualHandlingPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("pend explicit participant plan: %v", err)
		}

		var activity activityModel
		if err := db.Where("ticket_id = ?", ticket.ID).First(&activity).Error; err != nil {
			t.Fatalf("load pending activity: %v", err)
		}
		if activity.Name != "AI 低置信待处置：处理" || activity.NodeID != "manual-review" {
			t.Fatalf("expected explicit participant activity naming/node, got name=%q node=%q", activity.Name, activity.NodeID)
		}

		var assignment assignmentModel
		if err := db.Where("ticket_id = ? AND activity_id = ?", ticket.ID, activity.ID).First(&assignment).Error; err != nil {
			t.Fatalf("load explicit assignment: %v", err)
		}
		if assignment.ParticipantType != "user" || assignment.UserID == nil || *assignment.UserID != assigneeID || assignment.AssigneeID == nil || *assignment.AssigneeID != assigneeID {
			t.Fatalf("expected explicit participant assignment, got %+v", assignment)
		}
	})

	t.Run("fallback warning does not fabricate assignment when fallback user is invalid", func(t *testing.T) {
		db := newSmartContinuationDB(t)
		if err := db.Exec(`ALTER TABLE itsm_tickets ADD COLUMN assignee_id integer`).Error; err != nil {
			t.Fatalf("add assignee_id: %v", err)
		}
		if err := db.Exec(`CREATE TABLE users (id integer primary key, username text, is_active boolean, deleted_at datetime)`).Error; err != nil {
			t.Fatalf("create users: %v", err)
		}
		if err := db.Exec(`INSERT INTO users (id, username, is_active) VALUES (99, 'fallback-user', false)`).Error; err != nil {
			t.Fatalf("seed inactive fallback user: %v", err)
		}

		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		plan := &DecisionPlan{
			NextStepType: "process",
			Reasoning:    "低置信但参与人缺失",
			Confidence:   0.27,
			Activities: []DecisionActivity{{
				Type:            NodeProcess,
				ParticipantType: "department",
				Instructions:    "需要人工接管",
			}},
		}
		engine := NewSmartEngine(nil, nil, nil, nil, nil, fallbackOnlyConfigProvider{fallbackID: 99})

		if err := db.Transaction(func(tx *gorm.DB) error {
			return engine.pendManualHandlingPlan(tx, ticket.ID, plan)
		}); err != nil {
			t.Fatalf("pend fallback warning plan: %v", err)
		}

		var assignmentCount int64
		if err := db.Model(&assignmentModel{}).Where("ticket_id = ?", ticket.ID).Count(&assignmentCount).Error; err != nil {
			t.Fatalf("count assignments: %v", err)
		}
		if assignmentCount != 0 {
			t.Fatalf("expected no fabricated assignment for invalid fallback user, got %d", assignmentCount)
		}

		var timeline timelineModel
		if err := db.Where("ticket_id = ? AND event_type = ?", ticket.ID, "participant_fallback_warning").First(&timeline).Error; err != nil {
			t.Fatalf("load fallback warning timeline: %v", err)
		}
		if !strings.Contains(timeline.Message, "兜底处理人无效") {
			t.Fatalf("expected invalid fallback warning, got %q", timeline.Message)
		}
	})
}
