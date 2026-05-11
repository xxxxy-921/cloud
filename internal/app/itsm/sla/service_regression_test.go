package sla

import (
	"errors"
	"testing"

	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
	"gorm.io/gorm"
)

func newEscalationRuleServiceForTest(t *testing.T) (*EscalationRuleService, *database.DB) {
	t.Helper()
	db := &database.DB{DB: newTestDB(t)}
	return &EscalationRuleService{
		repo: &EscalationRuleRepo{db: db},
		db:   db,
	}, db
}

func seedEscalationRuleTestSLA(t *testing.T, db *database.DB, code string) SLATemplate {
	t.Helper()
	sla := SLATemplate{
		Name:              "SLA " + code,
		Code:              code,
		ResponseMinutes:   5,
		ResolutionMinutes: 30,
		IsActive:          true,
	}
	if err := db.Create(&sla).Error; err != nil {
		t.Fatalf("create test sla template %s: %v", code, err)
	}
	return sla
}

func TestEscalationRuleServiceCreateUpdateGetDeleteAndList(t *testing.T) {
	svc, db := newEscalationRuleServiceForTest(t)
	sla := seedEscalationRuleTestSLA(t, db, "esc-main")
	channel := Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444", IsActive: true}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	rule, err := svc.Create(&EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  5,
		ActionType:   "escalate_priority",
		TargetConfig: JSONField(`{"priorityId":1}`),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !rule.IsActive {
		t.Fatal("expected created rule to default to active")
	}

	if _, err := svc.Create(&EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  10,
		ActionType:   "escalate_priority",
		TargetConfig: JSONField(`{"priorityId":1}`),
	}); !errors.Is(err, ErrEscalationLevelExists) {
		t.Fatalf("duplicate create error = %v, want %v", err, ErrEscalationLevelExists)
	}

	second, err := svc.Create(&EscalationRule{
		SLAID:        sla.ID,
		TriggerType:  "resolution_timeout",
		Level:        2,
		WaitMinutes:  15,
		ActionType:   "escalate_priority",
		TargetConfig: JSONField(`{"priorityId":1}`),
	})
	if err != nil {
		t.Fatalf("create second rule: %v", err)
	}

	if _, err := svc.Update(second.ID, map[string]any{
		"trigger_type": "response_timeout",
		"level":        1,
		"target_config": JSONField(`{"priorityId":1}`),
	}); !errors.Is(err, ErrEscalationLevelExists) {
		t.Fatalf("conflicting update error = %v, want %v", err, ErrEscalationLevelExists)
	}

	updated, err := svc.Update(rule.ID, map[string]any{
		"wait_minutes": 20,
		"target_config": JSONField(`{"priorityId":1}`),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.WaitMinutes != 20 {
		t.Fatalf("WaitMinutes = %d, want 20", updated.WaitMinutes)
	}

	got, err := svc.Get(rule.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != rule.ID {
		t.Fatalf("Get ID = %d, want %d", got.ID, rule.ID)
	}

	items, err := svc.ListBySLA(sla.ID)
	if err != nil {
		t.Fatalf("ListBySLA: %v", err)
	}
	if len(items) != 2 || items[0].Level != 1 || items[1].Level != 2 {
		t.Fatalf("unexpected ListBySLA items: %+v", items)
	}

	if err := svc.Delete(rule.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(rule.ID); !errors.Is(err, ErrEscalationRuleNotFound) {
		t.Fatalf("Get deleted rule error = %v, want %v", err, ErrEscalationRuleNotFound)
	}
}

func TestEscalationRuleServiceRejectsMissingParentSLA(t *testing.T) {
	svc, db := newEscalationRuleServiceForTest(t)
	priority := Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	if _, err := svc.Create(&EscalationRule{
		SLAID:        999999,
		TriggerType:  "response_timeout",
		Level:        1,
		WaitMinutes:  5,
		ActionType:   "escalate_priority",
		TargetConfig: JSONField(`{"priorityId":1}`),
	}); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("create missing SLA error = %v, want %v", err, ErrSLATemplateNotFound)
	}

	if _, err := svc.ListBySLA(999999); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("list missing SLA error = %v, want %v", err, ErrSLATemplateNotFound)
	}
}

func TestPriorityServiceCreateUpdateGetDeleteAndList(t *testing.T) {
	svc, db := newSLATemplateServiceForTest(t)
	_ = svc
	prioritySvc := &PriorityService{repo: &PriorityRepo{db: db}}

	created, err := prioritySvc.Create(&Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444"})
	if err != nil {
		t.Fatalf("Create priority: %v", err)
	}
	if !created.IsActive {
		t.Fatal("expected priority to default active")
	}

	if _, err := prioritySvc.Create(&Priority{Name: "Dup", Code: "P1", Value: 2, Color: "#000"}); !errors.Is(err, ErrPriorityCodeExists) {
		t.Fatalf("duplicate priority error = %v, want %v", err, ErrPriorityCodeExists)
	}

	updated, err := prioritySvc.Update(created.ID, map[string]any{"name": "P1 updated"})
	if err != nil {
		t.Fatalf("Update priority: %v", err)
	}
	if updated.Name != "P1 updated" {
		t.Fatalf("priority name = %q, want P1 updated", updated.Name)
	}

	got, err := prioritySvc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get priority: %v", err)
	}
	if got.Code != "P1" {
		t.Fatalf("priority code = %q, want P1", got.Code)
	}

	list, err := prioritySvc.ListAll()
	if err != nil {
		t.Fatalf("ListAll priorities: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("priority list len = %d, want 1", len(list))
	}

	if err := prioritySvc.Delete(created.ID); err != nil {
		t.Fatalf("Delete priority: %v", err)
	}
	if _, err := prioritySvc.Get(created.ID); !errors.Is(err, ErrPriorityNotFound) {
		t.Fatalf("Get deleted priority error = %v, want %v", err, ErrPriorityNotFound)
	}

	if _, err := prioritySvc.Get(999999); !errors.Is(err, ErrPriorityNotFound) {
		t.Fatalf("Get missing priority error = %v, want %v", err, ErrPriorityNotFound)
	}
	if _, err := prioritySvc.Update(999999, map[string]any{"name": "ghost"}); !errors.Is(err, ErrPriorityNotFound) {
		t.Fatalf("Update missing priority error = %v, want %v", err, ErrPriorityNotFound)
	}
	if err := prioritySvc.Delete(999999); !errors.Is(err, ErrPriorityNotFound) {
		t.Fatalf("Delete missing priority error = %v, want %v", err, ErrPriorityNotFound)
	}

	second, err := prioritySvc.Create(&Priority{Name: "P2", Code: "P2", Value: 2, Color: "#0ea5e9"})
	if err != nil {
		t.Fatalf("create second priority: %v", err)
	}
	if _, err := prioritySvc.Update(second.ID, map[string]any{"code": "P1"}); !errors.Is(err, ErrPriorityCodeExists) {
		t.Fatalf("duplicate priority code update error = %v, want %v", err, ErrPriorityCodeExists)
	}
	if _, err := prioritySvc.Update(second.ID, map[string]any{"name": "   "}); !errors.Is(err, ErrPriorityInvalidIdentifier) {
		t.Fatalf("blank priority name update error = %v, want %v", err, ErrPriorityInvalidIdentifier)
	}
	if _, err := prioritySvc.Update(second.ID, map[string]any{"code": "   "}); !errors.Is(err, ErrPriorityInvalidIdentifier) {
		t.Fatalf("blank priority code update error = %v, want %v", err, ErrPriorityInvalidIdentifier)
	}
	trimmed, err := prioritySvc.Update(second.ID, map[string]any{"name": "  P2 Trimmed  ", "code": "  P2X  ", "value": 3})
	if err != nil {
		t.Fatalf("trimmed priority update: %v", err)
	}
	if trimmed.Name != "P2 Trimmed" || trimmed.Code != "P2X" || trimmed.Value != 3 {
		t.Fatalf("unexpected trimmed priority update result: %+v", trimmed)
	}
}

func TestPriorityServiceRejectsNonPositiveValue(t *testing.T) {
	_, db := newSLATemplateServiceForTest(t)
	prioritySvc := &PriorityService{repo: &PriorityRepo{db: db}}

	if _, err := prioritySvc.Create(&Priority{Name: "Broken", Code: "BROKEN", Value: 0, Color: "#000"}); !errors.Is(err, ErrPriorityInvalidValue) {
		t.Fatalf("create invalid priority value error = %v, want %v", err, ErrPriorityInvalidValue)
	}

	created, err := prioritySvc.Create(&Priority{Name: "P1", Code: "P1", Value: 1, Color: "#ef4444"})
	if err != nil {
		t.Fatalf("create valid priority: %v", err)
	}
	if _, err := prioritySvc.Update(created.ID, map[string]any{"value": -1}); !errors.Is(err, ErrPriorityInvalidValue) {
		t.Fatalf("update invalid priority value error = %v, want %v", err, ErrPriorityInvalidValue)
	}
}

func TestPriorityAndSLATemplateServicesRejectBlankBusinessIdentifiers(t *testing.T) {
	_, db := newSLATemplateServiceForTest(t)
	prioritySvc := &PriorityService{repo: &PriorityRepo{db: db}}
	slaSvc := &SLATemplateService{repo: &SLATemplateRepo{db: db}, db: db}

	if _, err := prioritySvc.Create(&Priority{Name: "   ", Code: " P1 ", Value: 1, Color: "#ef4444"}); !errors.Is(err, ErrPriorityInvalidIdentifier) {
		t.Fatalf("blank priority name create error = %v, want %v", err, ErrPriorityInvalidIdentifier)
	}
	createdPriority, err := prioritySvc.Create(&Priority{Name: " P1 ", Code: " P1 ", Value: 1, Color: "#ef4444"})
	if err != nil {
		t.Fatalf("create trimmed priority: %v", err)
	}
	if createdPriority.Name != "P1" || createdPriority.Code != "P1" {
		t.Fatalf("expected trimmed priority identifiers, got %+v", createdPriority)
	}
	if _, err := prioritySvc.Update(createdPriority.ID, map[string]any{"name": "   "}); !errors.Is(err, ErrPriorityInvalidIdentifier) {
		t.Fatalf("blank priority name update error = %v, want %v", err, ErrPriorityInvalidIdentifier)
	}

	if _, err := slaSvc.Create(&SLATemplate{Name: "   ", Code: " std ", ResponseMinutes: 5, ResolutionMinutes: 30}); !errors.Is(err, ErrSLATemplateInvalidIdentifier) {
		t.Fatalf("blank sla name create error = %v, want %v", err, ErrSLATemplateInvalidIdentifier)
	}
	createdSLA, err := slaSvc.Create(&SLATemplate{Name: " Standard ", Code: " std ", ResponseMinutes: 5, ResolutionMinutes: 30})
	if err != nil {
		t.Fatalf("create trimmed sla: %v", err)
	}
	if createdSLA.Name != "Standard" || createdSLA.Code != "std" {
		t.Fatalf("expected trimmed sla identifiers, got %+v", createdSLA)
	}
	if _, err := slaSvc.Update(createdSLA.ID, map[string]any{"code": "   "}); !errors.Is(err, ErrSLATemplateInvalidIdentifier) {
		t.Fatalf("blank sla code update error = %v, want %v", err, ErrSLATemplateInvalidIdentifier)
	}
}

func TestSLATemplateServiceGetUpdateDeleteMissingContracts(t *testing.T) {
	svc, _ := newSLATemplateServiceForTest(t)

	if _, err := svc.Get(999999); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("Get missing SLA template error = %v, want %v", err, ErrSLATemplateNotFound)
	}
	if _, err := svc.Update(999999, map[string]any{"name": "ghost"}); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("Update missing SLA template error = %v, want %v", err, ErrSLATemplateNotFound)
	}
	if err := svc.Delete(999999); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("Delete missing SLA template error = %v, want %v", err, ErrSLATemplateNotFound)
	}
}

func TestPriorityAndSLARepositoriesResolveOnlyActiveDefaults(t *testing.T) {
	_, db := newSLATemplateServiceForTest(t)
	priorityRepo := &PriorityRepo{db: db}
	slaRepo := &SLATemplateRepo{db: db}

	inactiveLow := &Priority{Name: "Inactive Low", Code: "P0", Value: 0, Color: "#111", IsActive: true}
	activeMid := &Priority{Name: "Active Mid", Code: "P2", Value: 2, Color: "#222", IsActive: true}
	activeHigh := &Priority{Name: "Active High", Code: "P3", Value: 3, Color: "#333", IsActive: true}
	for _, p := range []*Priority{inactiveLow, activeHigh, activeMid} {
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("create priority %s: %v", p.Code, err)
		}
	}
	if err := db.Model(inactiveLow).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate priority %s: %v", inactiveLow.Code, err)
	}

	defaultPriority, err := priorityRepo.FindDefaultActive()
	if err != nil {
		t.Fatalf("FindDefaultActive: %v", err)
	}
	if defaultPriority.Code != "P2" {
		t.Fatalf("default active priority = %s, want P2", defaultPriority.Code)
	}

	activeByCode, err := priorityRepo.FindActiveByCode("P2")
	if err != nil {
		t.Fatalf("FindActiveByCode active: %v", err)
	}
	if activeByCode.ID != activeMid.ID {
		t.Fatalf("FindActiveByCode id = %d, want %d", activeByCode.ID, activeMid.ID)
	}
	if _, err := priorityRepo.FindActiveByCode("P0"); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindActiveByCode inactive error = %v, want record not found", err)
	}

	inactiveSLA := &SLATemplate{Name: "Inactive SLA", Code: "sla-inactive", ResponseMinutes: 5, ResolutionMinutes: 15, IsActive: true}
	activeSLA := &SLATemplate{Name: "Active SLA", Code: "sla-active", ResponseMinutes: 10, ResolutionMinutes: 30, IsActive: true}
	for _, sla := range []*SLATemplate{inactiveSLA, activeSLA} {
		if err := db.Create(sla).Error; err != nil {
			t.Fatalf("create sla %s: %v", sla.Code, err)
		}
	}
	if err := db.Model(inactiveSLA).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate sla %s: %v", inactiveSLA.Code, err)
	}

	gotActiveSLA, err := slaRepo.FindActiveByID(activeSLA.ID)
	if err != nil {
		t.Fatalf("FindActiveByID active: %v", err)
	}
	if gotActiveSLA.Code != activeSLA.Code {
		t.Fatalf("FindActiveByID code = %s, want %s", gotActiveSLA.Code, activeSLA.Code)
	}
	if _, err := slaRepo.FindActiveByID(inactiveSLA.ID); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindActiveByID inactive error = %v, want record not found", err)
	}
}
