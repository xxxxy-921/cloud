package license

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/database"
)

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&Product{},
		&Plan{},
		&ProductKey{},
		&License{},
		&Licensee{},
		&LicenseRegistration{},
	); err != nil {
		t.Fatalf("failed to auto-migrate license models: %v", err)
	}

	return &database.DB{DB: db}
}
