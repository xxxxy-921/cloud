package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/repository"
)

func newTestDBForAuditLogService(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.AuditLog{}, &model.SystemConfig{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedSystemConfigForAuditLog(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	if err := db.Create(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
		t.Fatalf("seed system config: %v", err)
	}
}

func seedAuditLogDirect(t *testing.T, db *gorm.DB, category model.AuditCategory, action, summary string, createdAt time.Time) {
	t.Helper()
	if err := db.Create(&model.AuditLog{
		Category:  category,
		Action:    action,
		Summary:   summary,
		Level:     model.AuditLevelInfo,
		CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("seed audit log: %v", err)
	}
}

func newAuditLogServiceForTest(t *testing.T, db *gorm.DB) *AuditLogService {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, NewAuditLog)
	return do.MustInvoke[*AuditLogService](injector)
}

func waitForAuditLog(t *testing.T, db *gorm.DB, expectedCount int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int64
		db.Model(&model.AuditLog{}).Count(&count)
		if count >= expectedCount {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for audit log count %d", expectedCount)
}

func TestAuditLogServiceLog_PersistsAsync(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)

	svc.Log(model.AuditLog{
		Category: model.AuditCategoryOperation,
		Action:   "create",
		Summary:  "created resource",
		Level:    model.AuditLevelWarn,
	})

	waitForAuditLog(t, db, 1, 100*time.Millisecond)

	var logs []model.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("find logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Action != "create" || logs[0].Level != model.AuditLevelWarn {
		t.Fatalf("unexpected log: %+v", logs[0])
	}
}

func TestAuditLogServiceLog_DefaultsLevelToInfo(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)

	svc.Log(model.AuditLog{
		Category: model.AuditCategoryOperation,
		Action:   "update",
		Summary:  "updated resource",
	})

	waitForAuditLog(t, db, 1, 100*time.Millisecond)

	var log model.AuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("find log: %v", err)
	}
	if log.Level != model.AuditLevelInfo {
		t.Fatalf("expected level info, got %s", log.Level)
	}
}

func TestAuditLogServiceLog_ErrorsSwallowed(t *testing.T) {
	// Errors are logged but not returned; the call itself should not panic or block.
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)

	// This should not panic or return error.
	svc.Log(model.AuditLog{
		Category: model.AuditCategoryOperation,
		Action:   "action",
		Summary:  "summary",
	})

	// Even with an invalid future state, Log() itself is fire-and-forget.
	waitForAuditLog(t, db, 1, 100*time.Millisecond)
}

func TestAuditLogServiceList_DelegatesToRepo(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)
	now := time.Now()
	seedAuditLogDirect(t, db, model.AuditCategoryAuth, "login", "login ok", now)

	result, err := svc.List(repository.AuditLogListParams{
		Category: model.AuditCategoryAuth,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Action != "login" {
		t.Fatalf("unexpected items: %+v", result.Items)
	}
}

func TestAuditLogServiceCleanup_DeletesExpiredByCategory(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)
	now := time.Now()

	seedSystemConfigForAuditLog(t, db, "audit.retention_days_auth", "7")
	seedSystemConfigForAuditLog(t, db, "audit.retention_days_operation", "30")

	// auth: older than 7 days should be deleted
	seedAuditLogDirect(t, db, model.AuditCategoryAuth, "login", "old auth", now.AddDate(0, 0, -10))
	seedAuditLogDirect(t, db, model.AuditCategoryAuth, "login", "recent auth", now.AddDate(0, 0, -1))

	// operation: older than 30 days should be deleted
	seedAuditLogDirect(t, db, model.AuditCategoryOperation, "create", "old op", now.AddDate(0, 0, -40))
	seedAuditLogDirect(t, db, model.AuditCategoryOperation, "create", "recent op", now.AddDate(0, 0, -5))

	summary := svc.Cleanup()

	// Verify remaining logs
	var authCount, opCount int64
	db.Model(&model.AuditLog{}).Where("category = ?", model.AuditCategoryAuth).Count(&authCount)
	db.Model(&model.AuditLog{}).Where("category = ?", model.AuditCategoryOperation).Count(&opCount)

	if authCount != 1 {
		t.Fatalf("expected 1 auth log, got %d", authCount)
	}
	if opCount != 1 {
		t.Fatalf("expected 1 operation log, got %d", opCount)
	}
	if summary == "" || summary == "无过期日志需要清理" {
		t.Fatalf("expected non-empty summary with deletions, got %s", summary)
	}
}

func TestAuditLogServiceCleanup_NoRetentionConfig(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)
	now := time.Now()
	seedAuditLogDirect(t, db, model.AuditCategoryAuth, "login", "old auth", now.AddDate(0, 0, -100))

	summary := svc.Cleanup()
	if summary != "无过期日志需要清理" {
		t.Fatalf("expected no cleanup summary, got %s", summary)
	}

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected log to remain when no retention configured, got %d", count)
	}
}

func TestAuditLogServiceCleanup_ZeroRetention(t *testing.T) {
	db := newTestDBForAuditLogService(t)
	svc := newAuditLogServiceForTest(t, db)
	now := time.Now()
	seedSystemConfigForAuditLog(t, db, "audit.retention_days_auth", "0")
	seedAuditLogDirect(t, db, model.AuditCategoryAuth, "login", "old auth", now.AddDate(0, 0, -100))

	summary := svc.Cleanup()
	if summary != "无过期日志需要清理" {
		t.Fatalf("expected no cleanup summary for zero retention, got %s", summary)
	}

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected log to remain with zero retention, got %d", count)
	}
}
