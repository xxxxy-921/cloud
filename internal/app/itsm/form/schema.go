package form

import (
	"encoding/json"
	"fmt"
)

// Supported field types (15 types).
const (
	FieldText        = "text"
	FieldTextarea    = "textarea"
	FieldNumber      = "number"
	FieldEmail       = "email"
	FieldURL         = "url"
	FieldSelect      = "select"
	FieldMultiSelect = "multi_select"
	FieldRadio       = "radio"
	FieldCheckbox    = "checkbox"
	FieldSwitch      = "switch"
	FieldDate        = "date"
	FieldDatetime    = "datetime"
	FieldDateRange   = "date_range"
	FieldUserPicker  = "user_picker"
	FieldDeptPicker  = "dept_picker"
	FieldRichText    = "rich_text"
)

var allowedFieldTypes = map[string]bool{
	FieldText: true, FieldTextarea: true, FieldNumber: true,
	FieldEmail: true, FieldURL: true, FieldSelect: true,
	FieldMultiSelect: true, FieldRadio: true, FieldCheckbox: true,
	FieldSwitch: true, FieldDate: true, FieldDatetime: true,
	FieldDateRange: true, FieldUserPicker: true, FieldDeptPicker: true,
	FieldRichText: true,
}

// Supported validation rule names.
var allowedValidationRules = map[string]bool{
	"required": true, "minLength": true, "maxLength": true,
	"min": true, "max": true, "pattern": true,
	"email": true, "url": true,
}

// FormSchema is the top-level schema definition.
type FormSchema struct {
	Version int         `json:"version"`
	Fields  []FormField `json:"fields"`
	Layout  *FormLayout `json:"layout,omitempty"`
}

// FormField defines a single field in the form.
type FormField struct {
	Key          string                 `json:"key"`
	Type         string                 `json:"type"`
	Label        string                 `json:"label"`
	Placeholder  string                 `json:"placeholder,omitempty"`
	Description  string                 `json:"description,omitempty"`
	DefaultValue any                    `json:"defaultValue,omitempty"`
	Required     bool                   `json:"required,omitempty"`
	Disabled     bool                   `json:"disabled,omitempty"`
	Validation   []ValidationRule       `json:"validation,omitempty"`
	Options      []FieldOption          `json:"options,omitempty"`
	Visibility   *VisibilityRule        `json:"visibility,omitempty"`
	Binding      string                 `json:"binding,omitempty"`
	Permissions  map[string]string      `json:"permissions,omitempty"`
	Width        string                 `json:"width,omitempty"`
	Props        map[string]any         `json:"props,omitempty"`
}

// ValidationRule defines a validation constraint on a field.
type ValidationRule struct {
	Rule    string `json:"rule"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message"`
}

// FieldOption defines a selectable option for select/radio/checkbox fields.
type FieldOption struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// VisibilityRule controls conditional field visibility.
type VisibilityRule struct {
	Conditions []VisibilityCondition `json:"conditions"`
	Logic      string                `json:"logic,omitempty"` // "and" | "or", default "and"
}

// VisibilityCondition is a single condition in a visibility rule.
type VisibilityCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // equals | not_equals | in | not_in | is_empty | is_not_empty
	Value    any    `json:"value,omitempty"`
}

// FormLayout defines the layout structure for the form.
type FormLayout struct {
	Columns  int             `json:"columns,omitempty"` // 1 | 2 | 3, default 1
	Sections []LayoutSection `json:"sections"`
}

// LayoutSection groups fields under a titled section.
type LayoutSection struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Collapsible bool     `json:"collapsible,omitempty"`
	Fields      []string `json:"fields"`
}

// ValidationError represents a schema structural validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ParseSchema parses raw JSON into a FormSchema.
func ParseSchema(raw json.RawMessage) (*FormSchema, error) {
	var schema FormSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("invalid schema JSON: %w", err)
	}
	return &schema, nil
}

// ValidateSchema checks the structural integrity of a FormSchema.
// It verifies: version, field key uniqueness, field type validity,
// layout field references, and validation rule names.
func ValidateSchema(schema FormSchema) []ValidationError {
	var errs []ValidationError

	// 1. Check version
	if schema.Version < 1 {
		errs = append(errs, ValidationError{Field: "version", Message: "version must be >= 1"})
	}

	// 2. Check fields
	if len(schema.Fields) == 0 {
		errs = append(errs, ValidationError{Field: "fields", Message: "at least one field is required"})
	}

	keySet := make(map[string]bool, len(schema.Fields))
	for i, f := range schema.Fields {
		prefix := fmt.Sprintf("fields[%d]", i)

		if f.Key == "" {
			errs = append(errs, ValidationError{Field: prefix + ".key", Message: "field key must not be empty"})
		} else if keySet[f.Key] {
			errs = append(errs, ValidationError{Field: prefix + ".key", Message: fmt.Sprintf("duplicate field key: %s", f.Key)})
		} else {
			keySet[f.Key] = true
		}

		if f.Label == "" {
			errs = append(errs, ValidationError{Field: prefix + ".label", Message: "field label must not be empty"})
		}

		if !allowedFieldTypes[f.Type] {
			errs = append(errs, ValidationError{Field: prefix + ".type", Message: fmt.Sprintf("unknown field type: %s", f.Type)})
		}

		// Check validation rules
		for j, r := range f.Validation {
			if !allowedValidationRules[r.Rule] {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("%s.validation[%d].rule", prefix, j),
					Message: fmt.Sprintf("unknown validation rule: %s", r.Rule),
				})
			}
		}
	}

	// 3. Check layout section field references
	if schema.Layout != nil {
		for i, section := range schema.Layout.Sections {
			sPrefix := fmt.Sprintf("layout.sections[%d]", i)
			if section.Title == "" {
				errs = append(errs, ValidationError{Field: sPrefix + ".title", Message: "section title must not be empty"})
			}
			for j, fieldKey := range section.Fields {
				if !keySet[fieldKey] {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("%s.fields[%d]", sPrefix, j),
						Message: fmt.Sprintf("layout references unknown field key: %s", fieldKey),
					})
				}
			}
		}
	}

	return errs
}
