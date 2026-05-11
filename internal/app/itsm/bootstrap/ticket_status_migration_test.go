package bootstrap

import (
	"testing"
	"time"

	. "metis/internal/app/itsm/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMigrateTicketStatusModelMapsLegacyStatusToNewStatusAndOutcome(t *testing.T) {
	db := newTicketStatusMigrationDB(t)
	now := time.Now()

	tickets := []Ticket{
		{Code: "T-1", Title: "legacy pending", ServiceID: 1, EngineType: "smart", Status: "pending", PriorityID: 1, RequesterID: 1},
		{Code: "T-2", Title: "legacy in_progress human", ServiceID: 1, EngineType: "smart", Status: "in_progress", PriorityID: 1, RequesterID: 1},
		{Code: "T-3", Title: "legacy waiting_action", ServiceID: 1, EngineType: "smart", Status: "waiting_action", PriorityID: 1, RequesterID: 1},
		{Code: "T-4", Title: "legacy completed approved", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1},
		{Code: "T-5", Title: "legacy completed rejected", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1},
		{Code: "T-6", Title: "legacy completed fulfilled", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1},
		{Code: "T-7", Title: "legacy cancelled withdrawn", ServiceID: 1, EngineType: "smart", Status: "cancelled", PriorityID: 1, RequesterID: 1},
		{Code: "T-8", Title: "legacy cancelled", ServiceID: 1, EngineType: "smart", Status: "cancelled", PriorityID: 1, RequesterID: 1},
		{Code: "T-9", Title: "legacy failed", ServiceID: 1, EngineType: "smart", Status: "failed", PriorityID: 1, RequesterID: 1},
		{Code: "T-10", Title: "already new status", ServiceID: 1, EngineType: "smart", Status: TicketStatusDecisioning, PriorityID: 1, RequesterID: 1},
	}
	if err := db.Create(&tickets).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}

	activityRows := []TicketActivity{
		{TicketID: tickets[1].ID, ActivityType: "approve", Status: "pending", TransitionOutcome: ""},
		{TicketID: tickets[2].ID, ActivityType: "action", Status: "pending", TransitionOutcome: ""},
		{TicketID: tickets[3].ID, ActivityType: "approve", Status: "completed", TransitionOutcome: TicketOutcomeApproved, FinishedAt: ptrTime(now.Add(-20 * time.Minute))},
		{TicketID: tickets[4].ID, ActivityType: "process", Status: "completed", TransitionOutcome: TicketOutcomeRejected, FinishedAt: ptrTime(now.Add(-15 * time.Minute))},
		{TicketID: tickets[5].ID, ActivityType: "action", Status: "completed", TransitionOutcome: "success", FinishedAt: ptrTime(now.Add(-10 * time.Minute))},
	}
	if err := db.Create(&activityRows).Error; err != nil {
		t.Fatalf("create activities: %v", err)
	}

	assignments := []TicketAssignment{
		{TicketID: tickets[3].ID, ActivityID: activityRows[2].ID, ParticipantType: "user", Status: "completed"},
		{TicketID: tickets[4].ID, ActivityID: activityRows[3].ID, ParticipantType: "user", Status: "completed"},
		{TicketID: tickets[5].ID, ActivityID: activityRows[4].ID, ParticipantType: "user", Status: "completed"},
	}
	if err := db.Create(&assignments).Error; err != nil {
		t.Fatalf("create assignments: %v", err)
	}

	if err := db.Create(&TicketTimeline{
		TicketID:   tickets[6].ID,
		OperatorID: 1,
		EventType:  "withdrawn",
		Message:    "用户撤回",
	}).Error; err != nil {
		t.Fatalf("create withdrawn timeline: %v", err)
	}

	if err := MigrateTicketStatusModel(db); err != nil {
		t.Fatalf("run migration: %v", err)
	}
	// 再跑一次，确认一次性迁移逻辑幂等。
	if err := MigrateTicketStatusModel(db); err != nil {
		t.Fatalf("run migration second time: %v", err)
	}

	assertTicket := func(ticketID uint, wantStatus, wantOutcome string, expectFinished bool) {
		t.Helper()
		var got Ticket
		if err := db.First(&got, ticketID).Error; err != nil {
			t.Fatalf("load ticket %d: %v", ticketID, err)
		}
		if got.Status != wantStatus || got.Outcome != wantOutcome {
			t.Fatalf("ticket %d status/outcome mismatch: got (%s,%s), want (%s,%s)", ticketID, got.Status, got.Outcome, wantStatus, wantOutcome)
		}
		if expectFinished && got.FinishedAt == nil {
			t.Fatalf("ticket %d expected finished_at to be set", ticketID)
		}
		if !expectFinished && got.FinishedAt != nil {
			t.Fatalf("ticket %d expected finished_at to be nil", ticketID)
		}
	}

	assertTicket(tickets[0].ID, TicketStatusSubmitted, "", false)
	assertTicket(tickets[1].ID, TicketStatusWaitingHuman, "", false)
	assertTicket(tickets[2].ID, TicketStatusExecutingAction, "", false)
	assertTicket(tickets[3].ID, TicketStatusCompleted, TicketOutcomeApproved, true)
	assertTicket(tickets[4].ID, TicketStatusRejected, TicketOutcomeRejected, true)
	assertTicket(tickets[5].ID, TicketStatusCompleted, TicketOutcomeFulfilled, true)
	assertTicket(tickets[6].ID, TicketStatusWithdrawn, TicketOutcomeWithdrawn, true)
	assertTicket(tickets[7].ID, TicketStatusCancelled, TicketOutcomeCancelled, true)
	assertTicket(tickets[8].ID, TicketStatusFailed, TicketOutcomeFailed, true)
	assertTicket(tickets[9].ID, TicketStatusDecisioning, "", false)

	var approvedActivity TicketActivity
	if err := db.First(&approvedActivity, activityRows[2].ID).Error; err != nil {
		t.Fatalf("load approved activity: %v", err)
	}
	if approvedActivity.Status != TicketOutcomeApproved {
		t.Fatalf("expected activity status approved, got %s", approvedActivity.Status)
	}

	var rejectedActivity TicketActivity
	if err := db.First(&rejectedActivity, activityRows[3].ID).Error; err != nil {
		t.Fatalf("load rejected activity: %v", err)
	}
	if rejectedActivity.Status != TicketOutcomeRejected {
		t.Fatalf("expected activity status rejected, got %s", rejectedActivity.Status)
	}

	var untouchedActivity TicketActivity
	if err := db.First(&untouchedActivity, activityRows[4].ID).Error; err != nil {
		t.Fatalf("load untouched activity: %v", err)
	}
	if untouchedActivity.Status != "completed" {
		t.Fatalf("expected action activity status unchanged, got %s", untouchedActivity.Status)
	}

	var approvedAssignment TicketAssignment
	if err := db.First(&approvedAssignment, assignments[0].ID).Error; err != nil {
		t.Fatalf("load approved assignment: %v", err)
	}
	if approvedAssignment.Status != AssignmentApproved {
		t.Fatalf("expected assignment status approved, got %s", approvedAssignment.Status)
	}

	var rejectedAssignment TicketAssignment
	if err := db.First(&rejectedAssignment, assignments[1].ID).Error; err != nil {
		t.Fatalf("load rejected assignment: %v", err)
	}
	if rejectedAssignment.Status != AssignmentRejected {
		t.Fatalf("expected assignment status rejected, got %s", rejectedAssignment.Status)
	}

	var untouchedAssignment TicketAssignment
	if err := db.First(&untouchedAssignment, assignments[2].ID).Error; err != nil {
		t.Fatalf("load untouched assignment: %v", err)
	}
	if untouchedAssignment.Status != "completed" {
		t.Fatalf("expected assignment status unchanged, got %s", untouchedAssignment.Status)
	}
}

func TestDeriveTicketStatusOutcomeAndHelpers(t *testing.T) {
	db := newTicketStatusMigrationDB(t)

	t.Run("legacy empty and unknown statuses fall back safely", func(t *testing.T) {
		status, outcome := deriveTicketStatusOutcome(db, Ticket{Status: "", Outcome: ""})
		if status != TicketStatusSubmitted || outcome != "" {
			t.Fatalf("empty legacy status => (%s,%s), want (%s,%s)", status, outcome, TicketStatusSubmitted, "")
		}

		status, outcome = deriveTicketStatusOutcome(db, Ticket{Status: "mystery", Outcome: "?"})
		if status != TicketStatusSubmitted || outcome != "" {
			t.Fatalf("unknown legacy status => (%s,%s), want (%s,%s)", status, outcome, TicketStatusSubmitted, "")
		}
	})

	t.Run("new statuses normalize outcome and withdrawn semantics", func(t *testing.T) {
		rejectedNew := Ticket{Code: "T-NEW-REJECT", Title: "new reject", ServiceID: 1, EngineType: "smart", Status: TicketStatusCompleted, Outcome: TicketOutcomeRejected, PriorityID: 1, RequesterID: 1}
		withdrawnNew := Ticket{Code: "T-NEW-WITHDRAW", Title: "new withdraw", ServiceID: 1, EngineType: "smart", Status: TicketStatusCancelled, Outcome: "", PriorityID: 1, RequesterID: 1}
		rejectedBare := Ticket{Code: "T-NEW-REJECT-BARE", Title: "new reject bare", ServiceID: 1, EngineType: "smart", Status: TicketStatusRejected, Outcome: "", PriorityID: 1, RequesterID: 1}
		failedBare := Ticket{Code: "T-NEW-FAILED-BARE", Title: "new failed bare", ServiceID: 1, EngineType: "smart", Status: TicketStatusFailed, Outcome: "", PriorityID: 1, RequesterID: 1}
		if err := db.Create([]*Ticket{&rejectedNew, &withdrawnNew, &rejectedBare, &failedBare}).Error; err != nil {
			t.Fatalf("create new-status tickets: %v", err)
		}
		if err := db.Create(&TicketTimeline{TicketID: withdrawnNew.ID, OperatorID: 1, EventType: "withdrawn", Message: "用户撤回"}).Error; err != nil {
			t.Fatalf("create withdrawn timeline: %v", err)
		}

		status, outcome := deriveTicketStatusOutcome(db, rejectedNew)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("new completed+rejected => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		status, outcome = deriveTicketStatusOutcome(db, withdrawnNew)
		if status != TicketStatusWithdrawn || outcome != TicketOutcomeWithdrawn {
			t.Fatalf("new cancelled+withdrawn => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWithdrawn, TicketOutcomeWithdrawn)
		}

		status, outcome = deriveTicketStatusOutcome(db, rejectedBare)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("new rejected bare => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		status, outcome = deriveTicketStatusOutcome(db, failedBare)
		if status != TicketStatusFailed || outcome != TicketOutcomeFailed {
			t.Fatalf("new failed bare => (%s,%s), want (%s,%s)", status, outcome, TicketStatusFailed, TicketOutcomeFailed)
		}
	})

	t.Run("legacy active statuses derive from latest pending activity type", func(t *testing.T) {
		waitTicket := Ticket{Code: "T-WAIT", Title: "wait", ServiceID: 1, EngineType: "smart", Status: "in_progress", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&waitTicket).Error; err != nil {
			t.Fatalf("create wait ticket: %v", err)
		}
		waitActivity := TicketActivity{TicketID: waitTicket.ID, ActivityType: "wait", Status: "pending"}
		if err := db.Create(&waitActivity).Error; err != nil {
			t.Fatalf("create wait activity: %v", err)
		}
		status, outcome := deriveTicketStatusOutcome(db, waitTicket)
		if status != TicketStatusWaitingHuman || outcome != "" {
			t.Fatalf("wait activity => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWaitingHuman, "")
		}

		noActivityTicket := Ticket{Code: "T-NO-ACT", Title: "no act", ServiceID: 1, EngineType: "smart", Status: "waiting_action", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&noActivityTicket).Error; err != nil {
			t.Fatalf("create no activity ticket: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, noActivityTicket)
		if status != TicketStatusDecisioning || outcome != "" {
			t.Fatalf("missing active activity => (%s,%s), want (%s,%s)", status, outcome, TicketStatusDecisioning, "")
		}

		notifyTicket := Ticket{Code: "T-NOTIFY", Title: "notify", ServiceID: 1, EngineType: "smart", Status: "in_progress", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&notifyTicket).Error; err != nil {
			t.Fatalf("create notify ticket: %v", err)
		}
		notifyActivity := TicketActivity{TicketID: notifyTicket.ID, ActivityType: "notify", Status: "pending"}
		if err := db.Create(&notifyActivity).Error; err != nil {
			t.Fatalf("create notify activity: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, notifyTicket)
		if status != TicketStatusExecutingAction || outcome != "" {
			t.Fatalf("notify activity => (%s,%s), want (%s,%s)", status, outcome, TicketStatusExecutingAction, "")
		}

		unknownTicket := Ticket{Code: "T-UNKNOWN", Title: "unknown", ServiceID: 1, EngineType: "smart", Status: "waiting_action", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&unknownTicket).Error; err != nil {
			t.Fatalf("create unknown ticket: %v", err)
		}
		unknownActivity := TicketActivity{TicketID: unknownTicket.ID, ActivityType: "script", Status: "in_progress"}
		if err := db.Create(&unknownActivity).Error; err != nil {
			t.Fatalf("create unknown activity: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, unknownTicket)
		if status != TicketStatusDecisioning || outcome != "" {
			t.Fatalf("unknown activity type => (%s,%s), want (%s,%s)", status, outcome, TicketStatusDecisioning, "")
		}
	})

	t.Run("legacy terminal statuses normalize by history and timeline", func(t *testing.T) {
		approvedTicket := Ticket{Code: "T-APPROVED", Title: "approved", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1}
		rejectedTicket := Ticket{Code: "T-REJECTED", Title: "rejected", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1}
		fulfilledTicket := Ticket{Code: "T-FULFILLED", Title: "fulfilled", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1}
		withdrawnTicket := Ticket{Code: "T-WITHDRAWN", Title: "withdrawn", ServiceID: 1, EngineType: "smart", Status: "cancelled", PriorityID: 1, RequesterID: 1}
		cancelledTicket := Ticket{Code: "T-CANCELLED", Title: "cancelled", ServiceID: 1, EngineType: "smart", Status: "cancelled", PriorityID: 1, RequesterID: 1}
		failedTicket := Ticket{Code: "T-FAILED", Title: "failed", ServiceID: 1, EngineType: "smart", Status: "failed", PriorityID: 1, RequesterID: 1}
		if err := db.Create([]*Ticket{&approvedTicket, &rejectedTicket, &fulfilledTicket, &withdrawnTicket, &cancelledTicket, &failedTicket}).Error; err != nil {
			t.Fatalf("create terminal tickets: %v", err)
		}

		approvedActivity := TicketActivity{TicketID: approvedTicket.ID, ActivityType: "approve", Status: "completed", TransitionOutcome: TicketOutcomeApproved, FinishedAt: ptrTime(time.Now())}
		rejectedActivity := TicketActivity{TicketID: rejectedTicket.ID, ActivityType: "process", Status: "completed", TransitionOutcome: TicketOutcomeRejected, FinishedAt: ptrTime(time.Now())}
		if err := db.Create([]*TicketActivity{&approvedActivity, &rejectedActivity}).Error; err != nil {
			t.Fatalf("create terminal activities: %v", err)
		}
		if err := db.Create(&TicketTimeline{TicketID: withdrawnTicket.ID, OperatorID: 1, EventType: "withdrawn", Message: "用户撤回"}).Error; err != nil {
			t.Fatalf("create withdrawn timeline: %v", err)
		}

		cases := []struct {
			ticket       Ticket
			wantStatus   string
			wantOutcome  string
		}{
			{ticket: approvedTicket, wantStatus: TicketStatusCompleted, wantOutcome: TicketOutcomeApproved},
			{ticket: rejectedTicket, wantStatus: TicketStatusRejected, wantOutcome: TicketOutcomeRejected},
			{ticket: fulfilledTicket, wantStatus: TicketStatusCompleted, wantOutcome: TicketOutcomeFulfilled},
			{ticket: withdrawnTicket, wantStatus: TicketStatusWithdrawn, wantOutcome: TicketOutcomeWithdrawn},
			{ticket: cancelledTicket, wantStatus: TicketStatusCancelled, wantOutcome: TicketOutcomeCancelled},
			{ticket: failedTicket, wantStatus: TicketStatusFailed, wantOutcome: TicketOutcomeFailed},
		}
		for _, tc := range cases {
			gotStatus, gotOutcome := deriveTicketStatusOutcome(db, tc.ticket)
			if gotStatus != tc.wantStatus || gotOutcome != tc.wantOutcome {
				t.Fatalf("ticket %s => (%s,%s), want (%s,%s)", tc.ticket.Code, gotStatus, gotOutcome, tc.wantStatus, tc.wantOutcome)
			}
		}

		trailingNoiseTicket := Ticket{Code: "T-NOISE", Title: "noise", ServiceID: 1, EngineType: "smart", Status: "completed", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&trailingNoiseTicket).Error; err != nil {
			t.Fatalf("create noise ticket: %v", err)
		}
		rejectedAt := time.Now().Add(-2 * time.Minute)
		noiseAt := time.Now().Add(-time.Minute)
		rejectedHuman := TicketActivity{TicketID: trailingNoiseTicket.ID, ActivityType: "approve", Status: "completed", TransitionOutcome: TicketOutcomeRejected, FinishedAt: ptrTime(rejectedAt)}
		noiseAction := TicketActivity{TicketID: trailingNoiseTicket.ID, ActivityType: "action", Status: "completed", TransitionOutcome: "success", FinishedAt: ptrTime(noiseAt)}
		if err := db.Create([]*TicketActivity{&rejectedHuman, &noiseAction}).Error; err != nil {
			t.Fatalf("create trailing noise activities: %v", err)
		}
		gotStatus, gotOutcome := deriveTicketStatusOutcome(db, trailingNoiseTicket)
		if gotStatus != TicketStatusRejected || gotOutcome != TicketOutcomeRejected {
			t.Fatalf("legacy completed with trailing non-human activity => (%s,%s), want (%s,%s)", gotStatus, gotOutcome, TicketStatusRejected, TicketOutcomeRejected)
		}
	})

	t.Run("normalize new status respects explicit outcomes and withdrawn signals", func(t *testing.T) {
		cancelledTicket := Ticket{Code: "T-CANCEL", Title: "cancel", ServiceID: 1, EngineType: "smart", Status: TicketStatusCancelled, PriorityID: 1, RequesterID: 1}
		if err := db.Create(&cancelledTicket).Error; err != nil {
			t.Fatalf("create cancelled ticket: %v", err)
		}
		if err := db.Create(&TicketTimeline{TicketID: cancelledTicket.ID, OperatorID: 1, EventType: "withdrawn", Message: "撤回"}).Error; err != nil {
			t.Fatalf("create withdrawn timeline: %v", err)
		}
		status, outcome := normalizeNewTicketStatusOutcome(db, cancelledTicket.ID, TicketStatusCancelled, "")
		if status != TicketStatusWithdrawn || outcome != TicketOutcomeWithdrawn {
			t.Fatalf("cancelled with withdrawn timeline => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWithdrawn, TicketOutcomeWithdrawn)
		}

		completedTicket := Ticket{Code: "T-COMP", Title: "completed", ServiceID: 1, EngineType: "smart", Status: TicketStatusCompleted, PriorityID: 1, RequesterID: 1}
		if err := db.Create(&completedTicket).Error; err != nil {
			t.Fatalf("create completed ticket: %v", err)
		}
		status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, TicketStatusCompleted, TicketOutcomeRejected)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("completed+rejected => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, TicketStatusCompleted, TicketOutcomeApproved)
		if status != TicketStatusCompleted || outcome != TicketOutcomeApproved {
			t.Fatalf("completed+approved => (%s,%s), want (%s,%s)", status, outcome, TicketStatusCompleted, TicketOutcomeApproved)
		}

		status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, TicketStatusRejected, "")
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("rejected without outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, TicketStatusFailed, "")
		if status != TicketStatusFailed || outcome != TicketOutcomeFailed {
			t.Fatalf("failed without outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusFailed, TicketOutcomeFailed)
		}

		status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, TicketStatusCancelled, TicketOutcomeWithdrawn)
		if status != TicketStatusWithdrawn || outcome != TicketOutcomeWithdrawn {
			t.Fatalf("cancelled with explicit withdrawn outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWithdrawn, TicketOutcomeWithdrawn)
		}

		for _, passthrough := range []string{
			TicketStatusSubmitted,
			TicketStatusWaitingHuman,
			TicketStatusApprovedDecisioning,
			TicketStatusRejectedDecisioning,
			TicketStatusDecisioning,
			TicketStatusExecutingAction,
		} {
			status, outcome = normalizeNewTicketStatusOutcome(db, completedTicket.ID, passthrough, "ignored")
			if status != passthrough || outcome != "" {
				t.Fatalf("passthrough status %s => (%s,%s), want (%s,%s)", passthrough, status, outcome, passthrough, "")
			}
		}
	})

	t.Run("derive ticket status honors explicit new status normalizer", func(t *testing.T) {
		withdrawn := Ticket{Code: "T-NEW-WITHDRAWN", Title: "withdrawn", ServiceID: 1, EngineType: "smart", Status: TicketStatusCancelled, Outcome: TicketOutcomeWithdrawn, PriorityID: 1, RequesterID: 1}
		if err := db.Create(&withdrawn).Error; err != nil {
			t.Fatalf("create withdrawn ticket: %v", err)
		}
		status, outcome := deriveTicketStatusOutcome(db, withdrawn)
		if status != TicketStatusWithdrawn || outcome != TicketOutcomeWithdrawn {
			t.Fatalf("new cancelled withdrawn => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWithdrawn, TicketOutcomeWithdrawn)
		}

		rejected := Ticket{Code: "T-NEW-REJECTED", Title: "rejected", ServiceID: 1, EngineType: "smart", Status: TicketStatusRejected, PriorityID: 1, RequesterID: 1}
		if err := db.Create(&rejected).Error; err != nil {
			t.Fatalf("create rejected ticket: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, rejected)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("new rejected without outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		completed := Ticket{Code: "T-NEW-COMPLETE", Title: "complete", ServiceID: 1, EngineType: "smart", Status: TicketStatusCompleted, Outcome: "", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&completed).Error; err != nil {
			t.Fatalf("create completed ticket: %v", err)
		}
		rejectedHuman := TicketActivity{TicketID: completed.ID, ActivityType: "process", Status: "completed", TransitionOutcome: TicketOutcomeRejected, FinishedAt: ptrTime(time.Now())}
		if err := db.Create(&rejectedHuman).Error; err != nil {
			t.Fatalf("create rejected human activity: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, completed)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("new completed without explicit outcome should reuse last human result => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}
	})
}

func newTicketStatusMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:itsm_ticket_status_migration?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Ticket{}, &TicketActivity{}, &TicketAssignment{}, &TicketTimeline{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
