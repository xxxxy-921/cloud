package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// evalContext holds field values for gateway condition evaluation.
type evalContext map[string]any

// buildEvalContext creates the evaluation context from ticket and latest activity data.
func buildEvalContext(ticket *ticketModel, latestActivity *activityModel) evalContext {
	ctx := evalContext{
		"ticket.priority_id":  ticket.PriorityID,
		"ticket.requester_id": ticket.RequesterID,
		"ticket.status":       ticket.Status,
	}

	// Parse ticket form data
	if ticket.FormData != "" {
		var formData map[string]any
		if json.Unmarshal([]byte(ticket.FormData), &formData) == nil {
			for k, v := range formData {
				ctx["form."+k] = v
			}
		}
	}

	// Parse latest activity form data
	if latestActivity != nil && latestActivity.FormData != "" {
		var actData map[string]any
		if json.Unmarshal([]byte(latestActivity.FormData), &actData) == nil {
			for k, v := range actData {
				ctx["form."+k] = v
			}
		}
	}

	// Also expose last activity outcome
	if latestActivity != nil {
		ctx["activity.outcome"] = latestActivity.TransitionOutcome
	}

	return ctx
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
