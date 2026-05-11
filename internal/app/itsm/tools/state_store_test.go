package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"metis/internal/model"
)

func TestSessionStateStoreInvalidJSONReturnsError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:itsm_state_invalid_json?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("CREATE TABLE ai_agent_sessions (id INTEGER PRIMARY KEY, state TEXT)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate audit log: %v", err)
	}
	if err := db.Exec("INSERT INTO ai_agent_sessions (id, state) VALUES (?, ?)", 42, "{bad-json").Error; err != nil {
		t.Fatalf("insert state: %v", err)
	}

	store := NewSessionStateStore(db)
	state, err := store.GetState(42)
	if err == nil {
		t.Fatalf("expected invalid state json error, got state=%+v", state)
	}
	if !strings.Contains(err.Error(), "invalid service desk state") {
		t.Fatalf("unexpected error: %v", err)
	}
	var audit model.AuditLog
	if err := db.Where("action = ? AND resource_id = ?", "itsm.service_desk.state_invalid", "42").First(&audit).Error; err != nil {
		t.Fatalf("expected invalid-state audit log: %v", err)
	}
	if audit.Level != model.AuditLevelError {
		t.Fatalf("expected error audit level, got %s", audit.Level)
	}
}

func TestSessionStateStoreDefaultsAndSaveRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:itsm_state_round_trip?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("CREATE TABLE ai_agent_sessions (id INTEGER PRIMARY KEY, state TEXT)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec("INSERT INTO ai_agent_sessions (id, state) VALUES (?, ?), (?, ?)", 1, "", 2, "{}").Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	store := NewSessionStateStore(db)

	defaulted, err := store.GetState(1)
	if err != nil {
		t.Fatalf("get default state: %v", err)
	}
	if defaulted == nil || defaulted.Stage != "idle" {
		t.Fatalf("expected idle default state, got %+v", defaulted)
	}

	state := &ServiceDeskState{
		Stage:                   "awaiting_confirmation",
		LoadedServiceID:         5,
		ServiceVersionID:        9,
		RequestText:             "申请 VPN",
		DraftSummary:            "VPN 开通申请",
		DraftVersion:            2,
		ConfirmedDraftVersion:   1,
		PendingNextRequiredTool: "itsm.draft_confirm",
		DraftFormData: map[string]any{
			"vpn_account": "tester@example.com",
		},
	}
	if err := store.SaveState(2, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	reloaded, err := store.GetState(2)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if reloaded.Stage != "awaiting_confirmation" || reloaded.LoadedServiceID != 5 || reloaded.ServiceVersionID != 9 {
		t.Fatalf("unexpected reloaded state: %+v", reloaded)
	}
	if reloaded.PendingNextRequiredTool != "itsm.draft_confirm" || reloaded.DraftVersion != 2 || reloaded.ConfirmedDraftVersion != 1 {
		t.Fatalf("expected draft metadata to round-trip, got %+v", reloaded)
	}
	if got := reloaded.DraftFormData["vpn_account"]; got != "tester@example.com" {
		t.Fatalf("expected form data to round-trip, got %+v", reloaded.DraftFormData)
	}

	var raw string
	if err := db.Table("ai_agent_sessions").Where("id = ?", 2).Select("state").Scan(&raw).Error; err != nil {
		t.Fatalf("read raw state: %v", err)
	}
	if !strings.Contains(raw, "\"loaded_service_id\":5") {
		t.Fatalf("expected saved json to contain loaded service id, got %s", raw)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("unmarshal saved state: %v", err)
	}
	if parsed["stage"] != "awaiting_confirmation" {
		t.Fatalf("expected persisted stage awaiting_confirmation, got %+v", parsed)
	}
}
