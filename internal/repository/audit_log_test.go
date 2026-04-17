package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

func newTestDBForAuditLog(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedAuditLog(t *testing.T, db *gorm.DB, category model.AuditCategory, action, resource, summary, username string, createdAt time.Time) *model.AuditLog {
	t.Helper()
	log := &model.AuditLog{
		Category:  category,
		Action:    action,
		Resource:  resource,
		Summary:   summary,
		Username:  username,
		Level:     model.AuditLevelInfo,
		CreatedAt: createdAt,
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("seed audit log: %v", err)
	}
	return log
}

func newAuditLogRepoForTest(t *testing.T, db *gorm.DB) *AuditLogRepo {
	t.Helper()
	return &AuditLogRepo{db: &database.DB{DB: db}}
}

func TestAuditLogRepoCreate_Success(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)

	entry := &model.AuditLog{
		Category: model.AuditCategoryOperation,
		Action:   "create_user",
		Resource: "user",
		Summary:  "created user alice",
		Level:    model.AuditLevelInfo,
	}
	if err := repo.Create(entry); err != nil {
		t.Fatalf("create: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected ID to be auto-generated")
	}

	var found model.AuditLog
	if err := db.First(&found, entry.ID).Error; err != nil {
		t.Fatalf("find created: %v", err)
	}
	if found.Action != "create_user" || found.Summary != "created user alice" {
		t.Fatalf("unexpected persisted data: %+v", found)
	}
}

func TestAuditLogRepoList_ByCategory(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "login success", "alice", now)
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "", "create resource", "bob", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryAuth, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Category != model.AuditCategoryAuth {
		t.Fatalf("expected auth log, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_Pagination(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	for i := 0; i < 25; i++ {
		seedAuditLog(t, db, model.AuditCategoryOperation, fmt.Sprintf("action_%d", i), "", "summary", "user", now.Add(time.Duration(i)*time.Second))
	}

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 25 {
		t.Fatalf("expected total 25, got %d", result.Total)
	}
	if len(result.Items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(result.Items))
	}

	result2, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Page: 3, PageSize: 10})
	if err != nil {
		t.Fatalf("list page 3: %v", err)
	}
	if len(result2.Items) != 5 {
		t.Fatalf("expected 5 items on page 3, got %d", len(result2.Items))
	}
}

func TestAuditLogRepoList_DefaultPageSize(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	for i := 0; i < 25; i++ {
		seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "summary", "user", now)
	}

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 25 {
		t.Fatalf("expected total 25, got %d", result.Total)
	}
	if len(result.Items) != 20 {
		t.Fatalf("expected default pageSize 20, got %d", len(result.Items))
	}
}

func TestAuditLogRepoList_KeywordAuthMatchesUsername(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "login success", "alice@example.com", now)
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "login success", "bob@example.com", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryAuth, Keyword: "alice", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Username != "alice@example.com" {
		t.Fatalf("expected alice log, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_KeywordOperationMatchesSummary(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "", "created project alpha", "alice", now)
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "", "created project beta", "bob", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Keyword: "alpha", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Summary != "created project alpha" {
		t.Fatalf("expected alpha log, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_ActionFilter(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "", "summary", "alice", now)
	seedAuditLog(t, db, model.AuditCategoryOperation, "delete", "", "summary", "bob", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Action: "delete", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Action != "delete" {
		t.Fatalf("expected delete log, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_ResourceFilter(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "user", "summary", "alice", now)
	seedAuditLog(t, db, model.AuditCategoryOperation, "create", "role", "summary", "bob", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Resource: "role", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Resource != "role" {
		t.Fatalf("expected role log, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_DateRange(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now().Truncate(time.Second)
	t1 := now.Add(-48 * time.Hour)
	t2 := now.Add(-24 * time.Hour)
	t3 := now
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "old", "alice", t1)
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "mid", "bob", t2)
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "new", "charlie", t3)

	from := t2.Add(-time.Hour)
	to := t3.Add(time.Hour)
	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, DateFrom: &from, DateTo: &to, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	summaries := map[string]bool{}
	for _, item := range result.Items {
		summaries[item.Summary] = true
	}
	if !summaries["mid"] || !summaries["new"] {
		t.Fatalf("expected mid and new, got %+v", result.Items)
	}
}

func TestAuditLogRepoList_OrderByCreatedAtDesc(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	first := seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "first", "alice", now.Add(-2*time.Hour))
	second := seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "second", "bob", now.Add(-1*time.Hour))
	third := seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "third", "charlie", now)

	result, err := repo.List(AuditLogListParams{Category: model.AuditCategoryOperation, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	expected := []uint{third.ID, second.ID, first.ID}
	for i, item := range result.Items {
		if item.ID != expected[i] {
			t.Fatalf("expected order %v at %d, got %d", expected, i, item.ID)
		}
	}
}

func TestAuditLogRepoDeleteBefore_RemovesOldLogs(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "old", "alice", now.Add(-48*time.Hour))
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "new", "bob", now)

	deleted, err := repo.DeleteBefore(model.AuditCategoryOperation, cutoff)
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	var count int64
	db.Model(&model.AuditLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 remaining log, got %d", count)
	}
}

func TestAuditLogRepoDeleteBefore_ReturnsRowsAffected(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "old1", "alice", now.Add(-48*time.Hour))
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "old2", "bob", now.Add(-72*time.Hour))

	deleted, err := repo.DeleteBefore(model.AuditCategoryAuth, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted, got %d", deleted)
	}
}

func TestAuditLogRepoDeleteBefore_DoesNotAffectOtherCategories(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "old op", "alice", now.Add(-48*time.Hour))
	seedAuditLog(t, db, model.AuditCategoryAuth, "login", "", "old auth", "bob", now.Add(-48*time.Hour))

	deleted, err := repo.DeleteBefore(model.AuditCategoryOperation, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	var authCount int64
	db.Model(&model.AuditLog{}).Where("category = ?", model.AuditCategoryAuth).Count(&authCount)
	if authCount != 1 {
		t.Fatalf("expected auth log to remain, got %d", authCount)
	}
}

func TestAuditLogRepoDeleteBefore_DoesNotAffectNewerLogs(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)
	now := time.Now()
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "older", "alice", now.Add(-48*time.Hour))
	seedAuditLog(t, db, model.AuditCategoryOperation, "action", "", "newer", "bob", now.Add(-12*time.Hour))

	deleted, err := repo.DeleteBefore(model.AuditCategoryOperation, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	var count int64
	db.Model(&model.AuditLog{}).Where("summary = ?", "newer").Count(&count)
	if count != 1 {
		t.Fatalf("expected newer log to remain")
	}
}

func TestAuditLogRepoMigrate_Idempotent(t *testing.T) {
	db := newTestDBForAuditLog(t)
	repo := newAuditLogRepoForTest(t, db)

	if err := repo.Migrate(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := repo.Migrate(); err != nil {
		t.Fatalf("second migrate (idempotent): %v", err)
	}
}
