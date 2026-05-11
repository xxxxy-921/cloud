package ticket

import (
	"encoding/json"
	"strings"
	"testing"

	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/form"
	"metis/internal/database"
	"gorm.io/gorm"
)

func TestVariableServiceBulkSetRejectsUnsupportedValueTypesAtomically(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ProcessVariable{}); err != nil {
		t.Fatalf("migrate process variables: %v", err)
	}
	svc := &VariableService{repo: &VariableRepository{db: &database.DB{DB: db}}}

	err := db.Transaction(func(tx *gorm.DB) error {
		return svc.BulkSet(tx, []ProcessVariable{
			{
				TicketID:  1,
				ScopeID:   "root",
				Key:       "requester",
				Value:     "alice",
				ValueType: ValueTypeString,
				Source:    "form",
			},
			{
				TicketID:  1,
				ScopeID:   "root",
				Key:       "broken",
				Value:     "???",
				ValueType: "unsupported",
				Source:    "form",
			},
		})
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported value_type") {
		t.Fatalf("expected unsupported value_type error, got %v", err)
	}

	var count int64
	if err := db.Model(&ProcessVariable{}).Count(&count).Error; err != nil {
		t.Fatalf("count variables after failed bulk set: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected failed bulk set to leave no variables, got %d", count)
	}
}

func TestVariableServiceBulkSetAndDeleteByTicketContracts(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ProcessVariable{}); err != nil {
		t.Fatalf("migrate process variables: %v", err)
	}
	repo := &VariableRepository{db: &database.DB{DB: db}}
	svc := &VariableService{repo: repo}

	err := db.Transaction(func(tx *gorm.DB) error {
		return svc.BulkSet(tx, []ProcessVariable{
			{TicketID: 7, ScopeID: "root", Key: "summary", Value: "VPN access", ValueType: ValueTypeString, Source: "form"},
			{TicketID: 7, ScopeID: "root", Key: "approved", Value: "true", ValueType: ValueTypeBoolean, Source: "form"},
			{TicketID: 7, ScopeID: "root", Key: "form_json", Value: SerializeValue(map[string]any{"request_kind": "vpn"}), ValueType: ValueTypeJSON, Source: "form"},
		})
	})
	if err != nil {
		t.Fatalf("BulkSet valid variables: %v", err)
	}

	items, err := svc.ListByTicket(7)
	if err != nil {
		t.Fatalf("ListByTicket after bulk set: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 variables after bulk set, got %+v", items)
	}

	if err := repo.DeleteByTicket(nil, 7); err != nil {
		t.Fatalf("DeleteByTicket: %v", err)
	}
	items, err = svc.ListByTicket(7)
	if err != nil {
		t.Fatalf("ListByTicket after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected variables deleted by ticket, got %+v", items)
	}
}

func TestVariableInferenceAndSerializationContracts(t *testing.T) {
	tests := []struct {
		name       string
		fieldType  string
		hasOptions bool
		want       string
	}{
		{name: "text becomes string", fieldType: form.FieldText, want: ValueTypeString},
		{name: "number becomes number", fieldType: form.FieldNumber, want: ValueTypeNumber},
		{name: "switch becomes boolean", fieldType: form.FieldSwitch, want: ValueTypeBoolean},
		{name: "checkbox without options becomes boolean", fieldType: form.FieldCheckbox, want: ValueTypeBoolean},
		{name: "checkbox with options becomes json", fieldType: form.FieldCheckbox, hasOptions: true, want: ValueTypeJSON},
		{name: "date range becomes json", fieldType: form.FieldDateRange, want: ValueTypeJSON},
		{name: "datetime becomes date", fieldType: form.FieldDatetime, want: ValueTypeDate},
	}
	for _, tt := range tests {
		if got := InferValueType(tt.fieldType, tt.hasOptions); got != tt.want {
			t.Fatalf("%s: InferValueType(%q,%v)=%q, want %q", tt.name, tt.fieldType, tt.hasOptions, got, tt.want)
		}
	}

	if got := SerializeValue(json.Number("42")); got != "42" {
		t.Fatalf("SerializeValue(json.Number)= %q, want 42", got)
	}
	if got := SerializeValue(true); got != "true" {
		t.Fatalf("SerializeValue(bool)= %q, want true", got)
	}
	got := SerializeValue(map[string]any{"roles": []string{"vpn"}})
	if !json.Valid([]byte(got)) || !strings.Contains(got, `"roles":["vpn"]`) {
		t.Fatalf("SerializeValue(map) = %q, want valid JSON payload", got)
	}
}
