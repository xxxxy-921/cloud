package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/samber/do/v2"

	"metis/internal/model"
	"metis/internal/repository"
)

type AuditLogService struct {
	repo          *repository.AuditLogRepo
	sysConfigRepo *repository.SysConfigRepo
}

func NewAuditLog(i do.Injector) (*AuditLogService, error) {
	return &AuditLogService{
		repo:          do.MustInvoke[*repository.AuditLogRepo](i),
		sysConfigRepo: do.MustInvoke[*repository.SysConfigRepo](i),
	}, nil
}

// Log asynchronously writes an audit log entry. Errors are logged but never propagated.
func (s *AuditLogService) Log(entry model.AuditLog) {
	go func() {
		if entry.Level == "" {
			entry.Level = model.AuditLevelInfo
		}
		if err := s.repo.Create(&entry); err != nil {
			slog.Error("audit log write failed",
				"action", entry.Action,
				"category", entry.Category,
				"error", err,
			)
		}
	}()
}

// List returns paginated audit logs.
func (s *AuditLogService) List(params repository.AuditLogListParams) (*repository.AuditLogListResult, error) {
	return s.repo.List(params)
}

// Cleanup deletes expired audit logs per category based on retention settings.
// Returns a summary string describing what was deleted.
func (s *AuditLogService) Cleanup() string {
	categories := []struct {
		category model.AuditCategory
		key      string
	}{
		{model.AuditCategoryAuth, "audit.retention_days_auth"},
		{model.AuditCategoryOperation, "audit.retention_days_operation"},
		{model.AuditCategoryApplication, "audit.retention_days_application"},
	}

	var parts []string
	for _, c := range categories {
		days := s.getRetentionDays(c.key)
		if days <= 0 {
			continue
		}

		before := time.Now().AddDate(0, 0, -days)
		deleted, err := s.repo.DeleteBefore(c.category, before)
		if err != nil {
			slog.Error("audit log cleanup failed", "category", c.category, "error", err)
			continue
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d 条", c.category, deleted))
		}
	}

	if len(parts) == 0 {
		return "无过期日志需要清理"
	}

	summary := "清理过期审计日志: "
	for i, p := range parts {
		if i > 0 {
			summary += ", "
		}
		summary += p
	}
	return summary
}

func (s *AuditLogService) getRetentionDays(key string) int {
	cfg, err := s.sysConfigRepo.Get(key)
	if err != nil {
		return 0
	}
	v, err := strconv.Atoi(cfg.Value)
	if err != nil {
		return 0
	}
	return v
}
