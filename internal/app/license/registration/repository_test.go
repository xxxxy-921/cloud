package registration

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
)

func TestLicenseRegistrationRepoListFiltersAvailableAndPaginates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	now := time.Now()
	boundID := uint(99)
	rows := []domain.LicenseRegistration{
		{Code: "RG-AVAILABLE-NEW", Source: "manual"},
		{Code: "RG-BOUND", Source: "manual", BoundLicenseID: &boundID},
		{Code: "RG-EXPIRED", Source: "manual", ExpiresAt: ptrTime(now.Add(-time.Minute))},
		{Code: "RG-AVAILABLE-OLD", Source: "manual"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("create registration %s: %v", rows[i].Code, err)
		}
	}
	if err := db.Model(&rows[0]).Update("created_at", now.Add(2*time.Hour)).Error; err != nil {
		t.Fatalf("update created_at for first row: %v", err)
	}
	if err := db.Model(&rows[1]).Update("created_at", now.Add(time.Hour)).Error; err != nil {
		t.Fatalf("update created_at for second row: %v", err)
	}
	if err := db.Model(&rows[2]).Update("created_at", now).Error; err != nil {
		t.Fatalf("update created_at for third row: %v", err)
	}
	if err := db.Model(&rows[3]).Update("created_at", now.Add(-time.Hour)).Error; err != nil {
		t.Fatalf("update created_at for fourth row: %v", err)
	}

	items, total, err := repo.List(LicenseRegistrationListParams{
		Available: true,
		Page:      1,
		PageSize:  1,
	})
	if err != nil {
		t.Fatalf("list available registrations: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 1 || items[0].Code != "RG-AVAILABLE-NEW" {
		t.Fatalf("unexpected first page items: %+v", items)
	}

	items, total, err = repo.List(LicenseRegistrationListParams{
		Available: true,
		Page:      2,
		PageSize:  1,
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if total != 2 {
		t.Fatalf("page 2 total = %d, want 2", total)
	}
	if len(items) != 1 || items[0].Code != "RG-AVAILABLE-OLD" {
		t.Fatalf("unexpected second page items: %+v", items)
	}
}

func TestLicenseRegistrationRepoBindUnbindAndCleanupExpired(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	licenseID := uint(7)
	expiredUnbound := domain.LicenseRegistration{
		Code:      "RG-EXPIRED-UNBOUND",
		Source:    "manual",
		ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
	}
	expiredBound := domain.LicenseRegistration{
		Code:           "RG-EXPIRED-BOUND",
		Source:         "manual",
		ExpiresAt:      ptrTime(time.Now().Add(-time.Hour)),
		BoundLicenseID: &licenseID,
	}
	active := domain.LicenseRegistration{
		Code:      "RG-ACTIVE",
		Source:    "manual",
		ExpiresAt: ptrTime(time.Now().Add(time.Hour)),
	}
	for _, row := range []*domain.LicenseRegistration{&expiredUnbound, &expiredBound, &active} {
		if err := repo.Create(row); err != nil {
			t.Fatalf("create registration %s: %v", row.Code, err)
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := repo.UpdateBoundLicenseInTx(tx, active.ID, 42); err != nil {
			return err
		}
		return repo.UnbindLicenseInTx(tx, active.Code)
	}); err != nil {
		t.Fatalf("bind/unbind in tx: %v", err)
	}

	found, err := repo.FindByCode(active.Code)
	if err != nil {
		t.Fatalf("find active code: %v", err)
	}
	if found.BoundLicenseID != nil {
		t.Fatalf("BoundLicenseID = %v, want nil", found.BoundLicenseID)
	}

	if err := repo.DeleteExpired(time.Now()); err != nil {
		t.Fatalf("delete expired: %v", err)
	}

	if _, err := repo.FindByCode(expiredUnbound.Code); err == nil {
		t.Fatal("expected expired unbound registration to be deleted")
	}
	if _, err := repo.FindByCode(expiredBound.Code); err != nil {
		t.Fatalf("expected bound expired registration to remain, got %v", err)
	}
	if _, err := repo.FindByCode(active.Code); err != nil {
		t.Fatalf("expected active registration to remain, got %v", err)
	}
}

func TestLicenseRegistrationRepoCodeExists(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	if err := repo.Create(&domain.LicenseRegistration{Code: "RG-EXISTS", Source: "manual"}); err != nil {
		t.Fatalf("create registration: %v", err)
	}

	exists, err := repo.CodeExists("RG-EXISTS")
	if err != nil {
		t.Fatalf("code exists: %v", err)
	}
	if !exists {
		t.Fatal("expected code to exist")
	}

	exists, err = repo.CodeExists("RG-MISSING")
	if err != nil {
		t.Fatalf("missing code exists: %v", err)
	}
	if exists {
		t.Fatal("expected missing code to not exist")
	}
}

func TestLicenseRegistrationRepoTxCreateAndLookupHelpers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return repo.CreateInTx(tx, &domain.LicenseRegistration{
			Code:      "RG-TX-1",
			Source:    "manual",
			ExpiresAt: ptrTime(time.Now().Add(time.Hour)),
		})
	}); err != nil {
		t.Fatalf("CreateInTx: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		lr, err := repo.FindByCodeInTx(tx, "RG-TX-1")
		if err != nil {
			return err
		}
		if lr.Code != "RG-TX-1" {
			t.Fatalf("unexpected code in tx lookup: %+v", lr)
		}
		return nil
	}); err != nil {
		t.Fatalf("FindByCodeInTx: %v", err)
	}

	lr, err := repo.FindByCode("RG-TX-1")
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	byID, err := repo.FindByID(lr.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if byID.Code != lr.Code {
		t.Fatalf("FindByID code=%q, want %q", byID.Code, lr.Code)
	}
}

func TestLicenseRegistrationRepoListScopesByProductAndLicensee(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	productA := uint(11)
	productB := uint(22)
	licenseeA := uint(101)
	licenseeB := uint(202)
	boundID := uint(9)
	now := time.Now()

	rows := []domain.LicenseRegistration{
		{Code: "RG-SCOPE-MATCH-NEW", Source: "manual", ProductID: &productA, LicenseeID: &licenseeA},
		{Code: "RG-SCOPE-MATCH-OLD", Source: "manual", ProductID: &productA, LicenseeID: &licenseeA},
		{Code: "RG-SCOPE-OTHER-PRODUCT", Source: "manual", ProductID: &productB, LicenseeID: &licenseeA},
		{Code: "RG-SCOPE-OTHER-LICENSEE", Source: "manual", ProductID: &productA, LicenseeID: &licenseeB},
		{Code: "RG-SCOPE-BOUND", Source: "manual", ProductID: &productA, LicenseeID: &licenseeA, BoundLicenseID: &boundID},
		{Code: "RG-SCOPE-EXPIRED", Source: "manual", ProductID: &productA, LicenseeID: &licenseeA, ExpiresAt: ptrTime(now.Add(-time.Minute))},
	}
	for i := range rows {
		if err := repo.Create(&rows[i]); err != nil {
			t.Fatalf("create scoped registration %s: %v", rows[i].Code, err)
		}
	}
	if err := db.Model(&rows[0]).Update("created_at", now.Add(2*time.Hour)).Error; err != nil {
		t.Fatalf("update created_at for first row: %v", err)
	}
	if err := db.Model(&rows[1]).Update("created_at", now.Add(time.Hour)).Error; err != nil {
		t.Fatalf("update created_at for second row: %v", err)
	}

	items, total, err := repo.List(LicenseRegistrationListParams{
		ProductID:  productA,
		LicenseeID: licenseeA,
		Available:  true,
	})
	if err != nil {
		t.Fatalf("scoped available list: %v", err)
	}
	if total != 2 {
		t.Fatalf("scoped available total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("scoped available items len = %d, want 2", len(items))
	}
	if items[0].Code != "RG-SCOPE-MATCH-NEW" || items[1].Code != "RG-SCOPE-MATCH-OLD" {
		t.Fatalf("unexpected scoped order/items: %+v", items)
	}
}

func TestLicenseRegistrationRepoLookupHelpersReturnNotFoundForMissingRows(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseRegistrationRepo{DB: db}

	if _, err := repo.FindByID(9999); err == nil {
		t.Fatal("expected FindByID missing row to fail")
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := repo.FindByCodeInTx(tx, "RG-MISSING-TX"); err == nil {
			t.Fatal("expected FindByCodeInTx missing row to fail")
		}
		return nil
	}); err != nil {
		t.Fatalf("missing tx lookup transaction failed: %v", err)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
