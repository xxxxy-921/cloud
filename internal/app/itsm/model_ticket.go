package itsm

import (
	"time"

	"metis/internal/model"
)

// Ticket status constants
const (
	TicketStatusPending         = "pending"
	TicketStatusInProgress      = "in_progress"
	TicketStatusWaitingApproval = "waiting_approval"
	TicketStatusWaitingAction   = "waiting_action"
	TicketStatusCompleted       = "completed"
	TicketStatusFailed          = "failed"
	TicketStatusCancelled       = "cancelled"
)

// Ticket source constants
const (
	TicketSourceCatalog = "catalog"
	TicketSourceAgent   = "agent"
)

// SLA status constants
const (
	SLAStatusOnTrack          = "on_track"
	SLAStatusBreachedResponse = "breached_response"
	SLAStatusBreachedResolve  = "breached_resolution"
)

// Ticket 工单
type Ticket struct {
	model.BaseModel
	Code                  string     `json:"code" gorm:"size:32;uniqueIndex;not null"`
	Title                 string     `json:"title" gorm:"size:256;not null"`
	Description           string     `json:"description" gorm:"type:text"`
	ServiceID             uint       `json:"serviceId" gorm:"not null;index"`
	EngineType            string     `json:"engineType" gorm:"size:16;not null"`
	Status                string     `json:"status" gorm:"size:32;not null;default:pending;index"`
	PriorityID            uint       `json:"priorityId" gorm:"not null;index"`
	RequesterID           uint       `json:"requesterId" gorm:"not null;index"`
	AssigneeID            *uint      `json:"assigneeId" gorm:"index"`
	CurrentActivityID     *uint      `json:"currentActivityId" gorm:"index"`
	Source                string     `json:"source" gorm:"size:16;not null;default:catalog"` // catalog | agent
	AgentSessionID        *uint      `json:"agentSessionId" gorm:"index"`
	FormData              JSONField  `json:"formData" gorm:"type:text"`
	WorkflowJSON          JSONField  `json:"workflowJson" gorm:"type:text"` // snapshot of workflow at creation
	SLAResponseDeadline   *time.Time `json:"slaResponseDeadline"`
	SLAResolutionDeadline *time.Time `json:"slaResolutionDeadline"`
	SLAStatus             string     `json:"slaStatus" gorm:"size:32;default:on_track"`
	FinishedAt            *time.Time `json:"finishedAt"`
}

func (Ticket) TableName() string { return "itsm_tickets" }

type TicketResponse struct {
	ID                    uint       `json:"id"`
	Code                  string     `json:"code"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	ServiceID             uint       `json:"serviceId"`
	EngineType            string     `json:"engineType"`
	Status                string     `json:"status"`
	PriorityID            uint       `json:"priorityId"`
	RequesterID           uint       `json:"requesterId"`
	AssigneeID            *uint      `json:"assigneeId"`
	CurrentActivityID     *uint      `json:"currentActivityId"`
	Source                string     `json:"source"`
	AgentSessionID        *uint      `json:"agentSessionId"`
	FormData              JSONField  `json:"formData"`
	WorkflowJSON          JSONField  `json:"workflowJson"`
	SLAResponseDeadline   *time.Time `json:"slaResponseDeadline"`
	SLAResolutionDeadline *time.Time `json:"slaResolutionDeadline"`
	SLAStatus             string     `json:"slaStatus"`
	FinishedAt            *time.Time `json:"finishedAt"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

func (t *Ticket) ToResponse() TicketResponse {
	return TicketResponse{
		ID:                    t.ID,
		Code:                  t.Code,
		Title:                 t.Title,
		Description:           t.Description,
		ServiceID:             t.ServiceID,
		EngineType:            t.EngineType,
		Status:                t.Status,
		PriorityID:            t.PriorityID,
		RequesterID:           t.RequesterID,
		AssigneeID:            t.AssigneeID,
		CurrentActivityID:     t.CurrentActivityID,
		Source:                t.Source,
		AgentSessionID:        t.AgentSessionID,
		FormData:              t.FormData,
		WorkflowJSON:          t.WorkflowJSON,
		SLAResponseDeadline:   t.SLAResponseDeadline,
		SLAResolutionDeadline: t.SLAResolutionDeadline,
		SLAStatus:             t.SLAStatus,
		FinishedAt:            t.FinishedAt,
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
	}
}

// IsTerminal returns true if the ticket is in a terminal state.
func (t *Ticket) IsTerminal() bool {
	return t.Status == TicketStatusCompleted ||
		t.Status == TicketStatusFailed ||
		t.Status == TicketStatusCancelled
}

// TicketActivity 工单活动（工作流步骤）
type TicketActivity struct {
	model.BaseModel
	TicketID          uint      `json:"ticketId" gorm:"not null;index"`
	Name              string    `json:"name" gorm:"size:128"`
	ActivityType      string    `json:"activityType" gorm:"size:16"` // form | approve | process | action | end
	Status            string    `json:"status" gorm:"size:16;default:pending"`
	NodeID            string    `json:"nodeId" gorm:"size:64"`              // classic mode: workflow_json node ID
	ExecutionMode     string    `json:"executionMode" gorm:"size:16"`       // single | parallel | serial
	FormSchema        JSONField `json:"formSchema" gorm:"type:text"`
	FormData          JSONField `json:"formData" gorm:"type:text"`
	TransitionOutcome string    `json:"transitionOutcome" gorm:"size:16"`   // submit | approve | reject | success | failure
	AIDecision        JSONField `json:"aiDecision" gorm:"type:text"`        // smart mode
	AIReasoning       string    `json:"aiReasoning" gorm:"type:text"`       // smart mode
	AIConfidence      float64   `json:"aiConfidence" gorm:"default:0"`      // smart mode
	OverriddenBy      *uint     `json:"overriddenBy"`                       // operator ID when overridden
	DecisionReasoning string    `json:"decisionReasoning" gorm:"type:text"`
	StartedAt         *time.Time `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt"`
}

func (TicketActivity) TableName() string { return "itsm_ticket_activities" }

// TicketAssignment 工单参与人分配
type TicketAssignment struct {
	model.BaseModel
	TicketID        uint       `json:"ticketId" gorm:"not null;index"`
	ActivityID      uint       `json:"activityId" gorm:"not null;index"`
	ParticipantType string     `json:"participantType" gorm:"size:32;not null"` // user | requester_manager | position | department
	UserID          *uint      `json:"userId" gorm:"index"`
	PositionID      *uint      `json:"positionId" gorm:"index"`
	DepartmentID    *uint      `json:"departmentId" gorm:"index"`
	AssigneeID      *uint      `json:"assigneeId" gorm:"index"` // actual claimed person
	Status          string     `json:"status" gorm:"size:16;default:pending"`
	Sequence        int        `json:"sequence" gorm:"default:0"`
	IsCurrent       bool       `json:"isCurrent" gorm:"default:false"`
	ClaimedAt       *time.Time `json:"claimedAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
}

func (TicketAssignment) TableName() string { return "itsm_ticket_assignments" }

// TicketTimeline 工单时间线
type TicketTimeline struct {
	model.BaseModel
	TicketID   uint      `json:"ticketId" gorm:"not null;index"`
	ActivityID *uint     `json:"activityId" gorm:"index"`
	OperatorID uint      `json:"operatorId" gorm:"not null"`
	EventType  string    `json:"eventType" gorm:"size:32;not null"`
	Message    string    `json:"message" gorm:"size:512"`
	Details    JSONField `json:"details" gorm:"type:text"`
	Reasoning  string    `json:"reasoning" gorm:"type:text"`
}

func (TicketTimeline) TableName() string { return "itsm_ticket_timelines" }

type TicketTimelineResponse struct {
	ID         uint      `json:"id"`
	TicketID   uint      `json:"ticketId"`
	ActivityID *uint     `json:"activityId"`
	OperatorID uint      `json:"operatorId"`
	EventType  string    `json:"eventType"`
	Message    string    `json:"message"`
	Details    JSONField `json:"details"`
	Reasoning  string    `json:"reasoning"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (t *TicketTimeline) ToResponse() TicketTimelineResponse {
	return TicketTimelineResponse{
		ID:         t.ID,
		TicketID:   t.TicketID,
		ActivityID: t.ActivityID,
		OperatorID: t.OperatorID,
		EventType:  t.EventType,
		Message:    t.Message,
		Details:    t.Details,
		Reasoning:  t.Reasoning,
		CreatedAt:  t.CreatedAt,
	}
}

// TicketActionExecution 动作执行记录
type TicketActionExecution struct {
	model.BaseModel
	TicketID        uint      `json:"ticketId" gorm:"not null;index"`
	ActivityID      uint      `json:"activityId" gorm:"not null;index"`
	ServiceActionID uint      `json:"serviceActionId" gorm:"not null"`
	Status          string    `json:"status" gorm:"size:16;default:pending"` // pending | success | failed
	RequestPayload  JSONField `json:"requestPayload" gorm:"type:text"`
	ResponsePayload JSONField `json:"responsePayload" gorm:"type:text"`
	FailureReason   string    `json:"failureReason" gorm:"type:text"`
	RetryCount      int       `json:"retryCount" gorm:"default:0"`
}

func (TicketActionExecution) TableName() string { return "itsm_ticket_action_executions" }

// TicketLink 工单关联
type TicketLink struct {
	model.BaseModel
	ParentTicketID uint   `json:"parentTicketId" gorm:"not null;index"`
	ChildTicketID  uint   `json:"childTicketId" gorm:"not null;index"`
	LinkType       string `json:"linkType" gorm:"size:16;not null"` // related | caused_by | blocked_by
}

func (TicketLink) TableName() string { return "itsm_ticket_links" }

// PostMortem 故障复盘
type PostMortem struct {
	model.BaseModel
	TicketID       uint      `json:"ticketId" gorm:"uniqueIndex;not null"`
	RootCause      string    `json:"rootCause" gorm:"type:text"`
	ImpactSummary  string    `json:"impactSummary" gorm:"type:text"`
	ActionItems    JSONField `json:"actionItems" gorm:"type:text"`
	LessonsLearned string    `json:"lessonsLearned" gorm:"type:text"`
	CreatedBy      uint      `json:"createdBy" gorm:"not null"`
}

func (PostMortem) TableName() string { return "itsm_post_mortems" }
