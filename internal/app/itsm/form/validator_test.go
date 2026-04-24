package form

import "testing"

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
