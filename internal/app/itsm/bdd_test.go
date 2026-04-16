package itsm

// bdd_test.go — godog BDD test suite entry point for ITSM.
//
// This file configures the godog test suite to run Gherkin .feature files
// from the features/ directory. Scenarios tagged @wip are excluded by default.
//
// Run BDD tests:
//   go test ./internal/app/itsm/ -run TestBDD -v
//   make test-bdd
//
// Future feature files will cover:
//   - Classic engine workflow execution (token progression, gateways)
//   - Smart engine agent-driven ticket handling
//   - Workflow generation via LLM pipeline
//   - SLA enforcement and escalation

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

	// Step definitions will be registered here as features are added.
	// Example:
	//   sc.Given(`^一个服务定义 "([^"]*)" 使用经典引擎$`, bc.givenServiceWithClassicEngine)
	//   sc.When(`^用户创建工单$`, bc.whenUserCreatesTicket)
	//   sc.Then(`^工单状态为 "([^"]*)"$`, bc.thenTicketStatusIs)
	_ = bc
}
