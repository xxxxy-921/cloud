package product

import (
	"testing"

	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
	"gorm.io/gorm"
)

func TestProductReposPersistAndQueryCurrentState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productRepo := &ProductRepo{DB: db}
	planRepo := &PlanRepo{DB: db}
	keyRepo := &ProductKeyRepo{DB: db}

	product := &domain.Product{Name: "Metis", Code: "metis-repo", Status: domain.StatusPublished}
	if err := productRepo.Create(product); err != nil {
		t.Fatalf("Create product: %v", err)
	}
	foundByCode, err := productRepo.FindByCode(product.Code)
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	if foundByCode.ID != product.ID {
		t.Fatalf("FindByCode id=%d, want %d", foundByCode.ID, product.ID)
	}

	planA := &domain.Plan{ProductID: product.ID, Name: "Basic", SortOrder: 20}
	planB := &domain.Plan{ProductID: product.ID, Name: "Pro", SortOrder: 10, IsDefault: true}
	if err := planRepo.Create(planA); err != nil {
		t.Fatalf("Create planA: %v", err)
	}
	if err := planRepo.Create(planB); err != nil {
		t.Fatalf("Create planB: %v", err)
	}
	plans, err := planRepo.ListByProductID(product.ID)
	if err != nil {
		t.Fatalf("ListByProductID: %v", err)
	}
	if len(plans) != 2 || plans[0].Name != "Pro" || plans[1].Name != "Basic" {
		t.Fatalf("unexpected plans ordering: %+v", plans)
	}

	items, total, err := productRepo.List(ProductListParams{Keyword: "Metis", Status: domain.StatusPublished, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List products: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].PlanCount != 2 {
		t.Fatalf("unexpected product list result: total=%d items=%+v", total, items)
	}

	key := &domain.ProductKey{ProductID: product.ID, Version: 1, PublicKey: "pub", EncryptedPrivateKey: "enc", IsCurrent: true}
	if err := keyRepo.Create(key); err != nil {
		t.Fatalf("Create key: %v", err)
	}
	current, err := keyRepo.FindCurrentByProductID(product.ID)
	if err != nil {
		t.Fatalf("FindCurrentByProductID: %v", err)
	}
	if current.Version != 1 {
		t.Fatalf("current key version=%d, want 1", current.Version)
	}
}

func TestProductReposLookupAndExistenceHelpers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	productRepo := &ProductRepo{DB: db}
	planRepo := &PlanRepo{DB: db}
	keyRepo := &ProductKeyRepo{DB: db}

	product := &domain.Product{Name: "Lookup Host", Code: "lookup-host", Status: domain.StatusPublished}
	if err := productRepo.Create(product); err != nil {
		t.Fatalf("Create product: %v", err)
	}

	planA := &domain.Plan{ProductID: product.ID, Name: "Standard", SortOrder: 20}
	planB := &domain.Plan{ProductID: product.ID, Name: "Enterprise", SortOrder: 10}
	if err := planRepo.Create(planA); err != nil {
		t.Fatalf("Create planA: %v", err)
	}
	if err := planRepo.Create(planB); err != nil {
		t.Fatalf("Create planB: %v", err)
	}

	foundWithPlans, err := productRepo.FindByIDWithPlans(product.ID)
	if err != nil {
		t.Fatalf("FindByIDWithPlans: %v", err)
	}
	if len(foundWithPlans.Plans) != 2 || foundWithPlans.Plans[0].Name != "Enterprise" || foundWithPlans.Plans[1].Name != "Standard" {
		t.Fatalf("unexpected preloaded plans ordering: %+v", foundWithPlans.Plans)
	}

	exists, err := productRepo.ExistsByCode(product.Code)
	if err != nil {
		t.Fatalf("ExistsByCode existing: %v", err)
	}
	if !exists {
		t.Fatal("expected existing product code to exist")
	}
	exists, err = productRepo.ExistsByCode("missing-code")
	if err != nil {
		t.Fatalf("ExistsByCode missing: %v", err)
	}
	if exists {
		t.Fatal("expected missing product code to not exist")
	}

	exists, err = planRepo.ExistsByName(product.ID, "Standard")
	if err != nil {
		t.Fatalf("ExistsByName existing: %v", err)
	}
	if !exists {
		t.Fatal("expected existing plan name to exist")
	}
	exists, err = planRepo.ExistsByName(product.ID, "Standard", planA.ID)
	if err != nil {
		t.Fatalf("ExistsByName exclude self: %v", err)
	}
	if exists {
		t.Fatal("expected self-excluded plan name check to be false")
	}

	key := &domain.ProductKey{ProductID: product.ID, Version: 2, PublicKey: "pub-2", EncryptedPrivateKey: "enc-2", IsCurrent: true}
	if err := keyRepo.Create(key); err != nil {
		t.Fatalf("Create key: %v", err)
	}
	foundKey, err := keyRepo.FindByProductIDAndVersion(product.ID, 2)
	if err != nil {
		t.Fatalf("FindByProductIDAndVersion: %v", err)
	}
	if foundKey.PublicKey != "pub-2" {
		t.Fatalf("unexpected key lookup result: %+v", foundKey)
	}
	if _, err := keyRepo.FindByProductIDAndVersion(product.ID, 99); err == nil {
		t.Fatal("expected missing product key lookup to fail")
	} else if err != gorm.ErrRecordNotFound {
		t.Fatalf("unexpected missing key lookup error: %v", err)
	}
}
