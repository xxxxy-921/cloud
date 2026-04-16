package license

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateConstraintSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  ConstraintSchema
		wantErr error
	}{
		{
			name: "valid schema with one module",
			schema: ConstraintSchema{
				{Key: "mod1", Label: "Module 1", Features: []ConstraintFeature{
					{Key: "users", Label: "Users", Type: FeatureTypeNumber, Min: ptrFloat64(0), Max: ptrFloat64(100)},
				}},
			},
			wantErr: nil,
		},
		{
			name: "valid schema with multiple types",
			schema: ConstraintSchema{
				{Key: "core", Label: "Core", Features: []ConstraintFeature{
					{Key: "seats", Type: FeatureTypeNumber},
					{Key: "tier", Type: FeatureTypeEnum, Options: []string{"basic", "pro"}},
					{Key: "addons", Type: FeatureTypeMultiSelect, Options: []string{"backup", "audit"}},
				}},
			},
			wantErr: nil,
		},
		{
			name: "duplicate module key",
			schema: ConstraintSchema{
				{Key: "mod1"},
				{Key: "mod1"},
			},
			wantErr: ErrInvalidConstraintSchema,
		},
		{
			name: "empty module key",
			schema: ConstraintSchema{
				{Key: "", Features: []ConstraintFeature{{Key: "f1", Type: FeatureTypeNumber}}},
			},
			wantErr: ErrInvalidConstraintSchema,
		},
		{
			name: "duplicate feature key in module",
			schema: ConstraintSchema{
				{Key: "mod1", Features: []ConstraintFeature{
					{Key: "f1", Type: FeatureTypeNumber},
					{Key: "f1", Type: FeatureTypeEnum},
				}},
			},
			wantErr: ErrInvalidConstraintSchema,
		},
		{
			name: "empty feature key",
			schema: ConstraintSchema{
				{Key: "mod1", Features: []ConstraintFeature{
					{Key: "", Type: FeatureTypeNumber},
				}},
			},
			wantErr: ErrInvalidConstraintSchema,
		},
		{
			name: "invalid feature type",
			schema: ConstraintSchema{
				{Key: "mod1", Features: []ConstraintFeature{
					{Key: "f1", Type: "unknownType"},
				}},
			},
			wantErr: ErrInvalidConstraintSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConstraintSchema(tt.schema)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateConstraintSchema() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraintValues(t *testing.T) {
	schema := []byte(`[
		{"key":"core","label":"Core","features":[
			{"key":"seats","label":"Seats","type":"number","min":1,"max":100},
			{"key":"tier","label":"Tier","type":"enum","options":["basic","pro"]},
			{"key":"addons","label":"Addons","type":"multiSelect","options":["backup","audit"]}
		]}
	]`)

	tests := []struct {
		name    string
		values  []byte
		wantErr error
	}{
		{
			name:    "valid values",
			values:  []byte(`{"core":{"seats":10,"tier":"basic","addons":["backup"]}}`),
			wantErr: nil,
		},
		{
			name:    "enabled toggle is allowed",
			values:  []byte(`{"core":{"enabled":true,"seats":5}}`),
			wantErr: nil,
		},
		{
			name:    "missing module key",
			values:  []byte(`{"unknown":{"seats":5}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "unknown feature key",
			values:  []byte(`{"core":{"unknownFeature":5}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "number below min",
			values:  []byte(`{"core":{"seats":0}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "number above max",
			values:  []byte(`{"core":{"seats":101}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "enum value not in options",
			values:  []byte(`{"core":{"tier":"enterprise"}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "multiSelect type mismatch",
			values:  []byte(`{"core":{"addons":"backup"}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "multiSelect item not in options",
			values:  []byte(`{"core":{"addons":["unknown"]}}`),
			wantErr: ErrInvalidConstraintValues,
		},
		{
			name:    "number type mismatch",
			values:  []byte(`{"core":{"seats":"ten"}}`),
			wantErr: ErrInvalidConstraintValues,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConstraintValues(schema, tt.values)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateConstraintValues() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func FuzzValidateConstraintSchemaNoPanic(f *testing.F) {
	f.Add([]byte(`[{"key":"mod1","features":[{"key":"f1","type":"number"}]}]`))
	f.Add([]byte(`[{}]`))
	f.Add([]byte(`invalid json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var schema ConstraintSchema
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Skip()
		}
		_ = validateConstraintSchema(schema)
	})
}

func ptrFloat64(v float64) *float64 {
	return &v
}
