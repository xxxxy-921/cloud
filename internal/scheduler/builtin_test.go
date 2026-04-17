package scheduler

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSetAuditLogCleanupHandler_InvokesCleaner(t *testing.T) {
	called := false
	cleaner := func() string {
		called = true
		return "cleaned: 10 rows"
	}

	task := &TaskDef{Name: "audit-log-cleanup"}
	SetAuditLogCleanupHandler(task, cleaner)

	if task.Handler == nil {
		t.Fatal("expected Handler to be set")
	}

	err := task.Handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatal("expected cleaner to be called")
	}
}
