package certificate

import (
	"testing"
	"time"

	"metis/internal/app/license/domain"
	licenseepkg "metis/internal/app/license/licensee"
	"metis/internal/app/license/testutil"
)

func TestLicenseRepoListAndStateQueries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)
	repo := &LicenseRepo{db: db}

	product, err := productSvc.CreateProduct("Metis", "metis-repo", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}
	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &product.ID, LicenseeID: &licensee.ID, Code: "RG-REPO-1"})
	if err != nil {
		t.Fatalf("create registration: %v", err)
	}
	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        time.Now().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue license: %v", err)
	}

	detail, err := repo.FindByID(license.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if detail.ProductCode != "metis-repo" || detail.LicenseeName != "Acme" {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	list, total, err := repo.List(LicenseListParams{Keyword: "RG-REPO", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].RegistrationCode != reg.Code {
		t.Fatalf("unexpected list result: total=%d list=%+v", total, list)
	}

	if _, err := productSvc.RotateKey(product.ID); err != nil {
		t.Fatalf("rotate key: %v", err)
	}
	count, err := repo.CountByProductAndKeyVersionLessThan(product.ID, license.KeyVersion+1)
	if err != nil {
		t.Fatalf("CountByProductAndKeyVersionLessThan: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	items, err := repo.FindReissueableByProductID(product.ID, license.KeyVersion+1)
	if err != nil {
		t.Fatalf("FindReissueableByProductID: %v", err)
	}
	if len(items) != 1 || items[0].ID != license.ID {
		t.Fatalf("unexpected reissueable items: %+v", items)
	}

	past := time.Now().Add(-time.Minute)
	if err := repo.UpdateStatus(license.ID, map[string]any{"valid_until": &past}); err != nil {
		t.Fatalf("set valid_until: %v", err)
	}
	if err := repo.UpdateExpiredStatus(time.Now(), []string{"active", "pending"}); err != nil {
		t.Fatalf("UpdateExpiredStatus: %v", err)
	}
	refreshed, err := repo.FindByID(license.ID)
	if err != nil {
		t.Fatalf("reload expired license: %v", err)
	}
	if refreshed.LifecycleStatus != "expired" {
		t.Fatalf("lifecycle status = %q, want expired", refreshed.LifecycleStatus)
	}

	driftReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &product.ID, LicenseeID: &licensee.ID, Code: "RG-REPO-DRIFT-EXPIRE"})
	if err != nil {
		t.Fatalf("create drift reg: %v", err)
	}
	driftLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Drift",
		RegistrationCode: driftReg.Code,
		ValidFrom:        time.Now().Add(-2 * time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue drift license: %v", err)
	}
	if err := repo.UpdateStatus(driftLicense.ID, map[string]any{
		"status":      domain.LicenseStatusRevoked,
		"valid_until": &past,
	}); err != nil {
		t.Fatalf("drift revoke status: %v", err)
	}
	if err := repo.UpdateExpiredStatus(time.Now(), []string{"active", "pending"}); err != nil {
		t.Fatalf("UpdateExpiredStatus drift revoked: %v", err)
	}
	driftRefreshed, err := repo.FindByID(driftLicense.ID)
	if err != nil {
		t.Fatalf("reload drift revoked license: %v", err)
	}
	if driftRefreshed.LifecycleStatus == domain.LicenseLifecycleExpired {
		t.Fatalf("revoked drift license lifecycle should not be rewritten to expired: %+v", driftRefreshed)
	}

	byProduct, err := repo.FindByProductID(product.ID)
	if err != nil {
		t.Fatalf("FindByProductID: %v", err)
	}
	if len(byProduct) != 2 {
		t.Fatalf("unexpected product license list: %+v", byProduct)
	}
	found := map[uint]bool{}
	for _, item := range byProduct {
		found[item.ID] = true
	}
	if !found[license.ID] || !found[driftLicense.ID] {
		t.Fatalf("expected product license list to contain both original and drift license, got %+v", byProduct)
	}
}

func TestLicenseRepoListFiltersAndReissueableSelection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)
	repo := &LicenseRepo{db: db}

	productA, err := productSvc.CreateProduct("Metis A", "metis-a", "")
	if err != nil {
		t.Fatalf("create product A: %v", err)
	}
	productB, err := productSvc.CreateProduct("Metis B", "metis-b", "")
	if err != nil {
		t.Fatalf("create product B: %v", err)
	}
	if err := productSvc.UpdateStatus(productA.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product A: %v", err)
	}
	if err := productSvc.UpdateStatus(productB.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product B: %v", err)
	}

	licenseeA, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme"})
	if err != nil {
		t.Fatalf("create licensee A: %v", err)
	}
	licenseeB, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Globex"})
	if err != nil {
		t.Fatalf("create licensee B: %v", err)
	}

	regA1, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &productA.ID, LicenseeID: &licenseeA.ID, Code: "RG-FILTER-A1"})
	if err != nil {
		t.Fatalf("create registration A1: %v", err)
	}
	regA2, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &productA.ID, LicenseeID: &licenseeA.ID, Code: "RG-FILTER-A2"})
	if err != nil {
		t.Fatalf("create registration A2: %v", err)
	}
	regA3, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &productA.ID, LicenseeID: &licenseeA.ID, Code: "RG-FILTER-A3"})
	if err != nil {
		t.Fatalf("create registration A3: %v", err)
	}
	regB1, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{ProductID: &productB.ID, LicenseeID: &licenseeB.ID, Code: "RG-FILTER-B1"})
	if err != nil {
		t.Fatalf("create registration B1: %v", err)
	}

	activeA, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        productA.ID,
		LicenseeID:       licenseeA.ID,
		PlanName:         "Enterprise Alpha",
		RegistrationCode: regA1.Code,
		ValidFrom:        time.Now().Add(-2 * time.Hour),
		ValidUntil:       ptrTime(time.Now().Add(24 * time.Hour)),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue active license A: %v", err)
	}
	pendingA, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        productA.ID,
		LicenseeID:       licenseeA.ID,
		PlanName:         "Enterprise Beta",
		RegistrationCode: regA2.Code,
		ValidFrom:        time.Now().Add(24 * time.Hour),
		ValidUntil:       ptrTime(time.Now().Add(48 * time.Hour)),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue pending license A: %v", err)
	}
	suspendedA, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        productA.ID,
		LicenseeID:       licenseeA.ID,
		PlanName:         "Support Plan",
		RegistrationCode: regA3.Code,
		ValidFrom:        time.Now().Add(-4 * time.Hour),
		ValidUntil:       ptrTime(time.Now().Add(48 * time.Hour)),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue suspended license A: %v", err)
	}
	if err := licenseSvc.SuspendLicense(suspendedA.ID, 9); err != nil {
		t.Fatalf("suspend license A: %v", err)
	}
	revokedB, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        productB.ID,
		LicenseeID:       licenseeB.ID,
		PlanName:         "External Plan",
		RegistrationCode: regB1.Code,
		ValidFrom:        time.Now().Add(-4 * time.Hour),
		ValidUntil:       ptrTime(time.Now().Add(48 * time.Hour)),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue revoked license B: %v", err)
	}
	if err := licenseSvc.RevokeLicense(revokedB.ID, 9); err != nil {
		t.Fatalf("revoke license B: %v", err)
	}

	if err := db.Model(&domain.License{}).Where("id = ?", activeA.ID).Update("created_at", time.Now().Add(-3*time.Hour)).Error; err != nil {
		t.Fatalf("backdate active license: %v", err)
	}
	if err := db.Model(&domain.License{}).Where("id = ?", pendingA.ID).Update("created_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("backdate pending license: %v", err)
	}
	if err := db.Model(&domain.License{}).Where("id = ?", suspendedA.ID).Update("created_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatalf("backdate suspended license: %v", err)
	}

	t.Run("filters by product licensee status lifecycle and keyword", func(t *testing.T) {
		items, total, err := repo.List(LicenseListParams{
			ProductID:       productA.ID,
			LicenseeID:      licenseeA.ID,
			Status:          domain.LicenseStatusIssued,
			LifecycleStatus: domain.LicenseLifecycleSuspended,
			Keyword:         "Support",
			Page:            1,
			PageSize:        10,
		})
		if err != nil {
			t.Fatalf("List filtered: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].ID != suspendedA.ID {
			t.Fatalf("unexpected filtered list result: total=%d items=%+v", total, items)
		}
	})

	t.Run("paginates newest first with defaults", func(t *testing.T) {
		items, total, err := repo.List(LicenseListParams{
			ProductID:  productA.ID,
			LicenseeID: licenseeA.ID,
			Page:       0,
			PageSize:   0,
		})
		if err != nil {
			t.Fatalf("List defaults: %v", err)
		}
		if total != 3 || len(items) != 3 {
			t.Fatalf("unexpected paged list result: total=%d items=%+v", total, items)
		}
		if items[0].ID != suspendedA.ID || items[1].ID != pendingA.ID || items[2].ID != activeA.ID {
			t.Fatalf("expected created_at desc ordering, got ids=%d,%d,%d", items[0].ID, items[1].ID, items[2].ID)
		}
	})

	t.Run("find by product returns all licenses for the product regardless of lifecycle", func(t *testing.T) {
		items, err := repo.FindByProductID(productA.ID)
		if err != nil {
			t.Fatalf("FindByProductID: %v", err)
		}
		if len(items) != 3 {
			t.Fatalf("expected 3 licenses for product A, got %+v", items)
		}
		gotIDs := map[uint]bool{}
		for _, item := range items {
			gotIDs[item.ID] = true
		}
		if !gotIDs[activeA.ID] || !gotIDs[pendingA.ID] || !gotIDs[suspendedA.ID] || gotIDs[revokedB.ID] {
			t.Fatalf("unexpected product-scoped license set: %+v", gotIDs)
		}
	})

	t.Run("reissueable selection excludes revoked and current version", func(t *testing.T) {
		if _, err := productSvc.RotateKey(productA.ID); err != nil {
			t.Fatalf("rotate key for product A: %v", err)
		}
		currentKey, err := productSvc.GetPublicKey(productA.ID)
		if err != nil {
			t.Fatalf("get current public key for product A: %v", err)
		}
		if err := repo.UpdateStatus(activeA.ID, map[string]any{"key_version": currentKey.Version}); err != nil {
			t.Fatalf("set active license to current version: %v", err)
		}

		items, err := repo.FindReissueableByProductID(productA.ID, currentKey.Version)
		if err != nil {
			t.Fatalf("FindReissueableByProductID: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected only pending+suspended licenses to be reissueable, got %+v", items)
		}
		gotIDs := map[uint]bool{}
		for _, item := range items {
			gotIDs[item.ID] = true
		}
		if !gotIDs[pendingA.ID] || !gotIDs[suspendedA.ID] || gotIDs[activeA.ID] || gotIDs[revokedB.ID] {
			t.Fatalf("unexpected reissueable ids: %+v", gotIDs)
		}
	})

	t.Run("reissueable selection returns empty when every license is already on current version", func(t *testing.T) {
		currentKey, err := productSvc.GetPublicKey(productA.ID)
		if err != nil {
			t.Fatalf("get current public key for product A: %v", err)
		}
		if err := repo.UpdateStatus(pendingA.ID, map[string]any{"key_version": currentKey.Version}); err != nil {
			t.Fatalf("set pending license to current version: %v", err)
		}
		if err := repo.UpdateStatus(suspendedA.ID, map[string]any{"key_version": currentKey.Version}); err != nil {
			t.Fatalf("set suspended license to current version: %v", err)
		}

		items, err := repo.FindReissueableByProductID(productA.ID, currentKey.Version)
		if err != nil {
			t.Fatalf("FindReissueableByProductID current-only: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected no reissueable items after all licenses moved to current version, got %+v", items)
		}
	})

	t.Run("reissueable selection ignores lifecycle-revoked drift even if status was not updated", func(t *testing.T) {
		driftProduct, err := productSvc.CreateProduct("Metis Drift", "metis-drift", "")
		if err != nil {
			t.Fatalf("create drift product: %v", err)
		}
		if err := productSvc.UpdateStatus(driftProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish drift product: %v", err)
		}
		driftLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Drift Corp"})
		if err != nil {
			t.Fatalf("create drift licensee: %v", err)
		}
		driftReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &driftProduct.ID,
			LicenseeID: &driftLicensee.ID,
			Code:       "RG-DRIFT-REVOCATION-001",
		})
		if err != nil {
			t.Fatalf("create drift registration: %v", err)
		}
		driftLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        driftProduct.ID,
			LicenseeID:       driftLicensee.ID,
			PlanName:         "Drift Plan",
			RegistrationCode: driftReg.Code,
			ValidFrom:        time.Now().Add(-time.Hour),
			ValidUntil:       ptrTime(time.Now().Add(24 * time.Hour)),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue drift license: %v", err)
		}
		if _, err := productSvc.RotateKey(driftProduct.ID); err != nil {
			t.Fatalf("rotate key for drift product: %v", err)
		}
		currentKey, err := productSvc.GetPublicKey(driftProduct.ID)
		if err != nil {
			t.Fatalf("get current public key for drift product: %v", err)
		}
		if err := repo.UpdateStatus(driftLicense.ID, map[string]any{"lifecycle_status": domain.LicenseLifecycleRevoked}); err != nil {
			t.Fatalf("simulate lifecycle drift revoke: %v", err)
		}

		count, err := repo.CountByProductAndKeyVersionLessThan(driftProduct.ID, currentKey.Version)
		if err != nil {
			t.Fatalf("CountByProductAndKeyVersionLessThan drift revoked: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected lifecycle-revoked drift license to be ignored by impact count, got %d", count)
		}

		items, err := repo.FindReissueableByProductID(driftProduct.ID, currentKey.Version)
		if err != nil {
			t.Fatalf("FindReissueableByProductID drift revoked: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected lifecycle-revoked drift license to be excluded from reissueable set, got %+v", items)
		}
	})
}
