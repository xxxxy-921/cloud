package form

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseSchemaAndValidationErrorContracts(t *testing.T) {
	raw := json.RawMessage(`{"version":1,"fields":[{"key":"title","type":"text","label":"标题"}]}`)
	schema, err := ParseSchema(raw)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	if schema.Version != 1 || len(schema.Fields) != 1 || schema.Fields[0].Key != "title" {
		t.Fatalf("unexpected parsed schema: %+v", schema)
	}

	if _, err := ParseSchema(json.RawMessage(`{"version":1,"fields":[`)); err == nil || !strings.Contains(err.Error(), "invalid schema JSON") {
		t.Fatalf("expected invalid schema JSON error, got %v", err)
	}

	if got := (ValidationError{Field: "fields[0].key", Message: "duplicate"}).Error(); got != "fields[0].key: duplicate" {
		t.Fatalf("ValidationError with field = %q", got)
	}
	if got := (ValidationError{Message: "standalone"}).Error(); got != "standalone" {
		t.Fatalf("ValidationError without field = %q", got)
	}
}

func TestValidateSchemaStructuredFields(t *testing.T) {
	tests := []struct {
		name   string
		schema FormSchema
		field  string
	}{
		{
			name: "select requires options",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "kind", Type: FieldSelect, Label: "类型"},
			}},
			field: "fields[0].options",
		},
		{
			name: "multi select duplicate option value",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "tags", Type: FieldMultiSelect, Label: "标签", Options: []FieldOption{{Label: "A", Value: "a"}, {Label: "B", Value: "a"}}},
			}},
			field: "fields[0].options[1].value",
		},
		{
			name: "table requires columns",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细"},
			}},
			field: "fields[0].props.columns",
		},
		{
			name: "table rejects nested table",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "nested", Type: FieldTable, Label: "嵌套"},
				}}},
			}},
			field: "fields[0].props.columns[0].type",
		},
		{
			name: "table column select requires options",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "kind", Type: FieldSelect, Label: "类型"},
				}}},
			}},
			field: "fields[0].props.columns[0].options",
		},
		{
			name: "duplicate table column key rejected",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "name", Type: FieldText, Label: "名称"},
					{Key: "name", Type: FieldText, Label: "名称2"},
				}}},
			}},
			field: "fields[0].props.columns[1].key",
		},
		{
			name: "table column label required",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "name", Type: FieldText, Label: ""},
				}}},
			}},
			field: "fields[0].props.columns[0].label",
		},
		{
			name: "table column rich text rejected",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "content", Type: FieldRichText, Label: "内容"},
				}}},
			}},
			field: "fields[0].props.columns[0].type",
		},
		{
			name: "table column unknown type rejected",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
					{Key: "owner", Type: "matrix", Label: "负责人"},
				}}},
			}},
			field: "fields[0].props.columns[0].type",
		},
		{
			name: "duplicate field key rejected",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "title", Type: FieldText, Label: "标题"},
				{Key: "title", Type: FieldTextarea, Label: "描述"},
			}},
			field: "fields[1].key",
		},
		{
			name: "unknown layout field reference rejected",
			schema: FormSchema{
				Version: 1,
				Fields:  []FormField{{Key: "title", Type: FieldText, Label: "标题"}},
				Layout:  &FormLayout{Sections: []LayoutSection{{Title: "基础信息", Fields: []string{"missing"}}}},
			},
			field: "layout.sections[0].fields[0]",
		},
		{
			name: "unknown validation rule rejected",
			schema: FormSchema{Version: 1, Fields: []FormField{
				{Key: "title", Type: FieldText, Label: "标题", Validation: []ValidationRule{{Rule: "unsupported", Message: "bad"}}},
			}},
			field: "fields[0].validation[0].rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSchema(tt.schema)
			if len(errs) == 0 {
				t.Fatal("expected schema validation error")
			}
			found := false
			for _, err := range errs {
				if err.Field == tt.field {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error field %s, got %+v", tt.field, errs)
			}
		})
	}

	valid := FormSchema{Version: 1, Fields: []FormField{
		{Key: "items", Type: FieldTable, Label: "明细", Props: map[string]any{"columns": []TableColumn{
			{Key: "name", Type: FieldText, Label: "名称", Required: true},
			{Key: "kind", Type: FieldSelect, Label: "类型", Options: []FieldOption{{Label: "网络", Value: "network"}}},
		}}},
	}}
	if errs := ValidateSchema(valid); len(errs) != 0 {
		t.Fatalf("expected valid schema, got %+v", errs)
	}
}

func TestTableColumnsContracts(t *testing.T) {
	t.Run("missing props columns rejected", func(t *testing.T) {
		_, err := TableColumns(FormField{Key: "items", Type: FieldTable, Label: "明细"})
		if err == nil || !strings.Contains(err.Error(), "props.columns") {
			t.Fatalf("expected missing props.columns error, got %v", err)
		}
	})

	t.Run("invalid columns payload rejected", func(t *testing.T) {
		_, err := TableColumns(FormField{
			Key:   "items",
			Type:  FieldTable,
			Label: "明细",
			Props: map[string]any{"columns": func() {}},
		})
		if err == nil || !strings.Contains(err.Error(), "marshal table columns") {
			t.Fatalf("expected marshal table columns error, got %v", err)
		}
	})

	t.Run("object shaped columns rejected", func(t *testing.T) {
		_, err := TableColumns(FormField{
			Key:   "items",
			Type:  FieldTable,
			Label: "明细",
			Props: map[string]any{"columns": map[string]any{"key": "name"}},
		})
		if err == nil || !strings.Contains(err.Error(), "invalid table columns") {
			t.Fatalf("expected invalid table columns error, got %v", err)
		}
	})
}
