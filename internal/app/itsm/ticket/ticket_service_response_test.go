package ticket

import (
	"encoding/json"
	"testing"
	"time"

	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	orgdomain "metis/internal/app/org/domain"
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

func TestBuildResponses_AssignmentFallbackAndDecisionExplanationSnapshot(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}

	catalog := ServiceCatalog{Name: "IT", Code: "it"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:       "Smart approval",
		Code:       "smart-approval",
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

	department := orgdomain.Department{Name: "网络部", Code: "network", IsActive: true}
	if err := db.Create(&department).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}
	position := orgdomain.Position{Name: "值班岗", Code: "duty", IsActive: true}
	if err := db.Create(&position).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	currentNamedID := uint(1)
	ticketNamed := Ticket{
		Code:              "TICK-FALLBACK-NAMED",
		Title:             "Fallback named owner",
		ServiceID:         service.ID,
		EngineType:        "smart",
		Status:            TicketStatusWaitingHuman,
		PriorityID:        priority.ID,
		RequesterID:       requester.ID,
		CurrentActivityID: &currentNamedID,
		Source:            TicketSourceCatalog,
		SLAStatus:         SLAStatusOnTrack,
	}
	if err := db.Create(&ticketNamed).Error; err != nil {
		t.Fatalf("create named ticket: %v", err)
	}
	activityNamed := TicketActivity{
		TicketID:     ticketNamed.ID,
		Name:         "网络审批",
		ActivityType: engine.NodeApprove,
		Status:       engine.ActivityPending,
	}
	if err := db.Create(&activityNamed).Error; err != nil {
		t.Fatalf("create named activity: %v", err)
	}
	if err := db.Model(&ticketNamed).Where("id = ?", ticketNamed.ID).Update("current_activity_id", activityNamed.ID).Error; err != nil {
		t.Fatalf("update named current activity: %v", err)
	}
	if err := db.Create(&TicketAssignment{
		TicketID:        ticketNamed.ID,
		ActivityID:      activityNamed.ID,
		ParticipantType: "position_department",
		PositionID:      &position.ID,
		DepartmentID:    &department.ID,
		Status:          AssignmentPending,
		IsCurrent:       true,
	}).Error; err != nil {
		t.Fatalf("create named assignment: %v", err)
	}
	snapshotDetails, _ := json.Marshal(map[string]any{
		"decision_explanation": map[string]any{
			"basis":    "知识库与协作规范",
			"trigger":  "ai_decision",
			"decision": "转网络岗处理",
			"nextStep": "等待网络部审批",
		},
	})
	if err := db.Create(&TicketTimeline{
		TicketID:   ticketNamed.ID,
		EventType:  "ai_decision_pending",
		Message:    "latest row without snapshot should be skipped",
		OperatorID: requester.ID,
	}).Error; err != nil {
		t.Fatalf("create latest empty timeline: %v", err)
	}
	if err := db.Create(&TicketTimeline{
		TicketID:   ticketNamed.ID,
		EventType:  "ai_decision_executed",
		Message:    "snapshot timeline",
		OperatorID: requester.ID,
		Details:    JSONField(snapshotDetails),
	}).Error; err != nil {
		t.Fatalf("create snapshot timeline: %v", err)
	}

	currentIDFallbackID := uint(2)
	ticketIDFallback := Ticket{
		Code:              "TICK-FALLBACK-ID",
		Title:             "Fallback id owner",
		ServiceID:         service.ID,
		EngineType:        "smart",
		Status:            TicketStatusWaitingHuman,
		PriorityID:        priority.ID,
		RequesterID:       requester.ID,
		CurrentActivityID: &currentIDFallbackID,
		Source:            TicketSourceCatalog,
		SLAStatus:         SLAStatusOnTrack,
	}
	if err := db.Create(&ticketIDFallback).Error; err != nil {
		t.Fatalf("create id fallback ticket: %v", err)
	}
	activityIDFallback := TicketActivity{
		TicketID:     ticketIDFallback.ID,
		Name:         "待人工指派",
		ActivityType: engine.NodeProcess,
		Status:       engine.ActivityPending,
	}
	if err := db.Create(&activityIDFallback).Error; err != nil {
		t.Fatalf("create id fallback activity: %v", err)
	}
	if err := db.Model(&ticketIDFallback).Where("id = ?", ticketIDFallback.ID).Update("current_activity_id", activityIDFallback.ID).Error; err != nil {
		t.Fatalf("update id fallback current activity: %v", err)
	}
	positionID := uint(999)
	departmentID := uint(998)
	if err := db.Create(&TicketAssignment{
		TicketID:        ticketIDFallback.ID,
		ActivityID:      activityIDFallback.ID,
		ParticipantType: "position_department",
		PositionID:      &positionID,
		DepartmentID:    &departmentID,
		Status:          AssignmentPending,
		IsCurrent:       true,
	}).Error; err != nil {
		t.Fatalf("create id fallback assignment: %v", err)
	}

	responses, err := svc.BuildResponses([]Ticket{ticketNamed, ticketIDFallback}, 0)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	byCode := map[string]TicketResponse{}
	for _, resp := range responses {
		byCode[resp.Code] = resp
	}

	namedResp := byCode[ticketNamed.Code]
	if namedResp.CurrentOwnerName != "网络部 / 值班岗" {
		t.Fatalf("expected named owner fallback, got %q", namedResp.CurrentOwnerName)
	}
	if namedResp.DecisionExplanation == nil || namedResp.DecisionExplanation.Basis != "知识库与协作规范" || namedResp.DecisionExplanation.NextStep != "等待网络部审批" {
		t.Fatalf("expected decision explanation snapshot, got %+v", namedResp.DecisionExplanation)
	}

	idFallbackResp := byCode[ticketIDFallback.Code]
	if idFallbackResp.CurrentOwnerName != "部门 #998 / 岗位 #999" {
		t.Fatalf("expected id fallback owner, got %q", idFallbackResp.CurrentOwnerName)
	}
}

func TestBuildResponse_DegradesGracefullyWhenDisplayJoinsAreMissing(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	ticket := &Ticket{
		Code:        "TICK-MISSING-DISPLAY",
		Title:       "Missing display joins",
		ServiceID:   999,
		EngineType:  "manual",
		Status:      TicketStatusSubmitted,
		PriorityID:  888,
		RequesterID: 777,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	resp, err := svc.BuildResponse(ticket, 0)
	if err != nil {
		t.Fatalf("BuildResponse: %v", err)
	}
	if resp.Code != ticket.Code || resp.ServiceName != "" || resp.PriorityName != "" || resp.RequesterName != "" {
		t.Fatalf("expected raw response fallback without joined displays, got %+v", resp)
	}
}

func TestBuildResponses_PopulateSmartSummaryStateContracts(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	assignee := model.User{Username: "approver", IsActive: true}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	if err := db.Create(&assignee).Error; err != nil {
		t.Fatalf("create assignee: %v", err)
	}
	catalog := ServiceCatalog{Name: "IT", Code: "it-smart-states"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{Name: "Smart States", Code: "smart-states", CatalogID: catalog.ID, EngineType: "smart", IsActive: true}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P2", Code: "p2", Value: 2, Color: "#0ea5e9", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	terminal := Ticket{
		Code:        "TICK-SMART-TERMINAL",
		Title:       "terminal",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusCompleted,
		Outcome:     TicketOutcomeFulfilled,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	aiDisabled := Ticket{
		Code:           "TICK-SMART-AI-DISABLED",
		Title:          "ai disabled",
		ServiceID:      service.ID,
		EngineType:     "smart",
		Status:         TicketStatusDecisioning,
		PriorityID:     priority.ID,
		RequesterID:    requester.ID,
		AIFailureCount: engine.MaxAIFailureCount,
		Source:         TicketSourceCatalog,
		SLAStatus:      SLAStatusOnTrack,
	}
	noCurrent := Ticket{
		Code:        "TICK-SMART-NO-CURRENT",
		Title:       "no current activity",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	actionActivity := Ticket{
		Code:        "TICK-SMART-ACTION",
		Title:       "action running",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusExecutingAction,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	waitingHuman := Ticket{
		Code:        "TICK-SMART-HUMAN",
		Title:       "waiting human",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	for _, ticket := range []*Ticket{&terminal, &aiDisabled, &noCurrent, &actionActivity, &waitingHuman} {
		if err := db.Create(ticket).Error; err != nil {
			t.Fatalf("create ticket %s: %v", ticket.Code, err)
		}
	}

	actionNode := TicketActivity{TicketID: actionActivity.ID, Name: "执行脚本", ActivityType: engine.NodeAction, Status: engine.ActivityInProgress}
	if err := db.Create(&actionNode).Error; err != nil {
		t.Fatalf("create action node: %v", err)
	}
	if err := db.Model(&actionActivity).Where("id = ?", actionActivity.ID).Update("current_activity_id", actionNode.ID).Error; err != nil {
		t.Fatalf("update action current activity: %v", err)
	}
	actionActivity.CurrentActivityID = &actionNode.ID

	humanNode := TicketActivity{TicketID: waitingHuman.ID, Name: "人工审批", ActivityType: engine.NodeApprove, Status: engine.ActivityPending}
	if err := db.Create(&humanNode).Error; err != nil {
		t.Fatalf("create human node: %v", err)
	}
	if err := db.Model(&waitingHuman).Where("id = ?", waitingHuman.ID).Update("current_activity_id", humanNode.ID).Error; err != nil {
		t.Fatalf("update human current activity: %v", err)
	}
	waitingHuman.CurrentActivityID = &humanNode.ID
	if err := db.Create(&TicketAssignment{
		TicketID:        waitingHuman.ID,
		ActivityID:      humanNode.ID,
		ParticipantType: "user",
		UserID:          &assignee.ID,
		AssigneeID:      &assignee.ID,
		Status:          AssignmentPending,
		IsCurrent:       true,
	}).Error; err != nil {
		t.Fatalf("create human assignment: %v", err)
	}

	responses, err := svc.BuildResponses([]Ticket{terminal, aiDisabled, noCurrent, actionActivity, waitingHuman}, assignee.ID)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	byCode := map[string]TicketResponse{}
	for _, resp := range responses {
		byCode[resp.Code] = resp
	}
	if got := byCode[terminal.Code]; got.SmartState != "terminal" || got.NextStepSummary != "流程已结束" {
		t.Fatalf("terminal smart summary = %+v", got)
	}
	if got := byCode[aiDisabled.Code]; got.SmartState != "ai_disabled" || got.NextStepSummary != "AI 连续失败，等待人工接管" {
		t.Fatalf("ai_disabled smart summary = %+v", got)
	}
	if got := byCode[noCurrent.Code]; got.SmartState != "ai_reasoning" || got.CurrentOwnerType != "ai" || got.CurrentOwnerName != "AI 智能引擎" {
		t.Fatalf("no-current smart summary = %+v", got)
	}
	if got := byCode[actionActivity.Code]; got.SmartState != "action_running" || got.CurrentOwnerType != "system" || got.CurrentOwnerName != "自动化动作" {
		t.Fatalf("action-running smart summary = %+v", got)
	}
	if got := byCode[waitingHuman.Code]; got.SmartState != "waiting_human" || got.CurrentOwnerName != assignee.Username || !got.CanAct {
		t.Fatalf("waiting-human smart summary = %+v", got)
	}
}

func TestBuildResponses_WaitingHumanInProgressAssignmentStillShowsOwnerAndCanAct(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	assignee := model.User{Username: "claimed-owner", IsActive: true}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	if err := db.Create(&assignee).Error; err != nil {
		t.Fatalf("create assignee: %v", err)
	}
	catalog := ServiceCatalog{Name: "IT", Code: "it-smart-in-progress-owner"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{Name: "Smart In Progress Owner", Code: "smart-in-progress-owner", CatalogID: catalog.ID, EngineType: "smart", IsActive: true}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P2", Code: "p2", Value: 2, Color: "#0ea5e9", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	ticket := Ticket{
		Code:        "TICK-SMART-INPROGRESS-OWNER",
		Title:       "waiting human with in-progress assignment",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	humanNode := TicketActivity{TicketID: ticket.ID, Name: "人工审批", ActivityType: engine.NodeApprove, Status: engine.ActivityInProgress}
	if err := db.Create(&humanNode).Error; err != nil {
		t.Fatalf("create human node: %v", err)
	}
	if err := db.Model(&ticket).Where("id = ?", ticket.ID).Update("current_activity_id", humanNode.ID).Error; err != nil {
		t.Fatalf("update current activity: %v", err)
	}
	ticket.CurrentActivityID = &humanNode.ID
	claimedAt := time.Now()
	if err := db.Create(&TicketAssignment{
		TicketID:        ticket.ID,
		ActivityID:      humanNode.ID,
		ParticipantType: "user",
		UserID:          &assignee.ID,
		AssigneeID:      &assignee.ID,
		Status:          AssignmentInProgress,
		IsCurrent:       true,
		ClaimedAt:       &claimedAt,
	}).Error; err != nil {
		t.Fatalf("create in-progress assignment: %v", err)
	}

	responses, err := svc.BuildResponses([]Ticket{ticket}, assignee.ID)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	got := responses[0]
	if got.SmartState != "waiting_human" || got.CurrentOwnerName != assignee.Username || !got.CanAct {
		t.Fatalf("waiting-human in-progress smart summary = %+v", got)
	}
}

func TestBuildResponses_PopulateSmartSummaryFallbackContracts(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	catalog := ServiceCatalog{Name: "IT", Code: "it-smart-fallbacks"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{Name: "Smart Fallbacks", Code: "smart-fallbacks", CatalogID: catalog.ID, EngineType: "smart", IsActive: true}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P3", Code: "p3", Value: 3, Color: "#22c55e", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	missingActivityID := uint(999999)
	missingCurrent := Ticket{
		Code:              "TICK-SMART-MISSING-ACTIVITY",
		Title:             "missing current activity",
		ServiceID:         service.ID,
		EngineType:        "smart",
		Status:            TicketStatusDecisioning,
		PriorityID:        priority.ID,
		RequesterID:       requester.ID,
		CurrentActivityID: &missingActivityID,
		Source:            TicketSourceCatalog,
		SLAStatus:         SLAStatusOnTrack,
	}
	humanUnassigned := Ticket{
		Code:        "TICK-SMART-UNASSIGNED-HUMAN",
		Title:       "human pending without assignment",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	aiDecided := Ticket{
		Code:        "TICK-SMART-AI-DECIDED",
		Title:       "non human handoff completed",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	for _, ticket := range []*Ticket{&missingCurrent, &humanUnassigned, &aiDecided} {
		if err := db.Create(ticket).Error; err != nil {
			t.Fatalf("create ticket %s: %v", ticket.Code, err)
		}
	}

	humanNode := TicketActivity{TicketID: humanUnassigned.ID, Name: "", ActivityType: engine.NodeApprove, Status: engine.ActivityPending}
	if err := db.Create(&humanNode).Error; err != nil {
		t.Fatalf("create human node: %v", err)
	}
	if err := db.Model(&humanUnassigned).Where("id = ?", humanUnassigned.ID).Update("current_activity_id", humanNode.ID).Error; err != nil {
		t.Fatalf("set unassigned current activity: %v", err)
	}
	humanUnassigned.CurrentActivityID = &humanNode.ID

	decisionNode := TicketActivity{TicketID: aiDecided.ID, Name: "", ActivityType: engine.NodeProcess, Status: engine.ActivityCompleted}
	if err := db.Create(&decisionNode).Error; err != nil {
		t.Fatalf("create decision node: %v", err)
	}
	if err := db.Model(&aiDecided).Where("id = ?", aiDecided.ID).Update("current_activity_id", decisionNode.ID).Error; err != nil {
		t.Fatalf("set ai decided current activity: %v", err)
	}
	aiDecided.CurrentActivityID = &decisionNode.ID

	responses, err := svc.BuildResponses([]Ticket{missingCurrent, humanUnassigned, aiDecided}, 0)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	byCode := map[string]TicketResponse{}
	for _, resp := range responses {
		byCode[resp.Code] = resp
	}

	if got := byCode[missingCurrent.Code]; got.SmartState != "ai_reasoning" || got.CurrentOwnerName != "AI 智能引擎" || got.NextStepSummary != TicketStatusLabel(got.Status, got.Outcome) {
		t.Fatalf("missing-current smart summary = %+v", got)
	}
	if got := byCode[humanUnassigned.Code]; got.SmartState != "waiting_human" || got.CurrentOwnerName != "待分配" || got.NextStepSummary != engine.NodeApprove {
		t.Fatalf("unassigned human smart summary = %+v", got)
	}
	if got := byCode[aiDecided.Code]; got.SmartState != "ai_decided" || got.CurrentOwnerType != "ai" || got.CurrentOwnerName != "AI 智能引擎" || got.NextStepSummary != engine.NodeProcess {
		t.Fatalf("ai-decided smart summary = %+v", got)
	}
}

func TestBuildResponses_IgnoresForeignCurrentActivity(t *testing.T) {
	db := newTestDB(t)
	svc := &TicketService{ticketRepo: &TicketRepo{db: &database.DB{DB: db}}}

	requester := model.User{Username: "requester", IsActive: true}
	foreignAssignee := model.User{Username: "foreign-owner", IsActive: true}
	if err := db.Create(&requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	if err := db.Create(&foreignAssignee).Error; err != nil {
		t.Fatalf("create foreign assignee: %v", err)
	}
	catalog := ServiceCatalog{Name: "IT", Code: "it-smart-foreign-current"}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{Name: "Smart Foreign Current", Code: "smart-foreign-current", CatalogID: catalog.ID, EngineType: "smart", IsActive: true}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	priority := Priority{Name: "P2", Code: "p2", Value: 2, Color: "#0ea5e9", IsActive: true}
	if err := db.Create(&priority).Error; err != nil {
		t.Fatalf("create priority: %v", err)
	}

	ticket := Ticket{
		Code:        "TICK-SMART-FOREIGN-CURRENT",
		Title:       "foreign current activity",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	foreignTicket := Ticket{
		Code:        "TICK-SMART-FOREIGN-OWNER",
		Title:       "foreign owner ticket",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusWaitingHuman,
		PriorityID:  priority.ID,
		RequesterID: requester.ID,
		Source:      TicketSourceCatalog,
		SLAStatus:   SLAStatusOnTrack,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create primary ticket: %v", err)
	}
	if err := db.Create(&foreignTicket).Error; err != nil {
		t.Fatalf("create foreign ticket: %v", err)
	}

	foreignActivity := TicketActivity{TicketID: foreignTicket.ID, Name: "外部审批", ActivityType: engine.NodeApprove, Status: engine.ActivityPending}
	if err := db.Create(&foreignActivity).Error; err != nil {
		t.Fatalf("create foreign activity: %v", err)
	}
	if err := db.Create(&TicketAssignment{
		TicketID:   foreignTicket.ID,
		ActivityID: foreignActivity.ID,
		UserID:     &foreignAssignee.ID,
		AssigneeID: &foreignAssignee.ID,
		Status:     AssignmentPending,
		IsCurrent:  true,
	}).Error; err != nil {
		t.Fatalf("create foreign assignment: %v", err)
	}
	if err := db.Model(&ticket).Where("id = ?", ticket.ID).Update("current_activity_id", foreignActivity.ID).Error; err != nil {
		t.Fatalf("set foreign current activity: %v", err)
	}
	ticket.CurrentActivityID = &foreignActivity.ID

	responses, err := svc.BuildResponses([]Ticket{ticket}, 0)
	if err != nil {
		t.Fatalf("BuildResponses: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("responses len = %d, want 1", len(responses))
	}
	got := responses[0]
	if got.SmartState != "ai_reasoning" || got.CurrentOwnerName != "AI 智能引擎" {
		t.Fatalf("foreign current activity should be ignored, got %+v", got)
	}
	if got.DecisionExplanation == nil {
		t.Fatalf("decision explanation should still be built, got nil")
	}
	if got.DecisionExplanation.ActivityID != nil && *got.DecisionExplanation.ActivityID == foreignActivity.ID {
		t.Fatalf("decision explanation should not reuse foreign activity, got %+v", got.DecisionExplanation)
	}
}
