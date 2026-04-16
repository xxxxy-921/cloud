package itsm

// steps_common_test.go — shared BDD test context and common step definitions.
//
// bddContext holds the state shared across steps within a single scenario.
// Each scenario gets a fresh context via reset().

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// bddContext is the shared state container for BDD scenarios.
// Step definitions read and write fields on this struct.
type bddContext struct {
	db      *gorm.DB
	lastErr error

	// Entities populated by Given/When steps, asserted by Then steps.
	// Uncomment and expand as features are added:
	// catalog *ServiceCatalog
	// service *ServiceDefinition
	// ticket  any // future Ticket type
}

func newBDDContext() *bddContext {
	return &bddContext{}
}

// reset clears all state for a new scenario. Called in ctx.Before.
func (bc *bddContext) reset() {
	bc.lastErr = nil

	// Create a fresh in-memory database for each scenario.
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:bdd_%p?mode=memory&cache=shared", bc)), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("bdd: failed to open test db: %v", err))
	}

	// AutoMigrate models needed for BDD scenarios.
	if err := db.AutoMigrate(
		&ServiceCatalog{},
		&ServiceDefinition{},
		&ServiceAction{},
	); err != nil {
		panic(fmt.Sprintf("bdd: failed to migrate: %v", err))
	}

	bc.db = db
}
