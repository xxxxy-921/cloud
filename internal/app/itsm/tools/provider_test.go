package tools

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	aiapp "metis/internal/app/ai/runtime"
	"metis/internal/model"
)

func setupSeedAgentsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&aiapp.Provider{},
		&aiapp.AIModel{},
		&aiapp.Agent{},
		&aiapp.Tool{},
		&aiapp.AgentTool{},
		&aiapp.MCPServer{},
		&aiapp.Skill{},
		&aiapp.AgentMCPServer{},
		&aiapp.AgentSkill{},
		&aiapp.KnowledgeAsset{},
		&aiapp.AgentKnowledgeBase{},
		&aiapp.AgentKnowledgeGraph{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func TestSeedAgentsCreatesPresetAgentsWithDefaultModelAndBindings(t *testing.T) {
	db := setupSeedAgentsTestDB(t)

	provider := aiapp.Provider{Name: "Test Provider", Type: "openai", Protocol: "openai", BaseURL: "https://example.com", Status: "active"}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	defaultLLM := aiapp.AIModel{
		ModelID:     "gpt-test",
		DisplayName: "GPT Test",
		ProviderID:  provider.ID,
		Type:        "llm",
		IsDefault:   true,
		Status:      "active",
	}
	if err := db.Create(&defaultLLM).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	for _, toolName := range []string{"itsm.service_match", "general.current_time"} {
		if err := db.Create(&aiapp.Tool{Name: toolName, DisplayName: toolName, ParametersSchema: model.JSONText("{}")}).Error; err != nil {
			t.Fatalf("create tool %s: %v", toolName, err)
		}
	}

	if err := SeedAgents(db); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	var got aiapp.Agent
	if err := db.Where("code = ?", "itsm.servicedesk").First(&got).Error; err != nil {
		t.Fatalf("load service desk agent: %v", err)
	}
	if got.ModelID == nil || *got.ModelID != defaultLLM.ID {
		t.Fatalf("expected default model %d, got %v", defaultLLM.ID, got.ModelID)
	}
	if got.SystemPrompt == "" {
		t.Fatal("expected default system prompt to be seeded")
	}

	var toolCount int64
	if err := db.Table("ai_agent_tools").Where("agent_id = ?", got.ID).Count(&toolCount).Error; err != nil {
		t.Fatalf("count tool bindings: %v", err)
	}
	if toolCount != 2 {
		t.Fatalf("expected 2 seeded tool bindings, got %d", toolCount)
	}
}

func TestSeedAgentsKeepsExistingPresetAgentConfiguration(t *testing.T) {
	db := setupSeedAgentsTestDB(t)

	provider := aiapp.Provider{Name: "Test Provider", Type: "openai", Protocol: "openai", BaseURL: "https://example.com", Status: "active"}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	defaultModel := aiapp.AIModel{
		ModelID:     "gpt-default",
		DisplayName: "GPT Default",
		ProviderID:  provider.ID,
		Type:        "llm",
		IsDefault:   true,
		Status:      "active",
	}
	if err := db.Create(&defaultModel).Error; err != nil {
		t.Fatalf("create default model: %v", err)
	}
	userModel := aiapp.AIModel{
		ModelID:     "gpt-user",
		DisplayName: "GPT User",
		ProviderID:  provider.ID,
		Type:        "llm",
		Status:      "active",
	}
	if err := db.Create(&userModel).Error; err != nil {
		t.Fatalf("create user model: %v", err)
	}

	customTool := aiapp.Tool{Name: "custom.tool", DisplayName: "Custom Tool", ParametersSchema: model.JSONText("{}")}
	defaultTool := aiapp.Tool{Name: "itsm.service_match", DisplayName: "itsm.service_match", ParametersSchema: model.JSONText("{}")}
	if err := db.Create(&customTool).Error; err != nil {
		t.Fatalf("create custom tool: %v", err)
	}
	if err := db.Create(&defaultTool).Error; err != nil {
		t.Fatalf("create default tool: %v", err)
	}

	code := "itsm.servicedesk"
	userModelID := userModel.ID
	agent := aiapp.Agent{
		Name:         "IT éˆå¶…å§Ÿé™ç‰ˆæ«¤é‘³æˆ’ç¶‹",
		Code:         &code,
		Type:         aiapp.AgentTypeAssistant,
		Visibility:   aiapp.AgentVisibilityPublic,
		SystemPrompt: "user customized prompt",
		Temperature:  0.91,
		MaxTokens:    777,
		MaxTurns:     5,
		ModelID:      &userModelID,
		IsActive:     false,
		CreatedBy:    9,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create preset agent: %v", err)
	}
	if err := db.Model(&agent).Update("is_active", false).Error; err != nil {
		t.Fatalf("mark preset agent inactive: %v", err)
	}
	if err := db.Create(&aiapp.AgentTool{AgentID: agent.ID, ToolID: customTool.ID}).Error; err != nil {
		t.Fatalf("create custom binding: %v", err)
	}

	if err := SeedAgents(db); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	var got aiapp.Agent
	if err := db.Where("code = ?", code).First(&got).Error; err != nil {
		t.Fatalf("reload preset agent: %v", err)
	}
	if got.ModelID == nil || *got.ModelID != userModel.ID {
		t.Fatalf("expected model to stay %d, got %v", userModel.ID, got.ModelID)
	}
	if got.SystemPrompt != "user customized prompt" {
		t.Fatalf("expected prompt to remain customized, got %q", got.SystemPrompt)
	}
	if got.Temperature != 0.91 || got.MaxTokens != 777 || got.MaxTurns != 5 {
		t.Fatalf("expected numeric config to remain customized, got temp=%v maxTokens=%d maxTurns=%d", got.Temperature, got.MaxTokens, got.MaxTurns)
	}
	if got.IsActive {
		t.Fatal("expected active flag to remain user-configured")
	}

	var bindings []aiapp.AgentTool
	if err := db.Where("agent_id = ?", agent.ID).Find(&bindings).Error; err != nil {
		t.Fatalf("load bindings: %v", err)
	}
	if len(bindings) != 1 || bindings[0].ToolID != customTool.ID {
		t.Fatalf("expected custom tool binding to remain untouched, got %+v", bindings)
	}
}

func TestSeedAgentsBackfillsPresetCodeByNameWithoutOverwritingUserConfig(t *testing.T) {
	db := setupSeedAgentsTestDB(t)

	provider := aiapp.Provider{Name: "Test Provider", Type: "openai", Protocol: "openai", BaseURL: "https://example.com", Status: "active"}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	defaultModel := aiapp.AIModel{
		ModelID:     "gpt-default",
		DisplayName: "GPT Default",
		ProviderID:  provider.ID,
		Type:        "llm",
		IsDefault:   true,
		Status:      "active",
	}
	if err := db.Create(&defaultModel).Error; err != nil {
		t.Fatalf("create default model: %v", err)
	}

	agent := aiapp.Agent{
		Name:         "SLA 保障智能体",
		Type:         aiapp.AgentTypeAssistant,
		Visibility:   aiapp.AgentVisibilityPrivate,
		SystemPrompt: "user custom prompt",
		Temperature:  0.66,
		MaxTokens:    333,
		MaxTurns:     4,
		IsActive:     false,
		CreatedBy:    9,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create name-only agent: %v", err)
	}

	if err := SeedAgents(db); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	var got aiapp.Agent
	if err := db.Where("name = ?", "SLA 保障智能体").First(&got).Error; err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if got.Code == nil || *got.Code != "itsm.sla_assurance" {
		t.Fatalf("expected preset code to be backfilled, got %+v", got.Code)
	}
	if got.SystemPrompt != "user custom prompt" || got.Temperature != 0.66 || got.MaxTokens != 333 || got.MaxTurns != 4 {
		t.Fatalf("expected user-owned config to stay unchanged, got %+v", got)
	}
}

func TestSeedPresetAgentBindingsAndDefaultModelResolverContracts(t *testing.T) {
	db := setupSeedAgentsTestDB(t)

	if modelID := resolveDefaultLLMModelID(db); modelID != nil {
		t.Fatalf("expected nil default model without active default llm, got %v", *modelID)
	}

	provider := aiapp.Provider{Name: "Test Provider", Type: "openai", Protocol: "openai", BaseURL: "https://example.com", Status: "active"}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	inactiveDefault := aiapp.AIModel{
		ModelID:     "gpt-inactive-default",
		DisplayName: "GPT Inactive Default",
		ProviderID:  provider.ID,
		Type:        "llm",
		IsDefault:   true,
		Status:      "deprecated",
	}
	if err := db.Create(&inactiveDefault).Error; err != nil {
		t.Fatalf("create inactive default model: %v", err)
	}
	if modelID := resolveDefaultLLMModelID(db); modelID != nil {
		t.Fatalf("expected nil default model when default llm is inactive, got %v", *modelID)
	}
	activeDefault := aiapp.AIModel{
		ModelID:     "gpt-active-default",
		DisplayName: "GPT Active Default",
		ProviderID:  provider.ID,
		Type:        "llm",
		IsDefault:   true,
		Status:      "active",
	}
	if err := db.Create(&activeDefault).Error; err != nil {
		t.Fatalf("create active default model: %v", err)
	}
	if modelID := resolveDefaultLLMModelID(db); modelID == nil || *modelID != activeDefault.ID {
		t.Fatalf("expected active default llm id %d, got %v", activeDefault.ID, modelID)
	}

	tool := aiapp.Tool{Name: "itsm.service_match", DisplayName: "服务匹配", ParametersSchema: model.JSONText("{}")}
	skill := aiapp.Skill{Name: "service-skill", DisplayName: "Service Skill", SourceType: aiapp.SkillSourceUpload, IsActive: true}
	mcp := aiapp.MCPServer{Name: "service-mcp", Transport: aiapp.MCPTransportSSE, URL: "https://example.com/sse", AuthType: aiapp.AuthTypeNone, IsActive: true}
	kb := aiapp.KnowledgeAsset{Name: "service-kb", Category: "kb", Type: "naive_chunk", Status: "ready"}
	kg := aiapp.KnowledgeAsset{Name: "service-kg", Category: "kg", Type: "concept_map", Status: "ready"}
	for _, row := range []any{&tool, &skill, &mcp, &kb, &kg} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed binding target %T: %v", row, err)
		}
	}

	agent := aiapp.Agent{Name: "Binding Agent", Type: aiapp.AgentTypeAssistant, Visibility: aiapp.AgentVisibilityPrivate, CreatedBy: 1, IsActive: true}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create binding agent: %v", err)
	}

	preset := presetAgent{
		ToolNames:  []string{"itsm.service_match", "missing-tool"},
		SkillNames: []string{"service-skill"},
		MCPNames:   []string{"service-mcp"},
		KBNames:    []string{"service-kb"},
		KGNames:    []string{"service-kg"},
	}
	if err := seedPresetAgentBindings(db, agent.ID, preset); err != nil {
		t.Fatalf("seed preset bindings: %v", err)
	}
	if err := seedPresetAgentBindings(db, agent.ID, preset); err != nil {
		t.Fatalf("seed preset bindings second pass: %v", err)
	}

	var toolBindings int64
	if err := db.Table("ai_agent_tools").Where("agent_id = ?", agent.ID).Count(&toolBindings).Error; err != nil {
		t.Fatalf("count tool bindings: %v", err)
	}
	if toolBindings != 1 {
		t.Fatalf("expected only existing tool to bind once, got %d", toolBindings)
	}
	for _, check := range []struct {
		table string
		want  int64
	}{
		{table: "ai_agent_skills", want: 1},
		{table: "ai_agent_mcp_servers", want: 1},
		{table: "ai_agent_knowledge_bases", want: 1},
		{table: "ai_agent_knowledge_graphs", want: 1},
	} {
		var count int64
		if err := db.Table(check.table).Where("agent_id = ?", agent.ID).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", check.table, err)
		}
		if count != check.want {
			t.Fatalf("expected %s count=%d, got %d", check.table, check.want, count)
		}
	}
}
