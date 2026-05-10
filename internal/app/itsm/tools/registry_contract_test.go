package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	appcore "metis/internal/app"
)

func TestRegistryRegistersAndExecutesServiceDeskHandlers(t *testing.T) {
	store := newMemStateStore()
	store.states[7] = &ServiceDeskState{
		Stage:                   "awaiting_confirmation",
		LoadedServiceID:         5,
		ServiceVersionID:        9,
		DraftVersion:            2,
		ConfirmedDraftVersion:   1,
		PendingNextRequiredTool: "itsm.draft_confirm",
	}
	op := &stubOperator{
		tickets: []TicketSummary{{TicketID: 1, TicketCode: "TICK-1", Summary: "VPN", Status: "submitted"}},
	}
	registry := NewRegistry(op, store)

	for _, name := range []string{
		"itsm.service_match",
		"itsm.service_confirm",
		"itsm.service_load",
		"itsm.current_request_context",
		"itsm.new_request",
		"itsm.draft_prepare",
		"itsm.draft_confirm",
		"itsm.validate_participants",
		"itsm.ticket_create",
		"itsm.my_tickets",
		"itsm.ticket_withdraw",
	} {
		if !registry.HasTool(name) {
			t.Fatalf("expected registry to contain %s", name)
		}
	}
	if registry.HasTool("itsm.unknown") {
		t.Fatal("did not expect unknown tool to be registered")
	}

	ctx := context.WithValue(context.Background(), appcore.SessionIDKey, uint(7))
	currentContext, err := registry.Execute(ctx, "itsm.current_request_context", 1, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute current_request_context: %v", err)
	}
	var current map[string]any
	if err := json.Unmarshal(currentContext, &current); err != nil {
		t.Fatalf("unmarshal current context: %v", err)
	}
	if current["next_expected_action"] != "itsm.draft_confirm" {
		t.Fatalf("expected next_expected_action from stored state, got %+v", current)
	}

	ticketsPayload, err := registry.Execute(ctx, "itsm.my_tickets", 7, json.RawMessage(`{"status":"submitted"}`))
	if err != nil {
		t.Fatalf("execute my_tickets: %v", err)
	}
	var ticketsResp map[string]any
	if err := json.Unmarshal(ticketsPayload, &ticketsResp); err != nil {
		t.Fatalf("unmarshal my_tickets payload: %v", err)
	}
	if ticketsResp["ok"] != true || op.listStatus != "submitted" {
		t.Fatalf("expected my_tickets handler to delegate status filter, got payload=%+v status=%q", ticketsResp, op.listStatus)
	}

	withdrawPayload, err := registry.Execute(ctx, "itsm.ticket_withdraw", 7, json.RawMessage(`{"ticket_code":"TICK-1","reason":"用户撤回"}`))
	if err != nil {
		t.Fatalf("execute ticket_withdraw: %v", err)
	}
	var withdrawResp map[string]any
	if err := json.Unmarshal(withdrawPayload, &withdrawResp); err != nil {
		t.Fatalf("unmarshal withdraw payload: %v", err)
	}
	if withdrawResp["ok"] != true || op.withdrawCode != "TICK-1" || op.withdrawReason != "用户撤回" {
		t.Fatalf("expected withdraw handler delegation, got payload=%+v code=%q reason=%q", withdrawResp, op.withdrawCode, op.withdrawReason)
	}

	if _, err := registry.Execute(ctx, "itsm.unknown", 1, nil); err == nil || !strings.Contains(err.Error(), "unknown ITSM tool") {
		t.Fatalf("expected unknown tool error, got %v", err)
	}
}

func TestHashFormDataIsDeterministicAcrossKeyOrder(t *testing.T) {
	left := map[string]any{
		"summary": "VPN",
		"nested":  map[string]any{"b": 2, "a": 1},
		"users":   []any{"alice", "bob"},
	}
	right := map[string]any{
		"users":   []any{"alice", "bob"},
		"nested":  map[string]any{"a": 1, "b": 2},
		"summary": "VPN",
	}
	different := map[string]any{
		"summary": "VPN",
		"nested":  map[string]any{"a": 1, "b": 3},
		"users":   []any{"alice", "bob"},
	}

	leftHash := hashFormData(left)
	rightHash := hashFormData(right)
	if leftHash == "" || rightHash == "" {
		t.Fatalf("expected non-empty hashes, got left=%q right=%q", leftHash, rightHash)
	}
	if leftHash != rightHash {
		t.Fatalf("expected same logical form data to hash identically, got %q vs %q", leftHash, rightHash)
	}
	if leftHash == hashFormData(different) {
		t.Fatalf("expected changed form data to alter hash, got %q", leftHash)
	}
}
