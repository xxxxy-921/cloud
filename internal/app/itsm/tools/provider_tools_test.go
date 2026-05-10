package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	aiapp "metis/internal/app/ai/runtime"
)

func setupSeedToolsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&aiapp.Tool{}, &aiapp.AgentTool{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func TestAllToolsExposeServiceDeskDecisionAndSLAToolContracts(t *testing.T) {
	tools := AllTools()
	if len(tools) == 0 {
		t.Fatal("expected tool definitions")
	}

	byName := make(map[string]ITSMTool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	serviceMatch, ok := byName["itsm.service_match"]
	if !ok {
		t.Fatal("expected itsm.service_match definition")
	}
	if len(serviceMatch.RuntimeConfigSchema) == 0 || len(serviceMatch.RuntimeConfig) == 0 {
		t.Fatalf("expected service_match to define runtime config, got %+v", serviceMatch)
	}
	var runtimeCfg map[string]any
	if err := json.Unmarshal(serviceMatch.RuntimeConfig, &runtimeCfg); err != nil {
		t.Fatalf("unmarshal runtime config: %v", err)
	}
	if runtimeCfg["temperature"] != 0.2 || runtimeCfg["maxTokens"] != float64(1024) {
		t.Fatalf("unexpected runtime config defaults: %+v", runtimeCfg)
	}

	if _, ok := byName["decision.ticket_context"]; !ok {
		t.Fatal("expected decision tool definitions to be included")
	}
	if _, ok := byName["sla.trigger_escalation"]; !ok {
		t.Fatal("expected SLA tool definitions to be included")
	}

	withRuntimeConfig := 0
	for _, tool := range tools {
		if strings.HasPrefix(tool.Name, "decision.") {
			if len(tool.RuntimeConfigSchema) != 0 || len(tool.RuntimeConfig) != 0 {
				t.Fatalf("expected decision tool %s to inherit engine config without per-tool runtime config", tool.Name)
			}
		}
		if len(tool.RuntimeConfigSchema) > 0 || len(tool.RuntimeConfig) > 0 {
			withRuntimeConfig++
		}
	}
	if withRuntimeConfig == 0 {
		t.Fatal("expected at least one tool with runtime config defaults")
	}
}

func TestSeedToolsCreatesUpdatesAndClassifiesToolRecords(t *testing.T) {
	db := setupSeedToolsTestDB(t)

	deprecated := aiapp.Tool{
		Toolkit:          "itsm",
		Name:             "itsm.cancel_ticket",
		DisplayName:      "旧撤回工具",
		ParametersSchema: []byte(`{}`),
		IsActive:         true,
	}
	if err := db.Create(&deprecated).Error; err != nil {
		t.Fatalf("create deprecated tool: %v", err)
	}
	if err := db.Create(&aiapp.AgentTool{AgentID: 1, ToolID: deprecated.ID}).Error; err != nil {
		t.Fatalf("create deprecated binding: %v", err)
	}

	existing := aiapp.Tool{
		Toolkit:             "legacy",
		Name:                "itsm.service_match",
		DisplayName:         "旧服务匹配",
		Description:         "legacy",
		ParametersSchema:    []byte(`{"type":"object"}`),
		RuntimeConfigSchema: []byte(`{"legacy":true}`),
		RuntimeConfig:       nil,
		IsActive:            true,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create existing tool: %v", err)
	}

	if err := SeedTools(db); err != nil {
		t.Fatalf("seed tools: %v", err)
	}

	var deprecatedCount int64
	if err := db.Table("ai_tools").Where("name = ?", "itsm.cancel_ticket").Count(&deprecatedCount).Error; err != nil {
		t.Fatalf("count deprecated tools: %v", err)
	}
	if deprecatedCount != 0 {
		t.Fatalf("expected deprecated tool to be removed, still have %d", deprecatedCount)
	}
	var deprecatedBindingCount int64
	if err := db.Table("ai_agent_tools").Where("tool_id = ?", deprecated.ID).Count(&deprecatedBindingCount).Error; err != nil {
		t.Fatalf("count deprecated bindings: %v", err)
	}
	if deprecatedBindingCount != 0 {
		t.Fatalf("expected deprecated bindings to be removed, still have %d", deprecatedBindingCount)
	}

	var serviceMatch aiapp.Tool
	if err := db.Where("name = ?", "itsm.service_match").First(&serviceMatch).Error; err != nil {
		t.Fatalf("load service_match tool: %v", err)
	}
	if serviceMatch.DisplayName != "服务匹配" || serviceMatch.Description == "legacy" {
		t.Fatalf("expected service_match metadata to be updated, got %+v", serviceMatch)
	}
	if len(serviceMatch.RuntimeConfig) == 0 {
		t.Fatalf("expected empty runtime_config to be backfilled, got %+v", serviceMatch)
	}
	var backfilled map[string]any
	if err := json.Unmarshal(serviceMatch.RuntimeConfig, &backfilled); err != nil {
		t.Fatalf("unmarshal backfilled runtime config: %v", err)
	}
	if backfilled["modelId"] != float64(0) {
		t.Fatalf("expected default runtime config modelId=0, got %+v", backfilled)
	}

	var decisionTool aiapp.Tool
	if err := db.Where("name = ?", "decision.ticket_context").First(&decisionTool).Error; err != nil {
		t.Fatalf("load decision tool: %v", err)
	}
	if decisionTool.Toolkit != "decision" {
		t.Fatalf("expected decision toolkit classification, got %+v", decisionTool)
	}

	var slaTool aiapp.Tool
	if err := db.Where("name = ?", "sla.trigger_escalation").First(&slaTool).Error; err != nil {
		t.Fatalf("load sla tool: %v", err)
	}
	if slaTool.Toolkit != "sla" {
		t.Fatalf("expected sla toolkit classification, got %+v", slaTool)
	}

	var createTool aiapp.Tool
	if err := db.Where("name = ?", "itsm.ticket_create").First(&createTool).Error; err != nil {
		t.Fatalf("load ticket create tool: %v", err)
	}
	if createTool.Toolkit != "itsm" || !createTool.IsActive {
		t.Fatalf("expected standard itsm tool to be active and classified, got %+v", createTool)
	}
}
