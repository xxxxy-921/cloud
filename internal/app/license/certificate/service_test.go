package certificate

import (
	"errors"
	licensecrypto "metis/internal/app/license/crypto"
	"metis/internal/app/license/domain"
	licenseepkg "metis/internal/app/license/licensee"
	productpkg "metis/internal/app/license/product"
	"metis/internal/app/license/registration"
	"metis/internal/app/license/testutil"
	"strings"
	"testing"
	"time"

	"github.com/samber/do/v2"

	"metis/internal/database"
)

func newLicenseService(db *database.DB) *LicenseService {
	return &LicenseService{
		licenseRepo:      &LicenseRepo{db: db},
		productRepo:      &productpkg.ProductRepo{DB: db},
		licenseeRepo:     &licenseepkg.LicenseeRepo{DB: db},
		keyRepo:          &productpkg.ProductKeyRepo{DB: db},
		regRepo:          &registration.LicenseRegistrationRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}
}

func newProductService(db *database.DB) *productpkg.ProductService {
	injector := do.New()
	do.ProvideValue(injector, db)
	do.ProvideValue[[]byte](injector, []byte("test-jwt-secret"))
	do.ProvideNamedValue(injector, "licenseKeySecret", []byte("test-license-secret"))
	do.Provide(injector, productpkg.NewProductRepo)
	do.Provide(injector, productpkg.NewPlanRepo)
	do.Provide(injector, productpkg.NewProductKeyRepo)
	do.Provide(injector, productpkg.NewProductService)
	return do.MustInvoke[*productpkg.ProductService](injector)
}

func newLicenseeService(db *database.DB) *licenseepkg.LicenseeService {
	injector := do.New()
	do.ProvideValue(injector, db)
	do.Provide(injector, licenseepkg.NewLicenseeRepo)
	do.Provide(injector, licenseepkg.NewLicenseeService)
	return do.MustInvoke[*licenseepkg.LicenseeService](injector)
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func TestLicenseService_IssueLicense_HappyPath(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-issue", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-ACME-001",
	})
	if err != nil {
		t.Fatalf("setup registration failed: %v", err)
	}

	planName := "Enterprise"
	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         planName,
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if license.PlanName != planName {
		t.Errorf("PlanName = %q, want %q", license.PlanName, planName)
	}
	if license.RegistrationCode != reg.Code {
		t.Errorf("RegistrationCode = %q, want %q", license.RegistrationCode, reg.Code)
	}
	if license.LifecycleStatus != domain.LicenseLifecycleActive {
		t.Errorf("LifecycleStatus = %q, want %q", license.LifecycleStatus, domain.LicenseLifecycleActive)
	}
	if license.ActivationCode == "" {
		t.Error("expected non-empty ActivationCode")
	}
	if license.Signature == "" {
		t.Error("expected non-empty Signature")
	}

	// Registration should now be bound
	var boundReg domain.LicenseRegistration
	db.First(&boundReg, reg.ID)
	if boundReg.BoundLicenseID == nil || *boundReg.BoundLicenseID != license.ID {
		t.Error("expected registration to be bound to issued license")
	}
}

func TestLicenseService_CheckExpiredLicensesAndCleanupRegistrations(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	fixedNow := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	currentNow := fixedNow
	oldTimeNow := timeNow
	timeNow = func() time.Time { return currentNow }
	defer func() { timeNow = oldTimeNow }()

	product, err := productSvc.CreateProduct("Metis", "metis-expire", "")
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
	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-EXPIRE-1",
	})
	if err != nil {
		t.Fatalf("create expiring registration: %v", err)
	}
	if err := db.Model(&domain.LicenseRegistration{}).Where("id = ?", reg.ID).Update("expires_at", fixedNow.Add(-time.Hour)).Error; err != nil {
		t.Fatalf("expire registration in setup: %v", err)
	}
	keepReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-KEEP-1",
		ExpiresAt:  ptrTime(fixedNow.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("create active registration: %v", err)
	}

	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: keepReg.Code,
		ValidFrom:        fixedNow.Add(-48 * time.Hour),
		ValidUntil:       ptrTime(fixedNow.Add(time.Hour)),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue expired license: %v", err)
	}
	if license.LifecycleStatus != domain.LicenseLifecycleActive {
		t.Fatalf("expected issued lifecycle active before check, got %q", license.LifecycleStatus)
	}

	currentNow = fixedNow.Add(2 * time.Hour)
	if err := licenseSvc.CheckExpiredLicenses(); err != nil {
		t.Fatalf("CheckExpiredLicenses: %v", err)
	}
	detail, err := licenseSvc.GetLicense(license.ID)
	if err != nil {
		t.Fatalf("GetLicense: %v", err)
	}
	if detail.LifecycleStatus != domain.LicenseLifecycleExpired {
		t.Fatalf("lifecycle status=%q, want expired", detail.LifecycleStatus)
	}

	if err := licenseSvc.CleanupExpiredRegistrations(); err != nil {
		t.Fatalf("CleanupExpiredRegistrations: %v", err)
	}
	if _, err := licenseSvc.regRepo.FindByCode(reg.Code); err == nil {
		t.Fatal("expected expired unbound registration to be deleted")
	}
	if _, err := licenseSvc.regRepo.FindByCode(keepReg.Code); err != nil {
		t.Fatalf("expected bound active registration to remain: %v", err)
	}
}

func TestLicenseService_IssueLicense_Guards(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-guard", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	// Test unpublished product
	t.Run("unpublished product", func(t *testing.T) {
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: "RG-TEST-001",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrProductNotPublished) {
			t.Errorf("expected ErrProductNotPublished, got %v", err)
		}
	})

	t.Run("missing product", func(t *testing.T) {
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID + 999,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: "RG-MISSING-PRODUCT",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, productpkg.ErrProductNotFound) {
			t.Errorf("expected ErrProductNotFound, got %v", err)
		}
	})

	// Publish product for remaining tests
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	// Test inactive licensee
	t.Run("inactive licensee", func(t *testing.T) {
		if err := licenseeSvc.UpdateLicenseeStatus(licensee.ID, domain.LicenseeStatusArchived); err != nil {
			t.Fatalf("archive licensee failed: %v", err)
		}
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: "RG-TEST-002",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrLicenseeNotActive) {
			t.Errorf("expected ErrLicenseeNotActive, got %v", err)
		}
		// Restore for other tests
		if err := licenseeSvc.UpdateLicenseeStatus(licensee.ID, domain.LicenseeStatusActive); err != nil {
			t.Fatalf("reactivate licensee failed: %v", err)
		}
	})

	t.Run("missing licensee", func(t *testing.T) {
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID + 999,
			PlanName:         "Basic",
			RegistrationCode: "RG-MISSING-LICENSEE",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, licenseepkg.ErrLicenseeNotFound) {
			t.Errorf("expected ErrLicenseeNotFound, got %v", err)
		}
	})

	t.Run("missing registration without auto create is rejected", func(t *testing.T) {
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: "RG-NOT-FOUND-001",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrRegistrationNotFound) {
			t.Errorf("expected ErrRegistrationNotFound, got %v", err)
		}
	})

	t.Run("validity window must move forward", func(t *testing.T) {
		validFrom := timeNow().Add(48 * time.Hour)
		invalidUntil := validFrom.Add(-time.Minute)
		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:              product.ID,
			LicenseeID:             licensee.ID,
			PlanName:               "Basic",
			RegistrationCode:       "RG-BAD-WINDOW-001",
			AutoCreateRegistration: true,
			ValidFrom:              validFrom,
			ValidUntil:             &invalidUntil,
			IssuedBy:               1,
		})
		if !errors.Is(err, ErrInvalidValidityPeriod) {
			t.Errorf("expected ErrInvalidValidityPeriod, got %v", err)
		}
	})

	// Test already-bound registration code
	t.Run("already bound registration code", func(t *testing.T) {
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-BOUND-001",
		})
		if err != nil {
			t.Fatalf("setup registration failed: %v", err)
		}
		// Issue first license to bind the code
		_, err = licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("first issue failed: %v", err)
		}
		// Second issue with same code should fail
		_, err = licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Pro",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrRegistrationAlreadyBound) {
			t.Errorf("expected ErrRegistrationAlreadyBound, got %v", err)
		}
	})

	// Test expired registration code
	t.Run("expired registration code", func(t *testing.T) {
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-EXPIRED-001",
		})
		if err != nil {
			t.Fatalf("setup registration failed: %v", err)
		}
		past := timeNow().Add(-24 * time.Hour)
		if err := db.Model(&domain.LicenseRegistration{}).Where("id = ?", reg.ID).Update("expires_at", past).Error; err != nil {
			t.Fatalf("expire registration in setup: %v", err)
		}
		_, err = licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrRegistrationExpired) {
			t.Errorf("expected ErrRegistrationExpired, got %v", err)
		}
	})

	t.Run("registration ownership mismatch", func(t *testing.T) {
		otherProduct, err := productSvc.CreateProduct("Other", "other-product", "")
		if err != nil {
			t.Fatalf("create other product failed: %v", err)
		}
		if err := productSvc.UpdateStatus(otherProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish other product failed: %v", err)
		}
		otherLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Other Licensee", Notes: ""})
		if err != nil {
			t.Fatalf("create other licensee failed: %v", err)
		}
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &otherProduct.ID,
			LicenseeID: &otherLicensee.ID,
			Code:       "RG-FOREIGN-001",
		})
		if err != nil {
			t.Fatalf("create foreign registration failed: %v", err)
		}

		_, err = licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if err == nil {
			t.Fatal("expected foreign registration ownership mismatch to fail")
		}
	})

	// Test missing key by deleting the auto-generated key
	t.Run("missing product key", func(t *testing.T) {
		key, _ := licenseSvc.keyRepo.FindCurrentByProductID(product.ID)
		db.Delete(&domain.ProductKey{}, key.ID)

		_, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: "RG-MISSING-001",
			ValidFrom:        timeNow(),
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrProductKeyNotFound) {
			t.Errorf("expected ErrProductKeyNotFound, got %v", err)
		}
	})

	t.Run("auto create registration when allowed", func(t *testing.T) {
		autoProduct, err := productSvc.CreateProduct("Auto Product", "prod-auto-create", "")
		if err != nil {
			t.Fatalf("create auto product: %v", err)
		}
		if err := productSvc.UpdateStatus(autoProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish auto product: %v", err)
		}
		autoLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Auto Corp"})
		if err != nil {
			t.Fatalf("create auto licensee: %v", err)
		}
		license, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:              autoProduct.ID,
			LicenseeID:             autoLicensee.ID,
			PlanName:               "AutoCreate",
			RegistrationCode:       "RG-AUTO-CREATE-001",
			AutoCreateRegistration: true,
			ValidFrom:              timeNow(),
			IssuedBy:               1,
		})
		if err != nil {
			t.Fatalf("expected auto-create issue to succeed, got %v", err)
		}
		reg, err := licenseSvc.regRepo.FindByCode("RG-AUTO-CREATE-001")
		if err != nil {
			t.Fatalf("expected auto-created registration, got %v", err)
		}
		if reg.Source != "manual_input" {
			t.Fatalf("registration source = %q, want manual_input", reg.Source)
		}
		if reg.BoundLicenseID == nil || *reg.BoundLicenseID != license.ID {
			t.Fatalf("expected registration bound to issued license, got %+v", reg)
		}
		if reg.ProductID == nil || *reg.ProductID != autoProduct.ID || reg.LicenseeID == nil || *reg.LicenseeID != autoLicensee.ID {
			t.Fatalf("unexpected auto-created registration ownership: %+v", reg)
		}
	})

	t.Run("issue rejects blank registration code even with auto create enabled", func(t *testing.T) {
		autoProduct, err := productSvc.CreateProduct("Blank Product", "prod-auto-create-blank", "")
		if err != nil {
			t.Fatalf("create blank auto product: %v", err)
		}
		if err := productSvc.UpdateStatus(autoProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish blank auto product: %v", err)
		}
		autoLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Blank Auto Corp"})
		if err != nil {
			t.Fatalf("create blank auto licensee: %v", err)
		}

		if _, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:              autoProduct.ID,
			LicenseeID:             autoLicensee.ID,
			PlanName:               "AutoCreate",
			RegistrationCode:       "   ",
			AutoCreateRegistration: true,
			ValidFrom:              timeNow(),
			IssuedBy:               1,
		}); !errors.Is(err, ErrRegistrationCodeRequired) {
			t.Fatalf("blank registration issue error = %v, want %v", err, ErrRegistrationCodeRequired)
		}
	})
}

func TestLicenseService_CreateLicenseRegistration_RejectsAlreadyExpiredCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("Expired Registration Product", "expired-reg-product", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Expired Registration Corp"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}

	expiredAt := timeNow().Add(-time.Minute)
	if _, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-EXPIRED-CREATE-001",
		ExpiresAt:  &expiredAt,
	}); !errors.Is(err, ErrRegistrationExpired) {
		t.Fatalf("CreateLicenseRegistration expired-at-past error = %v, want %v", err, ErrRegistrationExpired)
	}
}

func TestLicenseService_UpgradeLicense(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-upgrade", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-UPGRADE-001",
	})
	if err != nil {
		t.Fatalf("setup registration failed: %v", err)
	}

	original, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}

	newLicense, err := licenseSvc.UpgradeLicense(original.ID, IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Pro",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow(),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newLicense.PlanName != "Pro" {
		t.Errorf("PlanName = %q, want %q", newLicense.PlanName, "Pro")
	}
	if newLicense.OriginalLicenseID == nil || *newLicense.OriginalLicenseID != original.ID {
		t.Error("expected new license to reference original license")
	}

	// Original should be revoked
	var orig domain.License
	db.First(&orig, original.ID)
	if orig.LifecycleStatus != domain.LicenseLifecycleRevoked {
		t.Errorf("original lifecycle = %q, want %q", orig.LifecycleStatus, domain.LicenseLifecycleRevoked)
	}
	if orig.Status != domain.LicenseStatusRevoked {
		t.Errorf("original status = %q, want %q", orig.Status, domain.LicenseStatusRevoked)
	}

	// Registration should be bound to new license
	var boundReg domain.LicenseRegistration
	db.First(&boundReg, reg.ID)
	if boundReg.BoundLicenseID == nil || *boundReg.BoundLicenseID != newLicense.ID {
		t.Error("expected registration to be bound to upgraded license")
	}

	t.Run("rejects non-forward validity window", func(t *testing.T) {
		badUntil := timeNow().Add(-2 * time.Hour)
		_, err := licenseSvc.UpgradeLicense(newLicense.ID, IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Broken",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			ValidUntil:       &badUntil,
			IssuedBy:         1,
		})
		if !errors.Is(err, ErrInvalidValidityPeriod) {
			t.Fatalf("expected ErrInvalidValidityPeriod, got %v", err)
		}
	})

	foreignProduct, err := productSvc.CreateProduct("Foreign Product", "foreign-upgrade-product", "")
	if err != nil {
		t.Fatalf("create foreign product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(foreignProduct.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish foreign product failed: %v", err)
	}
	foreignLicensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Foreign Upgrade Licensee"})
	if err != nil {
		t.Fatalf("create foreign licensee failed: %v", err)
	}
	foreignReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &foreignProduct.ID,
		LicenseeID: &foreignLicensee.ID,
		Code:       "RG-UPGRADE-FOREIGN-001",
	})
	if err != nil {
		t.Fatalf("create foreign registration failed: %v", err)
	}
	freshReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-UPGRADE-FOREIGN-SRC-001",
	})
	if err != nil {
		t.Fatalf("create fresh source registration failed: %v", err)
	}
	freshOriginal, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: freshReg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue fresh source license failed: %v", err)
	}

	_, err = licenseSvc.UpgradeLicense(freshOriginal.ID, IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "ForeignScope",
		RegistrationCode: foreignReg.Code,
		ValidFrom:        timeNow(),
		IssuedBy:         1,
	})
	if !errors.Is(err, ErrRegistrationOwnership) {
		t.Fatalf("expected ErrRegistrationOwnership on foreign upgrade registration, got %v", err)
	}

	_, err = licenseSvc.UpgradeLicense(freshOriginal.ID, IssueLicenseParams{
		ProductID:        foreignProduct.ID,
		LicenseeID:       foreignLicensee.ID,
		PlanName:         "CrossScope",
		RegistrationCode: foreignReg.Code,
		ValidFrom:        timeNow(),
		IssuedBy:         1,
	})
	if !errors.Is(err, ErrUpgradeScopeMismatch) {
		t.Fatalf("expected ErrUpgradeScopeMismatch on cross-scope upgrade, got %v", err)
	}
}

func TestLicenseService_UpgradeLicense_ReusesOriginalRegistrationCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-upgrade-reuse", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Reuse Corp"})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-UPGRADE-REUSE-001",
	})
	if err != nil {
		t.Fatalf("setup registration failed: %v", err)
	}

	original, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}

	upgraded, err := licenseSvc.UpgradeLicense(original.ID, IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Enterprise",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow(),
		IssuedBy:         2,
	})
	if err != nil {
		t.Fatalf("upgrade with reused registration code failed: %v", err)
	}
	if upgraded.ID == original.ID || upgraded.OriginalLicenseID == nil || *upgraded.OriginalLicenseID != original.ID {
		t.Fatalf("unexpected upgraded license linkage: original=%+v upgraded=%+v", original, upgraded)
	}
	if upgraded.RegistrationCode != reg.Code || upgraded.PlanName != "Enterprise" {
		t.Fatalf("unexpected upgraded license payload: %+v", upgraded)
	}

	var originalReloaded domain.License
	if err := db.First(&originalReloaded, original.ID).Error; err != nil {
		t.Fatalf("reload original license: %v", err)
	}
	if originalReloaded.LifecycleStatus != domain.LicenseLifecycleRevoked || originalReloaded.RevokedBy == nil || *originalReloaded.RevokedBy != 2 {
		t.Fatalf("expected original license revoked by upgrader, got %+v", originalReloaded)
	}

	var rebound domain.LicenseRegistration
	if err := db.First(&rebound, reg.ID).Error; err != nil {
		t.Fatalf("reload registration: %v", err)
	}
	if rebound.BoundLicenseID == nil || *rebound.BoundLicenseID != upgraded.ID {
		t.Fatalf("expected registration rebound to upgraded license, got %+v", rebound)
	}
}

func TestLicenseService_UpgradeLicense_PreservesSuspendedLifecycle(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-upgrade-suspended", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Suspended Upgrade Corp"})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-UPGRADE-SUSPEND-001",
	})
	if err != nil {
		t.Fatalf("setup registration failed: %v", err)
	}

	original, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}
	if err := licenseSvc.SuspendLicense(original.ID, 9); err != nil {
		t.Fatalf("suspend original before upgrade: %v", err)
	}

	upgraded, err := licenseSvc.UpgradeLicense(original.ID, IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Enterprise",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-30 * time.Minute),
		IssuedBy:         2,
	})
	if err != nil {
		t.Fatalf("upgrade suspended license: %v", err)
	}

	var detail domain.License
	if err := db.First(&detail, upgraded.ID).Error; err != nil {
		t.Fatalf("reload upgraded license: %v", err)
	}
	if detail.LifecycleStatus != domain.LicenseLifecycleSuspended {
		t.Fatalf("upgraded lifecycle = %q, want %q", detail.LifecycleStatus, domain.LicenseLifecycleSuspended)
	}
	if detail.SuspendedAt == nil || detail.SuspendedBy == nil || *detail.SuspendedBy != 9 {
		t.Fatalf("expected upgraded suspension metadata to be preserved, got %+v", detail)
	}
}

func TestLicenseService_UpgradeLicense_PreservesPendingLifecycle(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-upgrade-pending", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Pending Upgrade Corp"})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-UPGRADE-PENDING-001",
	})
	if err != nil {
		t.Fatalf("setup registration failed: %v", err)
	}

	validFrom := timeNow().Add(48 * time.Hour)
	validUntil := timeNow().Add(96 * time.Hour)
	original, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        validFrom,
		ValidUntil:       &validUntil,
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}
	if original.LifecycleStatus != domain.LicenseLifecyclePending {
		t.Fatalf("original lifecycle = %q, want %q", original.LifecycleStatus, domain.LicenseLifecyclePending)
	}

	upgraded, err := licenseSvc.UpgradeLicense(original.ID, IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Enterprise",
		RegistrationCode: reg.Code,
		ValidFrom:        validFrom,
		ValidUntil:       &validUntil,
		IssuedBy:         2,
	})
	if err != nil {
		t.Fatalf("upgrade pending license: %v", err)
	}

	var detail domain.License
	if err := db.First(&detail, upgraded.ID).Error; err != nil {
		t.Fatalf("reload upgraded license: %v", err)
	}
	if detail.LifecycleStatus != domain.LicenseLifecyclePending {
		t.Fatalf("upgraded lifecycle = %q, want %q", detail.LifecycleStatus, domain.LicenseLifecyclePending)
	}
	if !detail.ValidFrom.Equal(validFrom) {
		t.Fatalf("upgraded valid_from = %v, want %v", detail.ValidFrom, validFrom)
	}
	if detail.ValidUntil == nil || !detail.ValidUntil.Equal(validUntil) {
		t.Fatalf("upgraded valid_until = %v, want %v", detail.ValidUntil, validUntil)
	}
}

func TestLicenseService_LifecycleStateMachine(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-lifecycle", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	issue := func(code string) *domain.License {
		reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       code,
		})
		l, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue failed: %v", err)
		}
		return l
	}

	t.Run("revoke sets timestamps", func(t *testing.T) {
		l := issue("RG-REVOKE-001")
		if err := licenseSvc.RevokeLicense(l.ID, 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var updated domain.License
		db.First(&updated, l.ID)
		if updated.Status != domain.LicenseStatusRevoked {
			t.Errorf("status = %q, want %q", updated.Status, domain.LicenseStatusRevoked)
		}
		if updated.LifecycleStatus != domain.LicenseLifecycleRevoked {
			t.Errorf("lifecycle = %q, want %q", updated.LifecycleStatus, domain.LicenseLifecycleRevoked)
		}
		if updated.RevokedAt == nil || updated.RevokedBy == nil || *updated.RevokedBy != 2 {
			t.Error("expected revoked_at and revoked_by to be set")
		}
		// Double revoke should fail
		if !errors.Is(licenseSvc.RevokeLicense(l.ID, 2), ErrLicenseAlreadyRevoked) {
			t.Error("expected ErrLicenseAlreadyRevoked on second revoke")
		}
	})

	t.Run("suspend and reactivate", func(t *testing.T) {
		l := issue("RG-SUSPEND-001")
		if err := licenseSvc.SuspendLicense(l.ID, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var suspended domain.License
		db.First(&suspended, l.ID)
		if suspended.LifecycleStatus != domain.LicenseLifecycleSuspended {
			t.Errorf("lifecycle = %q, want %q", suspended.LifecycleStatus, domain.LicenseLifecycleSuspended)
		}
		if suspended.SuspendedAt == nil || suspended.SuspendedBy == nil || *suspended.SuspendedBy != 3 {
			t.Error("expected suspended_at and suspended_by to be set")
		}

		// Double suspend should fail
		if !errors.Is(licenseSvc.SuspendLicense(l.ID, 3), ErrLicenseAlreadySuspended) {
			t.Error("expected ErrLicenseAlreadySuspended on second suspend")
		}

		// Reactivate
		if err := licenseSvc.ReactivateLicense(l.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var reactivated domain.License
		db.First(&reactivated, l.ID)
		if reactivated.LifecycleStatus != domain.LicenseLifecycleActive {
			t.Errorf("lifecycle = %q, want %q", reactivated.LifecycleStatus, domain.LicenseLifecycleActive)
		}
		if reactivated.SuspendedAt != nil {
			t.Error("expected suspended_at to be cleared")
		}

		// Reactivate non-suspended should fail
		if !errors.Is(licenseSvc.ReactivateLicense(l.ID), ErrLicenseNotSuspended) {
			t.Error("expected ErrLicenseNotSuspended")
		}
	})

	t.Run("renew extends expiration", func(t *testing.T) {
		l := issue("RG-RENEW-001")
		originalActivationCode := l.ActivationCode
		originalSignature := l.Signature
		originalKeyVersion := l.KeyVersion

		if _, err := productSvc.RotateKey(product.ID); err != nil {
			t.Fatalf("rotate key failed: %v", err)
		}

		newExpiry := timeNow().Add(30 * 24 * time.Hour)
		if err := licenseSvc.RenewLicense(l.ID, &newExpiry, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var renewed domain.License
		db.First(&renewed, l.ID)
		if renewed.LifecycleStatus != domain.LicenseLifecycleActive {
			t.Errorf("lifecycle = %q, want %q", renewed.LifecycleStatus, domain.LicenseLifecycleActive)
		}
		if renewed.ValidUntil == nil || !renewed.ValidUntil.Equal(newExpiry) {
			t.Error("expected valid_until to be updated")
		}
		if renewed.ID != l.ID {
			t.Errorf("ID = %d, want original ID %d", renewed.ID, l.ID)
		}
		if renewed.ActivationCode == originalActivationCode {
			t.Error("expected activation code to change after renew")
		}
		if renewed.Signature == originalSignature {
			t.Error("expected signature to change after renew")
		}
		if renewed.KeyVersion != originalKeyVersion+1 {
			t.Errorf("KeyVersion = %d, want %d", renewed.KeyVersion, originalKeyVersion+1)
		}

		claims, err := licensecrypto.DecodeActivationCode(renewed.ActivationCode)
		if err != nil {
			t.Fatalf("decode renewed activation code failed: %v", err)
		}
		exp, ok := claims["exp"].(float64)
		if !ok {
			t.Fatalf("exp claim type = %T, want number", claims["exp"])
		}
		if int64(exp) != newExpiry.Unix() {
			t.Errorf("exp = %d, want %d", int64(exp), newExpiry.Unix())
		}
		sig, ok := claims["sig"].(string)
		if !ok || sig == "" {
			t.Fatalf("sig claim missing")
		}
		delete(claims, "sig")
		key, err := licenseSvc.keyRepo.FindByProductIDAndVersion(product.ID, renewed.KeyVersion)
		if err != nil {
			t.Fatalf("find renewed key failed: %v", err)
		}
		valid, err := licensecrypto.VerifyLicenseSignature(claims, sig, key.PublicKey)
		if err != nil {
			t.Fatalf("verify renewed signature failed: %v", err)
		}
		if !valid {
			t.Fatal("expected renewed signature to be valid")
		}
	})

	t.Run("renew rejects validUntil at or before validFrom", func(t *testing.T) {
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-RENEW-BAD-WINDOW-001",
		})
		if err != nil {
			t.Fatalf("create registration: %v", err)
		}
		validFrom := timeNow().Add(24 * time.Hour)
		license, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Bad Renew Window",
			RegistrationCode: reg.Code,
			ValidFrom:        validFrom,
			ValidUntil:       ptrTime(validFrom.Add(24 * time.Hour)),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue pending license: %v", err)
		}
		invalidUntil := validFrom
		if err := licenseSvc.RenewLicense(license.ID, &invalidUntil, 1); !errors.Is(err, ErrInvalidValidityPeriod) {
			t.Fatalf("expected ErrInvalidValidityPeriod, got %v", err)
		}
	})

	t.Run("status revoked drift blocks renew and preserves payload", func(t *testing.T) {
		l := issue("RG-RENEW-REVOKED-DRIFT-001")
		beforeActivation := l.ActivationCode
		beforeSignature := l.Signature
		beforeVersion := l.KeyVersion
		newExpiry := timeNow().Add(45 * 24 * time.Hour)

		if err := db.Model(&domain.License{}).Where("id = ?", l.ID).Update("status", domain.LicenseStatusRevoked).Error; err != nil {
			t.Fatalf("simulate status revoked drift: %v", err)
		}

		if err := licenseSvc.RenewLicense(l.ID, &newExpiry, 4); !errors.Is(err, ErrLicenseAlreadyRevoked) {
			t.Fatalf("expected ErrLicenseAlreadyRevoked, got %v", err)
		}

		var reloaded domain.License
		if err := db.First(&reloaded, l.ID).Error; err != nil {
			t.Fatalf("reload drift license: %v", err)
		}
		if reloaded.KeyVersion != beforeVersion || reloaded.ActivationCode != beforeActivation || reloaded.Signature != beforeSignature {
			t.Fatalf("revoked drift license should not be renewed, got %+v", reloaded)
		}
	})

	t.Run("renew to permanent clears expiration in certificate", func(t *testing.T) {
		reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-RENEW-PERM-001",
		})
		oldExpiry := timeNow().Add(24 * time.Hour)
		l, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			ValidUntil:       &oldExpiry,
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue failed: %v", err)
		}
		if err := licenseSvc.RenewLicense(l.ID, nil, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var renewed domain.License
		db.First(&renewed, l.ID)
		if renewed.ValidUntil != nil {
			t.Error("expected valid_until to be cleared")
		}
		claims, err := licensecrypto.DecodeActivationCode(renewed.ActivationCode)
		if err != nil {
			t.Fatalf("decode renewed activation code failed: %v", err)
		}
		if claims["exp"] != nil {
			t.Errorf("exp = %#v, want nil", claims["exp"])
		}
		sig := claims["sig"].(string)
		delete(claims, "sig")
		key, err := licenseSvc.keyRepo.FindByProductIDAndVersion(product.ID, renewed.KeyVersion)
		if err != nil {
			t.Fatalf("find renewed key failed: %v", err)
		}
		valid, err := licensecrypto.VerifyLicenseSignature(claims, sig, key.PublicKey)
		if err != nil {
			t.Fatalf("verify renewed signature failed: %v", err)
		}
		if !valid {
			t.Fatal("expected permanent renewed signature to be valid")
		}
	})

	t.Run("renew to past expiration marks expired", func(t *testing.T) {
		l := issue("RG-RENEW-EXPIRED-001")
		newExpiry := timeNow().Add(-time.Minute)
		if err := licenseSvc.RenewLicense(l.ID, &newExpiry, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var renewed domain.License
		db.First(&renewed, l.ID)
		if renewed.LifecycleStatus != domain.LicenseLifecycleExpired {
			t.Errorf("lifecycle = %q, want %q", renewed.LifecycleStatus, domain.LicenseLifecycleExpired)
		}
	})

	t.Run("renew keeps future-dated license pending", func(t *testing.T) {
		reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-RENEW-PENDING-001",
		})
		validFrom := timeNow().Add(48 * time.Hour)
		initialExpiry := timeNow().Add(72 * time.Hour)
		l, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        validFrom,
			ValidUntil:       &initialExpiry,
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue pending license: %v", err)
		}
		if l.LifecycleStatus != domain.LicenseLifecyclePending {
			t.Fatalf("issued lifecycle = %q, want %q", l.LifecycleStatus, domain.LicenseLifecyclePending)
		}

		newExpiry := timeNow().Add(96 * time.Hour)
		if err := licenseSvc.RenewLicense(l.ID, &newExpiry, 4); err != nil {
			t.Fatalf("renew pending license: %v", err)
		}

		var renewed domain.License
		db.First(&renewed, l.ID)
		if renewed.LifecycleStatus != domain.LicenseLifecyclePending {
			t.Fatalf("renewed lifecycle = %q, want %q", renewed.LifecycleStatus, domain.LicenseLifecyclePending)
		}
		if renewed.ValidUntil == nil || !renewed.ValidUntil.Equal(newExpiry) {
			t.Fatalf("renewed valid_until = %v, want %v", renewed.ValidUntil, newExpiry)
		}
	})

	t.Run("renew keeps suspended license suspended", func(t *testing.T) {
		reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-RENEW-SUSPENDED-001",
		})
		initialExpiry := timeNow().Add(24 * time.Hour)
		l, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			ValidUntil:       &initialExpiry,
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue suspended-renew license: %v", err)
		}
		if err := licenseSvc.SuspendLicense(l.ID, 7); err != nil {
			t.Fatalf("suspend license before renew: %v", err)
		}

		newExpiry := timeNow().Add(96 * time.Hour)
		if err := licenseSvc.RenewLicense(l.ID, &newExpiry, 4); err != nil {
			t.Fatalf("renew suspended license: %v", err)
		}

		var renewed domain.License
		db.First(&renewed, l.ID)
		if renewed.LifecycleStatus != domain.LicenseLifecycleSuspended {
			t.Fatalf("renewed lifecycle = %q, want %q", renewed.LifecycleStatus, domain.LicenseLifecycleSuspended)
		}
		if renewed.ValidUntil == nil || !renewed.ValidUntil.Equal(newExpiry) {
			t.Fatalf("renewed valid_until = %v, want %v", renewed.ValidUntil, newExpiry)
		}
		if renewed.SuspendedAt == nil || renewed.SuspendedBy == nil || *renewed.SuspendedBy != 7 {
			t.Fatalf("renew should preserve suspension markers, got %+v", renewed)
		}
	})

	t.Run("reactivate keeps future-dated license pending under injected clock", func(t *testing.T) {
		fixedNow := time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC)
		oldTimeNow := timeNow
		timeNow = func() time.Time { return fixedNow }
		defer func() { timeNow = oldTimeNow }()

		reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-REACTIVATE-PENDING-001",
		})
		validFrom := fixedNow.Add(48 * time.Hour)
		initialExpiry := fixedNow.Add(72 * time.Hour)
		l, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: reg.Code,
			ValidFrom:        validFrom,
			ValidUntil:       &initialExpiry,
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue pending license: %v", err)
		}
		if l.LifecycleStatus != domain.LicenseLifecyclePending {
			t.Fatalf("issued lifecycle = %q, want %q", l.LifecycleStatus, domain.LicenseLifecyclePending)
		}
		if err := licenseSvc.SuspendLicense(l.ID, 2); err != nil {
			t.Fatalf("suspend pending license: %v", err)
		}
		if err := licenseSvc.ReactivateLicense(l.ID); err != nil {
			t.Fatalf("reactivate pending license: %v", err)
		}

		var reactivated domain.License
		db.First(&reactivated, l.ID)
		if reactivated.LifecycleStatus != domain.LicenseLifecyclePending {
			t.Fatalf("reactivated lifecycle = %q, want %q", reactivated.LifecycleStatus, domain.LicenseLifecyclePending)
		}
	})

	t.Run("operations on revoked license are blocked", func(t *testing.T) {
		l := issue("RG-REVOKED-OP-001")
		if err := licenseSvc.RevokeLicense(l.ID, 2); err != nil {
			t.Fatalf("setup revoke failed: %v", err)
		}
		if !errors.Is(licenseSvc.SuspendLicense(l.ID, 1), ErrLicenseAlreadyRevoked) {
			t.Error("expected ErrLicenseAlreadyRevoked on suspend")
		}
		if !errors.Is(licenseSvc.RenewLicense(l.ID, nil, 1), ErrLicenseAlreadyRevoked) {
			t.Error("expected ErrLicenseAlreadyRevoked on renew")
		}
		_, err := licenseSvc.UpgradeLicense(l.ID, IssueLicenseParams{})
		if !errors.Is(err, ErrLicenseAlreadyRevoked) {
			t.Errorf("expected ErrLicenseAlreadyRevoked on upgrade, got %v", err)
		}
		if !errors.Is(licenseSvc.ReactivateLicense(l.ID), ErrLicenseAlreadyRevoked) {
			t.Error("expected ErrLicenseAlreadyRevoked on reactivate")
		}
	})

	t.Run("missing license ids surface not found consistently", func(t *testing.T) {
		if !errors.Is(licenseSvc.RevokeLicense(999999, 1), ErrLicenseNotFound) {
			t.Error("expected ErrLicenseNotFound on revoke")
		}
		if !errors.Is(licenseSvc.SuspendLicense(999999, 1), ErrLicenseNotFound) {
			t.Error("expected ErrLicenseNotFound on suspend")
		}
		if !errors.Is(licenseSvc.ReactivateLicense(999999), ErrLicenseNotFound) {
			t.Error("expected ErrLicenseNotFound on reactivate")
		}
		if !errors.Is(licenseSvc.RenewLicense(999999, nil, 1), ErrLicenseNotFound) {
			t.Error("expected ErrLicenseNotFound on renew")
		}
		if _, err := licenseSvc.GetLicense(999999); !errors.Is(err, ErrLicenseNotFound) {
			t.Errorf("expected ErrLicenseNotFound on get, got %v", err)
		}
		if _, _, err := licenseSvc.ExportLicFile(999999, "v1"); !errors.Is(err, ErrLicenseNotFound) {
			t.Errorf("expected ErrLicenseNotFound on export, got %v", err)
		}
	})
}

func TestLicenseService_BulkReissueLicenses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-reissue", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	// Issue a license
	reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-REISSUE-001",
	})
	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}

	originalVersion := license.KeyVersion

	// Rotate key so the license becomes reissueable
	_, err = productSvc.RotateKey(product.ID)
	if err != nil {
		t.Fatalf("rotate key failed: %v", err)
	}

	// Reissue by explicit ID
	reissued, err := licenseSvc.BulkReissueLicenses(product.ID, []uint{license.ID}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reissued != 1 {
		t.Errorf("reissued = %d, want 1", reissued)
	}

	var updated domain.License
	db.First(&updated, license.ID)
	if updated.KeyVersion != originalVersion+1 {
		t.Errorf("KeyVersion = %d, want %d", updated.KeyVersion, originalVersion+1)
	}
	if updated.ActivationCode == license.ActivationCode {
		t.Error("expected activation code to change after reissue")
	}

	// Test 100-item limit
	manyIDs := make([]uint, 101)
	for i := range manyIDs {
		manyIDs[i] = uint(i + 1)
	}
	_, err = licenseSvc.BulkReissueLicenses(product.ID, manyIDs, 1)
	if !errors.Is(err, ErrBulkReissueTooMany) {
		t.Errorf("expected ErrBulkReissueTooMany, got %v", err)
	}

	t.Run("explicit ids skip missing foreign and revoked licenses", func(t *testing.T) {
		otherProduct, err := productSvc.CreateProduct("Other", "prod-reissue-other", "")
		if err != nil {
			t.Fatalf("create other product: %v", err)
		}
		if err := productSvc.UpdateStatus(otherProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish other product: %v", err)
		}

		otherReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &otherProduct.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-REISSUE-OTHER-001",
		})
		if err != nil {
			t.Fatalf("create other registration: %v", err)
		}
		otherLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        otherProduct.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Other",
			RegistrationCode: otherReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue other product license: %v", err)
		}

		revokedReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-REISSUE-REVOKED-001",
		})
		if err != nil {
			t.Fatalf("create revoked registration: %v", err)
		}
		revokedLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Revoked",
			RegistrationCode: revokedReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue revoked candidate: %v", err)
		}
		if err := licenseSvc.RevokeLicense(revokedLicense.ID, 8); err != nil {
			t.Fatalf("revoke explicit candidate: %v", err)
		}

		beforeOtherActivation := otherLicense.ActivationCode
		beforeRevokedActivation := revokedLicense.ActivationCode
		if _, err := productSvc.RotateKey(product.ID); err != nil {
			t.Fatalf("rotate key for explicit mixed candidate: %v", err)
		}

		reissued, err := licenseSvc.BulkReissueLicenses(product.ID, []uint{license.ID, 999999, otherLicense.ID, revokedLicense.ID}, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses mixed explicit ids: %v", err)
		}
		if reissued != 1 {
			t.Fatalf("reissued = %d, want 1", reissued)
		}

		var reloadedOther domain.License
		if err := db.First(&reloadedOther, otherLicense.ID).Error; err != nil {
			t.Fatalf("reload other license: %v", err)
		}
		if reloadedOther.ActivationCode != beforeOtherActivation {
			t.Fatalf("foreign product license should be skipped, got %+v", reloadedOther)
		}

		var reloadedRevoked domain.License
		if err := db.First(&reloadedRevoked, revokedLicense.ID).Error; err != nil {
			t.Fatalf("reload revoked license: %v", err)
		}
		if reloadedRevoked.ActivationCode != beforeRevokedActivation {
			t.Fatalf("revoked explicit license should be skipped, got %+v", reloadedRevoked)
		}
	})

	t.Run("explicit ids skip licenses already on current key version", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("CurrentOnly", "prod-reissue-current-only", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Current Only Corp"})
		if err != nil {
			t.Fatalf("create licensee: %v", err)
		}
		localReg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-REISSUE-CURRENT-ONLY-001",
		})
		if err != nil {
			t.Fatalf("create registration: %v", err)
		}
		currentOnly, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue current-version license: %v", err)
		}

		beforeCurrentActivation := currentOnly.ActivationCode
		beforeCurrentSignature := currentOnly.Signature
		beforeCurrentVersion := currentOnly.KeyVersion

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, []uint{currentOnly.ID}, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses current-version explicit id: %v", err)
		}
		if reissued != 0 {
			t.Fatalf("reissued = %d, want 0", reissued)
		}

		var currentReloaded domain.License
		if err := localDB.First(&currentReloaded, currentOnly.ID).Error; err != nil {
			t.Fatalf("reload current-version license: %v", err)
		}
		if currentReloaded.KeyVersion != beforeCurrentVersion {
			t.Fatalf("current-version key changed unexpectedly: got %d want %d", currentReloaded.KeyVersion, beforeCurrentVersion)
		}
		if currentReloaded.ActivationCode != beforeCurrentActivation || currentReloaded.Signature != beforeCurrentSignature {
			t.Fatalf("current-version license should be skipped, got %+v", currentReloaded)
		}
	})

	t.Run("explicit ids skip status-revoked drift licenses", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("StatusDrift", "prod-reissue-status-drift", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Status Drift Corp"})
		if err != nil {
			t.Fatalf("create licensee: %v", err)
		}
		localReg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-REISSUE-STATUS-DRIFT-001",
		})
		if err != nil {
			t.Fatalf("create registration: %v", err)
		}
		statusDrift, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue license: %v", err)
		}
		if _, err := localProductSvc.RotateKey(localProduct.ID); err != nil {
			t.Fatalf("rotate key: %v", err)
		}
		if err := localDB.Model(&domain.License{}).Where("id = ?", statusDrift.ID).Update("status", domain.LicenseStatusRevoked).Error; err != nil {
			t.Fatalf("simulate status revoked drift: %v", err)
		}

		beforeActivation := statusDrift.ActivationCode
		beforeSignature := statusDrift.Signature
		beforeVersion := statusDrift.KeyVersion

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, []uint{statusDrift.ID}, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses status revoked drift: %v", err)
		}
		if reissued != 0 {
			t.Fatalf("reissued = %d, want 0", reissued)
		}

		var reloaded domain.License
		if err := localDB.First(&reloaded, statusDrift.ID).Error; err != nil {
			t.Fatalf("reload status drift license: %v", err)
		}
		if reloaded.KeyVersion != beforeVersion || reloaded.ActivationCode != beforeActivation || reloaded.Signature != beforeSignature {
			t.Fatalf("status-revoked drift license should be skipped, got %+v", reloaded)
		}
	})

	t.Run("explicit ids are deduplicated before limit and processing", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("Dedup", "prod-reissue-dedup", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Dedup Corp"})
		if err != nil {
			t.Fatalf("create licensee: %v", err)
		}
		localReg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-REISSUE-DEDUP-001",
		})
		if err != nil {
			t.Fatalf("create registration: %v", err)
		}
		localLicense, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue license: %v", err)
		}
		if _, err := localProductSvc.RotateKey(localProduct.ID); err != nil {
			t.Fatalf("rotate key: %v", err)
		}

		duplicateIDs := make([]uint, 101)
		for i := range duplicateIDs {
			duplicateIDs[i] = localLicense.ID
		}

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, duplicateIDs, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses duplicate ids: %v", err)
		}
		if reissued != 1 {
			t.Fatalf("reissued = %d, want 1", reissued)
		}

		var refreshed domain.License
		if err := localDB.First(&refreshed, localLicense.ID).Error; err != nil {
			t.Fatalf("reload license: %v", err)
		}
		if refreshed.KeyVersion != 2 {
			t.Fatalf("key version = %d, want 2", refreshed.KeyVersion)
		}
	})

	t.Run("auto select returns zero when no outdated active licenses remain", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("AutoZero", "prod-reissue-zero", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Zero Corp"})
		if err != nil {
			t.Fatalf("create licensee: %v", err)
		}

		createIssued := func(code string) *domain.License {
			t.Helper()
			reg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
				ProductID:  &localProduct.ID,
				LicenseeID: &localLicensee.ID,
				Code:       code,
			})
			if err != nil {
				t.Fatalf("create registration %s: %v", code, err)
			}
			issued, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
				ProductID:        localProduct.ID,
				LicenseeID:       localLicensee.ID,
				PlanName:         "Enterprise",
				RegistrationCode: reg.Code,
				ValidFrom:        timeNow().Add(-time.Hour),
				IssuedBy:         1,
			})
			if err != nil {
				t.Fatalf("issue license %s: %v", code, err)
			}
			return issued
		}

		current := createIssued("RG-REISSUE-ZERO-CURRENT")
		revoked := createIssued("RG-REISSUE-ZERO-REVOKED")
		if err := localLicenseSvc.RevokeLicense(revoked.ID, 9); err != nil {
			t.Fatalf("revoke candidate: %v", err)
		}

		impact, err := localLicenseSvc.AssessKeyRotationImpact(localProduct.ID)
		if err != nil {
			t.Fatalf("AssessKeyRotationImpact: %v", err)
		}
		if impact.AffectedCount != 0 {
			t.Fatalf("affected count = %d, want 0", impact.AffectedCount)
		}

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, nil, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses zero-auto-select: %v", err)
		}
		if reissued != 0 {
			t.Fatalf("reissued = %d, want 0", reissued)
		}

		var refreshedCurrent domain.License
		if err := localDB.First(&refreshedCurrent, current.ID).Error; err != nil {
			t.Fatalf("reload current license: %v", err)
		}
		if refreshedCurrent.KeyVersion != current.KeyVersion {
			t.Fatalf("current key version changed unexpectedly: got %d want %d", refreshedCurrent.KeyVersion, current.KeyVersion)
		}
	})
}

func TestLicenseService_ExportLicFile(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("domain.Product", "prod-export", "")
	if err != nil {
		t.Fatalf("setup product failed: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product failed: %v", err)
	}

	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Acme Corp", Notes: ""})
	if err != nil {
		t.Fatalf("setup licensee failed: %v", err)
	}

	reg, _ := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-EXPORT-001",
	})
	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Basic",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("setup issue failed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		content, filename, err := licenseSvc.ExportLicFile(license.ID, "v1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content == "" {
			t.Error("expected non-empty content")
		}
		if filename == "" {
			t.Error("expected non-empty filename")
		}
	})

	t.Run("v2 export decrypts with product license key", func(t *testing.T) {
		content, filename, err := licenseSvc.ExportLicFile(license.ID, "v2")
		if err != nil {
			t.Fatalf("unexpected v2 export error: %v", err)
		}
		if filename == "" {
			t.Fatal("expected non-empty filename")
		}
		plain, err := licensecrypto.DecryptLicenseFileV2(content, reg.Code, product.LicenseKey)
		if err != nil {
			t.Fatalf("DecryptLicenseFileV2: %v", err)
		}
		if !strings.Contains(string(plain), license.ActivationCode) {
			t.Fatalf("expected exported v2 file to contain activation code, got %s", string(plain))
		}
	})

	t.Run("falls back to product code identity and generic filename", func(t *testing.T) {
		if err := db.Model(&domain.Product{}).Where("id = ?", product.ID).Updates(map[string]any{
			"name": "",
			"code": "",
		}).Error; err != nil {
			t.Fatalf("clear product name/code: %v", err)
		}
		content, filename, err := licenseSvc.ExportLicFile(license.ID, "v1")
		if err != nil {
			t.Fatalf("ExportLicFile fallback identity: %v", err)
		}
		if !strings.HasPrefix(filename, "license_") || !strings.HasSuffix(filename, ".lic") {
			t.Fatalf("expected generic license filename, got %q", filename)
		}

		plain, err := licensecrypto.DecryptLicenseFile(content, reg.Code)
		if err != nil {
			t.Fatalf("DecryptLicenseFile fallback identity: %v", err)
		}
		if !strings.Contains(string(plain), license.ActivationCode) {
			t.Fatalf("expected fallback export to contain activation code, got %s", string(plain))
		}
	})

	t.Run("revoked license cannot be exported", func(t *testing.T) {
		if err := licenseSvc.RevokeLicense(license.ID, 1); err != nil {
			t.Fatalf("setup revoke failed: %v", err)
		}
		_, _, err := licenseSvc.ExportLicFile(license.ID, "v1")
		if !errors.Is(err, ErrRevokedLicenseNoExport) {
			t.Errorf("expected ErrRevokedLicenseNoExport, got %v", err)
		}
	})

	t.Run("status revoked drift still blocks export", func(t *testing.T) {
		freshReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       "RG-EXPORT-DRIFT-001",
		})
		if err != nil {
			t.Fatalf("create drift reg: %v", err)
		}
		driftLicense, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Basic",
			RegistrationCode: freshReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue drift license: %v", err)
		}
		if err := db.Model(&domain.License{}).Where("id = ?", driftLicense.ID).Update("status", domain.LicenseStatusRevoked).Error; err != nil {
			t.Fatalf("drift revoke status: %v", err)
		}
		_, _, err = licenseSvc.ExportLicFile(driftLicense.ID, "v1")
		if !errors.Is(err, ErrRevokedLicenseNoExport) {
			t.Fatalf("status drift export error = %v, want %v", err, ErrRevokedLicenseNoExport)
		}
	})
}

func TestLicenseService_CreateAndGenerateRegistrations(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("Registration Product", "registration-product", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Registration Corp"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}

	t.Run("create defaults source and rejects duplicates", func(t *testing.T) {
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{Code: "RG-CREATE-001"})
		if err != nil {
			t.Fatalf("CreateLicenseRegistration: %v", err)
		}
		if reg.Source != "pre_registered" {
			t.Fatalf("registration source = %q, want pre_registered", reg.Source)
		}
		if _, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{Code: "RG-CREATE-001"}); err == nil {
			t.Fatal("expected duplicate registration code to fail")
		}
	})

	t.Run("generate marks source auto_generated", func(t *testing.T) {
		reg, err := licenseSvc.GenerateLicenseRegistration(nil, nil)
		if err != nil {
			t.Fatalf("GenerateLicenseRegistration: %v", err)
		}
		if reg.Source != "auto_generated" {
			t.Fatalf("generated registration source = %q, want auto_generated", reg.Source)
		}
		if reg.Code == "" || !strings.HasPrefix(reg.Code, "RG-") {
			t.Fatalf("unexpected generated registration code: %+v", reg)
		}
	})

	t.Run("create without code auto generates prefix and keeps metadata", func(t *testing.T) {
		productID := product.ID
		licenseeID := licensee.ID
		expiresAt := timeNow().Add(24 * time.Hour).Truncate(time.Second)
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &productID,
			LicenseeID: &licenseeID,
			Source:     "manual_import",
			ExpiresAt:  &expiresAt,
		})
		if err != nil {
			t.Fatalf("CreateLicenseRegistration auto code: %v", err)
		}
		if reg.Code == "" || !strings.HasPrefix(reg.Code, "RG-") {
			t.Fatalf("expected generated RG- code, got %+v", reg)
		}
		if reg.Source != "manual_import" || reg.ProductID == nil || *reg.ProductID != productID || reg.LicenseeID == nil || *reg.LicenseeID != licenseeID {
			t.Fatalf("expected metadata to be preserved, got %+v", reg)
		}
		if reg.ExpiresAt == nil || !reg.ExpiresAt.Equal(expiresAt) {
			t.Fatalf("expected expiresAt to be preserved, got %+v", reg.ExpiresAt)
		}
	})

	t.Run("create trims blank code and auto generates registration", func(t *testing.T) {
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			Code: "   ",
		})
		if err != nil {
			t.Fatalf("CreateLicenseRegistration blank code: %v", err)
		}
		if reg.Code == "" || !strings.HasPrefix(reg.Code, "RG-") {
			t.Fatalf("expected generated RG- code for blank input, got %+v", reg)
		}
		if strings.TrimSpace(reg.Code) != reg.Code {
			t.Fatalf("expected persisted registration code to be trimmed, got %q", reg.Code)
		}
	})

	t.Run("generate keeps product and licensee references", func(t *testing.T) {
		productID := product.ID
		licenseeID := licensee.ID
		reg, err := licenseSvc.GenerateLicenseRegistration(&productID, &licenseeID)
		if err != nil {
			t.Fatalf("GenerateLicenseRegistration refs: %v", err)
		}
		if reg.ProductID == nil || *reg.ProductID != productID || reg.LicenseeID == nil || *reg.LicenseeID != licenseeID {
			t.Fatalf("expected generated registration refs to persist, got %+v", reg)
		}
	})

	t.Run("create rejects missing product or licensee references", func(t *testing.T) {
		missingProduct := uint(999999)
		if _, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID: &missingProduct,
			Code:      "RG-MISSING-PRODUCT",
		}); !errors.Is(err, productpkg.ErrProductNotFound) {
			t.Fatalf("CreateLicenseRegistration missing product error = %v, want %v", err, productpkg.ErrProductNotFound)
		}

		missingLicensee := uint(888888)
		if _, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			LicenseeID: &missingLicensee,
			Code:       "RG-MISSING-LICENSEE",
		}); !errors.Is(err, licenseepkg.ErrLicenseeNotFound) {
			t.Fatalf("CreateLicenseRegistration missing licensee error = %v, want %v", err, licenseepkg.ErrLicenseeNotFound)
		}
	})

	t.Run("generate rejects missing product or licensee references", func(t *testing.T) {
		missingProduct := uint(777777)
		if _, err := licenseSvc.GenerateLicenseRegistration(&missingProduct, nil); !errors.Is(err, productpkg.ErrProductNotFound) {
			t.Fatalf("GenerateLicenseRegistration missing product error = %v, want %v", err, productpkg.ErrProductNotFound)
		}

		missingLicensee := uint(666666)
		if _, err := licenseSvc.GenerateLicenseRegistration(nil, &missingLicensee); !errors.Is(err, licenseepkg.ErrLicenseeNotFound) {
			t.Fatalf("GenerateLicenseRegistration missing licensee error = %v, want %v", err, licenseepkg.ErrLicenseeNotFound)
		}
	})
}

func TestLicenseService_LifecycleEdgeCases(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("Metis", "metis-license-edges", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Edge Corp"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}
	reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-EDGE-001",
	})
	if err != nil {
		t.Fatalf("create registration: %v", err)
	}
	license, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Edge",
		RegistrationCode: reg.Code,
		ValidFrom:        timeNow().Add(-time.Hour),
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue license: %v", err)
	}

	t.Run("renew requires current product key", func(t *testing.T) {
		if err := db.Where("product_id = ? AND is_current = ?", product.ID, true).Delete(&domain.ProductKey{}).Error; err != nil {
			t.Fatalf("delete current key: %v", err)
		}
		err := licenseSvc.RenewLicense(license.ID, ptrTime(timeNow().Add(48*time.Hour)), 2)
		if !errors.Is(err, ErrProductKeyNotFound) {
			t.Fatalf("expected ErrProductKeyNotFound, got %v", err)
		}
	})

	t.Run("renew rejects license missing product association", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, _ := localProductSvc.CreateProduct("Metis", "metis-license-no-product", "")
		_ = localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished)
		localLicensee, _ := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "NoProduct Corp"})
		localReg, _ := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-EDGE-NOPROD",
		})
		localLicense, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Edge",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue local license: %v", err)
		}
		if err := localDB.Model(&domain.License{}).Where("id = ?", localLicense.ID).Update("product_id", nil).Error; err != nil {
			t.Fatalf("clear product_id: %v", err)
		}

		err = localLicenseSvc.RenewLicense(localLicense.ID, ptrTime(timeNow().Add(24*time.Hour)), 3)
		if err == nil || err.Error() != "license has no associated product" {
			t.Fatalf("expected missing product association error, got %v", err)
		}
	})

	t.Run("export requires signing key version and product association", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, _ := localProductSvc.CreateProduct("Metis", "metis-license-export-edge", "")
		_ = localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished)
		localLicensee, _ := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Export Corp"})
		localReg, _ := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-EDGE-EXPORT",
		})
		localLicense, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Edge",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue local export license: %v", err)
		}

		if err := localDB.Where("product_id = ? AND version = ?", localProduct.ID, localLicense.KeyVersion).Delete(&domain.ProductKey{}).Error; err != nil {
			t.Fatalf("delete signing key version: %v", err)
		}
		_, _, err = localLicenseSvc.ExportLicFile(localLicense.ID, "v1")
		if !errors.Is(err, ErrProductKeyNotFound) {
			t.Fatalf("expected ErrProductKeyNotFound, got %v", err)
		}

		refreshedDB := testutil.SetupTestDB(t)
		refreshedProductSvc := newProductService(refreshedDB)
		refreshedLicenseeSvc := newLicenseeService(refreshedDB)
		refreshedLicenseSvc := newLicenseService(refreshedDB)
		refreshedProduct, _ := refreshedProductSvc.CreateProduct("Metis", "metis-license-export-noproduct", "")
		_ = refreshedProductSvc.UpdateStatus(refreshedProduct.ID, domain.StatusPublished)
		refreshedLicensee, _ := refreshedLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Export NoProduct"})
		refreshedReg, _ := refreshedLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &refreshedProduct.ID,
			LicenseeID: &refreshedLicensee.ID,
			Code:       "RG-EDGE-EXPORT-NOPROD",
		})
		refreshedLicense, err := refreshedLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        refreshedProduct.ID,
			LicenseeID:       refreshedLicensee.ID,
			PlanName:         "Edge",
			RegistrationCode: refreshedReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue refreshed export license: %v", err)
		}
		if err := refreshedDB.Model(&domain.License{}).Where("id = ?", refreshedLicense.ID).Update("product_id", nil).Error; err != nil {
			t.Fatalf("clear export product_id: %v", err)
		}

		_, _, err = refreshedLicenseSvc.ExportLicFile(refreshedLicense.ID, "v1")
		if err == nil || err.Error() != "license has no associated product" {
			t.Fatalf("expected missing product association export error, got %v", err)
		}
	})
}

func TestLicenseService_KeyRotationImpactAndAutoBulkReissue(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productSvc := newProductService(db)
	licenseeSvc := newLicenseeService(db)
	licenseSvc := newLicenseService(db)

	product, err := productSvc.CreateProduct("Metis", "prod-rotate", "")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := productSvc.UpdateStatus(product.ID, domain.StatusPublished); err != nil {
		t.Fatalf("publish product: %v", err)
	}
	licensee, err := licenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Rotate Corp"})
	if err != nil {
		t.Fatalf("create licensee: %v", err)
	}

	issue := func(code string) *domain.License {
		t.Helper()
		reg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &product.ID,
			LicenseeID: &licensee.ID,
			Code:       code,
		})
		if err != nil {
			t.Fatalf("create registration %s: %v", code, err)
		}
		license, err := licenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        product.ID,
			LicenseeID:       licensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: reg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue license %s: %v", code, err)
		}
		return license
	}

	oldActive := issue("RG-ROTATE-ACTIVE-001")
	pendingReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-ROTATE-PENDING-001",
	})
	if err != nil {
		t.Fatalf("create pending registration: %v", err)
	}
	pendingFrom := timeNow().Add(36 * time.Hour)
	pendingUntil := timeNow().Add(72 * time.Hour)
	oldPending, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Enterprise",
		RegistrationCode: pendingReg.Code,
		ValidFrom:        pendingFrom,
		ValidUntil:       &pendingUntil,
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue pending license: %v", err)
	}
	if oldPending.LifecycleStatus != domain.LicenseLifecyclePending {
		t.Fatalf("pending lifecycle = %q, want %q", oldPending.LifecycleStatus, domain.LicenseLifecyclePending)
	}
	oldSuspended := issue("RG-ROTATE-SUSPENDED-001")
	if err := licenseSvc.SuspendLicense(oldSuspended.ID, 7); err != nil {
		t.Fatalf("suspend old license: %v", err)
	}
	expiredUntil := timeNow().Add(-30 * time.Minute)
	expiredReg, err := licenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
		ProductID:  &product.ID,
		LicenseeID: &licensee.ID,
		Code:       "RG-ROTATE-EXPIRED-001",
	})
	if err != nil {
		t.Fatalf("create expired registration: %v", err)
	}
	oldExpired, err := licenseSvc.IssueLicense(IssueLicenseParams{
		ProductID:        product.ID,
		LicenseeID:       licensee.ID,
		PlanName:         "Enterprise",
		RegistrationCode: expiredReg.Code,
		ValidFrom:        timeNow().Add(-48 * time.Hour),
		ValidUntil:       &expiredUntil,
		IssuedBy:         1,
	})
	if err != nil {
		t.Fatalf("issue expired license: %v", err)
	}
	if oldExpired.LifecycleStatus != domain.LicenseLifecycleExpired {
		t.Fatalf("expired lifecycle = %q, want %q", oldExpired.LifecycleStatus, domain.LicenseLifecycleExpired)
	}
	oldRevoked := issue("RG-ROTATE-REVOKED-001")
	if err := licenseSvc.RevokeLicense(oldRevoked.ID, 9); err != nil {
		t.Fatalf("revoke old license: %v", err)
	}

	if _, err := productSvc.RotateKey(product.ID); err != nil {
		t.Fatalf("rotate key: %v", err)
	}
	current := issue("RG-ROTATE-CURRENT-001")

	impact, err := licenseSvc.AssessKeyRotationImpact(product.ID)
	if err != nil {
		t.Fatalf("AssessKeyRotationImpact: %v", err)
	}
	if impact.CurrentVersion != current.KeyVersion {
		t.Fatalf("impact current version = %d, want %d", impact.CurrentVersion, current.KeyVersion)
	}
	if impact.AffectedCount != 4 {
		t.Fatalf("impact affected count = %d, want 4", impact.AffectedCount)
	}

	reissued, err := licenseSvc.BulkReissueLicenses(product.ID, nil, 1)
	if err != nil {
		t.Fatalf("BulkReissueLicenses auto-select: %v", err)
	}
	if reissued != 4 {
		t.Fatalf("reissued = %d, want 4", reissued)
	}

	var refreshedActive domain.License
	if err := db.First(&refreshedActive, oldActive.ID).Error; err != nil {
		t.Fatalf("reload active old license: %v", err)
	}
	if refreshedActive.KeyVersion != current.KeyVersion {
		t.Fatalf("active old key version = %d, want %d", refreshedActive.KeyVersion, current.KeyVersion)
	}

	var refreshedPending domain.License
	if err := db.First(&refreshedPending, oldPending.ID).Error; err != nil {
		t.Fatalf("reload pending old license: %v", err)
	}
	if refreshedPending.KeyVersion != current.KeyVersion {
		t.Fatalf("pending old key version = %d, want %d", refreshedPending.KeyVersion, current.KeyVersion)
	}
	if refreshedPending.LifecycleStatus != domain.LicenseLifecyclePending {
		t.Fatalf("pending old lifecycle = %q, want %q", refreshedPending.LifecycleStatus, domain.LicenseLifecyclePending)
	}
	if !refreshedPending.ValidFrom.Equal(pendingFrom) {
		t.Fatalf("pending valid_from changed unexpectedly: got %v want %v", refreshedPending.ValidFrom, pendingFrom)
	}
	if refreshedPending.ValidUntil == nil || !refreshedPending.ValidUntil.Equal(pendingUntil) {
		t.Fatalf("pending valid_until changed unexpectedly: got %v want %v", refreshedPending.ValidUntil, pendingUntil)
	}

	var refreshedSuspended domain.License
	if err := db.First(&refreshedSuspended, oldSuspended.ID).Error; err != nil {
		t.Fatalf("reload suspended old license: %v", err)
	}
	if refreshedSuspended.KeyVersion != current.KeyVersion {
		t.Fatalf("suspended old key version = %d, want %d", refreshedSuspended.KeyVersion, current.KeyVersion)
	}
	if refreshedSuspended.LifecycleStatus != domain.LicenseLifecycleSuspended {
		t.Fatalf("suspended old lifecycle = %q, want %q", refreshedSuspended.LifecycleStatus, domain.LicenseLifecycleSuspended)
	}

	var refreshedExpired domain.License
	if err := db.First(&refreshedExpired, oldExpired.ID).Error; err != nil {
		t.Fatalf("reload expired old license: %v", err)
	}
	if refreshedExpired.KeyVersion != current.KeyVersion {
		t.Fatalf("expired old key version = %d, want %d", refreshedExpired.KeyVersion, current.KeyVersion)
	}
	if refreshedExpired.LifecycleStatus != domain.LicenseLifecycleExpired {
		t.Fatalf("expired old lifecycle = %q, want %q", refreshedExpired.LifecycleStatus, domain.LicenseLifecycleExpired)
	}

	var refreshedRevoked domain.License
	if err := db.First(&refreshedRevoked, oldRevoked.ID).Error; err != nil {
		t.Fatalf("reload revoked old license: %v", err)
	}
	if refreshedRevoked.KeyVersion == current.KeyVersion {
		t.Fatalf("revoked license should not be reissued: %+v", refreshedRevoked)
	}

	missingProductID := product.ID + 999
	if _, err := licenseSvc.AssessKeyRotationImpact(missingProductID); !errors.Is(err, ErrProductKeyNotFound) {
		t.Fatalf("AssessKeyRotationImpact missing product error = %v, want %v", err, ErrProductKeyNotFound)
	}
	if _, err := licenseSvc.BulkReissueLicenses(missingProductID, nil, 1); !errors.Is(err, ErrProductKeyNotFound) {
		t.Fatalf("BulkReissueLicenses missing product error = %v, want %v", err, ErrProductKeyNotFound)
	}

	t.Run("lifecycle revoked drift is excluded from impact and auto bulk reissue", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("domain.Product", "prod-rotate-drift-revoked", "")
		if err != nil {
			t.Fatalf("create drift product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish drift product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Drift Revoked Corp"})
		if err != nil {
			t.Fatalf("create drift licensee: %v", err)
		}
		localReg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-ROTATE-DRIFT-REV-001",
		})
		if err != nil {
			t.Fatalf("create drift registration: %v", err)
		}
		driftLicense, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			ValidUntil:       ptrTime(timeNow().Add(24 * time.Hour)),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue drift license: %v", err)
		}
		if _, err := localProductSvc.RotateKey(localProduct.ID); err != nil {
			t.Fatalf("rotate key: %v", err)
		}
		if err := localDB.Model(&domain.License{}).Where("id = ?", driftLicense.ID).Update("lifecycle_status", domain.LicenseLifecycleRevoked).Error; err != nil {
			t.Fatalf("simulate lifecycle revoked drift: %v", err)
		}

		impact, err := localLicenseSvc.AssessKeyRotationImpact(localProduct.ID)
		if err != nil {
			t.Fatalf("AssessKeyRotationImpact drift revoked: %v", err)
		}
		if impact.AffectedCount != 0 {
			t.Fatalf("expected lifecycle-revoked drift license to be excluded from impact count, got %d", impact.AffectedCount)
		}

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, nil, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses drift revoked: %v", err)
		}
		if reissued != 0 {
			t.Fatalf("expected lifecycle-revoked drift license to be excluded from auto bulk reissue, got %d", reissued)
		}
	})

	t.Run("status revoked drift is excluded from impact and auto bulk reissue", func(t *testing.T) {
		localDB := testutil.SetupTestDB(t)
		localProductSvc := newProductService(localDB)
		localLicenseeSvc := newLicenseeService(localDB)
		localLicenseSvc := newLicenseService(localDB)

		localProduct, err := localProductSvc.CreateProduct("domain.Product", "prod-rotate-drift-status-revoked", "")
		if err != nil {
			t.Fatalf("create drift product: %v", err)
		}
		if err := localProductSvc.UpdateStatus(localProduct.ID, domain.StatusPublished); err != nil {
			t.Fatalf("publish drift product: %v", err)
		}
		localLicensee, err := localLicenseeSvc.CreateLicensee(licenseepkg.CreateLicenseeParams{Name: "Status Revoked Drift Corp"})
		if err != nil {
			t.Fatalf("create drift licensee: %v", err)
		}
		localReg, err := localLicenseSvc.CreateLicenseRegistration(CreateLicenseRegistrationParams{
			ProductID:  &localProduct.ID,
			LicenseeID: &localLicensee.ID,
			Code:       "RG-ROTATE-DRIFT-STATUS-001",
		})
		if err != nil {
			t.Fatalf("create drift registration: %v", err)
		}
		driftLicense, err := localLicenseSvc.IssueLicense(IssueLicenseParams{
			ProductID:        localProduct.ID,
			LicenseeID:       localLicensee.ID,
			PlanName:         "Enterprise",
			RegistrationCode: localReg.Code,
			ValidFrom:        timeNow().Add(-time.Hour),
			ValidUntil:       ptrTime(timeNow().Add(24 * time.Hour)),
			IssuedBy:         1,
		})
		if err != nil {
			t.Fatalf("issue drift license: %v", err)
		}
		if _, err := localProductSvc.RotateKey(localProduct.ID); err != nil {
			t.Fatalf("rotate key: %v", err)
		}
		if err := localDB.Model(&domain.License{}).Where("id = ?", driftLicense.ID).Update("status", domain.LicenseStatusRevoked).Error; err != nil {
			t.Fatalf("simulate status revoked drift: %v", err)
		}

		impact, err := localLicenseSvc.AssessKeyRotationImpact(localProduct.ID)
		if err != nil {
			t.Fatalf("AssessKeyRotationImpact status drift revoked: %v", err)
		}
		if impact.AffectedCount != 0 {
			t.Fatalf("expected status-revoked drift license to be excluded from impact count, got %d", impact.AffectedCount)
		}

		reissued, err := localLicenseSvc.BulkReissueLicenses(localProduct.ID, nil, 1)
		if err != nil {
			t.Fatalf("BulkReissueLicenses status drift revoked: %v", err)
		}
		if reissued != 0 {
			t.Fatalf("expected status-revoked drift license to be excluded from auto bulk reissue, got %d", reissued)
		}
	})
}
