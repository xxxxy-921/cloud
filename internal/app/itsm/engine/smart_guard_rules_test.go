package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDBBackupWhitelistValidationRejectsIncompleteOrVagueWindows(t *testing.T) {
	t.Run("rejects placeholder values", func(t *testing.T) {
		err := validateDBBackupWhitelistFormJSON(`{
			"database_name":"prod-orders",
			"source_ip":"10.0.0.8",
			"whitelist_window":"{{ticket.form_data.whitelist_window}}",
			"access_reason":"应急备份"
		}`)
		if err == nil || !strings.Contains(err.Error(), "放行时间窗") {
			t.Fatalf("expected whitelist window placeholder rejection, got %v", err)
		}
	})

	t.Run("rejects vague time windows", func(t *testing.T) {
		err := validateDBBackupWhitelistFormJSON(`{
			"database_name":"prod-orders",
			"source_ip":"10.0.0.8",
			"whitelist_window":"今晚维护窗口",
			"access_reason":"应急备份"
		}`)
		if err == nil || !strings.Contains(err.Error(), "时间窗不明确") {
			t.Fatalf("expected vague window rejection, got %v", err)
		}
	})

	t.Run("accepts explicit range", func(t *testing.T) {
		err := validateDBBackupWhitelistFormJSON(`{
			"database_name":"prod-orders",
			"source_ip":"10.0.0.8",
			"whitelist_window":"2026-05-10 20:00 ~ 2026-05-10 22:00",
			"access_reason":"应急备份"
		}`)
		if err != nil {
			t.Fatalf("expected explicit window to pass, got %v", err)
		}
	})
}

func TestWhitelistWindowAndActionAliasHelpers(t *testing.T) {
	if isConcreteWhitelistWindow("明天晚上") {
		t.Fatal("expected vague natural language window to be rejected")
	}
	if !isConcreteWhitelistWindow("2026-05-10 20:00 至 2026-05-10 22:00") {
		t.Fatal("expected explicit start/end time window to be accepted")
	}

	precheckAliases := actionCodeAliases("db_backup_whitelist_precheck")
	if len(precheckAliases) != 2 || precheckAliases[0] != "db_backup_whitelist_precheck" || precheckAliases[1] != "backup_whitelist_precheck" {
		t.Fatalf("unexpected precheck aliases: %+v", precheckAliases)
	}

	defaultAliases := actionCodeAliases("custom_action")
	if len(defaultAliases) != 1 || defaultAliases[0] != "custom_action" {
		t.Fatalf("unexpected default aliases: %+v", defaultAliases)
	}
}

func TestDecisionToolDefsAndSmartPoliciesExposeBuiltins(t *testing.T) {
	tools := DecisionToolDefs()
	if len(tools) != 8 {
		t.Fatalf("expected 8 decision tools, got %d", len(tools))
	}
	wantNames := []string{
		"decision.ticket_context",
		"decision.knowledge_search",
		"decision.resolve_participant",
		"decision.user_workload",
		"decision.similar_history",
		"decision.sla_status",
		"decision.list_actions",
		"decision.execute_action",
	}
	for i, want := range wantNames {
		if tools[i].Name != want {
			t.Fatalf("tool %d name = %q, want %q", i, tools[i].Name, want)
		}
	}

	policies := builtInSmartDecisionPolicies()
	if len(policies) != 3 {
		t.Fatalf("expected 3 built-in smart policies, got %d", len(policies))
	}
}

func TestFindSnapshotServiceActionByCodeAliasesPrefersActiveMatches(t *testing.T) {
	svc := &serviceModel{
		ID: 7,
		ActionsJSON: `[
			{"id":1,"code":"backup_whitelist_precheck","name":"旧预检","description":"legacy","actionType":"http","configJson":{"method":"POST","url":"http://example.com"},"isActive":false},
			{"id":2,"code":"db_backup_whitelist_precheck","name":"预检","description":"current","actionType":"http","configJson":{"method":"POST","url":"http://example.com"},"isActive":true}
		]`,
	}

	var action serviceActionModel
	found, err := findSnapshotServiceActionByCodeAliases(svc, actionCodeAliases("backup_whitelist_precheck"), &action)
	if err != nil {
		t.Fatalf("find snapshot action: %v", err)
	}
	if !found {
		t.Fatal("expected active snapshot action to be found")
	}
	if action.ID != 2 || action.Code != "db_backup_whitelist_precheck" || action.ServiceID != 7 {
		t.Fatalf("unexpected action match: %+v", action)
	}
}

func TestSmartDecisionPolicyFuncApplyAndDecisionStoreResolveForTool(t *testing.T) {
	called := false
	policy := SmartDecisionPolicyFunc(func(ctx context.Context, e *SmartEngine, tx *gorm.DB, ticketID uint, plan *DecisionPlan, svc *serviceModel) (bool, error) {
		called = ctx != nil && e != nil && tx != nil && ticketID == 42 && plan != nil && svc != nil
		return true, nil
	})

	db := newResolveToolDB(t)
	applied, err := policy.Apply(context.Background(), &SmartEngine{}, db, 42, &DecisionPlan{NextStepType: NodeProcess}, &serviceModel{ID: 7})
	if err != nil {
		t.Fatalf("apply policy: %v", err)
	}
	if !applied || !called {
		t.Fatalf("expected policy func to be invoked, applied=%v called=%v", applied, called)
	}

	store := &decisionDataStore{db: db}
	resolver := NewParticipantResolver(nil)

	managerIDs, err := store.ResolveForTool(resolver, 42, []byte(`{"type":"requester_manager"}`))
	if err != nil {
		t.Fatalf("resolve requester manager: %v", err)
	}
	if len(managerIDs) != 1 || managerIDs[0] != 2 {
		t.Fatalf("expected requester manager 2, got %+v", managerIDs)
	}

	userIDs, err := store.ResolveForTool(resolver, 42, []byte(`{"type":"user","value":"alice"}`))
	if err != nil {
		t.Fatalf("resolve username participant: %v", err)
	}
	if len(userIDs) != 1 || userIDs[0] != 1 {
		t.Fatalf("expected username alice to resolve to 1, got %+v", userIDs)
	}
}

func newResolveToolDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		requester_id integer
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (
		id integer primary key,
		username text,
		is_active boolean,
		manager_id integer,
		deleted_at datetime
	)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, username, is_active, manager_id) VALUES
		(1, 'alice', 1, 2),
		(2, 'manager', 1, NULL)`).Error; err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, requester_id) VALUES (42, 1)`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	return db
}
