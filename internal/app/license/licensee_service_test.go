package license

import (
	"errors"
	"testing"

	"metis/internal/database"
)

func newLicenseeService(db *database.DB) *LicenseeService {
	return &LicenseeService{repo: &LicenseeRepo{db: db}}
}

func TestLicenseeService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: "Note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if licensee.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", licensee.Name, "Acme Corp")
	}
	if licensee.Status != LicenseeStatusActive {
		t.Errorf("Status = %q, want %q", licensee.Status, LicenseeStatusActive)
	}
	if licensee.Code == "" {
		t.Error("expected non-empty Code")
	}

	// Duplicate name
	_, err = svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if !errors.Is(err, ErrLicenseeNameExists) {
		t.Errorf("expected ErrLicenseeNameExists, got %v", err)
	}
}

func TestLicenseeService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	newName := "Acme Inc"
	updated, err := svc.UpdateLicensee(licensee.ID, UpdateLicenseeParams{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Name = %q, want %q", updated.Name, newName)
	}
}

func TestLicenseeService_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Archive
	if err := svc.UpdateLicenseeStatus(licensee.ID, LicenseeStatusArchived); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var archived Licensee
	db.First(&archived, licensee.ID)
	if archived.Status != LicenseeStatusArchived {
		t.Errorf("Status = %q, want %q", archived.Status, LicenseeStatusArchived)
	}

	// Reactivate
	if err := svc.UpdateLicenseeStatus(licensee.ID, LicenseeStatusActive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var active Licensee
	db.First(&active, licensee.ID)
	if active.Status != LicenseeStatusActive {
		t.Errorf("Status = %q, want %q", active.Status, LicenseeStatusActive)
	}

	// Invalid transition
	if err := svc.UpdateLicenseeStatus(licensee.ID, "invalid"); !errors.Is(err, ErrLicenseeInvalidStatus) {
		t.Errorf("expected ErrLicenseeInvalidStatus, got %v", err)
	}
}
