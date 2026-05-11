package engine

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/testutil"
)

func TestDecisionDataStoreRuntimeContextReaders(t *testing.T) {
	db := testutil.NewTestDB(t)
	service := testutil.SeedSmartSubmissionService(t, db)
	store := NewDecisionDataStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	responseDeadline := now.Add(30 * time.Minute)
	resolutionDeadline := now.Add(2 * time.Hour)

	if err := db.Exec(`INSERT INTO users (id, username, is_active) VALUES (7, 'alice', true), (9, 'bob', true)`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	ticket := domain.Ticket{
		Code:                  "TICK-CTX-001",
		Title:                 "VPN 开通",
		Status:                TicketStatusWaitingHuman,
		Outcome:               "",
		ServiceID:             service.ID,
		EngineType:            "smart",
		RequesterID:           7,
		PriorityID:            1,
		FormData:              domain.JSONField(`{"env":"prod","region":"gz"}`),
		SLAStatus:             slaBreachedResponse,
		SLAResponseDeadline:   &responseDeadline,
		SLAResolutionDeadline: &resolutionDeadline,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	completedAt := now.Add(-10 * time.Minute)
	activities := []domain.TicketActivity{
		{TicketID: ticket.ID, Name: "人工审批", ActivityType: NodeApprove, Status: ActivityCompleted, ActivityGroupID: "grp-1", FinishedAt: &completedAt},
		{TicketID: ticket.ID, Name: "网络处理", ActivityType: NodeProcess, Status: ActivityPending, ActivityGroupID: "grp-1"},
		{TicketID: ticket.ID, Name: "安全复核", ActivityType: NodeProcess, Status: ActivityInProgress, ActivityGroupID: "grp-1"},
		{TicketID: ticket.ID, Name: "动作执行", ActivityType: NodeAction, Status: ActivityCompleted},
	}
	for i := range activities {
		if err := db.Create(&activities[i]).Error; err != nil {
			t.Fatalf("create activity %d: %v", i, err)
		}
	}

	assignments := []domain.TicketAssignment{
		{TicketID: ticket.ID, ActivityID: activities[1].ID, ParticipantType: "user", AssigneeID: uintPtr(9), UserID: uintPtr(9), Status: "pending", IsCurrent: true},
		{TicketID: ticket.ID, ActivityID: activities[0].ID, ParticipantType: "user", AssigneeID: uintPtr(7), UserID: uintPtr(7), Status: "completed", IsCurrent: false, FinishedAt: &completedAt},
		{TicketID: ticket.ID, ActivityID: activities[2].ID, ParticipantType: "user", AssigneeID: uintPtr(9), UserID: uintPtr(9), Status: "pending", IsCurrent: false},
	}
	for i := range assignments {
		if err := db.Create(&assignments[i]).Error; err != nil {
			t.Fatalf("create assignment %d: %v", i, err)
		}
	}

	action := domain.ServiceAction{
		Name:       "发送通知",
		Code:       "notify",
		ActionType: "http",
		ConfigJSON: domain.JSONField(`{"url":"https://example.com/webhook","method":"POST"}`),
		ServiceID:  service.ID,
		IsActive:   true,
	}
	if err := db.Create(&action).Error; err != nil {
		t.Fatalf("create action: %v", err)
	}
	if err := db.Create(&domain.TicketActionExecution{TicketID: ticket.ID, ServiceActionID: action.ID, Status: "success"}).Error; err != nil {
		t.Fatalf("create action execution: %v", err)
	}
	if err := db.Create(&domain.TicketActionExecution{TicketID: ticket.ID, ServiceActionID: action.ID, Status: "failed"}).Error; err != nil {
		t.Fatalf("create failed action execution: %v", err)
	}

	finishedOld := now.Add(-2 * time.Hour)
	finishedNew := now.Add(-1 * time.Hour)
	otherTickets := []domain.Ticket{
		{Code: "TICK-HIST-OLD", Title: "旧工单", Status: TicketStatusCompleted, Outcome: TicketOutcomeApproved, ServiceID: service.ID, RequesterID: 7, PriorityID: 1, FinishedAt: &finishedOld},
		{Code: "TICK-HIST-NEW", Title: "新工单", Status: TicketStatusCompleted, Outcome: TicketOutcomeApproved, ServiceID: service.ID, RequesterID: 7, PriorityID: 1, FinishedAt: &finishedNew},
		{Code: "TICK-HIST-OTHER", Title: "其他状态", Status: TicketStatusRejected, Outcome: TicketOutcomeRejected, ServiceID: service.ID, RequesterID: 7, PriorityID: 1, FinishedAt: &finishedNew},
	}
	for i := range otherTickets {
		if err := db.Create(&otherTickets[i]).Error; err != nil {
			t.Fatalf("create history ticket %d: %v", i, err)
		}
	}

	ctx, err := store.GetTicketContext(ticket.ID)
	if err != nil {
		t.Fatalf("GetTicketContext: %v", err)
	}
	if ctx.Code != ticket.Code || ctx.FormData != string(ticket.FormData) {
		t.Fatalf("unexpected ticket context: %+v", ctx)
	}

	currentAssignment, err := store.GetCurrentAssignment(ticket.ID)
	if err != nil {
		t.Fatalf("GetCurrentAssignment: %v", err)
	}
	if currentAssignment == nil || currentAssignment.AssigneeID != 9 || currentAssignment.AssigneeName != "bob" {
		t.Fatalf("unexpected current assignment: %+v", currentAssignment)
	}

	currentActivities, err := store.GetCurrentActivities(ticket.ID)
	if err != nil {
		t.Fatalf("GetCurrentActivities: %v", err)
	}
	if len(currentActivities) != 2 || currentActivities[0].Name != "网络处理" || currentActivities[1].Name != "安全复核" {
		t.Fatalf("unexpected current activities: %+v", currentActivities)
	}

	history, err := store.GetDecisionHistory(ticket.ID)
	if err != nil {
		t.Fatalf("GetDecisionHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("decision history len = %d, want 2", len(history))
	}

	assignRows, err := store.GetActivityAssignments(activities[1].ID)
	if err != nil {
		t.Fatalf("GetActivityAssignments: %v", err)
	}
	if len(assignRows) != 1 || assignRows[0].AssigneeID == nil || *assignRows[0].AssigneeID != 9 {
		t.Fatalf("unexpected activity assignments: %+v", assignRows)
	}

	executedActions, err := store.GetExecutedActions(ticket.ID)
	if err != nil {
		t.Fatalf("GetExecutedActions: %v", err)
	}
	if len(executedActions) != 1 || executedActions[0].ActionCode != "notify" {
		t.Fatalf("unexpected executed actions: %+v", executedActions)
	}

	groups, err := store.GetParallelGroups(ticket.ID)
	if err != nil {
		t.Fatalf("GetParallelGroups: %v", err)
	}
	if len(groups) != 1 || groups[0].ActivityGroupID != "grp-1" || groups[0].Total != 3 || groups[0].Completed != 1 {
		t.Fatalf("unexpected parallel groups: %+v", groups)
	}

	pendingNames, err := store.GetPendingActivityNames(ticket.ID, "grp-1")
	if err != nil {
		t.Fatalf("GetPendingActivityNames: %v", err)
	}
	if len(pendingNames) != 2 || pendingNames[0] != "网络处理" || pendingNames[1] != "安全复核" {
		t.Fatalf("unexpected pending activity names: %+v", pendingNames)
	}

	user, err := store.GetUserBasicInfo(9)
	if err != nil {
		t.Fatalf("GetUserBasicInfo: %v", err)
	}
	if user.Username != "bob" || !user.IsActive {
		t.Fatalf("unexpected user basic info: %+v", user)
	}

	pendingCount, err := store.CountUserPendingActivities(9)
	if err != nil {
		t.Fatalf("CountUserPendingActivities: %v", err)
	}
	if pendingCount != 2 {
		t.Fatalf("pending activity count = %d, want 2", pendingCount)
	}

	similar, err := store.GetSimilarHistory(service.ID, ticket.ID, 2)
	if err != nil {
		t.Fatalf("GetSimilarHistory: %v", err)
	}
	if len(similar) != 2 || similar[0].Code != "TICK-HIST-NEW" || similar[1].Code != "TICK-HIST-OLD" {
		t.Fatalf("unexpected similar history: %+v", similar)
	}

	completedCount, err := store.CountCompletedTickets(service.ID)
	if err != nil {
		t.Fatalf("CountCompletedTickets: %v", err)
	}
	if completedCount != 2 {
		t.Fatalf("completed ticket count = %d, want 2", completedCount)
	}

	activityCount, err := store.CountTicketActivities(ticket.ID)
	if err != nil {
		t.Fatalf("CountTicketActivities: %v", err)
	}
	if activityCount != 4 {
		t.Fatalf("ticket activity count = %d, want 4", activityCount)
	}

	slaData, err := store.GetSLAData(ticket.ID)
	if err != nil {
		t.Fatalf("GetSLAData: %v", err)
	}
	if slaData.SLAStatus != slaBreachedResponse || slaData.SLAResponseDeadline == nil || !slaData.SLAResponseDeadline.Equal(responseDeadline) {
		t.Fatalf("unexpected SLA data: %+v", slaData)
	}
}

func TestDecisionDataStoreSnapshotCountsAndErrors(t *testing.T) {
	db := testutil.NewTestDB(t)
	service := testutil.SeedSmartSubmissionService(t, db)

	liveActions := []domain.ServiceAction{
		{
			Name:       "Live notify",
			Code:       "notify",
			ActionType: "http",
			ConfigJSON: domain.JSONField(`{"url":"https://example.com/live"}`),
			ServiceID:  service.ID,
			IsActive:   true,
		},
		{
			Name:       "Live pause",
			Code:       "pause",
			ActionType: "http",
			ConfigJSON: domain.JSONField(`{"url":"https://example.com/pause"}`),
			ServiceID:  service.ID,
			IsActive:   true,
		},
	}
	for i := range liveActions {
		if err := db.Create(&liveActions[i]).Error; err != nil {
			t.Fatalf("create live action %d: %v", i, err)
		}
	}
	if err := db.Exec("UPDATE itsm_service_actions SET is_active = ? WHERE id = ?", false, liveActions[1].ID).Error; err != nil {
		t.Fatalf("mark live disabled action inactive: %v", err)
	}

	version := domain.ServiceDefinitionVersion{
		ServiceID:   service.ID,
		Version:     1,
		ContentHash: "snapshot-actions-count",
		EngineType:  "smart",
		ActionsJSON: domain.JSONField(`[{"id":` + itoa(liveActions[0].ID) + `,"code":"notify","name":"Snapshot notify","description":"old","actionType":"http","configJson":{"url":"https://example.com/snapshot"},"isActive":true},{"id":9999,"code":"disabled","name":"Disabled","description":"old","actionType":"http","configJson":{"url":"https://example.com/disabled"},"isActive":false}]`),
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create version: %v", err)
	}
	ticket := domain.Ticket{
		Code:             "TICK-SNAPSHOT-COUNT",
		Title:            "snapshot count",
		ServiceID:        service.ID,
		ServiceVersionID: &version.ID,
		EngineType:       "smart",
		Status:           TicketStatusDecisioning,
		PriorityID:       1,
		RequesterID:      1,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	store := NewDecisionDataStore(db)
	count, err := store.CountActiveServiceActions(ticket.ID, service.ID)
	if err != nil {
		t.Fatalf("CountActiveServiceActions: %v", err)
	}
	if count != 1 {
		t.Fatalf("active action count = %d, want 1", count)
	}

	action, err := store.GetServiceAction(ticket.ID, liveActions[0].ID, service.ID)
	if err != nil {
		t.Fatalf("GetServiceAction: %v", err)
	}
	if action.Name != "Snapshot notify" {
		t.Fatalf("unexpected snapshot action: %+v", action)
	}

	if _, err := store.GetServiceAction(ticket.ID, liveActions[1].ID, service.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetServiceAction missing snapshot action error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestDecisionDataStoreServiceActionSnapshotContracts(t *testing.T) {
	db := testutil.NewTestDB(t)
	service := testutil.SeedSmartSubmissionService(t, db)

	liveActions := []domain.ServiceAction{
		{
			Name:       "Live notify",
			Code:       "notify",
			ActionType: "http",
			ConfigJSON: domain.JSONField(`{"url":"https://example.com/live"}`),
			ServiceID:  service.ID,
			IsActive:   true,
		},
	}
	for i := range liveActions {
		if err := db.Create(&liveActions[i]).Error; err != nil {
			t.Fatalf("create live action %d: %v", i, err)
		}
	}

	t.Run("falls back to live actions when ticket has no runtime version", func(t *testing.T) {
		ticket := domain.Ticket{
			Code:        "TICK-LIVE-ACTION-FALLBACK",
			Title:       "live fallback",
			ServiceID:   service.ID,
			EngineType:  "smart",
			Status:      TicketStatusDecisioning,
			PriorityID:  1,
			RequesterID: 1,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create fallback ticket: %v", err)
		}

		store := NewDecisionDataStore(db)
		rows, err := store.ListActiveServiceActions(ticket.ID, service.ID)
		if err != nil {
			t.Fatalf("ListActiveServiceActions fallback: %v", err)
		}
		if len(rows) != 1 || rows[0].Code != "notify" {
			t.Fatalf("unexpected live fallback actions: %+v", rows)
		}

		action, err := store.GetServiceAction(ticket.ID, liveActions[0].ID, service.ID)
		if err != nil {
			t.Fatalf("GetServiceAction fallback: %v", err)
		}
		if action.Code != "notify" || !action.IsActive {
			t.Fatalf("unexpected live fallback action: %+v", action)
		}
	})

	t.Run("malformed snapshot json fails closed", func(t *testing.T) {
		version := domain.ServiceDefinitionVersion{
			ServiceID:   service.ID,
			Version:     2,
			ContentHash: "snapshot-actions-bad-json",
			EngineType:  "smart",
			ActionsJSON: domain.JSONField(`{"broken":`),
		}
		if err := db.Create(&version).Error; err != nil {
			t.Fatalf("create malformed version: %v", err)
		}
		ticket := domain.Ticket{
			Code:             "TICK-SNAPSHOT-BAD-JSON",
			Title:            "snapshot bad json",
			ServiceID:        service.ID,
			ServiceVersionID: &version.ID,
			EngineType:       "smart",
			Status:           TicketStatusDecisioning,
			PriorityID:       1,
			RequesterID:      1,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create malformed snapshot ticket: %v", err)
		}

		store := NewDecisionDataStore(db)
		if _, err := store.ListActiveServiceActions(ticket.ID, service.ID); err == nil || err.Error() == "" {
			t.Fatalf("expected malformed snapshot list error, got %v", err)
		}
		if _, err := store.GetServiceAction(ticket.ID, liveActions[0].ID, service.ID); err == nil || err.Error() == "" {
			t.Fatalf("expected malformed snapshot get error, got %v", err)
		}
	})
}

func uintPtr(v uint) *uint {
	return &v
}
