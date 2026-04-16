package itsm

// bdd_test.go — godog BDD test suite entry point for ITSM.
//
// Run BDD tests:
//   go test ./internal/app/itsm/ -run TestBDD -v
//   make test-bdd

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

func TestBDD(t *testing.T) {
	suite := godog.TestSuite{
		Name:                "itsm-bdd",
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run BDD feature tests")
	}
}

func initializeScenario(sc *godog.ScenarioContext) {
	bc := newBDDContext()

	sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
		bc.reset()
		return ctx, nil
	})

	// Common Given steps
	sc.Given(`^已完成系统初始化$`, bc.givenSystemInitialized)
	sc.Given(`^已准备好以下参与人、岗位与职责$`, bc.givenParticipants)
}
