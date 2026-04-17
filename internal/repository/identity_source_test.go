package repository

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

func newTestDBForIdentitySource(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.IdentitySource{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedIdentitySource(t *testing.T, db *gorm.DB, name, sourceType, domains string, enabled bool, sortOrder int) *model.IdentitySource {
	t.Helper()
	s := &model.IdentitySource{
		Name:      name,
		Type:      sourceType,
		Domains:   domains,
		Enabled:   enabled,
		SortOrder: sortOrder,
	}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("seed identity source: %v", err)
	}
	return s
}

func newIdentitySourceRepoForTest(t *testing.T, db *gorm.DB) *IdentitySourceRepo {
	t.Helper()
	return &IdentitySourceRepo{db: &database.DB{DB: db}}
}

func TestIdentitySourceRepoList_Ordering(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "B", "oidc", "", true, 2)
	seedIdentitySource(t, db, "A", "ldap", "", true, 1)
	seedIdentitySource(t, db, "C", "oidc", "", true, 2)

	items, err := repo.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Expected order: A (sort 1), B (sort 2, id 1), C (sort 2, id 2)
	if items[0].Name != "A" || items[1].Name != "B" || items[2].Name != "C" {
		t.Fatalf("unexpected order: %v", items)
	}
}

func TestIdentitySourceRepoFindByID_Success(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seeded := seedIdentitySource(t, db, "Okta", "oidc", "", true, 0)

	found, err := repo.FindByID(seeded.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Name != "Okta" {
		t.Fatalf("expected name Okta, got %s", found.Name)
	}
}

func TestIdentitySourceRepoFindByID_NotFound(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)

	_, err := repo.FindByID(9999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestIdentitySourceRepoFindByDomain_Match(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	found, err := repo.FindByDomain("example.com")
	if err != nil {
		t.Fatalf("find by domain: %v", err)
	}
	if found.Name != "Okta" {
		t.Fatalf("expected Okta, got %s", found.Name)
	}
}

func TestIdentitySourceRepoFindByDomain_CaseInsensitive(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	found, err := repo.FindByDomain("EXAMPLE.COM")
	if err != nil {
		t.Fatalf("find by domain: %v", err)
	}
	if found.Name != "Okta" {
		t.Fatalf("expected Okta, got %s", found.Name)
	}
}

func TestIdentitySourceRepoFindByDomain_TrimWhitespace(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	found, err := repo.FindByDomain(" example.com ")
	if err != nil {
		t.Fatalf("find by domain: %v", err)
	}
	if found.Name != "Okta" {
		t.Fatalf("expected Okta, got %s", found.Name)
	}
}

func TestIdentitySourceRepoFindByDomain_NoMatch(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	_, err := repo.FindByDomain("other.com")
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestIdentitySourceRepoCheckDomainConflict_Detected(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	err := repo.CheckDomainConflict("example.com", 0)
	if err != ErrDomainConflict {
		t.Fatalf("expected ErrDomainConflict, got %v", err)
	}
}

func TestIdentitySourceRepoCheckDomainConflict_SelfUpdateAllowed(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seeded := seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	err := repo.CheckDomainConflict("example.com", seeded.ID)
	if err != nil {
		t.Fatalf("expected no error for self-update, got %v", err)
	}
}

func TestIdentitySourceRepoCheckDomainConflict_EmptyDomains(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seedIdentitySource(t, db, "Okta", "oidc", "example.com", true, 0)

	err := repo.CheckDomainConflict("", 0)
	if err != nil {
		t.Fatalf("expected no error for empty domains, got %v", err)
	}
}

func TestIdentitySourceRepoCreate(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	s := &model.IdentitySource{Name: "New", Type: "oidc"}

	if err := repo.Create(s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("expected ID to be generated")
	}
}

func TestIdentitySourceRepoUpdate(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seeded := seedIdentitySource(t, db, "Old", "oidc", "", true, 0)

	seeded.Name = "Updated"
	if err := repo.Update(seeded); err != nil {
		t.Fatalf("update: %v", err)
	}

	found, _ := repo.FindByID(seeded.ID)
	if found.Name != "Updated" {
		t.Fatalf("expected Updated, got %s", found.Name)
	}
}

func TestIdentitySourceRepoDelete(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seeded := seedIdentitySource(t, db, "Delete", "oidc", "", true, 0)

	if err := repo.Delete(seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := repo.FindByID(seeded.ID)
	if !isRecordNotFound(err) {
		t.Fatalf("expected record not found, got %v", err)
	}
}

func TestIdentitySourceRepoToggle(t *testing.T) {
	db := newTestDBForIdentitySource(t)
	repo := newIdentitySourceRepoForTest(t, db)
	seeded := seedIdentitySource(t, db, "Toggle", "oidc", "", true, 0)

	result, err := repo.Toggle(seeded.ID)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if result.Enabled {
		t.Fatal("expected disabled after toggle")
	}

	result, err = repo.Toggle(seeded.ID)
	if err != nil {
		t.Fatalf("toggle back: %v", err)
	}
	if !result.Enabled {
		t.Fatal("expected enabled after second toggle")
	}
}

func isRecordNotFound(err error) bool {
	return err != nil && err == gorm.ErrRecordNotFound
}
