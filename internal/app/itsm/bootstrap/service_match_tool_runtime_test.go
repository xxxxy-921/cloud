package bootstrap

import (
	"testing"

	aiapp "metis/internal/app/ai/runtime"
	"metis/internal/model"
)

func TestSeedServiceMatchToolRuntime_SeedsDefaultButPreservesCustomConfig(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&aiapp.Tool{}, &aiapp.Agent{}); err != nil {
		t.Fatalf("migrate ai runtime tables: %v", err)
	}

	tool := aiapp.Tool{
		Toolkit:       "itsm",
		Name:          "itsm.service_match",
		DisplayName:   "Service Match",
		Description:   "match services",
		RuntimeConfig: model.JSONText(`{"modelId":0,"temperature":0.2,"maxTokens":1024,"timeoutSeconds":30}`),
		IsActive:      true,
	}
	if err := db.Create(&tool).Error; err != nil {
		t.Fatalf("create tool: %v", err)
	}

	modelID := uint(42)
	code := "itsm.servicedesk"
	agent := aiapp.Agent{
		Name:        "Service Desk",
		Code:        &code,
		Type:        aiapp.AgentTypeAssistant,
		IsActive:    true,
		Visibility:  aiapp.AgentVisibilityTeam,
		CreatedBy:   1,
		ModelID:     &modelID,
		Temperature: 0.35,
		MaxTokens:   9000, // should clamp to default 1024
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}

	seedServiceMatchToolRuntime(db)

	var seeded aiapp.Tool
	if err := db.First(&seeded, tool.ID).Error; err != nil {
		t.Fatalf("reload seeded tool: %v", err)
	}
	wantSeeded := `{"modelId":42,"temperature":0.35,"maxTokens":1024,"timeoutSeconds":30}`
	if string(seeded.RuntimeConfig) != wantSeeded {
		t.Fatalf("runtime_config = %s, want %s", seeded.RuntimeConfig, wantSeeded)
	}

	custom := model.JSONText(`{"modelId":99,"temperature":0.6,"maxTokens":2048,"timeoutSeconds":45}`)
	if err := db.Model(&aiapp.Tool{}).Where("id = ?", tool.ID).Update("runtime_config", custom).Error; err != nil {
		t.Fatalf("set custom runtime: %v", err)
	}

	if err := db.Model(&aiapp.Agent{}).Where("id = ?", agent.ID).Updates(map[string]any{
		"model_id":     uint(7),
		"temperature":  0.1,
		"max_tokens":   256,
	}).Error; err != nil {
		t.Fatalf("mutate agent runtime: %v", err)
	}

	seedServiceMatchToolRuntime(db)

	var preserved aiapp.Tool
	if err := db.First(&preserved, tool.ID).Error; err != nil {
		t.Fatalf("reload preserved tool: %v", err)
	}
	if string(preserved.RuntimeConfig) != string(custom) {
		t.Fatalf("expected custom runtime to remain untouched, got %s want %s", preserved.RuntimeConfig, custom)
	}
}
