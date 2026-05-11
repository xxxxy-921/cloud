package definition

import (
	"testing"

	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
)

func TestServiceActionRepoUpdateDeleteAndListScopeContracts(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	repo := &ServiceActionRepo{db: &database.DB{DB: db}}

	root, err := catSvc.Create("Root", "root-action-repo", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	serviceA, err := serviceDefs.Create(&ServiceDefinition{Name: "A", Code: "svc-action-repo-a", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service A: %v", err)
	}
	serviceB, err := serviceDefs.Create(&ServiceDefinition{Name: "B", Code: "svc-action-repo-b", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service B: %v", err)
	}

	actionA1 := &ServiceAction{Name: "First", Code: "first", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/1"}`), ServiceID: serviceA.ID}
	actionA2 := &ServiceAction{Name: "Second", Code: "second", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/2"}`), ServiceID: serviceA.ID}
	actionB1 := &ServiceAction{Name: "Third", Code: "third", ActionType: "http", ConfigJSON: JSONField(`{"url":"https://example.com/3"}`), ServiceID: serviceB.ID}
	for _, action := range []*ServiceAction{actionA1, actionA2, actionB1} {
		if err := repo.Create(action); err != nil {
			t.Fatalf("create action %s: %v", action.Code, err)
		}
	}

	if err := repo.Update(actionA1.ID, map[string]any{"name": "First Updated", "is_active": false}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, err := repo.FindByID(actionA1.ID)
	if err != nil {
		t.Fatalf("FindByID updated action: %v", err)
	}
	if reloaded.Name != "First Updated" || reloaded.IsActive {
		t.Fatalf("unexpected updated action: %+v", reloaded)
	}

	if err := repo.UpdateByService(serviceA.ID, actionA1.ID, map[string]any{"code": "first-service-scoped"}); err != nil {
		t.Fatalf("UpdateByService: %v", err)
	}
	reloaded, err = repo.FindByID(actionA1.ID)
	if err != nil {
		t.Fatalf("FindByID reloaded action: %v", err)
	}
	if reloaded.Code != "first-service-scoped" {
		t.Fatalf("expected service-scoped update to persist, got %+v", reloaded)
	}

	if err := repo.UpdateByService(serviceB.ID, actionA1.ID, map[string]any{"code": "cross-service-write"}); err != nil {
		t.Fatalf("cross-service UpdateByService should not error, got %v", err)
	}
	reloaded, err = repo.FindByID(actionA1.ID)
	if err != nil {
		t.Fatalf("FindByID after cross-service update: %v", err)
	}
	if reloaded.Code != "first-service-scoped" {
		t.Fatalf("cross-service update should not modify row, got %+v", reloaded)
	}

	items, err := repo.ListByService(serviceA.ID)
	if err != nil {
		t.Fatalf("ListByService: %v", err)
	}
	if len(items) != 2 || items[0].ID != actionA1.ID || items[1].ID != actionA2.ID {
		t.Fatalf("expected service-scoped id ordering, got %+v", items)
	}

	if err := repo.DeleteByService(serviceB.ID, actionA2.ID); err != nil {
		t.Fatalf("cross-service DeleteByService should not error, got %v", err)
	}
	if _, err := repo.FindByID(actionA2.ID); err != nil {
		t.Fatalf("cross-service delete should not remove action: %v", err)
	}

	if err := repo.DeleteByService(serviceA.ID, actionA2.ID); err != nil {
		t.Fatalf("DeleteByService: %v", err)
	}
	if _, err := repo.FindByID(actionA2.ID); err == nil {
		t.Fatal("expected scoped delete to remove action")
	}

	if err := repo.Delete(actionB1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(actionB1.ID); err == nil {
		t.Fatal("expected delete by id to remove action")
	}
}
