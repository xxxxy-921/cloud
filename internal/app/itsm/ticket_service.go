package itsm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app/itsm/engine"
)

var (
	ErrTicketNotFound   = errors.New("ticket not found")
	ErrTicketTerminal   = errors.New("ticket is in a terminal state and cannot be modified")
	ErrServiceNotActive = errors.New("service is not active")
	ErrActivityNotOwner = errors.New("only the assignee or admin can progress this activity")
	ErrActivityNotWait  = errors.New("signal is only allowed on wait nodes")
)

type TicketService struct {
	ticketRepo   *TicketRepo
	timelineRepo *TimelineRepo
	serviceRepo  *ServiceDefRepo
	slaRepo      *SLATemplateRepo
	priorityRepo *PriorityRepo
	engine       *engine.ClassicEngine
}

func NewTicketService(i do.Injector) (*TicketService, error) {
	return &TicketService{
		ticketRepo:   do.MustInvoke[*TicketRepo](i),
		timelineRepo: do.MustInvoke[*TimelineRepo](i),
		serviceRepo:  do.MustInvoke[*ServiceDefRepo](i),
		slaRepo:      do.MustInvoke[*SLATemplateRepo](i),
		priorityRepo: do.MustInvoke[*PriorityRepo](i),
		engine:       do.MustInvoke[*engine.ClassicEngine](i),
	}, nil
}

type CreateTicketInput struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ServiceID   uint      `json:"serviceId"`
	PriorityID  uint      `json:"priorityId"`
	FormData    JSONField `json:"formData"`
}

func (s *TicketService) Create(input CreateTicketInput, requesterID uint) (*Ticket, error) {
	// Validate service
	svc, err := s.serviceRepo.FindByID(input.ServiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceDefNotFound
		}
		return nil, err
	}
	if !svc.IsActive {
		return nil, ErrServiceNotActive
	}

	// Validate priority
	if _, err := s.priorityRepo.FindByID(input.PriorityID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPriorityNotFound
		}
		return nil, err
	}

	// For classic engine, validate workflow_json before creating ticket
	if svc.EngineType == "classic" {
		if len(svc.WorkflowJSON) == 0 {
			return nil, errors.New("服务未配置工作流")
		}
		if errs := engine.ValidateWorkflow(json.RawMessage(svc.WorkflowJSON)); len(errs) > 0 {
			return nil, errors.New("工作流校验失败: " + errs[0].Message)
		}
	}

	// Generate ticket code
	code, err := s.ticketRepo.NextCode()
	if err != nil {
		return nil, err
	}

	ticket := &Ticket{
		Code:        code,
		Title:       input.Title,
		Description: input.Description,
		ServiceID:   input.ServiceID,
		EngineType:  svc.EngineType,
		Status:      TicketStatusPending,
		PriorityID:  input.PriorityID,
		RequesterID: requesterID,
		Source:      TicketSourceCatalog,
		FormData:    input.FormData,
		SLAStatus:   SLAStatusOnTrack,
	}

	// Snapshot workflow_json for classic engine
	if svc.EngineType == "classic" {
		ticket.WorkflowJSON = svc.WorkflowJSON
	}

	// Calculate SLA deadlines
	if svc.SLAID != nil {
		sla, err := s.slaRepo.FindByID(*svc.SLAID)
		if err == nil {
			now := time.Now()
			responseDeadline := now.Add(time.Duration(sla.ResponseMinutes) * time.Minute)
			resolutionDeadline := now.Add(time.Duration(sla.ResolutionMinutes) * time.Minute)
			ticket.SLAResponseDeadline = &responseDeadline
			ticket.SLAResolutionDeadline = &resolutionDeadline
		}
	}

	// Create in transaction
	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		// Record timeline
		tl := &TicketTimeline{
			TicketID:   ticket.ID,
			OperatorID: requesterID,
			EventType:  "ticket_created",
			Message:    "工单已创建",
		}
		if err := tx.Create(tl).Error; err != nil {
			return err
		}

		// For classic engine, start the workflow
		if svc.EngineType == "classic" {
			return s.engine.Start(context.Background(), tx, engine.StartParams{
				TicketID:     ticket.ID,
				WorkflowJSON: json.RawMessage(ticket.WorkflowJSON),
				RequesterID:  requesterID,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(ticket.ID)
}

// Progress advances a classic workflow ticket. The operator must be the assignee or have admin privileges.
func (s *TicketService) Progress(ticketID uint, activityID uint, outcome string, result json.RawMessage, operatorID uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	if t.IsTerminal() {
		return nil, ErrTicketTerminal
	}

	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		return s.engine.Progress(context.Background(), tx, engine.ProgressParams{
			TicketID:   ticketID,
			ActivityID: activityID,
			Outcome:    outcome,
			Result:     result,
			OperatorID: operatorID,
		})
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(ticketID)
}

// Signal triggers a wait node's continuation from an external source.
func (s *TicketService) Signal(ticketID uint, activityID uint, outcome string, data json.RawMessage, operatorID uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	if t.IsTerminal() {
		return nil, ErrTicketTerminal
	}

	// Verify the activity is a wait node and is pending
	var activity TicketActivity
	if err := s.ticketRepo.DB().First(&activity, activityID).Error; err != nil {
		return nil, engine.ErrActivityNotFound
	}
	if activity.ActivityType != engine.NodeWait {
		return nil, ErrActivityNotWait
	}
	if activity.Status != engine.ActivityPending && activity.Status != engine.ActivityInProgress {
		return nil, engine.ErrActivityNotActive
	}

	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		return s.engine.Progress(context.Background(), tx, engine.ProgressParams{
			TicketID:   ticketID,
			ActivityID: activityID,
			Outcome:    outcome,
			Result:     data,
			OperatorID: operatorID,
		})
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(ticketID)
}

// GetActivities returns all activities for a ticket.
func (s *TicketService) GetActivities(ticketID uint) ([]TicketActivity, error) {
	var activities []TicketActivity
	if err := s.ticketRepo.DB().Where("ticket_id = ?", ticketID).Order("id ASC").Find(&activities).Error; err != nil {
		return nil, err
	}
	return activities, nil
}

func (s *TicketService) Get(id uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *TicketService) List(params TicketListParams) ([]Ticket, int64, error) {
	return s.ticketRepo.List(params)
}

func (s *TicketService) Mine(requesterID uint, status string, page, pageSize int) ([]Ticket, int64, error) {
	params := TicketListParams{
		RequesterID: &requesterID,
		Status:      status,
		Page:        page,
		PageSize:    pageSize,
	}
	return s.ticketRepo.List(params)
}

func (s *TicketService) Todo(assigneeID uint, page, pageSize int) ([]Ticket, int64, error) {
	return s.ticketRepo.ListTodo(assigneeID, page, pageSize)
}

func (s *TicketService) History(params HistoryListParams) ([]Ticket, int64, error) {
	return s.ticketRepo.ListHistory(params)
}

func (s *TicketService) Assign(id uint, assigneeID uint, operatorID uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	if t.IsTerminal() {
		return nil, ErrTicketTerminal
	}

	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"assignee_id": assigneeID,
			"status":      TicketStatusInProgress,
		}
		if err := s.ticketRepo.UpdateInTx(tx, id, updates); err != nil {
			return err
		}
		tl := &TicketTimeline{
			TicketID:   id,
			OperatorID: operatorID,
			EventType:  "ticket_assigned",
			Message:    "工单已指派",
		}
		return s.timelineRepo.CreateInTx(tx, tl)
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(id)
}

func (s *TicketService) Complete(id uint, operatorID uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	if t.IsTerminal() {
		return nil, ErrTicketTerminal
	}

	now := time.Now()
	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":      TicketStatusCompleted,
			"finished_at": now,
		}
		if err := s.ticketRepo.UpdateInTx(tx, id, updates); err != nil {
			return err
		}
		tl := &TicketTimeline{
			TicketID:   id,
			OperatorID: operatorID,
			EventType:  "ticket_completed",
			Message:    "工单已完成",
		}
		return s.timelineRepo.CreateInTx(tx, tl)
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(id)
}

type CancelTicketInput struct {
	Reason string `json:"reason"`
}

func (s *TicketService) Cancel(id uint, reason string, operatorID uint) (*Ticket, error) {
	t, err := s.ticketRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	if t.IsTerminal() {
		return nil, ErrTicketTerminal
	}

	// For classic engine, use engine's Cancel to properly clean up activities
	if t.EngineType == "classic" {
		if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
			return s.engine.Cancel(context.Background(), tx, engine.CancelParams{
				TicketID:   id,
				Reason:     reason,
				OperatorID: operatorID,
			})
		}); err != nil {
			return nil, err
		}
		return s.ticketRepo.FindByID(id)
	}

	// Manual mode: original cancel logic
	now := time.Now()
	if err := s.ticketRepo.DB().Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":      TicketStatusCancelled,
			"finished_at": now,
		}
		if err := s.ticketRepo.UpdateInTx(tx, id, updates); err != nil {
			return err
		}
		tl := &TicketTimeline{
			TicketID:   id,
			OperatorID: operatorID,
			EventType:  "ticket_cancelled",
			Message:    "工单已取消: " + reason,
		}
		return s.timelineRepo.CreateInTx(tx, tl)
	}); err != nil {
		return nil, err
	}

	return s.ticketRepo.FindByID(id)
}
