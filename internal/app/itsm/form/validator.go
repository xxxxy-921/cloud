package form

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
)

// FieldValidationError represents a validation error for a specific field.
type FieldValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateFormData validates user-submitted data against the form schema.
// It checks each field's validation rules and returns all errors found.
func ValidateFormData(schema FormSchema, data map[string]any) []FieldValidationError {
	var errs []FieldValidationError

	for _, field := range schema.Fields {
		val, exists := data[field.Key]
		isEmpty := !exists || val == nil || val == ""

		// Check required (from field.Required flag or validation rules)
		isRequired := field.Required
		for _, rule := range field.Validation {
			if rule.Rule == "required" {
				isRequired = true
				break
			}
		}

		if isRequired && isEmpty {
			msg := "此字段为必填项"
			for _, rule := range field.Validation {
				if rule.Rule == "required" && rule.Message != "" {
					msg = rule.Message
					break
				}
			}
			errs = append(errs, FieldValidationError{Field: field.Key, Message: msg})
			continue // skip further checks if required field is missing
		}

		if isEmpty {
			continue // optional field with no value — skip validation
		}

		// Apply each validation rule
		for _, rule := range field.Validation {
			if rule.Rule == "required" {
				continue // already handled above
			}
			if err := applyRule(field.Key, rule, val); err != nil {
				errs = append(errs, *err)
			}
		}
	}

	return errs
}

func applyRule(key string, rule ValidationRule, val any) *FieldValidationError {
	switch rule.Rule {
	case "minLength":
		s := toString(val)
		limit := toInt(rule.Value)
		if len([]rune(s)) < limit {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, fmt.Sprintf("长度不能少于 %d 个字符", limit))}
		}
	case "maxLength":
		s := toString(val)
		limit := toInt(rule.Value)
		if len([]rune(s)) > limit {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, fmt.Sprintf("长度不能超过 %d 个字符", limit))}
		}
	case "min":
		n := toFloat(val)
		limit := toFloat(rule.Value)
		if n < limit {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, fmt.Sprintf("值不能小于 %v", rule.Value))}
		}
	case "max":
		n := toFloat(val)
		limit := toFloat(rule.Value)
		if n > limit {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, fmt.Sprintf("值不能大于 %v", rule.Value))}
		}
	case "pattern":
		s := toString(val)
		pattern := toString(rule.Value)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return &FieldValidationError{Field: key, Message: fmt.Sprintf("无效的正则表达式: %s", pattern)}
		}
		if !re.MatchString(s) {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, "格式不正确")}
		}
	case "email":
		s := toString(val)
		if _, err := mail.ParseAddress(s); err != nil {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, "请输入有效的邮箱地址")}
		}
	case "url":
		s := toString(val)
		u, err := url.ParseRequestURI(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return &FieldValidationError{Field: key, Message: ruleMessage(rule, "请输入有效的 URL")}
		}
	}
	return nil
}

func ruleMessage(rule ValidationRule, fallback string) string {
	if rule.Message != "" {
		return rule.Message
	}
	return fallback
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json_number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// json_number is a type alias to avoid importing encoding/json just for json.Number
type json_number = interface{ Float64() (float64, error) }

func toInt(v any) int {
	return int(toFloat(v))
}
