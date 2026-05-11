package bootstrap

import (
	"testing"

	. "metis/internal/app/itsm/domain"
	"metis/internal/model"
)

func TestBootstrapHelperContracts(t *testing.T) {
	t.Run("sameMenuParent compares nil and ids explicitly", func(t *testing.T) {
		if !sameMenuParent(nil, nil) {
			t.Fatal("nil parents should match")
		}

		a := uint(7)
		b := uint(7)
		c := uint(9)
		if !sameMenuParent(&a, &b) {
			t.Fatal("equal parent ids should match")
		}
		if sameMenuParent(&a, &c) {
			t.Fatal("different parent ids should not match")
		}
		if sameMenuParent(&a, nil) || sameMenuParent(nil, &a) {
			t.Fatal("nil and non-nil parent ids should not match")
		}
	})

	t.Run("ensureSystemConfig creates once and never overwrites", func(t *testing.T) {
		db := newTestDB(t)

		ensureSystemConfig(db, "bootstrap.helper.contract", "v1")
		ensureSystemConfig(db, "bootstrap.helper.contract", "v2")

		var cfg model.SystemConfig
		if err := db.Where("\"key\" = ?", "bootstrap.helper.contract").First(&cfg).Error; err != nil {
			t.Fatalf("load system config: %v", err)
		}
		if cfg.Value != "v1" {
			t.Fatalf("expected existing value to be preserved, got %q", cfg.Value)
		}

		var count int64
		if err := db.Model(&model.SystemConfig{}).Where("\"key\" = ?", "bootstrap.helper.contract").Count(&count).Error; err != nil {
			t.Fatalf("count system config: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected one system config row, got %d", count)
		}
	})

	t.Run("deleteLegacyPathBuilderAgents removes only legacy preset codes", func(t *testing.T) {
		db := newTestDB(t)

		if err := db.Exec(`
			INSERT INTO ai_agents (name, code, type, visibility, created_by, is_active) VALUES
			('Legacy Path Builder', 'itsm.path_builder', 'assistant', 'team', 1, true),
			('Legacy Generator', 'itsm.generator', 'assistant', 'team', 1, true),
			('Service Desk', 'itsm.servicedesk', 'assistant', 'team', 1, true)
		`).Error; err != nil {
			t.Fatalf("seed ai_agents rows: %v", err)
		}

		deleteLegacyPathBuilderAgents(db)

		type agentRow struct {
			Code string
		}
		var rows []agentRow
		if err := db.Table("ai_agents").Order("code ASC").Scan(&rows).Error; err != nil {
			t.Fatalf("list ai_agents: %v", err)
		}
		if len(rows) != 1 || rows[0].Code != "itsm.servicedesk" {
			t.Fatalf("expected only servicedesk agent to remain, got %+v", rows)
		}
	})
}

func TestDeriveTicketStatusOutcomeCoversLegacyDirectBranches(t *testing.T) {
	db := newTicketStatusMigrationDB(t)

	t.Run("legacy pending maps back to submitted", func(t *testing.T) {
		status, outcome := deriveTicketStatusOutcome(db, Ticket{Status: "pending"})
		if status != TicketStatusSubmitted || outcome != "" {
			t.Fatalf("pending => (%s,%s), want (%s,%s)", status, outcome, TicketStatusSubmitted, "")
		}
	})

	t.Run("legacy failed stays failed", func(t *testing.T) {
		status, outcome := deriveTicketStatusOutcome(db, Ticket{Status: "failed"})
		if status != TicketStatusFailed || outcome != TicketOutcomeFailed {
			t.Fatalf("failed => (%s,%s), want (%s,%s)", status, outcome, TicketStatusFailed, TicketOutcomeFailed)
		}
	})

	t.Run("legacy cancelled without withdrawn timeline stays cancelled", func(t *testing.T) {
		ticket := Ticket{Code: "T-CANCEL-ONLY", Title: "cancel", ServiceID: 1, EngineType: "smart", Status: "cancelled", PriorityID: 1, RequesterID: 1}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create cancelled ticket: %v", err)
		}
		status, outcome := deriveTicketStatusOutcome(db, ticket)
		if status != TicketStatusCancelled || outcome != TicketOutcomeCancelled {
			t.Fatalf("cancelled => (%s,%s), want (%s,%s)", status, outcome, TicketStatusCancelled, TicketOutcomeCancelled)
		}
	})

	t.Run("legacy terminal statuses honor explicit stored outcome before history inference", func(t *testing.T) {
		explicitRejected := Ticket{
			Code:        "T-LEGACY-EXPLICIT-REJECTED",
			Title:       "explicit rejected",
			ServiceID:   1,
			EngineType:  "smart",
			Status:      "completed",
			Outcome:     TicketOutcomeRejected,
			PriorityID:  1,
			RequesterID: 1,
		}
		if err := db.Create(&explicitRejected).Error; err != nil {
			t.Fatalf("create explicit rejected ticket: %v", err)
		}
		status, outcome := deriveTicketStatusOutcome(db, explicitRejected)
		if status != TicketStatusRejected || outcome != TicketOutcomeRejected {
			t.Fatalf("legacy completed with explicit rejected outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusRejected, TicketOutcomeRejected)
		}

		explicitWithdrawn := Ticket{
			Code:        "T-LEGACY-EXPLICIT-WITHDRAWN",
			Title:       "explicit withdrawn",
			ServiceID:   1,
			EngineType:  "smart",
			Status:      "cancelled",
			Outcome:     TicketOutcomeWithdrawn,
			PriorityID:  1,
			RequesterID: 1,
		}
		if err := db.Create(&explicitWithdrawn).Error; err != nil {
			t.Fatalf("create explicit withdrawn ticket: %v", err)
		}
		status, outcome = deriveTicketStatusOutcome(db, explicitWithdrawn)
		if status != TicketStatusWithdrawn || outcome != TicketOutcomeWithdrawn {
			t.Fatalf("legacy cancelled with explicit withdrawn outcome => (%s,%s), want (%s,%s)", status, outcome, TicketStatusWithdrawn, TicketOutcomeWithdrawn)
		}
	})
}
