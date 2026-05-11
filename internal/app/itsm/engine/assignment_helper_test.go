package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAssignmentHelpersAndStatusContracts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	t.Run("assignmentOperatorCondition matches direct and scoped assignees", func(t *testing.T) {
		activity := activityModel{TicketID: 1, ActivityType: NodeApprove, Status: ActivityPending}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		userID := uint(11)
		positionID := uint(22)
		departmentID := uint(33)
		otherDepartment := uint(44)

		assignments := []assignmentModel{
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "user", UserID: &userID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "position_department", PositionID: &positionID, DepartmentID: &departmentID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "", PositionID: &positionID, DepartmentID: &departmentID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "position", PositionID: &positionID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "department", DepartmentID: &departmentID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "", DepartmentID: &departmentID, Status: "pending"},
			{TicketID: 1, ActivityID: activity.ID, ParticipantType: "position_department", PositionID: &positionID, DepartmentID: &otherDepartment, Status: "pending"},
		}
		if err := db.Create(&assignments).Error; err != nil {
			t.Fatalf("create assignments: %v", err)
		}

		var matched []assignmentModel
		err := db.Model(&assignmentModel{}).
			Where("activity_id = ?", activity.ID).
			Where(assignmentOperatorCondition(db, "itsm_ticket_assignments", userID, []uint{positionID}, []uint{departmentID})).
			Order("id ASC").
			Find(&matched).Error
		if err != nil {
			t.Fatalf("query matches: %v", err)
		}
		if len(matched) != 6 {
			t.Fatalf("expected 6 matching assignments, got %d: %+v", len(matched), matched)
		}
	})

	t.Run("completePendingAssignment handles zero operator and missing assignments", func(t *testing.T) {
		if got, ok, err := completePendingAssignment(db, nil, 1, 0, TicketOutcomeApproved, time.Now(), nil, nil, true); err != nil || ok || got != nil {
			t.Fatalf("zero operator => (%+v,%v,%v), want (nil,false,nil)", got, ok, err)
		}

		activity := activityModel{TicketID: 2, ActivityType: NodeApprove, Status: ActivityPending}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		assignee := uint(99)
		assignment := assignmentModel{
			TicketID:        2,
			ActivityID:      activity.ID,
			ParticipantType: "user",
			UserID:          &assignee,
			AssigneeID:      &assignee,
			Status:          "pending",
			IsCurrent:       true,
		}
		if err := db.Create(&assignment).Error; err != nil {
			t.Fatalf("create mismatched assignment: %v", err)
		}

		got, ok, err := completePendingAssignment(db, nil, activity.ID, 100, TicketOutcomeApproved, time.Now(), nil, nil, true)
		if !errors.Is(err, ErrNoActiveAssignment) || ok || got != nil {
			t.Fatalf("missing active assignment => (%+v,%v,%v), want (nil,false,%v)", got, ok, err, ErrNoActiveAssignment)
		}
	})

	t.Run("completePendingAssignment updates claimed assignment and returns completion", func(t *testing.T) {
		now := time.Now()
		activity := activityModel{TicketID: 3, ActivityType: NodeProcess, Status: ActivityInProgress}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		operatorID := uint(77)
		positionID := uint(8)
		departmentID := uint(9)
		assignment := assignmentModel{
			TicketID:        3,
			ActivityID:      activity.ID,
			ParticipantType: "position_department",
			PositionID:      &positionID,
			DepartmentID:    &departmentID,
			Status:          "claimed",
			IsCurrent:       true,
		}
		if err := db.Create(&assignment).Error; err != nil {
			t.Fatalf("create assignment: %v", err)
		}

		completed, ok, err := completePendingAssignment(db, nil, activity.ID, operatorID, TicketOutcomeRejected, now, []uint{positionID}, []uint{departmentID}, true)
		if err != nil || !ok || completed == nil {
			t.Fatalf("complete assignment => (%+v,%v,%v), want completed assignment", completed, ok, err)
		}
		if completed.AssigneeID == nil || *completed.AssigneeID != operatorID || completed.Status != "rejected" {
			t.Fatalf("unexpected completed assignment: %+v", completed)
		}

		var stored assignmentModel
		if err := db.First(&stored, assignment.ID).Error; err != nil {
			t.Fatalf("reload assignment: %v", err)
		}
		if stored.Status != "rejected" || stored.IsCurrent || stored.AssigneeID == nil || *stored.AssigneeID != operatorID || stored.FinishedAt == nil {
			t.Fatalf("stored assignment not completed correctly: %+v", stored)
		}
	})

	t.Run("activityBecameInactive only returns true for non-active statuses", func(t *testing.T) {
		pending := activityModel{TicketID: 4, ActivityType: NodeApprove, Status: ActivityPending}
		done := activityModel{TicketID: 4, ActivityType: NodeApprove, Status: ActivityApproved}
		if err := db.Create(&pending).Error; err != nil {
			t.Fatalf("create pending activity: %v", err)
		}
		if err := db.Create(&done).Error; err != nil {
			t.Fatalf("create done activity: %v", err)
		}

		if activityBecameInactive(db, pending.ID) {
			t.Fatal("pending activity should still be active")
		}
		if !activityBecameInactive(db, done.ID) {
			t.Fatal("approved activity should be inactive")
		}
		if activityBecameInactive(db, 999999) {
			t.Fatal("missing activity should not be treated as inactive transition")
		}
	})

	t.Run("status helpers map outcomes consistently", func(t *testing.T) {
		if got := HumanActivityResultStatus(TicketOutcomeRejected); got != ActivityRejected {
			t.Fatalf("rejected outcome => %s, want %s", got, ActivityRejected)
		}
		if got := HumanActivityResultStatus("anything-else"); got != ActivityApproved {
			t.Fatalf("non-rejected outcome => %s, want %s", got, ActivityApproved)
		}

		if got := TicketDecisioningStatusForOutcome(ActivityRejected); got != TicketStatusRejectedDecisioning {
			t.Fatalf("rejected decision status => %s", got)
		}
		if got := TicketDecisioningStatusForOutcome(ActivityApproved); got != TicketStatusApprovedDecisioning {
			t.Fatalf("approved decision status => %s", got)
		}
		if got := TicketDecisioningStatusForOutcome("unknown"); got != TicketStatusDecisioning {
			t.Fatalf("fallback decision status => %s", got)
		}

		if got := humanOrCompletedActivityStatus(NodeApprove, TicketOutcomeRejected); got != ActivityRejected {
			t.Fatalf("human activity status => %s", got)
		}
		if got := humanOrCompletedActivityStatus(NodeAction, TicketOutcomeRejected); got != ActivityCompleted {
			t.Fatalf("non-human activity status => %s", got)
		}

		if !IsCompletedActivityStatus(ActivityApproved) || !IsCompletedActivityStatus(ActivityCancelled) {
			t.Fatal("approved/cancelled should count as completed activity statuses")
		}
		if IsCompletedActivityStatus(ActivityPending) {
			t.Fatal("pending should not count as completed activity status")
		}
	})
}
