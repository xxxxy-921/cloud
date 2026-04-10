package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"metis/internal/model"
)

// HistoryCleanupTask is the built-in task that cleans up old execution records.
var HistoryCleanupTask = &TaskDef{
	Name:        "scheduler-history-cleanup",
	Type:        TypeScheduled,
	Description: "清理过期的任务执行历史记录",
	CronExpr:    "0 4 * * *", // daily at 4:00 AM
	Timeout:     60 * time.Second,
	MaxRetries:  1,
	// Handler is set dynamically via SetCleanupHandler because it needs access to sysconfig repo
}

// ConfigReader reads a system config value by key.
type ConfigReader func(key string) (string, error)

// SetCleanupHandler sets up the history cleanup handler with a config reader.
func SetCleanupHandler(task *TaskDef, configReader ConfigReader, store *GormStore) {
	task.Handler = func(ctx context.Context, _ json.RawMessage) error {
		val, err := configReader("scheduler.history_retention_days")
		if err != nil || val == "" {
			slog.Info("scheduler: history cleanup skipped (no config)")
			return nil
		}

		days, err := strconv.Atoi(val)
		if err != nil || days <= 0 {
			slog.Info("scheduler: history cleanup skipped", "value", val)
			return nil
		}

		before := time.Now().AddDate(0, 0, -days)
		deleted, err := store.DeleteOlderThan(ctx, before)
		if err != nil {
			return err
		}

		slog.Info("scheduler: history cleanup done", "deleted", deleted, "olderThan", before.Format(time.DateOnly))
		return nil
	}
}

// SeedHistoryRetentionConfig creates the default config entry if it doesn't exist.
func SeedHistoryRetentionConfig(configSaver func(cfg *model.SystemConfig) error) error {
	return configSaver(&model.SystemConfig{
		Key:    "scheduler.history_retention_days",
		Value:  "30",
		Remark: "任务执行历史保留天数，0 表示永不清理",
	})
}

// BlacklistCleanupTask cleans up expired entries from the in-memory token blacklist.
var BlacklistCleanupTask = &TaskDef{
	Name:        "blacklist-cleanup",
	Type:        TypeScheduled,
	Description: "清理已过期的Token黑名单条目",
	CronExpr:    "*/5 * * * *", // every 5 minutes
	Timeout:     10 * time.Second,
	MaxRetries:  1,
	// Handler is set dynamically via SetBlacklistCleanupHandler
}

// BlacklistCleaner is a function that cleans up expired blacklist entries.
type BlacklistCleaner func() int

// SetBlacklistCleanupHandler sets up the blacklist cleanup handler.
func SetBlacklistCleanupHandler(task *TaskDef, cleaner BlacklistCleaner) {
	task.Handler = func(ctx context.Context, _ json.RawMessage) error {
		removed := cleaner()
		if removed > 0 {
			slog.Info("scheduler: blacklist cleanup done", "removed", removed)
		}
		return nil
	}
}

// ExpiredTokenCleanupTask hard-deletes old refresh token records from the database.
var ExpiredTokenCleanupTask = &TaskDef{
	Name:        "expired-token-cleanup",
	Type:        TypeScheduled,
	Description: "清理过期或已撤销的会话Token记录",
	CronExpr:    "0 3 * * *", // daily at 3:00 AM
	Timeout:     60 * time.Second,
	MaxRetries:  1,
	// Handler is set dynamically via SetExpiredTokenCleanupHandler
}

// TokenDeleter is a function that deletes expired/revoked tokens older than the given duration.
type TokenDeleter func(olderThan time.Duration) (int64, error)

// SetExpiredTokenCleanupHandler sets up the expired token cleanup handler.
func SetExpiredTokenCleanupHandler(task *TaskDef, deleter TokenDeleter) {
	task.Handler = func(ctx context.Context, _ json.RawMessage) error {
		deleted, err := deleter(7 * 24 * time.Hour) // 7 days
		if err != nil {
			return err
		}
		if deleted > 0 {
			slog.Info("scheduler: expired token cleanup done", "deleted", deleted)
		}
		return nil
	}
}

// AuditLogCleanupTask cleans up expired audit log entries based on per-category retention settings.
var AuditLogCleanupTask = &TaskDef{
	Name:        "audit-log-cleanup",
	Type:        TypeScheduled,
	Description: "按分类保留策略清理过期审计日志",
	CronExpr:    "0 3 * * *", // daily at 3:00 AM
	Timeout:     120 * time.Second,
	MaxRetries:  1,
	// Handler is set dynamically via SetAuditLogCleanupHandler
}

// AuditLogCleaner is a function that cleans up expired audit logs and returns a summary.
type AuditLogCleaner func() string

// SetAuditLogCleanupHandler sets up the audit log cleanup handler.
func SetAuditLogCleanupHandler(task *TaskDef, cleaner AuditLogCleaner) {
	task.Handler = func(ctx context.Context, _ json.RawMessage) error {
		summary := cleaner()
		slog.Info("scheduler: audit log cleanup done", "summary", summary)
		return nil
	}
}
