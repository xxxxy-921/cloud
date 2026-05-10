package engine

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEvaluateCondition_Equals(t *testing.T) {
	ctx := evalContext{"form.type": "vpn"}
	cond := GatewayCondition{Field: "form.type", Operator: "equals", Value: "vpn"}
	if !evaluateCondition(cond, ctx) {
		t.Error("equals should match")
	}
	cond.Value = "other"
	if evaluateCondition(cond, ctx) {
		t.Error("equals should not match different value")
	}
}

func TestEvaluateCondition_NotEquals(t *testing.T) {
	ctx := evalContext{"form.type": "vpn"}
	cond := GatewayCondition{Field: "form.type", Operator: "not_equals", Value: "other"}
	if !evaluateCondition(cond, ctx) {
		t.Error("not_equals should match")
	}
	cond.Value = "vpn"
	if evaluateCondition(cond, ctx) {
		t.Error("not_equals should not match same value")
	}
}

func TestEvaluateCondition_ContainsAny(t *testing.T) {
	ctx := evalContext{"form.type": "vpn_request"}
	// String contains
	cond := GatewayCondition{Field: "form.type", Operator: "contains_any", Value: "vpn"}
	if !evaluateCondition(cond, ctx) {
		t.Error("contains_any with string should match substring")
	}
	// Array contains
	cond.Value = []any{"vpn_request", "other"}
	if !evaluateCondition(cond, ctx) {
		t.Error("contains_any with array should match element")
	}
	cond.Value = []any{"other", "nope"}
	if evaluateCondition(cond, ctx) {
		t.Error("contains_any with array should not match")
	}

	ctx["form.multi"] = []string{"vpn_request", "db_backup"}
	cond = GatewayCondition{Field: "form.multi", Operator: "contains_any", Value: []string{"db_backup", "other"}}
	if !evaluateCondition(cond, ctx) {
		t.Error("contains_any should match []string field values")
	}
	cond.Value = []any{"other", "nope"}
	if evaluateCondition(cond, ctx) {
		t.Error("contains_any should reject unmatched []string field values")
	}
}

func TestEvaluateCondition_ContainsAnyStringFieldAgainstArrayUsesSubstringMatch(t *testing.T) {
	tests := []struct {
		name  string
		field string
		cond  any
		want  bool
	}{
		{
			name:  "ops long sentence matches log phrase",
			field: "生产发布后需要日志排查",
			cond:  []any{"日志排查", "磁盘清理"},
			want:  true,
		},
		{
			name:  "ops synonym phrase matches array",
			field: "需要进程排查",
			cond:  []any{"进程排障", "进程排查"},
			want:  true,
		},
		{
			name:  "ops reordered phrase matches array",
			field: "请协助清理磁盘空间",
			cond:  []string{"磁盘清理", "清理磁盘"},
			want:  true,
		},
		{
			name:  "vpn enum remains compatible",
			field: "network_access_issue",
			cond:  []string{"online_support", "network_access_issue"},
			want:  true,
		},
		{
			name:  "unrelated security phrase does not match network",
			field: "安全审计",
			cond:  []string{"网络抓包", "连通性诊断"},
			want:  false,
		},
		{
			name:  "generic patrol does not match ops keywords",
			field: "普通巡检",
			cond:  []string{"进程排障", "磁盘清理"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := evalContext{"form.access_reason": tt.field}
			cond := GatewayCondition{Field: "form.access_reason", Operator: "contains_any", Value: tt.cond}
			got := evaluateCondition(cond, ctx)
			if got != tt.want {
				t.Fatalf("evaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_NumericComparisons(t *testing.T) {
	ctx := evalContext{"ticket.priority_id": float64(3)}

	tests := []struct {
		op   string
		val  any
		want bool
	}{
		{"gt", float64(2), true},
		{"gt", float64(3), false},
		{"lt", float64(4), true},
		{"lt", float64(3), false},
		{"gte", float64(3), true},
		{"gte", float64(4), false},
		{"lte", float64(3), true},
		{"lte", float64(2), false},
	}
	for _, tt := range tests {
		cond := GatewayCondition{Field: "ticket.priority_id", Operator: tt.op, Value: tt.val}
		got := evaluateCondition(cond, ctx)
		if got != tt.want {
			t.Errorf("%s %v: got %v, want %v", tt.op, tt.val, got, tt.want)
		}
	}
}

func TestEvaluateCondition_In(t *testing.T) {
	ctx := evalContext{"form.status": "open"}
	cond := GatewayCondition{Field: "form.status", Operator: "in", Value: []any{"open", "pending"}}
	if !evaluateCondition(cond, ctx) {
		t.Error("in should match")
	}
	cond.Value = []any{"closed", "resolved"}
	if evaluateCondition(cond, ctx) {
		t.Error("in should not match")
	}
}

func TestEvaluateCondition_NotIn(t *testing.T) {
	ctx := evalContext{"form.status": "open"}
	cond := GatewayCondition{Field: "form.status", Operator: "not_in", Value: []any{"closed", "resolved"}}
	if !evaluateCondition(cond, ctx) {
		t.Error("not_in should match when value not in set")
	}
	cond.Value = []any{"open", "pending"}
	if evaluateCondition(cond, ctx) {
		t.Error("not_in should not match when value in set")
	}
}

func TestEvaluateCondition_IsEmpty(t *testing.T) {
	ctx := evalContext{"form.name": ""}
	cond := GatewayCondition{Field: "form.name", Operator: "is_empty"}
	if !evaluateCondition(cond, ctx) {
		t.Error("is_empty should match empty string")
	}
	// Missing field
	cond.Field = "form.missing"
	if !evaluateCondition(cond, ctx) {
		t.Error("is_empty should match missing field")
	}
	// Non-empty
	ctx["form.name"] = "hello"
	cond.Field = "form.name"
	if evaluateCondition(cond, ctx) {
		t.Error("is_empty should not match non-empty")
	}

	ctx["form.selection"] = []any{}
	cond.Field = "form.selection"
	if !evaluateCondition(cond, ctx) {
		t.Error("is_empty should match empty array-like selections")
	}
}

func TestEvaluateCondition_IsNotEmpty(t *testing.T) {
	ctx := evalContext{"form.name": "hello"}
	cond := GatewayCondition{Field: "form.name", Operator: "is_not_empty"}
	if !evaluateCondition(cond, ctx) {
		t.Error("is_not_empty should match non-empty")
	}
	ctx["form.name"] = ""
	if evaluateCondition(cond, ctx) {
		t.Error("is_not_empty should not match empty string")
	}
	// Missing field
	cond.Field = "form.missing"
	if evaluateCondition(cond, ctx) {
		t.Error("is_not_empty should not match missing field")
	}

	ctx["form.selection"] = []string{}
	cond.Field = "form.selection"
	if evaluateCondition(cond, ctx) {
		t.Error("is_not_empty should not match empty string slices")
	}
}

func TestEvaluateCondition_Between(t *testing.T) {
	ctx := evalContext{"ticket.priority_id": float64(3)}
	cond := GatewayCondition{Field: "ticket.priority_id", Operator: "between", Value: []any{float64(1), float64(5)}}
	if !evaluateCondition(cond, ctx) {
		t.Error("between should match inclusive range")
	}
	cond.Value = []any{float64(4), float64(6)}
	if evaluateCondition(cond, ctx) {
		t.Error("between should not match out of range")
	}
	// Boundary inclusive
	cond.Value = []any{float64(3), float64(5)}
	if !evaluateCondition(cond, ctx) {
		t.Error("between should be inclusive on lower bound")
	}
}

func TestEvaluateCondition_Matches(t *testing.T) {
	ctx := evalContext{"form.email": "user@example.com"}
	cond := GatewayCondition{Field: "form.email", Operator: "matches", Value: `^[a-z]+@example\.com$`}
	if !evaluateCondition(cond, ctx) {
		t.Error("matches should match valid regex")
	}
	cond.Value = `^admin@`
	if evaluateCondition(cond, ctx) {
		t.Error("matches should not match non-matching regex")
	}
	// Invalid regex should return false
	cond.Value = `[invalid`
	if evaluateCondition(cond, ctx) {
		t.Error("matches should return false for invalid regex")
	}
}

func TestEvaluateCondition_CompoundAnd(t *testing.T) {
	ctx := evalContext{"form.type": "vpn", "ticket.priority_id": float64(3)}
	cond := GatewayCondition{
		Logic: "and",
		Conditions: []GatewayCondition{
			{Field: "form.type", Operator: "equals", Value: "vpn"},
			{Field: "ticket.priority_id", Operator: "gt", Value: float64(2)},
		},
	}
	if !evaluateCondition(cond, ctx) {
		t.Error("AND compound should match when all true")
	}
	// One false
	cond.Conditions[1].Value = float64(5)
	if evaluateCondition(cond, ctx) {
		t.Error("AND compound should not match when one false")
	}
}

func TestEvaluateCondition_CompoundOr(t *testing.T) {
	ctx := evalContext{"form.type": "vpn", "ticket.priority_id": float64(3)}
	cond := GatewayCondition{
		Logic: "or",
		Conditions: []GatewayCondition{
			{Field: "form.type", Operator: "equals", Value: "other"},
			{Field: "ticket.priority_id", Operator: "gt", Value: float64(2)},
		},
	}
	if !evaluateCondition(cond, ctx) {
		t.Error("OR compound should match when any true")
	}
	// All false
	cond.Conditions[1].Value = float64(5)
	if evaluateCondition(cond, ctx) {
		t.Error("OR compound should not match when all false")
	}
}

func TestEvaluateCondition_NestedCompound(t *testing.T) {
	ctx := evalContext{"form.type": "vpn", "ticket.priority_id": float64(3), "form.status": "open"}
	cond := GatewayCondition{
		Logic: "and",
		Conditions: []GatewayCondition{
			{Field: "form.type", Operator: "equals", Value: "vpn"},
			{
				Logic: "or",
				Conditions: []GatewayCondition{
					{Field: "ticket.priority_id", Operator: "gt", Value: float64(5)},
					{Field: "form.status", Operator: "equals", Value: "open"},
				},
			},
		},
	}
	if !evaluateCondition(cond, ctx) {
		t.Error("nested compound should match (type=vpn AND (prio>5 OR status=open))")
	}
}

func TestEvaluateCondition_BackwardCompatSingleCondition(t *testing.T) {
	// A simple condition without Logic/Conditions should still work
	ctx := evalContext{"form.type": "vpn"}
	cond := GatewayCondition{Field: "form.type", Operator: "equals", Value: "vpn"}
	if !evaluateCondition(cond, ctx) {
		t.Error("backward-compat single condition should work")
	}
}

func TestEvaluateCondition_MissingField(t *testing.T) {
	ctx := evalContext{}
	cond := GatewayCondition{Field: "form.missing", Operator: "equals", Value: "x"}
	if evaluateCondition(cond, ctx) {
		t.Error("missing field should return false for equals")
	}
}

func TestEvaluateCondition_UnknownOperator(t *testing.T) {
	ctx := evalContext{"form.type": "vpn"}
	cond := GatewayCondition{Field: "form.type", Operator: "unknown_op", Value: "vpn"}
	if evaluateCondition(cond, ctx) {
		t.Error("unknown operator should return false")
	}
}

func TestDeserializeVarValue(t *testing.T) {
	tests := []struct {
		raw       string
		valueType string
		want      any
	}{
		{"", "string", nil},
		{"hello", "string", "hello"},
		{"42.5", "number", float64(42.5)},
		{"true", "boolean", true},
		{`{"key":"val"}`, "json", map[string]any{"key": "val"}},
		{"2024-01-01", "date", "2024-01-01"},
	}
	for _, tt := range tests {
		got := deserializeVarValue(tt.raw, tt.valueType)
		if got == nil && tt.want == nil {
			continue
		}
		// Compare via sprintf for simplicity
		gotStr, wantStr := fmt.Sprintf("%v", got), fmt.Sprintf("%v", tt.want)
		if gotStr != wantStr {
			t.Errorf("deserializeVarValue(%q, %q) = %v, want %v", tt.raw, tt.valueType, got, tt.want)
		}
	}
}

func TestBuildEvalContextContracts(t *testing.T) {
	newDB := func(t *testing.T) *gorm.DB {
		t.Helper()
		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &processVariableModel{}); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
		return db
	}

	t.Run("process variables take precedence over legacy form data", func(t *testing.T) {
		db := newDB(t)
		ticket := ticketModel{
			Code:        "T-COND-1",
			Title:       "condition",
			Status:      "decisioning",
			ServiceID:   1,
			EngineType:  "smart",
			RequesterID: 9,
			PriorityID:  3,
			FormData:    `{"env":"legacy","count":"1"}`,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		activity := activityModel{
			TicketID:          ticket.ID,
			Name:              "审批",
			ActivityType:      "approve",
			Status:            "completed",
			TransitionOutcome: "approved",
			FormData:          `{"env":"activity-legacy","fromActivity":"yes"}`,
		}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		vars := []processVariableModel{
			{TicketID: ticket.ID, ScopeID: "root", Key: "env", Value: "runtime", ValueType: "string"},
			{TicketID: ticket.ID, ScopeID: "root", Key: "count", Value: "42", ValueType: "number"},
			{TicketID: ticket.ID, ScopeID: "root", Key: "approved", Value: "true", ValueType: "boolean"},
			{TicketID: ticket.ID, ScopeID: "root", Key: "payload", Value: `{"kind":"json"}`, ValueType: "json"},
		}
		if err := db.Create(&vars).Error; err != nil {
			t.Fatalf("create process vars: %v", err)
		}

		ctx := buildEvalContext(db, &ticket, &activity)
		if ctx["ticket.priority_id"] != uint(3) || ctx["ticket.requester_id"] != uint(9) || ctx["ticket.status"] != "decisioning" {
			t.Fatalf("unexpected ticket context: %+v", ctx)
		}
		if ctx["var.env"] != "runtime" || ctx["form.env"] != "runtime" {
			t.Fatalf("expected process variable to override legacy form data, got %+v", ctx)
		}
		if got, ok := ctx["var.count"].(float64); !ok || got != 42 {
			t.Fatalf("var.count = %#v, want float64(42)", ctx["var.count"])
		}
		if got, ok := ctx["form.approved"].(bool); !ok || !got {
			t.Fatalf("form.approved = %#v, want true", ctx["form.approved"])
		}
		payload, ok := ctx["var.payload"].(map[string]any)
		if !ok || payload["kind"] != "json" {
			t.Fatalf("var.payload = %#v, want decoded json map", ctx["var.payload"])
		}
		if _, exists := ctx["form.fromActivity"]; exists {
			t.Fatalf("expected legacy activity form data to be ignored when process vars exist, got %+v", ctx)
		}
		if ctx["activity.outcome"] != "approved" {
			t.Fatalf("activity.outcome = %#v, want approved", ctx["activity.outcome"])
		}
	})

	t.Run("legacy tickets fall back to ticket and activity form data", func(t *testing.T) {
		db := newDB(t)
		ticket := ticketModel{
			Code:        "T-COND-2",
			Title:       "legacy",
			Status:      "submitted",
			ServiceID:   1,
			EngineType:  "classic",
			RequesterID: 5,
			PriorityID:  1,
			FormData:    `{"region":"ticket","count":1}`,
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		activity := activityModel{
			TicketID:          ticket.ID,
			Name:              "表单",
			ActivityType:      "form",
			Status:            "pending",
			TransitionOutcome: "submitted",
			FormData:          `{"region":"activity","comment":"latest"}`,
		}
		if err := db.Create(&activity).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}

		ctx := buildEvalContext(db, &ticket, &activity)
		if ctx["form.region"] != "activity" {
			t.Fatalf("expected latest activity form data to override ticket fallback, got %#v", ctx["form.region"])
		}
		if ctx["form.comment"] != "latest" {
			t.Fatalf("expected activity-only field to be present, got %#v", ctx["form.comment"])
		}
		if ctx["form.count"] != float64(1) {
			t.Fatalf("expected ticket form_data numeric fallback, got %#v", ctx["form.count"])
		}
		if ctx["activity.outcome"] != "submitted" {
			t.Fatalf("activity.outcome = %#v, want submitted", ctx["activity.outcome"])
		}
	})

	t.Run("nil transaction and malformed legacy json degrade safely", func(t *testing.T) {
		ticket := ticketModel{
			ID:          7,
			Status:      "submitted",
			RequesterID: 2,
			PriorityID:  8,
			FormData:    `{"broken"`,
		}
		activity := activityModel{
			TransitionOutcome: "rejected",
			FormData:          `{"still":"broken"`,
		}
		ctx := buildEvalContext(nil, &ticket, &activity)
		if ctx["ticket.priority_id"] != uint(8) || ctx["ticket.requester_id"] != uint(2) || ctx["ticket.status"] != "submitted" {
			t.Fatalf("unexpected base context without tx: %+v", ctx)
		}
		if _, exists := ctx["form.broken"]; exists {
			t.Fatalf("expected malformed json to be ignored, got %+v", ctx)
		}
		if ctx["activity.outcome"] != "rejected" {
			t.Fatalf("activity.outcome = %#v, want rejected", ctx["activity.outcome"])
		}
	})
}

func TestConditionHelpersHandleTypedCollectionsAndScalars(t *testing.T) {
	if got := toFloat64(int32(7)); got != 7 {
		t.Fatalf("toFloat64(int32(7)) = %v, want 7", got)
	}
	if got := toFloat64(uint32(8)); got != 8 {
		t.Fatalf("toFloat64(uint32(8)) = %v, want 8", got)
	}
	if got := toFloat64("12.5"); got != 12.5 {
		t.Fatalf("toFloat64(\"12.5\") = %v, want 12.5", got)
	}

	if !isEmpty([]any{}) {
		t.Fatal("expected empty []any to be treated as empty")
	}
	if !isEmpty([]string{}) {
		t.Fatal("expected empty []string to be treated as empty")
	}
	if !isEmpty(map[string]any{}) {
		t.Fatal("expected empty map to be treated as empty")
	}
	if isEmpty([]any{"value"}) {
		t.Fatal("expected populated slice not to be empty")
	}
	if isEmpty(map[string]any{"k": "v"}) {
		t.Fatal("expected populated map not to be empty")
	}
	if !isEmpty([0]string{}) {
		t.Fatal("expected empty array to be treated as empty")
	}
	if isEmpty([1]string{"v"}) {
		t.Fatal("expected populated array not to be empty")
	}
	if !isEmpty(false) {
		t.Fatal("expected false boolean to be treated as empty")
	}
	if isEmpty(true) {
		t.Fatal("expected true boolean not to be treated as empty")
	}

	if !inSet("open", []string{"pending", "open"}) {
		t.Fatal("expected inSet to match string slice")
	}
	if inSet("closed", []any{"pending", "open"}) {
		t.Fatal("expected inSet to reject non-members")
	}

	if !containsAny([]string{"db_backup", "vpn"}, []string{"vpn", "network"}) {
		t.Fatal("expected containsAny to match []string field against []string condition")
	}
	if !containsAny([]string{"db_backup", "vpn"}, []any{"ops", "db_backup"}) {
		t.Fatal("expected containsAny to match []string field against []any condition")
	}
	if containsAny([]string{"db_backup", "vpn"}, []string{"ops", "network"}) {
		t.Fatal("expected containsAny to reject non-overlapping []string field values")
	}

	if got := toFloat64(float32(1.5)); got != 1.5 {
		t.Fatalf("toFloat64(float32(1.5)) = %v, want 1.5", got)
	}
	if got := toFloat64(int64(9)); got != 9 {
		t.Fatalf("toFloat64(int64(9)) = %v, want 9", got)
	}
	if got := toFloat64(json.Number("bad")); got != 0 {
		t.Fatalf("toFloat64(json.Number(\"bad\")) = %v, want 0", got)
	}
	if got := toFloat64("bad"); got != 0 {
		t.Fatalf("toFloat64(\"bad\") = %v, want 0", got)
	}
}

func TestContainsAnyCoversCollectionAndScalarCombinations(t *testing.T) {
	t.Run("[]any field supports []string and substring string conditions", func(t *testing.T) {
		field := []any{"network_admin", "db_backup"}
		if !containsAny(field, []string{"ops_admin", "db_backup"}) {
			t.Fatal("expected []any field to match []string condition by exact element")
		}
		if !containsAny(field, "network") {
			t.Fatal("expected []any field to match substring string condition")
		}
		if containsAny(field, []string{"ops_admin", "security"}) {
			t.Fatal("expected []any field not to match unrelated []string condition")
		}
	})

	t.Run("scalar field supports exact and substring checks against arrays", func(t *testing.T) {
		if !containsAny("db_backup_whitelist_apply", []any{"backup", "other"}) {
			t.Fatal("expected scalar field to match []any condition by substring")
		}
		if !containsAny("network_access_issue", []string{"vpn", "access_issue"}) {
			t.Fatal("expected scalar field to match []string condition by substring")
		}
		if containsAny("serial_change", []any{"vpn", "network"}) {
			t.Fatal("expected scalar field not to match unrelated []any condition")
		}
	})
}
