package bootstrap

import (
	"testing"
	"time"

	. "metis/internal/app/itsm/domain"
)

func TestMigrateServiceDeskSubmissionIndex_RebuildsLegacyDraftUniqueness(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ServiceDeskSubmission{}); err != nil {
		t.Fatalf("migrate submissions: %v", err)
	}

	if err := db.Migrator().DropIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft"); err != nil {
		t.Fatalf("drop modern index: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_itsm_submission_draft ON itsm_service_desk_submissions(session_id, draft_version, fields_hash)`).Error; err != nil {
		t.Fatalf("create legacy index: %v", err)
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex before migrate: %v", err)
	}
	if !legacy {
		t.Fatal("expected handcrafted legacy index to be detected")
	}

	first := ServiceDeskSubmission{
		SessionID:    77,
		DraftVersion: 3,
		FieldsHash:   "fields-same",
		RequestHash:  "request-a",
		Status:       "submitted",
		SubmittedBy:  9,
		SubmittedAt:  time.Now(),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first submission: %v", err)
	}

	second := ServiceDeskSubmission{
		SessionID:    77,
		DraftVersion: 3,
		FieldsHash:   "fields-same",
		RequestHash:  "request-b",
		Status:       "submitted",
		SubmittedBy:  9,
		SubmittedAt:  time.Now(),
	}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected legacy three-column uniqueness to reject different request_hash")
	}

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex: %v", err)
	}

	legacy, err = hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex after migrate: %v", err)
	}
	if legacy {
		t.Fatal("expected legacy index to be rebuilt with request_hash")
	}

	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second submission after migration: %v", err)
	}

	var count int64
	if err := db.Model(&ServiceDeskSubmission{}).
		Where("session_id = ? AND draft_version = ? AND fields_hash = ?", first.SessionID, first.DraftVersion, first.FieldsHash).
		Count(&count).Error; err != nil {
		t.Fatalf("count migrated submissions: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two submissions after rebuilt index, got %d", count)
	}
}

func TestMigrateServiceDeskSubmissionIndex_NoopsWithoutSubmissionTable(t *testing.T) {
	db := newTestDB(t)

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex without table: %v", err)
	}
}

func TestMigrateServiceDeskSubmissionIndex_AutoMigratesWhenDraftIndexMissing(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ServiceDeskSubmission{}); err != nil {
		t.Fatalf("migrate submissions: %v", err)
	}
	if err := db.Migrator().DropIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft"); err != nil {
		t.Fatalf("drop draft index: %v", err)
	}

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex without draft index: %v", err)
	}

	if !db.Migrator().HasIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft") {
		t.Fatal("expected draft uniqueness index to be recreated by AutoMigrate")
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex after AutoMigrate: %v", err)
	}
	if legacy {
		t.Fatal("expected recreated draft index to use modern four-column shape")
	}
}

func TestMigrateServiceDeskSubmissionIndex_RebuildsWrongOrderedDraftIndex(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ServiceDeskSubmission{}); err != nil {
		t.Fatalf("migrate submissions: %v", err)
	}
	if err := db.Migrator().DropIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft"); err != nil {
		t.Fatalf("drop modern index: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_itsm_submission_draft ON itsm_service_desk_submissions(session_id, fields_hash, draft_version, request_hash)`).Error; err != nil {
		t.Fatalf("create wrong-ordered index: %v", err)
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex on wrong-ordered index: %v", err)
	}
	if !legacy {
		t.Fatal("expected wrong-ordered draft index to be treated as legacy")
	}

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex wrong-ordered index: %v", err)
	}

	legacy, err = hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex after wrong-ordered rebuild: %v", err)
	}
	if legacy {
		t.Fatal("expected wrong-ordered draft index to be rebuilt to modern shape")
	}
}

func TestMigrateServiceDeskSubmissionIndex_LeavesModernIndexIntact(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ServiceDeskSubmission{}); err != nil {
		t.Fatalf("migrate submissions: %v", err)
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex on modern index: %v", err)
	}
	if legacy {
		t.Fatal("expected current four-column index to be treated as modern")
	}

	first := ServiceDeskSubmission{
		SessionID:    88,
		DraftVersion: 5,
		FieldsHash:   "fields-modern",
		RequestHash:  "request-a",
		Status:       "submitted",
		SubmittedBy:  9,
		SubmittedAt:  time.Now(),
	}
	second := ServiceDeskSubmission{
		SessionID:    88,
		DraftVersion: 5,
		FieldsHash:   "fields-modern",
		RequestHash:  "request-b",
		Status:       "submitted",
		SubmittedBy:  9,
		SubmittedAt:  time.Now(),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first modern submission: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second modern submission: %v", err)
	}

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex on modern index: %v", err)
	}

	legacy, err = hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex after no-op migrate: %v", err)
	}
	if legacy {
		t.Fatal("expected modern index to remain modern after migration")
	}
}

func TestMigrateServiceDeskSubmissionIndex_RebuildsMalformedDraftIndexShape(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ServiceDeskSubmission{}); err != nil {
		t.Fatalf("migrate submissions: %v", err)
	}

	if err := db.Migrator().DropIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft"); err != nil {
		t.Fatalf("drop modern index: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_itsm_submission_draft ON itsm_service_desk_submissions(session_id, draft_version)`).Error; err != nil {
		t.Fatalf("create malformed index: %v", err)
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex on malformed index: %v", err)
	}
	if !legacy {
		t.Fatal("expected malformed draft index to be treated as legacy")
	}

	if err := migrateServiceDeskSubmissionIndex(db); err != nil {
		t.Fatalf("migrateServiceDeskSubmissionIndex malformed index: %v", err)
	}

	legacy, err = hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex after malformed rebuild: %v", err)
	}
	if legacy {
		t.Fatal("expected malformed draft index to be rebuilt to modern shape")
	}

	first := ServiceDeskSubmission{
		SessionID:    91,
		DraftVersion: 1,
		FieldsHash:   "same-fields",
		RequestHash:  "request-a",
		Status:       "submitted",
		SubmittedBy:  3,
		SubmittedAt:  time.Now(),
	}
	second := ServiceDeskSubmission{
		SessionID:    91,
		DraftVersion: 1,
		FieldsHash:   "same-fields",
		RequestHash:  "request-b",
		Status:       "submitted",
		SubmittedBy:  3,
		SubmittedAt:  time.Now(),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first submission after malformed rebuild: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second submission after malformed rebuild: %v", err)
	}
}

func TestRepairCompletedHumanAssignments_RepairsRejectedAssignmentsUsingTimelineTime(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&Priority{}, &Ticket{}, &TicketActivity{}, &TicketAssignment{}, &TicketTimeline{}); err != nil {
		t.Fatalf("migrate ticket tables: %v", err)
	}

	operatorID := uint(21)
	priority := Priority{Name: "普通", Code: "bootstrap-normal", Value: 10, Color: "#666"}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}
	ticket := Ticket{
		Code:        "TICK-BOOTSTRAP-REPAIR",
		Title:       "审批驳回修复",
		ServiceID:   1,
		EngineType:  "smart",
		Status:      TicketStatusRejected,
		PriorityID:  priority.ID,
		RequesterID: operatorID,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := TicketActivity{
		TicketID:          ticket.ID,
		Name:              "审批",
		ActivityType:      "approve",
		Status:            TicketOutcomeRejected,
		TransitionOutcome: TicketOutcomeRejected,
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	assignment := TicketAssignment{
		TicketID:        ticket.ID,
		ActivityID:      activity.ID,
		ParticipantType: "user",
		UserID:          &operatorID,
		AssigneeID:      &operatorID,
		Status:          AssignmentPending,
		IsCurrent:       true,
	}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	timelineAt := time.Now().Add(-time.Minute)
	timeline := TicketTimeline{
		TicketID:   ticket.ID,
		ActivityID: &activity.ID,
		OperatorID: operatorID,
		EventType:  "activity_completed",
		Message:    "活动 [审批] 完成，结果: rejected",
	}
	if err := db.Create(&timeline).Error; err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	if err := db.Model(&timeline).Update("created_at", timelineAt).Error; err != nil {
		t.Fatalf("set timeline created_at: %v", err)
	}

	if err := RepairCompletedHumanAssignments(db); err != nil {
		t.Fatalf("RepairCompletedHumanAssignments: %v", err)
	}

	var refreshed TicketAssignment
	if err := db.First(&refreshed, assignment.ID).Error; err != nil {
		t.Fatalf("reload assignment: %v", err)
	}
	if refreshed.Status != AssignmentRejected {
		t.Fatalf("assignment status = %q, want %q", refreshed.Status, AssignmentRejected)
	}
	if refreshed.FinishedAt == nil || !refreshed.FinishedAt.Equal(timelineAt) {
		t.Fatalf("assignment finished_at = %v, want %v", refreshed.FinishedAt, timelineAt)
	}
	if refreshed.IsCurrent {
		t.Fatalf("expected repaired assignment not current: %+v", refreshed)
	}
}
