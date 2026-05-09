package ticket

import (
	"encoding/json"
	"testing"

	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	"metis/internal/database"
	"metis/internal/model"
)

func TestBuildResponses_IncludesIntakeFormSchema(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	user := model.User{Username: "schema-viewer", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	catalog := ServiceCatalog{Name: "IT", Code: "it"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	intakeSchema := JSONField(`{"version":1,"fields":[{"key":"access_window","type":"date_range","label":"访问时段","props":{"withTime":true,"mode":"datetime"}}]}`)
	service := ServiceDefinition{
		Name:             "Server Access",
		Code:             "server-access",
		CatalogID:        catalog.ID,
		EngineType:       "smart",
		IsActive:         true,
		IntakeFormSchema: intakeSchema,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P1", Code: "p1", Value: 1, Color: "#f00", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}
	ticket := Ticket{
		Code:        "TICK-SCHEMA",
		Title:       "Temporary access",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusSubmitted,
		PriorityID:  priority.ID,
		RequesterID: user.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	responses, err := svc.BuildResponses([]Ticket{ticket}, user.ID)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if string(responses[0].IntakeFormSchema) != string(intakeSchema) {
		t.Fatalf("expected intake schema %s, got %s", intakeSchema, responses[0].IntakeFormSchema)
	}

	var payload map[string]any
	if err := json.Unmarshal(responses[0].IntakeFormSchema, &payload); err != nil {
		t.Fatalf("unmarshal intake schema: %v", err)
	}
	if payload["version"] != float64(1) {
		t.Fatalf("unexpected intake schema payload: %+v", payload)
	}
}

func TestBuildResponses_ParallelCurrentOwnerUsesRemainingPendingApprover(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	completedApprover := model.User{Username: "network_admin", IsActive: true}
	pendingApprover := model.User{Username: "security_admin", IsActive: true}
	for _, user := range []*model.User{&requester, &completedApprover, &pendingApprover} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Username, err)
		}
	}

	catalog := ServiceCatalog{Name: "IT", Code: "it"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:       "Parallel Approval",
		Code:       "parallel-approval",
		CatalogID:  catalog.ID,
		EngineType: "smart",
		IsActive:   true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P1", Code: "p1", Value: 1, Color: "#f00", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	completedActivityID := uint(1)
	ticket := Ticket{
		Code:              "TICK-PARALLEL-OWNER",
		Title:             "Parallel owner should show remaining approver",
		ServiceID:         service.ID,
		EngineType:        "smart",
		Status:            TicketStatusWaitingHuman,
		PriorityID:        priority.ID,
		RequesterID:       requester.ID,
		CurrentActivityID: &completedActivityID,
		Source:            TicketSourceCatalog,
		SLAStatus:         SLAStatusOnTrack,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	completedActivity := TicketActivity{
		TicketID:        ticket.ID,
		Name:            "Network approval",
		ActivityType:    engine.NodeApprove,
		Status:          engine.ActivityApproved,
		ActivityGroupID: "parallel-group-1",
	}
	if err := db.Create(&completedActivity).Error; err != nil {
		t.Fatalf("create completed activity: %v", err)
	}
	pendingActivity := TicketActivity{
		TicketID:        ticket.ID,
		Name:            "Security approval",
		ActivityType:    engine.NodeApprove,
		Status:          engine.ActivityPending,
		ActivityGroupID: "parallel-group-1",
	}
	if err := db.Create(&pendingActivity).Error; err != nil {
		t.Fatalf("create pending activity: %v", err)
	}

	if err := db.Model(&Ticket{}).Where("id = ?", ticket.ID).Update("current_activity_id", completedActivity.ID).Error; err != nil {
		t.Fatalf("update ticket current activity: %v", err)
	}
	ticket.CurrentActivityID = &completedActivity.ID

	completedAssignment := TicketAssignment{
		TicketID:        ticket.ID,
		ActivityID:      completedActivity.ID,
		ParticipantType: "user",
		UserID:          &completedApprover.ID,
		AssigneeID:      &completedApprover.ID,
		Status:          AssignmentApproved,
		IsCurrent:       true,
	}
	if err := db.Create(&completedAssignment).Error; err != nil {
		t.Fatalf("create completed assignment: %v", err)
	}
	pendingAssignment := TicketAssignment{
		TicketID:        ticket.ID,
		ActivityID:      pendingActivity.ID,
		ParticipantType: "user",
		UserID:          &pendingApprover.ID,
		AssigneeID:      &pendingApprover.ID,
		Status:          AssignmentPending,
		IsCurrent:       true,
	}
	if err := db.Create(&pendingAssignment).Error; err != nil {
		t.Fatalf("create pending assignment: %v", err)
	}

	responses, err := svc.BuildResponses([]Ticket{ticket}, pendingApprover.ID)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].CurrentOwnerName != pendingApprover.Username {
		t.Fatalf("expected current owner %q, got %q", pendingApprover.Username, responses[0].CurrentOwnerName)
	}
	if responses[0].CurrentOwnerType != "parallel" {
		t.Fatalf("expected current owner type parallel, got %q", responses[0].CurrentOwnerType)
	}
}
