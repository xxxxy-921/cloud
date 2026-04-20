package repository

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

func TestSysConfigRepoGetReturnsErrNotFound(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file:sys-config-repo?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: gdb})

	repo, err := NewSysConfig(injector)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}

	_, err = repo.Get("system.logo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
