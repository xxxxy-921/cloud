package licensee

import (
	"testing"
	"time"

	"metis/internal/app/license/domain"
	"metis/internal/app/license/testutil"
)

func TestLicenseeRepoListDefaultsToActiveAndSupportsFilters(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := &LicenseeRepo{DB: db}

	items := []domain.Licensee{
		{Name: "Acme", Code: "LS-ACME", Status: domain.LicenseeStatusActive},
		{Name: "Beta", Code: "LS-BETA", Status: domain.LicenseeStatusArchived},
		{Name: "Gamma", Code: "LS-GAMMA", Status: domain.LicenseeStatusActive},
	}
	for i := range items {
		if err := repo.Create(&items[i]); err != nil {
			t.Fatalf("create licensee %s: %v", items[i].Name, err)
		}
	}
	if err := db.Model(&items[0]).Update("created_at", time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatalf("update created_at: %v", err)
	}

	activeItems, total, err := repo.List(LicenseeListParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list default active: %v", err)
	}
	if total != 2 || len(activeItems) != 2 {
		t.Fatalf("default active list = total %d items %+v", total, activeItems)
	}
	if activeItems[0].Code != "LS-ACME" {
		t.Fatalf("expected newest active first, got %+v", activeItems)
	}

	allItems, total, err := repo.List(LicenseeListParams{Status: "all", Keyword: "BETA", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list all filtered: %v", err)
	}
	if total != 1 || len(allItems) != 1 || allItems[0].Code != "LS-BETA" {
		t.Fatalf("unexpected all-items filter result: total=%d items=%+v", total, allItems)
	}
}
