package form

import (
	"encoding/json"
	"testing"
)

func TestValidateFormDataRules(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "title", Type: FieldText, Label: "标题", Required: true, Validation: []ValidationRule{{Rule: "required", Message: "请输入标题"}, {Rule: "minLength", Value: 3, Message: "标题太短"}}},
			{Key: "age", Type: FieldNumber, Label: "年龄", Validation: []ValidationRule{{Rule: "min", Value: 18, Message: "年龄太小"}, {Rule: "max", Value: 60, Message: "年龄太大"}}},
			{Key: "code", Type: FieldText, Label: "编码", Validation: []ValidationRule{{Rule: "pattern", Value: `^[A-Z]+$`, Message: "编码必须大写"}}},
			{Key: "email", Type: FieldEmail, Label: "邮箱", Validation: []ValidationRule{{Rule: "email", Message: "邮箱格式错误"}}},
			{Key: "homepage", Type: FieldURL, Label: "主页", Validation: []ValidationRule{{Rule: "url", Message: "URL格式错误"}}},
			{Key: "optional", Type: FieldText, Label: "选填"},
		},
	}

	errs := ValidateFormData(schema, map[string]any{
		"title":    "",
		"age":      17,
		"code":     "abc",
		"email":    "bad",
		"homepage": "ftp://example.test",
		"optional": "",
	})
	got := map[string]string{}
	for _, err := range errs {
		got[err.Field] = err.Message
	}

	want := map[string]string{
		"title":    "请输入标题",
		"age":      "年龄太小",
		"code":     "编码必须大写",
		"email":    "邮箱格式错误",
		"homepage": "URL格式错误",
	}
	if len(got) != len(want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
	for field, message := range want {
		if got[field] != message {
			t.Fatalf("field %s message = %q, want %q; all=%+v", field, got[field], message, got)
		}
	}
	if _, ok := got["optional"]; ok {
		t.Fatalf("optional empty field should not fail: %+v", got)
	}
}

func TestValidateFormDataAcceptsValidValues(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "title", Type: FieldText, Label: "标题", Required: true, Validation: []ValidationRule{{Rule: "maxLength", Value: 8, Message: "标题太长"}}},
			{Key: "age", Type: FieldNumber, Label: "年龄", Validation: []ValidationRule{{Rule: "min", Value: 18, Message: "年龄太小"}, {Rule: "max", Value: 60, Message: "年龄太大"}}},
			{Key: "email", Type: FieldEmail, Label: "邮箱", Validation: []ValidationRule{{Rule: "email", Message: "邮箱格式错误"}}},
			{Key: "homepage", Type: FieldURL, Label: "主页", Validation: []ValidationRule{{Rule: "url", Message: "URL格式错误"}}},
		},
	}

	errs := ValidateFormData(schema, map[string]any{
		"title":    "VPN",
		"age":      30,
		"email":    "ops@example.test",
		"homepage": "https://example.test",
	})
	if len(errs) != 0 {
		t.Fatalf("expected valid data, got %+v", errs)
	}
}

func TestValidateFormDataStructuredValues(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "tags", Type: FieldMultiSelect, Label: "标签", Required: true, Options: []FieldOption{{Label: "VPN", Value: "vpn"}, {Label: "网络", Value: "network"}}},
			{Key: "agree", Type: FieldCheckbox, Label: "同意", Required: true},
			{Key: "systems", Type: FieldCheckbox, Label: "系统", Required: true, Options: []FieldOption{{Label: "ERP", Value: "erp"}}},
			{Key: "range", Type: FieldDateRange, Label: "日期范围", Required: true},
			{Key: "items", Type: FieldTable, Label: "明细", Required: true, Props: map[string]any{
				"columns": []TableColumn{
					{Key: "name", Type: FieldText, Label: "名称", Required: true},
					{Key: "kind", Type: FieldSelect, Label: "类型", Required: true, Options: []FieldOption{{Label: "网络", Value: "network"}}},
				},
			}},
		},
	}

	tests := []struct {
		name string
		data map[string]any
		want string
	}{
		{
			name: "empty array fails required",
			data: map[string]any{"tags": []any{}},
			want: "tags",
		},
		{
			name: "multi select must be array",
			data: map[string]any{"tags": "vpn", "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"name": "A", "kind": "network"}}},
			want: "tags",
		},
		{
			name: "multi select option membership",
			data: map[string]any{"tags": []any{"bad"}, "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"name": "A", "kind": "network"}}},
			want: "tags",
		},
		{
			name: "checkbox without options must be boolean",
			data: map[string]any{"tags": []any{"vpn"}, "agree": "yes", "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"name": "A", "kind": "network"}}},
			want: "agree",
		},
		{
			name: "checkbox with options must be array",
			data: map[string]any{"tags": []any{"vpn"}, "agree": true, "systems": "erp", "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"name": "A", "kind": "network"}}},
			want: "systems",
		},
		{
			name: "date range requires start and end",
			data: map[string]any{"tags": []any{"vpn"}, "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01"}, "items": []any{map[string]any{"name": "A", "kind": "network"}}},
			want: "range",
		},
		{
			name: "table must be row array",
			data: map[string]any{"tags": []any{"vpn"}, "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": map[string]any{"name": "A"}},
			want: "items",
		},
		{
			name: "table column required",
			data: map[string]any{"tags": []any{"vpn"}, "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"kind": "network"}}},
			want: "items[0].name",
		},
		{
			name: "table column option membership",
			data: map[string]any{"tags": []any{"vpn"}, "agree": true, "systems": []any{"erp"}, "range": map[string]any{"start": "2026-01-01", "end": "2026-01-02"}, "items": []any{map[string]any{"name": "A", "kind": "bad"}}},
			want: "items[0].kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateFormData(schema, tt.data)
			if len(errs) == 0 {
				t.Fatalf("expected validation error for %s", tt.want)
			}
			if errs[0].Field != tt.want {
				t.Fatalf("first error field = %s, want %s; all=%+v", errs[0].Field, tt.want, errs)
			}
		})
	}

	valid := map[string]any{
		"tags":    []any{"vpn", "network"},
		"agree":   true,
		"systems": []any{"erp"},
		"range":   map[string]any{"start": "2026-01-01", "end": "2026-01-02"},
		"items":   []any{map[string]any{"name": "A", "kind": "network"}},
	}
	if errs := ValidateFormData(schema, valid); len(errs) != 0 {
		t.Fatalf("expected structured values to be valid, got %+v", errs)
	}
}

func TestValidateFormDataStructuredValueShapes(t *testing.T) {
	type dateRange struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}

	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "duration", Type: FieldNumber, Label: "时长", Validation: []ValidationRule{{Rule: "min", Value: json.Number("1"), Message: "时长不能小于 1"}, {Rule: "max", Value: json.Number("8"), Message: "时长不能大于 8"}}},
			{Key: "range", Type: FieldDateRange, Label: "日期范围", Required: true},
			{Key: "items", Type: FieldTable, Label: "明细", Required: true, Props: map[string]any{
				"columns": []TableColumn{
					{Key: "owner", Type: FieldText, Label: "负责人", Validation: []ValidationRule{{Rule: "required", Message: "请填写负责人"}}},
				},
			}},
		},
	}

	t.Run("json number respects numeric rules", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"duration": json.Number("0"),
			"range":    dateRange{Start: "2026-01-01", End: "2026-01-02"},
			"items":    []map[string]any{{"owner": "alice"}},
		})
		if len(errs) != 1 || errs[0].Field != "duration" || errs[0].Message != "时长不能小于 1" {
			t.Fatalf("unexpected json number validation errors: %+v", errs)
		}
	})

	t.Run("date range struct shape is accepted", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"duration": json.Number("4"),
			"range":    dateRange{Start: "2026-01-01", End: "2026-01-02"},
			"items":    []map[string]any{{"owner": "alice"}},
		})
		if len(errs) != 0 {
			t.Fatalf("expected struct-shaped date range to pass, got %+v", errs)
		}
	})

	t.Run("empty date range object triggers required", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"duration": json.Number("4"),
			"range":    map[string]any{"start": "", "end": ""},
			"items":    []map[string]any{{"owner": "alice"}},
		})
		if len(errs) != 1 || errs[0].Field != "range" || errs[0].Message != "此字段为必填项" {
			t.Fatalf("unexpected empty date range errors: %+v", errs)
		}
	})

	t.Run("table column required message uses nested required rule", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"duration": json.Number("4"),
			"range":    dateRange{Start: "2026-01-01", End: "2026-01-02"},
			"items":    []map[string]any{{"owner": "", "note": "present"}},
		})
		if len(errs) != 1 || errs[0].Field != "items[0].owner" || errs[0].Message != "请填写负责人" {
			t.Fatalf("unexpected nested required errors: %+v", errs)
		}
	})
}

func TestValidateFormDataNumericAndTypedTableContracts(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "quota", Type: FieldNumber, Label: "配额", Validation: []ValidationRule{{Rule: "min", Value: 1, Message: "配额不能小于 1"}, {Rule: "max", Value: 10, Message: "配额不能大于 10"}}},
			{Key: "items", Type: FieldTable, Label: "明细", Required: true, Props: map[string]any{
				"columns": []TableColumn{
					{Key: "name", Type: FieldText, Label: "名称", Required: true},
				},
			}},
		},
	}

	t.Run("unsigned integers respect numeric rules", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"quota": uint(5),
			"items": []any{map[string]any{"name": "vpn"}},
		})
		if len(errs) != 0 {
			t.Fatalf("expected uint numeric value to pass, got %+v", errs)
		}
	})

	t.Run("typed empty table slice triggers required", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"quota": uint(5),
			"items": []map[string]any{},
		})
		if len(errs) != 1 || errs[0].Field != "items" || errs[0].Message != "此字段为必填项" {
			t.Fatalf("unexpected empty typed table errors: %+v", errs)
		}
	})
}

func TestValidateFormDataNumericTypeVariants(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "quota", Type: FieldNumber, Label: "配额", Validation: []ValidationRule{{Rule: "min", Value: 1, Message: "配额不能小于 1"}, {Rule: "max", Value: 10, Message: "配额不能大于 10"}}},
		},
	}

	tests := []struct {
		name string
		val  any
	}{
		{name: "int32", val: int32(5)},
		{name: "uint64", val: uint64(5)},
		{name: "json number", val: json.Number("5")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateFormData(schema, map[string]any{"quota": tt.val})
			if len(errs) != 0 {
				t.Fatalf("expected numeric variant to pass, got %+v", errs)
			}
		})
	}
}

func TestValidateFormDataOptionValuesAcceptNonStringScalars(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "priority", Type: FieldSelect, Label: "优先级", Options: []FieldOption{{Label: "P1", Value: 1}, {Label: "P2", Value: 2}}},
			{Key: "risk", Type: FieldRadio, Label: "风险", Options: []FieldOption{{Label: "高", Value: 10}, {Label: "中", Value: 20}}},
			{Key: "modules", Type: FieldMultiSelect, Label: "影响模块", Options: []FieldOption{{Label: "网关", Value: 100}, {Label: "支付", Value: 200}}},
			{Key: "systems", Type: FieldCheckbox, Label: "系统", Options: []FieldOption{{Label: "主集群", Value: 7}, {Label: "备集群", Value: 8}}},
		},
	}

	t.Run("numeric option values round-trip as submitted scalars", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"priority": 1,
			"risk":     10,
			"modules":  []any{100, 200},
			"systems":  []any{7},
		})
		if len(errs) != 0 {
			t.Fatalf("expected numeric option values to pass, got %+v", errs)
		}
	})

	t.Run("unknown numeric option still rejected", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"priority": 3,
		})
		if len(errs) != 1 || errs[0].Field != "priority" {
			t.Fatalf("expected priority option membership failure, got %+v", errs)
		}
	})

	t.Run("object item in multi select still rejected", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"modules": []any{map[string]any{"value": 100}},
		})
		if len(errs) != 1 || errs[0].Field != "modules" {
			t.Fatalf("expected invalid multi-select payload to fail, got %+v", errs)
		}
	})

	t.Run("string option values keep passing", func(t *testing.T) {
		stringSchema := FormSchema{
			Version: 1,
			Fields: []FormField{
				{Key: "category", Type: FieldSelect, Label: "分类", Options: []FieldOption{{Label: "VPN", Value: "vpn"}}},
				{Key: "tags", Type: FieldMultiSelect, Label: "标签", Options: []FieldOption{{Label: "网络", Value: "network"}}},
			},
		}
		errs := ValidateFormData(stringSchema, map[string]any{
			"category": "vpn",
			"tags":     []any{"network"},
		})
		if len(errs) != 0 {
			t.Fatalf("expected string option values to remain valid, got %+v", errs)
		}
	})
}

func TestValidateFormDataRuleFallbacksAndTypedCollections(t *testing.T) {
	schema := FormSchema{
		Version: 1,
		Fields: []FormField{
			{Key: "title", Type: FieldText, Label: "标题", Validation: []ValidationRule{{Rule: "minLength", Value: 2}, {Rule: "maxLength", Value: 4}}},
			{Key: "code", Type: FieldText, Label: "编码", Validation: []ValidationRule{{Rule: "pattern", Value: "["}}},
			{Key: "website", Type: FieldURL, Label: "站点", Validation: []ValidationRule{{Rule: "url"}}},
			{Key: "mailbox", Type: FieldEmail, Label: "邮箱", Validation: []ValidationRule{{Rule: "email"}}},
			{Key: "quota", Type: FieldNumber, Label: "配额", Validation: []ValidationRule{{Rule: "min", Value: uint32(2)}, {Rule: "max", Value: uint64(8)}}},
			{Key: "tags", Type: FieldMultiSelect, Label: "标签", Options: []FieldOption{{Label: "VPN", Value: "vpn"}, {Label: "DB", Value: "db"}}},
			{Key: "systems", Type: FieldCheckbox, Label: "系统", Options: []FieldOption{{Label: "ERP", Value: "erp"}, {Label: "OA", Value: "oa"}}},
		},
	}

	t.Run("built-in fallback messages are preserved", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"title":   "A",
			"code":    "OPS",
			"website": "ssh://bad.example",
			"mailbox": "bad",
			"quota":   uint32(1),
			"tags":    []string{"vpn"},
			"systems": []string{"erp"},
		})
		got := map[string]string{}
		for _, err := range errs {
			got[err.Field] = err.Message
		}
		if got["title"] != "长度不能少于 2 个字符" {
			t.Fatalf("title fallback message = %q, want minLength fallback; all=%+v", got["title"], got)
		}
		if got["code"] != "无效的正则表达式: [" {
			t.Fatalf("code fallback message = %q, want invalid regex fallback; all=%+v", got["code"], got)
		}
		if got["website"] != "请输入有效的 URL" {
			t.Fatalf("website fallback message = %q, want URL fallback; all=%+v", got["website"], got)
		}
		if got["mailbox"] != "请输入有效的邮箱地址" {
			t.Fatalf("mailbox fallback message = %q, want email fallback; all=%+v", got["mailbox"], got)
		}
		if got["quota"] != "值不能小于 2" {
			t.Fatalf("quota fallback message = %q, want min fallback; all=%+v", got["quota"], got)
		}
	})

	t.Run("typed string slices remain valid option collections", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"title":   "OPS",
			"code":    "OPS",
			"website": "https://example.test",
			"mailbox": "ops@example.test",
			"quota":   uint64(8),
			"tags":    []string{"vpn", "db"},
			"systems": []string{"erp", "oa"},
		})
		if len(errs) != 1 || errs[0].Field != "code" {
			t.Fatalf("expected only invalid regex error from code rule setup, got %+v", errs)
		}
	})

	t.Run("typed string slice option misses are still rejected", func(t *testing.T) {
		errs := ValidateFormData(schema, map[string]any{
			"title":   "OPS",
			"code":    "OPS",
			"website": "https://example.test",
			"mailbox": "ops@example.test",
			"quota":   uint64(4),
			"tags":    []string{"vpn", "ghost"},
			"systems": []string{"erp"},
		})
		fields := map[string]bool{}
		for _, err := range errs {
			fields[err.Field] = true
		}
		if !fields["tags"] {
			t.Fatalf("expected tags option membership error, got %+v", errs)
		}
	})
}

func TestValidatorHelperTypedInputAdapters(t *testing.T) {
	t.Run("toStringSlice accepts string collections and rejects mixed arrays", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
			want  []string
			ok    bool
		}{
			{name: "string slice", input: []string{"vpn", "db"}, want: []string{"vpn", "db"}, ok: true},
			{name: "any slice", input: []any{"vpn", "db"}, want: []string{"vpn", "db"}, ok: true},
			{name: "mixed any slice", input: []any{"vpn", 1}, ok: false},
			{name: "non array", input: "vpn", ok: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, ok := toStringSlice(tt.input)
				if ok != tt.ok {
					t.Fatalf("ok = %v, want %v; got=%v", ok, tt.ok, got)
				}
				if !tt.ok {
					return
				}
				if len(got) != len(tt.want) {
					t.Fatalf("len = %d, want %d; got=%v", len(got), len(tt.want), got)
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Fatalf("got[%d] = %q, want %q; all=%v", i, got[i], tt.want[i], got)
					}
				}
			})
		}
	})

	t.Run("toString normalizes nil string and scalar values", func(t *testing.T) {
		if got := toString(nil); got != "" {
			t.Fatalf("toString(nil) = %q, want empty string", got)
		}
		if got := toString("vpn"); got != "vpn" {
			t.Fatalf("toString(string) = %q, want vpn", got)
		}
		if got := toString(42); got != "42" {
			t.Fatalf("toString(int) = %q, want 42", got)
		}
		if got := toString(true); got != "true" {
			t.Fatalf("toString(bool) = %q, want true", got)
		}
	})

	t.Run("toFloat converts typed numerics and degrades unsupported values to zero", func(t *testing.T) {
		tests := []struct {
			name  string
			input any
			want  float64
		}{
			{name: "float64", input: 1.5, want: 1.5},
			{name: "int32", input: int32(7), want: 7},
			{name: "uint64", input: uint64(9), want: 9},
			{name: "json number", input: json.Number("12.5"), want: 12.5},
			{name: "unsupported string", input: "12.5", want: 0},
			{name: "unsupported bool", input: true, want: 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := toFloat(tt.input); got != tt.want {
					t.Fatalf("toFloat(%T) = %v, want %v", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("isNumber matches typed numerics and rejects non numeric values", func(t *testing.T) {
		numericValues := []any{int32(1), uint64(2), float32(3.5), json.Number("4")}
		for _, input := range numericValues {
			if !isNumber(input) {
				t.Fatalf("expected %T to be recognized as number", input)
			}
		}
		nonNumericValues := []any{"4", true, map[string]any{"value": 1}}
		for _, input := range nonNumericValues {
			if isNumber(input) {
				t.Fatalf("expected %T to be rejected as number", input)
			}
		}
	})
}
