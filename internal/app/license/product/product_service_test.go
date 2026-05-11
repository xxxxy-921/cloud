package product

import (
	"encoding/json"
	"errors"
	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
	"testing"
)

func TestProductService_CreateProduct(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	product, err := svc.CreateProduct("Metis Enterprise", "metis-ent", "Enterprise edition")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product.Name != "Metis Enterprise" {
		t.Errorf("Name = %q, want %q", product.Name, "Metis Enterprise")
	}
	if product.Code != "metis-ent" {
		t.Errorf("Code = %q, want %q", product.Code, "metis-ent")
	}
	if product.Status != domain.StatusUnpublished {
		t.Errorf("Status = %q, want %q", product.Status, domain.StatusUnpublished)
	}

	// Assert that exactly one domain.ProductKey was created and marked current
	keys, err := svc.keyRepo.FindByProductIDAndVersion(product.ID, 1)
	if err != nil {
		t.Fatalf("failed to find product key: %v", err)
	}
	if !keys.IsCurrent {
		t.Error("expected initial key to be current")
	}
	if keys.PublicKey == "" {
		t.Error("expected public key to be non-empty")
	}
}

func TestProductService_UpdateStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	product, err := svc.CreateProduct("domain.Product", "prod-1", "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name      string
		from      string
		to        string
		wantErr   bool
		wantErrIs error
	}{
		{"unpublished -> published", domain.StatusUnpublished, domain.StatusPublished, false, nil},
		{"published -> unpublished", domain.StatusPublished, domain.StatusUnpublished, false, nil},
		{"published -> archived", domain.StatusPublished, domain.StatusArchived, false, nil},
		{"archived -> unpublished", domain.StatusArchived, domain.StatusUnpublished, false, nil},
		{"unpublished -> archived", domain.StatusUnpublished, domain.StatusArchived, false, nil},
		{"archived -> published", domain.StatusArchived, domain.StatusPublished, true, ErrInvalidStatusTransition},
		{"published -> published", domain.StatusPublished, domain.StatusPublished, true, ErrInvalidStatusTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset product status for each case
			db.Model(&domain.Product{}).Where("id = ?", product.ID).Update("status", tt.from)

			err := svc.UpdateStatus(product.ID, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var updated domain.Product
			db.First(&updated, product.ID)
			if updated.Status != tt.to {
				t.Errorf("status = %q, want %q", updated.Status, tt.to)
			}
		})
	}
}

func TestProductService_RotateKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	product, err := svc.CreateProduct("domain.Product", "prod-rotate", "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	oldKey, err := svc.keyRepo.FindCurrentByProductID(product.ID)
	if err != nil {
		t.Fatalf("failed to find initial key: %v", err)
	}

	newKey, err := svc.RotateKey(product.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newKey.Version != oldKey.Version+1 {
		t.Errorf("newKey.Version = %d, want %d", newKey.Version, oldKey.Version+1)
	}
	if !newKey.IsCurrent {
		t.Error("expected new key to be current")
	}

	// Old key should no longer be current
	refreshedOld, err := svc.keyRepo.FindByProductIDAndVersion(product.ID, oldKey.Version)
	if err != nil {
		t.Fatalf("failed to find old key: %v", err)
	}
	if refreshedOld.IsCurrent {
		t.Error("expected old key to be revoked (not current)")
	}
	if refreshedOld.RevokedAt == nil {
		t.Error("expected old key to have revoked_at set")
	}
	firstRevokedAt := *refreshedOld.RevokedAt

	secondKey, err := svc.RotateKey(product.ID)
	if err != nil {
		t.Fatalf("second rotate unexpected error: %v", err)
	}
	if secondKey.Version != newKey.Version+1 {
		t.Fatalf("secondKey.Version = %d, want %d", secondKey.Version, newKey.Version+1)
	}
	if !secondKey.IsCurrent {
		t.Fatal("expected second key to be current")
	}
	if secondKey.PublicKey == newKey.PublicKey {
		t.Fatal("expected rotated public key to change across versions")
	}

	currentKey, err := svc.GetPublicKey(product.ID)
	if err != nil {
		t.Fatalf("GetPublicKey after second rotate: %v", err)
	}
	if currentKey.ID != secondKey.ID || currentKey.Version != secondKey.Version || !currentKey.IsCurrent {
		t.Fatalf("unexpected current key after second rotate: %+v want %+v", currentKey, secondKey)
	}

	refreshedFirstRotated, err := svc.keyRepo.FindByProductIDAndVersion(product.ID, newKey.Version)
	if err != nil {
		t.Fatalf("failed to find first rotated key: %v", err)
	}
	if refreshedFirstRotated.IsCurrent {
		t.Fatal("expected first rotated key to be revoked after second rotation")
	}
	if refreshedFirstRotated.RevokedAt == nil {
		t.Fatal("expected first rotated key to have revoked_at set")
	}

	refreshedOriginal, err := svc.keyRepo.FindByProductIDAndVersion(product.ID, oldKey.Version)
	if err != nil {
		t.Fatalf("reload original key after second rotation: %v", err)
	}
	if refreshedOriginal.RevokedAt == nil || !refreshedOriginal.RevokedAt.Equal(firstRevokedAt) {
		t.Fatalf("expected original key revoked_at to remain stable, got %v want %v", refreshedOriginal.RevokedAt, firstRevokedAt)
	}

	var currentCount int64
	if err := db.Model(&domain.ProductKey{}).Where("product_id = ? AND is_current = ?", product.ID, true).Count(&currentCount).Error; err != nil {
		t.Fatalf("count current keys: %v", err)
	}
	if currentCount != 1 {
		t.Fatalf("current key count = %d, want 1", currentCount)
	}
}

func TestProductService_CreateProduct_DuplicateCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	_, err := svc.CreateProduct("domain.Product A", "same-code", "")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.CreateProduct("domain.Product B", "same-code", "")
	if err == nil {
		t.Fatal("expected error for duplicate code, got nil")
	}
	if err != ErrProductCodeExists {
		t.Errorf("expected ErrProductCodeExists, got %v", err)
	}
}

func TestProductService_TransactionalKeyBootstrapContracts(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	t.Run("create product rolls back product row when initial key insert fails", func(t *testing.T) {
		if err := db.Exec(`
			CREATE TRIGGER fail_initial_product_key_insert
			BEFORE INSERT ON license_product_keys
			BEGIN
				SELECT RAISE(ABORT, 'fail initial product key insert');
			END;
		`).Error; err != nil {
			t.Fatalf("create trigger: %v", err)
		}
		defer func() {
			if err := db.Exec(`DROP TRIGGER IF EXISTS fail_initial_product_key_insert`).Error; err != nil {
				t.Fatalf("drop trigger: %v", err)
			}
		}()

		_, err := svc.CreateProduct("Rollback Product", "rollback-product", "desc")
		if err == nil {
			t.Fatal("expected CreateProduct to fail when initial key insert aborts")
		}

		var productCount int64
		if err := db.Model(&domain.Product{}).Where("code = ?", "rollback-product").Count(&productCount).Error; err != nil {
			t.Fatalf("count rolled back product: %v", err)
		}
		if productCount != 0 {
			t.Fatalf("productCount = %d, want 0 after transaction rollback", productCount)
		}

		var keyCount int64
		if err := db.Model(&domain.ProductKey{}).Count(&keyCount).Error; err != nil {
			t.Fatalf("count product keys after rollback: %v", err)
		}
		if keyCount != 0 {
			t.Fatalf("keyCount = %d, want 0 after failed bootstrap", keyCount)
		}
	})

	t.Run("rotate key rolls back revoke when new key insert fails", func(t *testing.T) {
		product, err := svc.CreateProduct("Rotate Rollback Product", "rotate-rollback-product", "desc")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		before, err := svc.keyRepo.FindCurrentByProductID(product.ID)
		if err != nil {
			t.Fatalf("find current key before failed rotate: %v", err)
		}

		if err := db.Exec(`
			CREATE TRIGGER fail_rotated_product_key_insert
			BEFORE INSERT ON license_product_keys
			WHEN NEW.version > 1
			BEGIN
				SELECT RAISE(ABORT, 'fail rotated product key insert');
			END;
		`).Error; err != nil {
			t.Fatalf("create rotate trigger: %v", err)
		}
		defer func() {
			if err := db.Exec(`DROP TRIGGER IF EXISTS fail_rotated_product_key_insert`).Error; err != nil {
				t.Fatalf("drop rotate trigger: %v", err)
			}
		}()

		if _, err := svc.RotateKey(product.ID); err == nil {
			t.Fatal("expected RotateKey to fail when rotated key insert aborts")
		}

		after, err := svc.keyRepo.FindCurrentByProductID(product.ID)
		if err != nil {
			t.Fatalf("find current key after failed rotate: %v", err)
		}
		if after.ID != before.ID || after.Version != before.Version || !after.IsCurrent {
			t.Fatalf("current key changed after failed rotate: before=%+v after=%+v", before, after)
		}
		if after.RevokedAt != nil {
			t.Fatalf("expected original current key to remain unrecalled after failed rotate, got revoked_at=%v", after.RevokedAt)
		}

		var keyCount int64
		if err := db.Model(&domain.ProductKey{}).Where("product_id = ?", product.ID).Count(&keyCount).Error; err != nil {
			t.Fatalf("count product keys after failed rotate: %v", err)
		}
		if keyCount != 1 {
			t.Fatalf("keyCount = %d, want 1 after failed rotate", keyCount)
		}
	})
}

func TestProductService_ReadAndUpdateFlows(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	product, err := svc.CreateProduct("Metis", "metis", "desc")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := db.Model(&domain.Product{}).Where("id = ?", product.ID).Update("license_key", "").Error; err != nil {
		t.Fatalf("clear license key: %v", err)
	}

	got, err := svc.GetProduct(product.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if got.LicenseKey == "" {
		t.Fatal("expected GetProduct to backfill missing license key")
	}

	newName := "Metis Updated"
	newDesc := "new-desc"
	updated, err := svc.UpdateProduct(product.ID, UpdateProductParams{Name: &newName, Description: &newDesc})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if updated.Name != newName || updated.Description != newDesc {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	items, total, err := svc.ListProducts(ProductListParams{Keyword: "Updated", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != product.ID {
		t.Fatalf("unexpected list result: total=%d items=%+v", total, items)
	}

	if err := svc.UpdateConstraintSchema(product.ID, json.RawMessage(`[{"key":"vpn","label":"VPN","features":[{"key":"seat","label":"Seat","type":"number","min":1,"max":10}]}]`)); err != nil {
		t.Fatalf("UpdateConstraintSchema: %v", err)
	}
	key, err := svc.GetPublicKey(product.ID)
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	if key.PublicKey == "" {
		t.Fatal("expected public key to be non-empty")
	}
}

func TestProductService_GuardsForMissingProductsAndKeys(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := &ProductService{
		productRepo:      &ProductRepo{DB: db},
		planRepo:         &PlanRepo{DB: db},
		keyRepo:          &ProductKeyRepo{DB: db},
		db:               db,
		jwtSecret:        []byte("test-jwt-secret"),
		licenseKeySecret: []byte("test-license-secret"),
	}

	t.Run("missing product status update returns not found", func(t *testing.T) {
		err := svc.UpdateStatus(9999, domain.StatusPublished)
		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("UpdateStatus missing product error = %v, want %v", err, ErrProductNotFound)
		}
	})

	t.Run("get and update reject missing product", func(t *testing.T) {
		if _, err := svc.GetProduct(9999); !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("GetProduct missing product error = %v, want %v", err, ErrProductNotFound)
		}
		name := "Missing"
		if _, err := svc.UpdateProduct(9999, UpdateProductParams{Name: &name}); !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("UpdateProduct missing product error = %v, want %v", err, ErrProductNotFound)
		}
	})

	t.Run("update schema rejects missing product and invalid payload", func(t *testing.T) {
		if err := svc.UpdateConstraintSchema(9999, json.RawMessage(`[{"key":"vpn","label":"VPN","features":[]}]`)); !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("UpdateConstraintSchema missing product error = %v, want %v", err, ErrProductNotFound)
		}
		product, err := svc.CreateProduct("Schema Guard Product", "schema-guard-product", "")
		if err != nil {
			t.Fatalf("create schema guard product: %v", err)
		}
		if err := svc.UpdateConstraintSchema(product.ID, json.RawMessage(`{"bad":true}`)); !errors.Is(err, ErrInvalidConstraintSchema) {
			t.Fatalf("UpdateConstraintSchema invalid payload error = %v, want %v", err, ErrInvalidConstraintSchema)
		}
	})

	t.Run("rotate key rejects missing product", func(t *testing.T) {
		_, err := svc.RotateKey(9999)
		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("RotateKey missing product error = %v, want %v", err, ErrProductNotFound)
		}
	})

	t.Run("rotate key rejects product without current key", func(t *testing.T) {
		product, err := svc.CreateProduct("Orphan Key Product", "orphan-key", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		if err := db.Where("product_id = ?", product.ID).Delete(&domain.ProductKey{}).Error; err != nil {
			t.Fatalf("delete current key: %v", err)
		}

		_, err = svc.RotateKey(product.ID)
		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("RotateKey missing current key error = %v, want %v", err, ErrProductNotFound)
		}
	})

	t.Run("rotate key fails without encryption secrets and keeps current key intact", func(t *testing.T) {
		product, err := svc.CreateProduct("Secretless Rotate Product", "secretless-rotate", "")
		if err != nil {
			t.Fatalf("create product: %v", err)
		}
		before, err := svc.GetPublicKey(product.ID)
		if err != nil {
			t.Fatalf("GetPublicKey before failed rotate: %v", err)
		}

		broken := &ProductService{
			productRepo:      svc.productRepo,
			planRepo:         svc.planRepo,
			keyRepo:          svc.keyRepo,
			db:               db,
			jwtSecret:        nil,
			licenseKeySecret: nil,
		}
		if _, err := broken.RotateKey(product.ID); err == nil {
			t.Fatal("expected RotateKey to fail when encryption secrets are missing")
		}

		after, err := svc.GetPublicKey(product.ID)
		if err != nil {
			t.Fatalf("GetPublicKey after failed rotate: %v", err)
		}
		if after.ID != before.ID || after.Version != before.Version || !after.IsCurrent {
			t.Fatalf("expected current key to remain unchanged after failed rotate: before=%+v after=%+v", before, after)
		}

		var currentCount int64
		if err := db.Model(&domain.ProductKey{}).Where("product_id = ? AND is_current = ?", product.ID, true).Count(&currentCount).Error; err != nil {
			t.Fatalf("count current keys after failed rotate: %v", err)
		}
		if currentCount != 1 {
			t.Fatalf("current key count after failed rotate = %d, want 1", currentCount)
		}
	})

	t.Run("get public key rejects missing product and missing current key", func(t *testing.T) {
		if _, err := svc.GetPublicKey(9999); !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("GetPublicKey missing product error = %v, want %v", err, ErrProductNotFound)
		}

		product, err := svc.CreateProduct("Keyless Product", "keyless-product", "")
		if err != nil {
			t.Fatalf("create keyless product: %v", err)
		}
		if err := db.Where("product_id = ?", product.ID).Delete(&domain.ProductKey{}).Error; err != nil {
			t.Fatalf("delete product key: %v", err)
		}

		if _, err := svc.GetPublicKey(product.ID); !errors.Is(err, ErrProductNotFound) {
			t.Fatalf("GetPublicKey missing current key error = %v, want %v", err, ErrProductNotFound)
		}
	})

	t.Run("create and update reject blank business identifiers", func(t *testing.T) {
		if _, err := svc.CreateProduct("   ", "valid-code", "desc"); !errors.Is(err, ErrInvalidProductName) {
			t.Fatalf("CreateProduct blank name error = %v, want %v", err, ErrInvalidProductName)
		}
		if _, err := svc.CreateProduct("Valid Name", "   ", "desc"); !errors.Is(err, ErrInvalidProductCode) {
			t.Fatalf("CreateProduct blank code error = %v, want %v", err, ErrInvalidProductCode)
		}

		product, err := svc.CreateProduct("Trim Guard", "trim-guard", "desc")
		if err != nil {
			t.Fatalf("create trim guard product: %v", err)
		}

		blank := "   "
		if _, err := svc.UpdateProduct(product.ID, UpdateProductParams{Name: &blank}); !errors.Is(err, ErrInvalidProductName) {
			t.Fatalf("UpdateProduct blank name error = %v, want %v", err, ErrInvalidProductName)
		}
	})
}
