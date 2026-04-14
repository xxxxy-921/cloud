package itsm

import (
	"fmt"
	"sync"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type TicketRepo struct {
	db     *database.DB
	seqMu  sync.Mutex
}

func NewTicketRepo(i do.Injector) (*TicketRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &TicketRepo{db: db}, nil
}

// NextCode generates the next ticket code (TICK-XXXXXX) in a concurrency-safe manner.
func (r *TicketRepo) NextCode() (string, error) {
	r.seqMu.Lock()
	defer r.seqMu.Unlock()

	var maxID uint
	if err := r.db.Model(&Ticket{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("TICK-%06d", maxID+1), nil
}

func (r *TicketRepo) Create(t *Ticket) error {
	return r.db.Create(t).Error
}

func (r *TicketRepo) FindByID(id uint) (*Ticket, error) {
	var t Ticket
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepo) FindByCode(code string) (*Ticket, error) {
	var t Ticket
	if err := r.db.Where("code = ?", code).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepo) Update(id uint, updates map[string]any) error {
	return r.db.Model(&Ticket{}).Where("id = ?", id).Updates(updates).Error
}

type TicketListParams struct {
	Keyword     string
	Status      string
	PriorityID  *uint
	ServiceID   *uint
	AssigneeID  *uint
	RequesterID *uint
	Page        int
	PageSize    int
	DeptScope   *[]uint
}

func (r *TicketRepo) List(params TicketListParams) ([]Ticket, int64, error) {
	query := r.db.Model(&Ticket{})

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("code LIKE ? OR title LIKE ? OR description LIKE ?", like, like, like)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.PriorityID != nil {
		query = query.Where("priority_id = ?", *params.PriorityID)
	}
	if params.ServiceID != nil {
		query = query.Where("service_id = ?", *params.ServiceID)
	}
	if params.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *params.AssigneeID)
	}
	if params.RequesterID != nil {
		query = query.Where("requester_id = ?", *params.RequesterID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	var items []Ticket
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListTodo returns active tickets assigned to a user, ordered by priority value then creation time.
func (r *TicketRepo) ListTodo(assigneeID uint, page, pageSize int) ([]Ticket, int64, error) {
	activeStatuses := []string{TicketStatusPending, TicketStatusInProgress, TicketStatusWaitingApproval}
	query := r.db.Model(&Ticket{}).
		Where("assignee_id = ? AND status IN ?", assigneeID, activeStatuses)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var items []Ticket
	offset := (page - 1) * pageSize
	if err := r.db.
		Joins("LEFT JOIN itsm_priorities ON itsm_priorities.id = itsm_tickets.priority_id").
		Where("itsm_tickets.assignee_id = ? AND itsm_tickets.status IN ?", assigneeID, activeStatuses).
		Order("itsm_priorities.value ASC, itsm_tickets.created_at ASC").
		Offset(offset).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type HistoryListParams struct {
	AssigneeID *uint
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

// ListHistory returns terminal-state tickets with optional filters.
func (r *TicketRepo) ListHistory(params HistoryListParams) ([]Ticket, int64, error) {
	terminalStatuses := []string{TicketStatusCompleted, TicketStatusFailed, TicketStatusCancelled}
	query := r.db.Model(&Ticket{}).Where("status IN ?", terminalStatuses)

	if params.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *params.AssigneeID)
	}
	if params.StartDate != nil {
		query = query.Where("finished_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		query = query.Where("finished_at <= ?", *params.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	var items []Ticket
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("finished_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateInTx performs an update within a provided transaction.
func (r *TicketRepo) UpdateInTx(tx *gorm.DB, id uint, updates map[string]any) error {
	return tx.Model(&Ticket{}).Where("id = ?", id).Updates(updates).Error
}

func (r *TicketRepo) DB() *gorm.DB {
	return r.db.DB
}
