package sla

import (
	"errors"
	"fmt"
	. "metis/internal/app/itsm/domain"
	"testing"

	"metis/internal/database"
	"metis/internal/model"
)

func TestValidateEscalationTargetConfigRequiresActionTargets(t *testing.T) {
	tests := []struct {
		name       string
		actionType string
		raw        JSONField
		wantErr    bool
	}{
		{
			name:       "notify valid",
			actionType: "notify",
			raw:        JSONField(`{"recipients":[{"type":"user","value":"1"}],"channelId":2}`),
		},
		{
			name:       "notify requester valid",
			actionType: "notify",
			raw:        JSONField(`{"recipients":[{"type":"requester"}],"channelId":2}`),
		},
		{
			name:       "notify requires recipients",
			actionType: "notify",
			raw:        JSONField(`{"channelId":2}`),
			wantErr:    true,
		},
		{
			name:       "notify requires channel",
			actionType: "notify",
			raw:        JSONField(`{"recipients":[{"type":"user","value":"1"}]}`),
			wantErr:    true,
		},
		{
			name:       "reassign valid",
			actionType: "reassign",
			raw:        JSONField(`{"assigneeCandidates":[{"type":"department","value":"10"}]}`),
		},
		{
			name:       "reassign requires candidates",
			actionType: "reassign",
			raw:        JSONField(`{}`),
			wantErr:    true,
		},
		{
			name:       "priority valid",
			actionType: "escalate_priority",
			raw:        JSONField(`{"priorityId":1}`),
		},
		{
			name:       "priority requires target",
			actionType: "escalate_priority",
			raw:        JSONField(`{}`),
			wantErr:    true,
		},
		{
			name:       "position department requires codes",
			actionType: "notify",
			raw:        JSONField(`{"recipients":[{"type":"position_department","position_code":"ops"}],"channelId":2}`),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEscalationTargetConfig(tt.actionType, tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrEscalationTargetConfig) {
					t.Fatalf("error = %v, want ErrEscalationTargetConfig", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate config: %v", err)
			}
		})
	}
}

func TestEscalationTargetReferenceValidation(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate message channels: %v", err)
	}
	svc := &EscalationRuleService{db: &database.DB{DB: db}}

	disabled := &model.MessageChannel{Name: "Disabled SMTP", Type: "smtp", Config: `{}`, Enabled: false}
	if err := db.Create(disabled).Error; err != nil {
		t.Fatalf("create disabled channel: %v", err)
	}
	if err := db.Model(disabled).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable channel: %v", err)
	}
	if err := svc.validateEscalationTargetReferences("notify", JSONField(fmt.Sprintf(`{"recipients":[{"type":"user","value":"1"}],"channelId":%d}`, disabled.ID))); !errors.Is(err, ErrEscalationTargetConfig) {
		t.Fatalf("disabled channel error = %v, want ErrEscalationTargetConfig", err)
	}

	enabled := &model.MessageChannel{Name: "SMTP", Type: "smtp", Config: `{}`, Enabled: true}
	if err := db.Create(enabled).Error; err != nil {
		t.Fatalf("create enabled channel: %v", err)
	}
	if err := svc.validateEscalationTargetReferences("notify", JSONField(fmt.Sprintf(`{"recipients":[{"type":"user","value":"1"}],"channelId":%d}`, enabled.ID))); err != nil {
		t.Fatalf("enabled channel rejected: %v", err)
	}

	if err := svc.validateEscalationTargetReferences("escalate_priority", JSONField(`{"priorityId":99}`)); !errors.Is(err, ErrEscalationTargetConfig) {
		t.Fatalf("missing priority error = %v, want ErrEscalationTargetConfig", err)
	}
	priority := &Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444", IsActive: true}
	if err := db.Create(priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}
	if err := svc.validateEscalationTargetReferences("escalate_priority", JSONField(fmt.Sprintf(`{"priorityId":%d}`, priority.ID))); err != nil {
		t.Fatalf("enabled priority rejected: %v", err)
	}
}

func TestEscalationRuleServiceRejectsInvalidLevelAndWaitMinutes(t *testing.T) {
	svc, db := newEscalationRuleServiceForTest(t)
	sla := seedEscalationRuleTestSLA(t, db, "esc-invalid")
	if err := db.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate message channels: %v", err)
	}
	channel := model.MessageChannel{Name: "Email", Type: "smtp", Config: `{}`, Enabled: true}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	validTarget := JSONField(fmt.Sprintf(`{"recipients":[{"type":"user","value":"1"}],"channelId":%d}`, channel.ID))
	if _, err := svc.Create(&EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "response_timeout",
		Level:        0,
		WaitMinutes:  5,
		ActionType:   "notify",
		TargetConfig: validTarget,
	}); !errors.Is(err, ErrEscalationRuleInvalid) {
		t.Fatalf("invalid level create error = %v, want %v", err, ErrEscalationRuleInvalid)
	}

	rule, err := svc.Create(&EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  5,
		ActionType:   "notify",
		TargetConfig: validTarget,
	})
	if err != nil {
		t.Fatalf("seed valid rule: %v", err)
	}

	if _, err := svc.Update(rule.ID, map[string]any{"wait_minutes": -1}); !errors.Is(err, ErrEscalationRuleInvalid) {
		t.Fatalf("invalid wait_minutes update error = %v, want %v", err, ErrEscalationRuleInvalid)
	}
}

func TestEscalationRuleServiceDeleteMissingReturnsBusinessNotFound(t *testing.T) {
	svc, _ := newEscalationRuleServiceForTest(t)
	if err := svc.Delete(999999); !errors.Is(err, ErrEscalationRuleNotFound) {
		t.Fatalf("delete missing escalation error = %v, want %v", err, ErrEscalationRuleNotFound)
	}
}
