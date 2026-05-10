package catalog

import (
	"errors"
	. "metis/internal/app/itsm/domain"
	"testing"
)

func TestCatalogServiceUpdate_RejectsMissingParent(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	missingParentID := uint(999)
	_, err = svc.Update(root.ID, map[string]any{"parent_id": missingParentID})
	if !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("expected ErrCatalogNotFound, got %v", err)
	}
}

func TestCatalogServiceUpdate_RejectsThirdLevelParent(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, _ := svc.Create("Root", "root", "", "", nil, 10)
	if root == nil {
		t.Fatal("expected root catalog")
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 10)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	anotherRoot, err := svc.Create("Another Root", "another-root", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create another root: %v", err)
	}

	_, err = svc.Update(anotherRoot.ID, map[string]any{"parent_id": child.ID})
	if !errors.Is(err, ErrCatalogTooDeep) {
		t.Fatalf("expected ErrCatalogTooDeep, got %v", err)
	}
}

func TestCatalogServiceUpdate_RejectsSelfParent(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	_, err = svc.Update(root.ID, map[string]any{"parent_id": root.ID})
	if err == nil || err.Error() != "invalid catalog parent" {
		t.Fatalf("expected invalid catalog parent error, got %v", err)
	}
}

func TestCatalogServiceUpdate_RejectsDescendantParent(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 10)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	_, err = svc.Update(root.ID, map[string]any{"parent_id": child.ID})
	if err == nil || err.Error() != "invalid catalog parent" {
		t.Fatalf("expected invalid catalog parent error, got %v", err)
	}
}

func TestCatalogServiceCreate_RejectsDuplicateCode(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	if _, err := svc.Create("Root", "root", "", "", nil, 10); err != nil {
		t.Fatalf("create root: %v", err)
	}

	_, err := svc.Create("Other", "root", "", "", nil, 20)
	if err == nil || err.Error() != "catalog code already exists" {
		t.Fatalf("expected catalog code already exists, got %v", err)
	}
}

func TestCatalogServiceUpdate_RejectsDuplicateCode(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	first, err := svc.Create("First", "first", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := svc.Create("Second", "second", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	_, err = svc.Update(second.ID, map[string]any{"code": first.Code})
	if err == nil || err.Error() != "catalog code already exists" {
		t.Fatalf("expected catalog code already exists, got %v", err)
	}
}

func TestCatalogServiceTree_PreservesSortOrder(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	late, err := svc.Create("Late", "late", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create late: %v", err)
	}
	early, err := svc.Create("Early", "early", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create early: %v", err)
	}
	if _, err := svc.Create("Child B", "child-b", "", "", &early.ID, 20); err != nil {
		t.Fatalf("create child b: %v", err)
	}
	if _, err := svc.Create("Child A", "child-a", "", "", &early.ID, 10); err != nil {
		t.Fatalf("create child a: %v", err)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
	if tree[0].ID != early.ID || tree[1].ID != late.ID {
		t.Fatalf("unexpected root order: %+v", tree)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Code != "child-a" || tree[0].Children[1].Code != "child-b" {
		t.Fatalf("unexpected child order: %+v", tree[0].Children)
	}
}

func TestCatalogServiceServiceCounts_AggregatesDirectAndRootCounts(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, err := svc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := svc.Create("Child", "child", "", "", &root.ID, 10)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	emptyRoot, err := svc.Create("Empty", "empty", "", "", nil, 20)
	if err != nil {
		t.Fatalf("create empty root: %v", err)
	}
	services := []ServiceDefinition{
		{Name: "Root Direct", Code: "root-direct", CatalogID: root.ID, EngineType: "classic", IsActive: true},
		{Name: "Child Service", Code: "child-service", CatalogID: child.ID, EngineType: "classic", IsActive: true},
	}
	for i := range services {
		if err := db.Create(&services[i]).Error; err != nil {
			t.Fatalf("create service %d: %v", i, err)
		}
	}

	counts, err := svc.ServiceCounts()
	if err != nil {
		t.Fatalf("service counts: %v", err)
	}
	if counts.Total != 2 {
		t.Fatalf("expected total 2, got %d", counts.Total)
	}
	if counts.ByCatalogID[root.ID] != 1 || counts.ByCatalogID[child.ID] != 1 || counts.ByCatalogID[emptyRoot.ID] != 0 {
		t.Fatalf("unexpected direct counts: %+v", counts.ByCatalogID)
	}
	if counts.ByRootCatalogID[root.ID] != 2 || counts.ByRootCatalogID[emptyRoot.ID] != 0 {
		t.Fatalf("unexpected root counts: %+v", counts.ByRootCatalogID)
	}
}

func TestCatalogServiceTreeAndCounts_EmptyStateContracts(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	tree, err := svc.Tree()
	if err != nil {
		t.Fatalf("Tree on empty state: %v", err)
	}
	if len(tree) != 0 {
		t.Fatalf("expected empty tree, got %+v", tree)
	}

	counts, err := svc.ServiceCounts()
	if err != nil {
		t.Fatalf("ServiceCounts on empty state: %v", err)
	}
	if counts.Total != 0 || len(counts.ByCatalogID) != 0 || len(counts.ByRootCatalogID) != 0 {
		t.Fatalf("unexpected empty counts: %+v", counts)
	}
}

func TestCatalogServiceCreateGetAndUpdateContracts(t *testing.T) {
	db := newTestDB(t)
	svc := newCatalogServiceForTest(t, db)

	root, err := svc.Create("Root", "root", "root desc", "folder", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	t.Run("create rejects missing parent and third level", func(t *testing.T) {
		missingParentID := uint(999)
		if _, err := svc.Create("Ghost", "ghost", "", "", &missingParentID, 0); !errors.Is(err, ErrCatalogNotFound) {
			t.Fatalf("expected ErrCatalogNotFound, got %v", err)
		}

		child, err := svc.Create("Child", "child", "", "", &root.ID, 20)
		if err != nil {
			t.Fatalf("create child: %v", err)
		}
		if _, err := svc.Create("Grand", "grand", "", "", &child.ID, 30); !errors.Is(err, ErrCatalogTooDeep) {
			t.Fatalf("expected ErrCatalogTooDeep, got %v", err)
		}
	})

	t.Run("get returns stored catalog and missing id", func(t *testing.T) {
		got, err := svc.Get(root.ID)
		if err != nil {
			t.Fatalf("Get root: %v", err)
		}
		if got.Name != "Root" || got.Description != "root desc" || got.Icon != "folder" {
			t.Fatalf("unexpected catalog payload: %+v", got)
		}

		if _, err := svc.Get(999); !errors.Is(err, ErrCatalogNotFound) {
			t.Fatalf("expected ErrCatalogNotFound, got %v", err)
		}
	})

	t.Run("update persists scalar fields and active flag", func(t *testing.T) {
		updated, err := svc.Update(root.ID, map[string]any{
			"name":        "Root Updated",
			"description": "updated desc",
			"icon":        "grid",
			"sort_order":  99,
			"is_active":   false,
		})
		if err != nil {
			t.Fatalf("Update root: %v", err)
		}
		if updated.Name != "Root Updated" || updated.Description != "updated desc" || updated.Icon != "grid" || updated.SortOrder != 99 || updated.IsActive {
			t.Fatalf("unexpected updated catalog: %+v", updated)
		}

		if _, err := svc.Update(999, map[string]any{"name": "ghost"}); !errors.Is(err, ErrCatalogNotFound) {
			t.Fatalf("expected ErrCatalogNotFound on update missing catalog, got %v", err)
		}
	})
}
