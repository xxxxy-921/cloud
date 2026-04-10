package repository

import (
	"time"

	"github.com/samber/do/v2"

	"metis/internal/database"
	"metis/internal/model"
)

type AuditLogRepo struct {
	db *database.DB
}

func NewAuditLog(i do.Injector) (*AuditLogRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &AuditLogRepo{db: db}, nil
}

func (r *AuditLogRepo) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

// AuditLogListParams holds query parameters for listing audit logs.
type AuditLogListParams struct {
	Category model.AuditCategory
	Keyword  string
	Action   string
	Resource string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PageSize int
}

// AuditLogListResult holds the paginated result.
type AuditLogListResult struct {
	Items []model.AuditLog `json:"items"`
	Total int64            `json:"total"`
}

func (r *AuditLogRepo) List(params AuditLogListParams) (*AuditLogListResult, error) {
	query := r.db.Model(&model.AuditLog{}).Where("category = ?", params.Category)

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		if params.Category == model.AuditCategoryAuth {
			query = query.Where("username LIKE ?", like)
		} else {
			query = query.Where("summary LIKE ?", like)
		}
	}

	if params.Action != "" {
		query = query.Where("action = ?", params.Action)
	}

	if params.Resource != "" {
		query = query.Where("resource = ?", params.Resource)
	}

	if params.DateFrom != nil {
		query = query.Where("created_at >= ?", *params.DateFrom)
	}

	if params.DateTo != nil {
		query = query.Where("created_at <= ?", *params.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	var logs []model.AuditLog
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	return &AuditLogListResult{Items: logs, Total: total}, nil
}

// DeleteBefore deletes audit logs of the given category older than the cutoff time.
// Returns the number of deleted records.
func (r *AuditLogRepo) DeleteBefore(category model.AuditCategory, before time.Time) (int64, error) {
	result := r.db.Where("category = ? AND created_at < ?", category, before).Delete(&model.AuditLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// Migrate creates composite indexes that GORM tags alone cannot express.
func (r *AuditLogRepo) Migrate() error {
	return r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_category_created ON audit_logs(category, created_at);
		CREATE INDEX IF NOT EXISTS idx_audit_user_created ON audit_logs(user_id, created_at);
		CREATE INDEX IF NOT EXISTS idx_audit_resource_id ON audit_logs(resource, resource_id);
	`).Error
}
