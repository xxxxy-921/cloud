package definition

import (
	"errors"
	"strings"
	"testing"

	. "metis/internal/app/itsm/domain"
)

func createServiceActionTestService(t *testing.T) (*ServiceActionService, *ServiceDefinition) {
	t.Helper()

	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	actionSvc := newServiceActionServiceForTest(t, db, serviceDefs)

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{
		Name:              "Webhook Service",
		Code:              "webhook-service",
		CatalogID:         root.ID,
		EngineType:        "smart",
		CollaborationSpec: "spec",
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return actionSvc, service
}

func TestServiceActionServiceCreateUpdateListAndDelete(t *testing.T) {
	svc, service := createServiceActionTestService(t)

	created, err := svc.Create(&ServiceAction{
		Name:       "SendWebhook",
		Code:       "send-webhook",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/hook","headers":{"X-Test":"1"}}`),
		ServiceID:  service.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !created.IsActive {
		t.Fatal("expected action to be activated on create")
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Code != "send-webhook" || got.ServiceID != service.ID {
		t.Fatalf("unexpected Get result: %+v", got)
	}

	updated, err := svc.Update(service.ID, created.ID, map[string]any{
		"name":        "SendWebhookV2",
		"config_json": JSONField(`{"url":"https://example.com/hook","method":"PUT","timeout":45}`),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "SendWebhookV2" {
		t.Fatalf("updated name = %q, want SendWebhookV2", updated.Name)
	}

	items, err := svc.ListByService(service.ID)
	if err != nil {
		t.Fatalf("ListByService: %v", err)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("unexpected list result: %+v", items)
	}

	if err := svc.Delete(service.ID, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(created.ID); !errors.Is(err, ErrServiceActionNotFound) {
		t.Fatalf("expected deleted action to be missing, got %v", err)
	}
}

func TestServiceActionServiceRejectsDuplicateCodeAndInvalidConfig(t *testing.T) {
	svc, service := createServiceActionTestService(t)

	if _, err := svc.Create(&ServiceAction{
		Name:       "SendWebhook",
		Code:       "send-webhook",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/hook"}`),
		ServiceID:  service.ID,
	}); err != nil {
		t.Fatalf("seed action: %v", err)
	}

	if _, err := svc.Create(&ServiceAction{
		Name:       "Duplicate",
		Code:       "send-webhook",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/other"}`),
		ServiceID:  service.ID,
	}); !errors.Is(err, ErrActionCodeExists) {
		t.Fatalf("expected duplicate action code error, got %v", err)
	}

	if _, err := svc.Create(&ServiceAction{
		Name:       "BadType",
		Code:       "bad-type",
		ActionType: "shell",
		ConfigJSON: JSONField(`{"url":"https://example.com"}`),
		ServiceID:  service.ID,
	}); !errors.Is(err, ErrInvalidActionConfig) {
		t.Fatalf("expected invalid action config error, got %v", err)
	}
}

func TestServiceActionServiceUpdateRejectsCrossServiceConflictAndInvalidConfig(t *testing.T) {
	svc, service := createServiceActionTestService(t)

	first, err := svc.Create(&ServiceAction{
		Name:       "First",
		Code:       "first",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/first"}`),
		ServiceID:  service.ID,
	})
	if err != nil {
		t.Fatalf("create first action: %v", err)
	}
	second, err := svc.Create(&ServiceAction{
		Name:       "Second",
		Code:       "second",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/second"}`),
		ServiceID:  service.ID,
	})
	if err != nil {
		t.Fatalf("create second action: %v", err)
	}

	if _, err := svc.Update(service.ID, second.ID, map[string]any{"code": first.Code}); !errors.Is(err, ErrActionCodeExists) {
		t.Fatalf("expected duplicate code on update, got %v", err)
	}

	if _, err := svc.Update(service.ID, second.ID, map[string]any{
		"action_type": "http",
		"config_json": JSONField(`{"url":"ftp://example.com"}`),
	}); !errors.Is(err, ErrInvalidActionConfig) {
		t.Fatalf("expected invalid action config on update, got %v", err)
	}

	if _, err := svc.GetByService(service.ID, first.ID); err != nil {
		t.Fatalf("GetByService: %v", err)
	}
	if _, err := svc.GetByService(service.ID+999, first.ID); !errors.Is(err, ErrServiceDefNotFound) {
		t.Fatalf("expected missing service error, got %v", err)
	}
}

func TestServiceActionServiceRejectsBlankBusinessIdentifiers(t *testing.T) {
	svc, service := createServiceActionTestService(t)

	tests := []struct {
		name   string
		action *ServiceAction
		want   error
	}{
		{
			name: "blank name",
			action: &ServiceAction{
				Name:       "   ",
				Code:       "notify-blank-name",
				ActionType: "http",
				ConfigJSON: JSONField(`{"url":"https://example.com/hook"}`),
				ServiceID:  service.ID,
			},
			want: ErrInvalidActionName,
		},
		{
			name: "blank code",
			action: &ServiceAction{
				Name:       "Notify",
				Code:       "   ",
				ActionType: "http",
				ConfigJSON: JSONField(`{"url":"https://example.com/hook"}`),
				ServiceID:  service.ID,
			},
			want: ErrInvalidActionCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(tt.action)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Create error = %v, want %v", err, tt.want)
			}
		})
	}

	valid, err := svc.Create(&ServiceAction{
		Name:       "  Notify Ops  ",
		Code:       "  notify-ops  ",
		ActionType: "http",
		ConfigJSON: JSONField(`{"url":"https://example.com/hook"}`),
		ServiceID:  service.ID,
	})
	if err != nil {
		t.Fatalf("create trimmed valid action: %v", err)
	}
	if valid.Name != strings.TrimSpace(valid.Name) || valid.Code != strings.TrimSpace(valid.Code) {
		t.Fatalf("expected action identifiers to be trimmed, got %+v", valid)
	}
}
