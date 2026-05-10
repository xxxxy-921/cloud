package licensee

import (
	"errors"
	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
	"testing"

	"metis/internal/database"
)

func newLicenseeService(db *database.DB) *LicenseeService {
	return &LicenseeService{repo: &LicenseeRepo{DB: db}}
}

func TestLicenseeService_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: "Note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if licensee.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", licensee.Name, "Acme Corp")
	}
	if licensee.Status != domain.LicenseeStatusActive {
		t.Errorf("Status = %q, want %q", licensee.Status, domain.LicenseeStatusActive)
	}
	if licensee.Code == "" {
		t.Error("expected non-empty Code")
	}

	// Duplicate name
	_, err = svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if !errors.Is(err, ErrLicenseeNameExists) {
		t.Errorf("expected ErrLicenseeNameExists, got %v", err)
	}

	_, err = svc.CreateLicensee(CreateLicenseeParams{Name: "   ", Notes: ""})
	if !errors.Is(err, ErrInvalidLicenseeName) {
		t.Errorf("blank name create error = %v, want %v", err, ErrInvalidLicenseeName)
	}

	trimmed, err := svc.CreateLicensee(CreateLicenseeParams{Name: "  Trimmed Corp  ", Notes: ""})
	if err != nil {
		t.Fatalf("trimmed create unexpected error: %v", err)
	}
	if trimmed.Name != "Trimmed Corp" {
		t.Fatalf("trimmed create name = %q, want %q", trimmed.Name, "Trimmed Corp")
	}
}

func TestLicenseeService_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	other, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Beta Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup other failed: %v", err)
	}

	newName := "Acme Inc"
	updated, err := svc.UpdateLicensee(licensee.ID, UpdateLicenseeParams{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Name = %q, want %q", updated.Name, newName)
	}

	duplicateName := other.Name
	if _, err := svc.UpdateLicensee(licensee.ID, UpdateLicenseeParams{Name: &duplicateName}); !errors.Is(err, ErrLicenseeNameExists) {
		t.Fatalf("duplicate update error = %v, want %v", err, ErrLicenseeNameExists)
	}
	blank := "   "
	if _, err := svc.UpdateLicensee(licensee.ID, UpdateLicenseeParams{Name: &blank}); !errors.Is(err, ErrInvalidLicenseeName) {
		t.Fatalf("blank update error = %v, want %v", err, ErrInvalidLicenseeName)
	}
	trimmedName := "  Acme Prime  "
	updated, err = svc.UpdateLicensee(licensee.ID, UpdateLicenseeParams{Name: &trimmedName})
	if err != nil {
		t.Fatalf("trimmed update unexpected error: %v", err)
	}
	if updated.Name != "Acme Prime" {
		t.Fatalf("trimmed update name = %q, want %q", updated.Name, "Acme Prime")
	}
	if _, err := svc.UpdateLicensee(999, UpdateLicenseeParams{Name: &newName}); !errors.Is(err, ErrLicenseeNotFound) {
		t.Fatalf("missing update error = %v, want %v", err, ErrLicenseeNotFound)
	}
}

func TestLicenseeService_UpdateStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := newLicenseeService(db)

	licensee, err := svc.CreateLicensee(CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Archive
	if err := svc.UpdateLicenseeStatus(licensee.ID, domain.LicenseeStatusArchived); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var archived domain.Licensee
	db.First(&archived, licensee.ID)
	if archived.Status != domain.LicenseeStatusArchived {
		t.Errorf("Status = %q, want %q", archived.Status, domain.LicenseeStatusArchived)
	}

	// Reactivate
	if err := svc.UpdateLicenseeStatus(licensee.ID, domain.LicenseeStatusActive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var active domain.Licensee
	db.First(&active, licensee.ID)
	if active.Status != domain.LicenseeStatusActive {
		t.Errorf("Status = %q, want %q", active.Status, domain.LicenseeStatusActive)
	}

	// Invalid transition
	if err := svc.UpdateLicenseeStatus(licensee.ID, "invalid"); !errors.Is(err, ErrLicenseeInvalidStatus) {
		t.Errorf("expected ErrLicenseeInvalidStatus, got %v", err)
	}
	if err := svc.UpdateLicenseeStatus(999, domain.LicenseeStatusArchived); !errors.Is(err, ErrLicenseeNotFound) {
		t.Errorf("expected ErrLicenseeNotFound, got %v", err)
	}
	if _, err := svc.GetLicensee(999); !errors.Is(err, ErrLicenseeNotFound) {
		t.Errorf("expected ErrLicenseeNotFound on get, got %v", err)
	}
}
