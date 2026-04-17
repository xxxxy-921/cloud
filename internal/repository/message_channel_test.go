package repository

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

func newTestDBForMessageChannel(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedMessageChannel(t *testing.T, db *gorm.DB, name, channelType, config string) *model.MessageChannel {
	t.Helper()
	ch := &model.MessageChannel{
		Name:    name,
		Type:    channelType,
		Config:  config,
		Enabled: true,
	}
	if err := db.Create(ch).Error; err != nil {
		t.Fatalf("seed message channel: %v", err)
	}
	return ch
}

func newMessageChannelRepoForTest(t *testing.T, db *gorm.DB) *MessageChannelRepo {
	t.Helper()
	return &MessageChannelRepo{db: &database.DB{DB: db}}
}

func TestMessageChannelRepoCreate(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)

	ch := &model.MessageChannel{Name: "SMTP", Type: "email", Config: `{}`}
	if err := repo.Create(ch); err != nil {
		t.Fatalf("create: %v", err)
	}
	if ch.ID == 0 {
		t.Fatal("expected ID to be auto-generated")
	}
	if !ch.Enabled {
		t.Fatal("expected Enabled to be true")
	}
}

func TestMessageChannelRepoFindByID_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `{}`)

	found, err := repo.FindByID(seeded.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.Name != "SMTP" {
		t.Fatalf("expected name SMTP, got %s", found.Name)
	}
}

func TestMessageChannelRepoFindByID_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)

	_, err := repo.FindByID(9999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestMessageChannelRepoList_Pagination(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)

	for i := 0; i < 25; i++ {
		seedMessageChannel(t, db, fmt.Sprintf("Channel %d", i), "email", `{}`)
	}

	items, total, err := repo.List(ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(items))
	}
}

func TestMessageChannelRepoList_KeywordFilter(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)

	seedMessageChannel(t, db, "Production SMTP", "email", `{}`)
	seedMessageChannel(t, db, "Dev Webhook", "webhook", `{}`)
	seedMessageChannel(t, db, "Staging SMTP", "email", `{}`)

	items, total, err := repo.List(ListParams{Keyword: "SMTP", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestMessageChannelRepoUpdate(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)
	seeded := seedMessageChannel(t, db, "Old", "email", `{}`)

	seeded.Name = "New"
	seeded.Config = `{"host":"localhost"}`
	if err := repo.Update(seeded); err != nil {
		t.Fatalf("update: %v", err)
	}

	found, _ := repo.FindByID(seeded.ID)
	if found.Name != "New" || found.Config != `{"host":"localhost"}` {
		t.Fatalf("update not persisted: %+v", found)
	}
}

func TestMessageChannelRepoDelete_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)
	seeded := seedMessageChannel(t, db, "ToDelete", "email", `{}`)

	if err := repo.Delete(seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := repo.FindByID(seeded.ID)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected record not found after delete, got %v", err)
	}
}

func TestMessageChannelRepoDelete_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)

	err := repo.Delete(9999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestMessageChannelRepoToggleEnabled(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	repo := newMessageChannelRepoForTest(t, db)
	seeded := seedMessageChannel(t, db, "Toggle", "email", `{}`)

	if !seeded.Enabled {
		t.Fatal("expected seeded channel to be enabled")
	}

	ch, err := repo.ToggleEnabled(seeded.ID)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if ch.Enabled {
		t.Fatal("expected channel to be disabled after toggle")
	}

	ch, err = repo.ToggleEnabled(seeded.ID)
	if err != nil {
		t.Fatalf("toggle back: %v", err)
	}
	if !ch.Enabled {
		t.Fatal("expected channel to be re-enabled after second toggle")
	}
}

func TestMaskConfig_HappyPath(t *testing.T) {
	input := `{"host":"smtp.example.com","port":587,"password":"secret123"}`
	expected := `{"host":"smtp.example.com","password":"******","port":587}`
	got := MaskConfig(input)
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestMaskConfig_MalformedJSON(t *testing.T) {
	input := `not-json`
	got := MaskConfig(input)
	if got != input {
		t.Fatalf("expected original string for malformed JSON, got %s", got)
	}
}
