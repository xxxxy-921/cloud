package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// evalContext holds field values for gateway condition evaluation.
type evalContext map[string]any

// buildEvalContext creates the evaluation context from process variables and ticket data.
// It first tries to load from itsm_process_variables; if none exist (legacy ticket),
// falls back to parsing form_data JSON from ticket and latest activity.
func buildEvalContext(tx *gorm.DB, ticket *ticketModel, latestActivity *activityModel) evalContext {
	ctx := evalContext{
		"ticket.priority_id":  ticket.PriorityID,
		"ticket.requester_id": ticket.RequesterID,
		"ticket.status":       ticket.Status,
	}

	// Try to load process variables from the table
	var vars []processVariableModel
	if tx != nil {
		tx.Where("ticket_id = ? AND scope_id = ?", ticket.ID, "root").Find(&vars)
	}

	if len(vars) > 0 {
		// Populate var.<key> and form.<key> (backward compat) from process variables
		for _, v := range vars {
			deserialized := deserializeVarValue(v.Value, v.ValueType)
			ctx["var."+v.Key] = deserialized
			ctx["form."+v.Key] = deserialized // backward compatibility
		}
	} else {
		// Fallback: parse form_data JSON from ticket and latest activity (legacy behavior)
		if ticket.FormData != "" {
			var formData map[string]any
			if json.Unmarshal([]byte(ticket.FormData), &formData) == nil {
				for k, v := range formData {
					ctx["form."+k] = v
				}
			}
		}

		if latestActivity != nil && latestActivity.FormData != "" {
			var actData map[string]any
			if json.Unmarshal([]byte(latestActivity.FormData), &actData) == nil {
				for k, v := range actData {
					ctx["form."+k] = v
				}
			}
		}
	}

	// Also expose last activity outcome
	if latestActivity != nil {
		ctx["activity.outcome"] = latestActivity.TransitionOutcome
	}

	return ctx
}

// deserializeVarValue restores typed value from stored TEXT for eval context.
func deserializeVarValue(raw string, valueType string) any {
	if raw == "" {
		return nil
	}
	switch valueType {
	case "number":
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f
		}
		return raw
	case "boolean":
		if b, err := strconv.ParseBool(raw); err == nil {
			return b
		}
		return raw
	case "json":
		var v any
		if json.Unmarshal([]byte(raw), &v) == nil {
			return v
		}
		return raw
	default: // string, date
		return raw
	}
}

// evaluateCondition checks a single gateway condition against the context.
func evaluateCondition(cond GatewayCondition, ctx evalContext) bool {
	fieldVal, exists := ctx[cond.Field]
	if !exists {
		return false
	}

	switch cond.Operator {
	case "equals":
		return compareEqual(fieldVal, cond.Value)
	case "not_equals":
		return !compareEqual(fieldVal, cond.Value)
	case "contains_any":
		return containsAny(fieldVal, cond.Value)
	case "gt":
		return compareNumeric(fieldVal, cond.Value) > 0
	case "lt":
		return compareNumeric(fieldVal, cond.Value) < 0
	case "gte":
		return compareNumeric(fieldVal, cond.Value) >= 0
	case "lte":
		return compareNumeric(fieldVal, cond.Value) <= 0
	default:
		return false
	}
}

func compareEqual(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func containsAny(fieldVal, condVal any) bool {
	fieldStr := fmt.Sprintf("%v", fieldVal)

	switch v := condVal.(type) {
	case []any:
		for _, item := range v {
			if strings.EqualFold(fieldStr, fmt.Sprintf("%v", item)) {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if strings.EqualFold(fieldStr, item) {
				return true
			}
		}
	case string:
		return strings.Contains(strings.ToLower(fieldStr), strings.ToLower(v))
	}
	return false
}

func compareNumeric(a, b any) int {
	af := toFloat64(a)
	bf := toFloat64(b)
	if af > bf {
		return 1
	}
	if af < bf {
		return -1
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
