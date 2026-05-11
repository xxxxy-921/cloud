package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	aiapp "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/prompts"
	"metis/internal/app/itsm/testutil"
	"metis/internal/app/itsm/tools"
	"metis/internal/database"
	"metis/internal/handler"
	coremodel "metis/internal/model"
	"metis/internal/pkg/crypto"
	"metis/internal/repository"
)

func newEngineConfigTestService(t *testing.T) (*EngineConfigService, *database.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	if err := tools.SeedTools(db); err != nil {
		t.Fatalf("seed tools: %v", err)
	}
	if err := tools.SeedAgents(db); err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	if err := seedEngineConfigForTest(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, aiapp.NewAgentRepo)
	do.Provide(injector, aiapp.NewAgentService)
	do.Provide(injector, aiapp.NewModelRepo)
	do.Provide(injector, aiapp.NewProviderRepo)
	do.ProvideValue(injector, crypto.EncryptionKey(crypto.DeriveKey("test-secret")))
	do.Provide(injector, aiapp.NewProviderService)
	do.Provide(injector, aiapp.NewToolRepo)
	do.Provide(injector, aiapp.NewToolRuntimeService)
	do.Provide(injector, NewEngineConfigService)
	return do.MustInvoke[*EngineConfigService](injector), &database.DB{DB: db}
}

func seedEngineConfigForTest(db *gorm.DB) error {
	defaults := map[string]string{
		SmartTicketIntakeAgentKey:              "0",
		SmartTicketDecisionAgentKey:            "0",
		SmartTicketSLAAssuranceAgentKey:        "0",
		SmartTicketDecisionModeKey:             "direct_first",
		SmartTicketPathModelKey:                "0",
		SmartTicketPathTemperatureKey:          "0.3",
		SmartTicketPathMaxRetriesKey:           "1",
		SmartTicketPathTimeoutKey:              "60",
		SmartTicketPathSystemPromptKey:         prompts.PathBuilderSystemPromptDefault,
		SmartTicketSessionTitleModelKey:        "0",
		SmartTicketSessionTitleTemperatureKey:  "0.2",
		SmartTicketSessionTitleMaxRetriesKey:   "1",
		SmartTicketSessionTitleTimeoutKey:      "30",
		SmartTicketSessionTitlePromptKey:       SessionTitleSystemPromptDefault,
		SmartTicketPublishHealthModelKey:       "0",
		SmartTicketPublishHealthTemperatureKey: "0.2",
		SmartTicketPublishHealthMaxRetriesKey:  "1",
		SmartTicketPublishHealthTimeoutKey:     "45",
		SmartTicketPublishHealthPromptKey:      prompts.PublishHealthSystemPromptDefault,
		SmartTicketGuardAuditLevelKey:          "full",
		SmartTicketGuardFallbackKey:            "0",
	}
	for key, value := range defaults {
		if err := db.FirstOrCreate(&coremodel.SystemConfig{}, coremodel.SystemConfig{Key: key, Value: value}).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestEngineConfigServiceReadsAndUpdatesSmartStaffingAndEngineSettings(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	provider := aiapp.Provider{
		Name:     "OpenAI",
		Type:     aiapp.ProviderTypeOpenAI,
		Protocol: "openai",
		BaseURL:  "https://example.test",
		Status:   aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-test",
		DisplayName: "GPT Test",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	var intakeAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.servicedesk").First(&intakeAgent).Error; err != nil {
		t.Fatalf("load intake agent: %v", err)
	}
	var decisionAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.decision").First(&decisionAgent).Error; err != nil {
		t.Fatalf("load decision agent: %v", err)
	}
	var slaAssuranceAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.sla_assurance").First(&slaAssuranceAgent).Error; err != nil {
		t.Fatalf("load SLA assurance agent: %v", err)
	}
	if err := db.Model(&aiapp.Agent{}).Where("id IN ?", []uint{intakeAgent.ID, decisionAgent.ID, slaAssuranceAgent.ID}).Update("model_id", model.ID).Error; err != nil {
		t.Fatalf("bind staffing agent models: %v", err)
	}
	if err := db.Table("ai_tools").Where("name = ?", "itsm.service_match").Update("runtime_config",
		`{"modelId":`+strconv.FormatUint(uint64(model.ID), 10)+`,"temperature":0.2,"maxTokens":1024,"timeoutSeconds":30}`).Error; err != nil {
		t.Fatalf("configure service match tool runtime: %v", err)
	}

	fallback := coremodel.User{Username: "fallback", IsActive: true}
	if err := db.Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback user: %v", err)
	}

	var staffingReq UpdateSmartStaffingRequest
	staffingReq.Posts.Intake.AgentID = intakeAgent.ID
	staffingReq.Posts.Decision.AgentID = decisionAgent.ID
	staffingReq.Posts.Decision.Mode = "ai_only"
	staffingReq.Posts.SLAAssurance.AgentID = slaAssuranceAgent.ID
	if err := svc.UpdateSmartStaffingConfig(&staffingReq); err != nil {
		t.Fatalf("update smart staffing config: %v", err)
	}

	var engineReq UpdateEngineSettingsRequest
	engineReq.Runtime.PathBuilder.ModelID = model.ID
	engineReq.Runtime.PathBuilder.Temperature = 0.25
	engineReq.Runtime.PathBuilder.MaxRetries = 4
	engineReq.Runtime.PathBuilder.TimeoutSeconds = 90
	engineReq.Runtime.PathBuilder.SystemPrompt = "path prompt"
	engineReq.Runtime.TitleBuilder.ModelID = model.ID
	engineReq.Runtime.TitleBuilder.Temperature = 0.15
	engineReq.Runtime.TitleBuilder.MaxRetries = 2
	engineReq.Runtime.TitleBuilder.TimeoutSeconds = 45
	engineReq.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	engineReq.Runtime.HealthChecker.ModelID = model.ID
	engineReq.Runtime.HealthChecker.Temperature = 0.2
	engineReq.Runtime.HealthChecker.MaxRetries = 1
	engineReq.Runtime.HealthChecker.TimeoutSeconds = 55
	engineReq.Runtime.HealthChecker.SystemPrompt = "health prompt"
	engineReq.Runtime.Guard.AuditLevel = "summary"
	engineReq.Runtime.Guard.FallbackAssignee = fallback.ID
	if err := svc.UpdateEngineSettingsConfig(&engineReq); err != nil {
		t.Fatalf("update engine settings config: %v", err)
	}

	staffing, err := svc.GetSmartStaffingConfig()
	if err != nil {
		t.Fatalf("get smart staffing config: %v", err)
	}
	if staffing.Posts.Intake.AgentID != intakeAgent.ID || staffing.Posts.Decision.AgentID != decisionAgent.ID || staffing.Posts.Decision.Mode != "ai_only" || staffing.Posts.SLAAssurance.AgentID != slaAssuranceAgent.ID {
		t.Fatalf("unexpected smart staffing posts: %+v", staffing)
	}
	if len(staffing.Health.Items) != 3 {
		t.Fatalf("expected staffing health to contain only three posts, got %+v", staffing.Health.Items)
	}

	engineSettings, err := svc.GetEngineSettingsConfig()
	if err != nil {
		t.Fatalf("get engine settings config: %v", err)
	}
	if engineSettings.Runtime.PathBuilder.ModelID != model.ID || engineSettings.Runtime.PathBuilder.ProviderID != provider.ID || engineSettings.Runtime.PathBuilder.MaxRetries != 4 || engineSettings.Runtime.PathBuilder.TimeoutSeconds != 90 {
		t.Fatalf("unexpected path config: %+v", engineSettings.Runtime.PathBuilder)
	}
	if engineSettings.Runtime.PathBuilder.SystemPrompt != "path prompt" {
		t.Fatalf("unexpected path prompt: %q", engineSettings.Runtime.PathBuilder.SystemPrompt)
	}
	if engineSettings.Runtime.TitleBuilder.ModelID != model.ID || engineSettings.Runtime.TitleBuilder.ProviderID != provider.ID || engineSettings.Runtime.TitleBuilder.MaxRetries != 2 || engineSettings.Runtime.TitleBuilder.TimeoutSeconds != 45 {
		t.Fatalf("unexpected title builder config: %+v", engineSettings.Runtime.TitleBuilder)
	}
	if engineSettings.Runtime.TitleBuilder.SystemPrompt != "title prompt" {
		t.Fatalf("unexpected title builder prompt: %q", engineSettings.Runtime.TitleBuilder.SystemPrompt)
	}
	if engineSettings.Runtime.HealthChecker.ModelID != model.ID || engineSettings.Runtime.HealthChecker.ProviderID != provider.ID || engineSettings.Runtime.HealthChecker.MaxRetries != 1 || engineSettings.Runtime.HealthChecker.TimeoutSeconds != 55 {
		t.Fatalf("unexpected health checker config: %+v", engineSettings.Runtime.HealthChecker)
	}
	if engineSettings.Runtime.HealthChecker.SystemPrompt != "health prompt" {
		t.Fatalf("unexpected health checker prompt: %q", engineSettings.Runtime.HealthChecker.SystemPrompt)
	}
	if engineSettings.Runtime.Guard.AuditLevel != "summary" || engineSettings.Runtime.Guard.FallbackAssignee != fallback.ID {
		t.Fatalf("unexpected guard config: %+v", engineSettings.Runtime.Guard)
	}

	expectedKeys := map[string]string{
		SmartTicketIntakeAgentKey:              strconv.FormatUint(uint64(intakeAgent.ID), 10),
		SmartTicketDecisionAgentKey:            strconv.FormatUint(uint64(decisionAgent.ID), 10),
		SmartTicketSLAAssuranceAgentKey:        strconv.FormatUint(uint64(slaAssuranceAgent.ID), 10),
		SmartTicketDecisionModeKey:             "ai_only",
		SmartTicketPathModelKey:                strconv.FormatUint(uint64(model.ID), 10),
		SmartTicketPathTemperatureKey:          "0.25",
		SmartTicketPathMaxRetriesKey:           "4",
		SmartTicketPathTimeoutKey:              "90",
		SmartTicketPathSystemPromptKey:         "path prompt",
		SmartTicketSessionTitleModelKey:        strconv.FormatUint(uint64(model.ID), 10),
		SmartTicketSessionTitleTemperatureKey:  "0.15",
		SmartTicketSessionTitleMaxRetriesKey:   "2",
		SmartTicketSessionTitleTimeoutKey:      "45",
		SmartTicketSessionTitlePromptKey:       "title prompt",
		SmartTicketPublishHealthModelKey:       strconv.FormatUint(uint64(model.ID), 10),
		SmartTicketPublishHealthTemperatureKey: "0.2",
		SmartTicketPublishHealthMaxRetriesKey:  "1",
		SmartTicketPublishHealthTimeoutKey:     "55",
		SmartTicketPublishHealthPromptKey:      "health prompt",
		SmartTicketGuardAuditLevelKey:          "summary",
		SmartTicketGuardFallbackKey:            strconv.FormatUint(uint64(fallback.ID), 10),
	}
	for key, value := range expectedKeys {
		var cfg coremodel.SystemConfig
		if err := db.Where("\"key\" = ?", key).First(&cfg).Error; err != nil {
			t.Fatalf("load system config %s: %v", key, err)
		}
		if cfg.Value != value {
			t.Fatalf("expected %s=%s, got %s", key, value, cfg.Value)
		}
	}
}

func TestEngineConfigServiceRejectsInvalidFallbackAssignee(t *testing.T) {
	svc, _ := newEngineConfigTestService(t)
	var req UpdateEngineSettingsRequest
	req.Runtime.PathBuilder.MaxRetries = 3
	req.Runtime.PathBuilder.TimeoutSeconds = 120
	req.Runtime.PathBuilder.SystemPrompt = "path prompt"
	req.Runtime.TitleBuilder.MaxRetries = 1
	req.Runtime.TitleBuilder.TimeoutSeconds = 30
	req.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	req.Runtime.HealthChecker.MaxRetries = 1
	req.Runtime.HealthChecker.TimeoutSeconds = 45
	req.Runtime.HealthChecker.SystemPrompt = "health prompt"
	req.Runtime.Guard.AuditLevel = "full"
	req.Runtime.Guard.FallbackAssignee = 999
	if err := svc.UpdateEngineSettingsConfig(&req); err != ErrFallbackUserNotFound {
		t.Fatalf("expected ErrFallbackUserNotFound, got %v", err)
	}
}

func TestEngineConfigServiceRejectsInvalidUpdateInputs(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	provider := aiapp.Provider{
		Name:     "OpenAI Invalids",
		Type:     aiapp.ProviderTypeOpenAI,
		Protocol: "openai",
		BaseURL:  "https://invalids.example.test",
		Status:   aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-invalids",
		DisplayName: "GPT Invalids",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	var badStaffing UpdateSmartStaffingRequest
	badStaffing.Posts.Decision.Mode = "manual_only"
	if err := svc.UpdateSmartStaffingConfig(&badStaffing); !errors.Is(err, ErrInvalidEngineConfig) {
		t.Fatalf("expected invalid decision mode error, got %v", err)
	}

	base := UpdateEngineSettingsRequest{}
	base.Runtime.PathBuilder.ModelID = model.ID
	base.Runtime.PathBuilder.Temperature = 0.2
	base.Runtime.PathBuilder.MaxRetries = 1
	base.Runtime.PathBuilder.TimeoutSeconds = 30
	base.Runtime.PathBuilder.SystemPrompt = "path prompt"
	base.Runtime.TitleBuilder.ModelID = model.ID
	base.Runtime.TitleBuilder.Temperature = 0.2
	base.Runtime.TitleBuilder.MaxRetries = 1
	base.Runtime.TitleBuilder.TimeoutSeconds = 30
	base.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	base.Runtime.HealthChecker.ModelID = model.ID
	base.Runtime.HealthChecker.Temperature = 0.2
	base.Runtime.HealthChecker.MaxRetries = 1
	base.Runtime.HealthChecker.TimeoutSeconds = 30
	base.Runtime.HealthChecker.SystemPrompt = "health prompt"
	base.Runtime.Guard.AuditLevel = "full"

	t.Run("rejects invalid audit level", func(t *testing.T) {
		req := base
		req.Runtime.Guard.AuditLevel = "verbose"
		if err := svc.UpdateEngineSettingsConfig(&req); !errors.Is(err, ErrInvalidEngineConfig) {
			t.Fatalf("expected invalid audit level error, got %v", err)
		}
	})

	t.Run("rejects invalid temperature", func(t *testing.T) {
		req := base
		req.Runtime.PathBuilder.Temperature = 1.5
		if err := svc.UpdateEngineSettingsConfig(&req); !errors.Is(err, ErrInvalidEngineConfig) {
			t.Fatalf("expected invalid temperature error, got %v", err)
		}
	})

	t.Run("rejects invalid retry count", func(t *testing.T) {
		req := base
		req.Runtime.TitleBuilder.MaxRetries = 11
		if err := svc.UpdateEngineSettingsConfig(&req); !errors.Is(err, ErrInvalidEngineConfig) {
			t.Fatalf("expected invalid retries error, got %v", err)
		}
	})

	t.Run("rejects invalid timeout", func(t *testing.T) {
		req := base
		req.Runtime.HealthChecker.TimeoutSeconds = 301
		if err := svc.UpdateEngineSettingsConfig(&req); !errors.Is(err, ErrInvalidEngineConfig) {
			t.Fatalf("expected invalid timeout error, got %v", err)
		}
	})
}

func TestEngineConfigServiceSmartStaffingUpdateRollsBackOnWriteFailure(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	var intakeAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.servicedesk").First(&intakeAgent).Error; err != nil {
		t.Fatalf("load intake agent: %v", err)
	}
	var decisionAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.decision").First(&decisionAgent).Error; err != nil {
		t.Fatalf("load decision agent: %v", err)
	}

	if err := db.Exec(`CREATE TRIGGER reject_decision_agent_config_update
BEFORE UPDATE ON system_configs
WHEN NEW.key = '` + SmartTicketDecisionAgentKey + `'
BEGIN
	SELECT RAISE(FAIL, 'reject decision agent config');
END;`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	var req UpdateSmartStaffingRequest
	req.Posts.Intake.AgentID = intakeAgent.ID
	req.Posts.Decision.AgentID = decisionAgent.ID
	req.Posts.Decision.Mode = "direct_first"

	err := svc.UpdateSmartStaffingConfig(&req)
	if err == nil {
		t.Fatalf("expected write failure")
	}
	if got := svc.IntakeAgentID(); got != 0 {
		t.Fatalf("expected intake agent rollback to 0, got %d", got)
	}
}

func TestEngineConfigServiceEngineSettingsUpdateRollsBackOnWriteFailure(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	provider := aiapp.Provider{
		Name:     "OpenAI",
		Type:     aiapp.ProviderTypeOpenAI,
		Protocol: "openai",
		BaseURL:  "https://example.test",
		Status:   aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-test",
		DisplayName: "GPT Test",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	if err := db.Exec(`CREATE TRIGGER reject_path_temperature_config_update
BEFORE UPDATE ON system_configs
WHEN NEW.key = '` + SmartTicketPathTemperatureKey + `'
BEGIN
	SELECT RAISE(FAIL, 'reject path temperature config');
END;`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	var req UpdateEngineSettingsRequest
	req.Runtime.PathBuilder.ModelID = model.ID
	req.Runtime.PathBuilder.Temperature = 0.25
	req.Runtime.PathBuilder.MaxRetries = 4
	req.Runtime.PathBuilder.TimeoutSeconds = 90
	req.Runtime.PathBuilder.SystemPrompt = "path prompt"
	req.Runtime.TitleBuilder.ModelID = model.ID
	req.Runtime.TitleBuilder.Temperature = 0.15
	req.Runtime.TitleBuilder.MaxRetries = 2
	req.Runtime.TitleBuilder.TimeoutSeconds = 45
	req.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	req.Runtime.HealthChecker.ModelID = model.ID
	req.Runtime.HealthChecker.Temperature = 0.2
	req.Runtime.HealthChecker.MaxRetries = 1
	req.Runtime.HealthChecker.TimeoutSeconds = 55
	req.Runtime.HealthChecker.SystemPrompt = "health prompt"
	req.Runtime.Guard.AuditLevel = "summary"

	err := svc.UpdateEngineSettingsConfig(&req)
	if err == nil {
		t.Fatalf("expected write failure")
	}
	if got := svc.readPathConfig().ModelID; got != 0 {
		t.Fatalf("expected path model rollback to 0, got %d", got)
	}
}

func TestEngineConfigServiceRuntimeConfigRequiresDBPrompt(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	provider := aiapp.Provider{
		Name:     "OpenAI",
		Type:     aiapp.ProviderTypeOpenAI,
		Protocol: "openai",
		BaseURL:  "https://example.test",
		Status:   aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-test",
		DisplayName: "GPT Test",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	var req UpdateEngineSettingsRequest
	req.Runtime.PathBuilder.ModelID = model.ID
	req.Runtime.PathBuilder.Temperature = 0.2
	req.Runtime.PathBuilder.MaxRetries = 1
	req.Runtime.PathBuilder.TimeoutSeconds = 60
	req.Runtime.PathBuilder.SystemPrompt = "path prompt"
	req.Runtime.TitleBuilder.ModelID = model.ID
	req.Runtime.TitleBuilder.Temperature = 0.2
	req.Runtime.TitleBuilder.MaxRetries = 1
	req.Runtime.TitleBuilder.TimeoutSeconds = 30
	req.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	req.Runtime.HealthChecker.ModelID = model.ID
	req.Runtime.HealthChecker.Temperature = 0.2
	req.Runtime.HealthChecker.MaxRetries = 1
	req.Runtime.HealthChecker.TimeoutSeconds = 45
	req.Runtime.HealthChecker.SystemPrompt = "health prompt"
	req.Runtime.Guard.AuditLevel = "full"
	req.Runtime.Guard.FallbackAssignee = 0
	if err := svc.UpdateEngineSettingsConfig(&req); err != nil {
		t.Fatalf("update engine settings: %v", err)
	}

	if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPathSystemPromptKey).Update("value", "").Error; err != nil {
		t.Fatalf("clear path system prompt: %v", err)
	}
	_, err := svc.PathBuilderRuntimeConfig()
	if err == nil || !errors.Is(err, ErrEngineNotConfigured) {
		t.Fatalf("expected ErrEngineNotConfigured for path builder prompt, got %v", err)
	}

	if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketSessionTitlePromptKey).Update("value", "").Error; err != nil {
		t.Fatalf("clear title system prompt: %v", err)
	}
	_, err = svc.SessionTitleRuntimeConfig()
	if err == nil || !errors.Is(err, ErrEngineNotConfigured) {
		t.Fatalf("expected ErrEngineNotConfigured for title builder prompt, got %v", err)
	}

	if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPublishHealthPromptKey).Update("value", "").Error; err != nil {
		t.Fatalf("clear health checker system prompt: %v", err)
	}
	_, err = svc.HealthCheckRuntimeConfig()
	if err == nil || !errors.Is(err, ErrEngineNotConfigured) {
		t.Fatalf("expected ErrEngineNotConfigured for health checker prompt, got %v", err)
	}
}

func TestEngineConfigServiceRuntimeAccessorsAndLLMConfigs(t *testing.T) {
	svc, db := newEngineConfigTestService(t)
	key := crypto.EncryptionKey(crypto.DeriveKey("test-secret"))
	encrypted, err := crypto.Encrypt([]byte("secret-key"), key)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	provider := aiapp.Provider{
		Name:            "OpenAI Runtime",
		Type:            aiapp.ProviderTypeOpenAI,
		Protocol:        "openai",
		BaseURL:         "https://runtime.example.test",
		APIKeyEncrypted: encrypted,
		Status:          aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-runtime",
		DisplayName: "GPT Runtime",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	fallback := coremodel.User{Username: "runtime-fallback", IsActive: true}
	if err := db.Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback: %v", err)
	}

	var intakeAgent aiapp.Agent
	var decisionAgent aiapp.Agent
	var slaAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.servicedesk").First(&intakeAgent).Error; err != nil {
		t.Fatalf("load intake agent: %v", err)
	}
	if err := db.Where("code = ?", "itsm.decision").First(&decisionAgent).Error; err != nil {
		t.Fatalf("load decision agent: %v", err)
	}
	if err := db.Where("code = ?", "itsm.sla_assurance").First(&slaAgent).Error; err != nil {
		t.Fatalf("load sla agent: %v", err)
	}

	var staffingReq UpdateSmartStaffingRequest
	staffingReq.Posts.Intake.AgentID = intakeAgent.ID
	staffingReq.Posts.Decision.AgentID = decisionAgent.ID
	staffingReq.Posts.Decision.Mode = "ai_only"
	staffingReq.Posts.SLAAssurance.AgentID = slaAgent.ID
	if err := svc.UpdateSmartStaffingConfig(&staffingReq); err != nil {
		t.Fatalf("update staffing: %v", err)
	}

	var engineReq UpdateEngineSettingsRequest
	engineReq.Runtime.PathBuilder.ModelID = model.ID
	engineReq.Runtime.PathBuilder.Temperature = 0.25
	engineReq.Runtime.PathBuilder.MaxRetries = 2
	engineReq.Runtime.PathBuilder.TimeoutSeconds = 90
	engineReq.Runtime.PathBuilder.SystemPrompt = "path prompt"
	engineReq.Runtime.TitleBuilder.ModelID = model.ID
	engineReq.Runtime.TitleBuilder.Temperature = 0.1
	engineReq.Runtime.TitleBuilder.MaxRetries = 1
	engineReq.Runtime.TitleBuilder.TimeoutSeconds = 30
	engineReq.Runtime.TitleBuilder.SystemPrompt = "title prompt"
	engineReq.Runtime.HealthChecker.ModelID = model.ID
	engineReq.Runtime.HealthChecker.Temperature = 0.2
	engineReq.Runtime.HealthChecker.MaxRetries = 3
	engineReq.Runtime.HealthChecker.TimeoutSeconds = 45
	engineReq.Runtime.HealthChecker.SystemPrompt = "health prompt"
	engineReq.Runtime.Guard.AuditLevel = "summary"
	engineReq.Runtime.Guard.FallbackAssignee = fallback.ID
	if err := svc.UpdateEngineSettingsConfig(&engineReq); err != nil {
		t.Fatalf("update engine settings: %v", err)
	}

	pathCfg, err := svc.PathBuilderRuntimeConfig()
	if err != nil {
		t.Fatalf("PathBuilderRuntimeConfig: %v", err)
	}
	if pathCfg.Model != model.ModelID || pathCfg.BaseURL != provider.BaseURL || pathCfg.APIKey != "secret-key" || pathCfg.MaxTokens != 4096 || pathCfg.SystemPrompt != "path prompt" {
		t.Fatalf("unexpected path cfg: %+v", pathCfg)
	}
	titleCfg, err := svc.SessionTitleRuntimeConfig()
	if err != nil {
		t.Fatalf("SessionTitleRuntimeConfig: %v", err)
	}
	if titleCfg.MaxTokens != 96 || titleCfg.SystemPrompt != "title prompt" {
		t.Fatalf("unexpected title cfg: %+v", titleCfg)
	}
	healthCfg, err := svc.HealthCheckRuntimeConfig()
	if err != nil {
		t.Fatalf("HealthCheckRuntimeConfig: %v", err)
	}
	if healthCfg.MaxTokens != 1024 || healthCfg.SystemPrompt != "health prompt" {
		t.Fatalf("unexpected health cfg: %+v", healthCfg)
	}

	if svc.FallbackAssigneeID() != fallback.ID || svc.DecisionAgentID() != decisionAgent.ID || svc.SLAAssuranceAgentID() != slaAgent.ID || svc.IntakeAgentID() != intakeAgent.ID {
		t.Fatalf("unexpected accessor ids: fallback=%d decision=%d sla=%d intake=%d", svc.FallbackAssigneeID(), svc.DecisionAgentID(), svc.SLAAssuranceAgentID(), svc.IntakeAgentID())
	}
	if svc.DecisionMode() != "ai_only" || svc.AuditLevel() != "summary" || svc.SLACriticalThresholdSeconds() != 1800 || svc.SLAWarningThresholdSeconds() != 3600 || svc.SimilarHistoryLimit() != 5 || svc.ParallelConvergenceTimeout().Hours() != 72 {
		t.Fatalf("unexpected engine accessors")
	}
}

func TestEngineConfigServiceAccessorDefaultsRemainStable(t *testing.T) {
	svc, _ := newEngineConfigTestService(t)

	if svc.FallbackAssigneeID() != 0 {
		t.Fatalf("FallbackAssigneeID default = %d, want 0", svc.FallbackAssigneeID())
	}
	if svc.DecisionAgentID() != 0 || svc.SLAAssuranceAgentID() != 0 || svc.IntakeAgentID() != 0 {
		t.Fatalf("expected zero agent defaults, got intake=%d decision=%d sla=%d", svc.IntakeAgentID(), svc.DecisionAgentID(), svc.SLAAssuranceAgentID())
	}
	if svc.DecisionMode() != "direct_first" {
		t.Fatalf("DecisionMode default = %q, want %q", svc.DecisionMode(), "direct_first")
	}
	if svc.AuditLevel() != "full" {
		t.Fatalf("AuditLevel default = %q, want %q", svc.AuditLevel(), "full")
	}
	if svc.SLACriticalThresholdSeconds() != 1800 || svc.SLAWarningThresholdSeconds() != 3600 {
		t.Fatalf("unexpected SLA thresholds: critical=%d warning=%d", svc.SLACriticalThresholdSeconds(), svc.SLAWarningThresholdSeconds())
	}
	if svc.SimilarHistoryLimit() != 5 {
		t.Fatalf("SimilarHistoryLimit = %d, want 5", svc.SimilarHistoryLimit())
	}
	if svc.ParallelConvergenceTimeout() != 72*time.Hour {
		t.Fatalf("ParallelConvergenceTimeout = %s, want %s", svc.ParallelConvergenceTimeout(), 72*time.Hour)
	}
}

func TestEngineConfigServiceGettersDegradeGracefullyOnPersistedInvalidValues(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	updates := map[string]string{
		SmartTicketDecisionAgentKey:            "999999",
		SmartTicketDecisionModeKey:             "",
		SmartTicketPathModelKey:                "bad-model-id",
		SmartTicketPathTemperatureKey:          "not-a-float",
		SmartTicketPathMaxRetriesKey:           "oops",
		SmartTicketPathTimeoutKey:              "broken",
		SmartTicketPathSystemPromptKey:         "   ",
		SmartTicketSessionTitleModelKey:        "0",
		SmartTicketSessionTitleTemperatureKey:  "bad-title-temp",
		SmartTicketSessionTitleMaxRetriesKey:   "bad-title-retries",
		SmartTicketSessionTitleTimeoutKey:      "bad-title-timeout",
		SmartTicketSessionTitlePromptKey:       "   ",
		SmartTicketPublishHealthModelKey:       "123456",
		SmartTicketPublishHealthTemperatureKey: "bad-health-temp",
		SmartTicketPublishHealthMaxRetriesKey:  "bad-health-retries",
		SmartTicketPublishHealthTimeoutKey:     "bad-health-timeout",
		SmartTicketPublishHealthPromptKey:      "   ",
		SmartTicketGuardAuditLevelKey:          "",
		SmartTicketGuardFallbackKey:            "bad-user-id",
	}
	for key, value := range updates {
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", key).Update("value", value).Error; err != nil {
			t.Fatalf("update system config %s: %v", key, err)
		}
	}

	staffing, err := svc.GetSmartStaffingConfig()
	if err != nil {
		t.Fatalf("GetSmartStaffingConfig: %v", err)
	}
	if staffing.Posts.Decision.AgentID != 999999 || staffing.Posts.Decision.AgentName != "" {
		t.Fatalf("unexpected degraded decision agent selector: %+v", staffing.Posts.Decision)
	}
	if staffing.Posts.Decision.Mode != "direct_first" {
		t.Fatalf("decision mode fallback = %q, want %q", staffing.Posts.Decision.Mode, "direct_first")
	}

	settings, err := svc.GetEngineSettingsConfig()
	if err != nil {
		t.Fatalf("GetEngineSettingsConfig: %v", err)
	}
	if settings.Runtime.PathBuilder.ModelID != 0 || settings.Runtime.PathBuilder.ModelName != "" || settings.Runtime.PathBuilder.ProviderName != "" {
		t.Fatalf("unexpected degraded path builder model metadata: %+v", settings.Runtime.PathBuilder)
	}
	if settings.Runtime.PathBuilder.Temperature != 0 || settings.Runtime.PathBuilder.MaxRetries != 0 || settings.Runtime.PathBuilder.TimeoutSeconds != 0 {
		t.Fatalf("unexpected degraded path builder numeric defaults: %+v", settings.Runtime.PathBuilder)
	}
	if settings.Runtime.PathBuilder.SystemPrompt != "" {
		t.Fatalf("expected blank path builder prompt after trim, got %q", settings.Runtime.PathBuilder.SystemPrompt)
	}
	if settings.Runtime.TitleBuilder.ModelID != 0 || settings.Runtime.TitleBuilder.Temperature != 0 || settings.Runtime.TitleBuilder.MaxRetries != 0 || settings.Runtime.TitleBuilder.TimeoutSeconds != 0 || settings.Runtime.TitleBuilder.SystemPrompt != "" {
		t.Fatalf("unexpected degraded title builder config: %+v", settings.Runtime.TitleBuilder)
	}
	if settings.Runtime.HealthChecker.ModelID != 123456 || settings.Runtime.HealthChecker.ModelName != "" || settings.Runtime.HealthChecker.ProviderName != "" {
		t.Fatalf("unexpected degraded health checker model metadata: %+v", settings.Runtime.HealthChecker)
	}
	if settings.Runtime.HealthChecker.Temperature != 0 || settings.Runtime.HealthChecker.MaxRetries != 0 || settings.Runtime.HealthChecker.TimeoutSeconds != 0 || settings.Runtime.HealthChecker.SystemPrompt != "" {
		t.Fatalf("unexpected degraded health checker config: %+v", settings.Runtime.HealthChecker)
	}
	if settings.Runtime.Guard.AuditLevel != "full" || settings.Runtime.Guard.FallbackAssignee != 0 {
		t.Fatalf("unexpected degraded guard config: %+v", settings.Runtime.Guard)
	}
}

func TestEngineConfigHandlerMapsValidationAndSuccess(t *testing.T) {
	svc, db := newEngineConfigTestService(t)
	key := crypto.EncryptionKey(crypto.DeriveKey("test-secret"))
	encrypted, err := crypto.Encrypt([]byte("secret-key"), key)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	provider := aiapp.Provider{
		Name:            "OpenAI Handler",
		Type:            aiapp.ProviderTypeOpenAI,
		Protocol:        "openai",
		BaseURL:         "https://handler.example.test",
		APIKeyEncrypted: encrypted,
		Status:          aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-handler",
		DisplayName: "GPT Handler",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	fallback := coremodel.User{Username: "handler-fallback", IsActive: true}
	if err := db.Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback: %v", err)
	}
	var intakeAgent aiapp.Agent
	var decisionAgent aiapp.Agent
	var slaAgent aiapp.Agent
	_ = db.Where("code = ?", "itsm.servicedesk").First(&intakeAgent).Error
	_ = db.Where("code = ?", "itsm.decision").First(&decisionAgent).Error
	_ = db.Where("code = ?", "itsm.sla_assurance").First(&slaAgent).Error

	h := &EngineConfigHandler{svc: svc}
	perform := func(method, path string, body []byte, routes func(*gin.Engine)) *httptest.ResponseRecorder {
		t.Helper()
		gin.SetMode(gin.TestMode)
		r := gin.New()
		routes(r)
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	rec := perform(http.MethodPut, "/smart-staffing", []byte(`{"posts":{"intake":{"agentId":999}}}`), func(r *gin.Engine) {
		r.PUT("/smart-staffing", h.UpdateSmartStaffing)
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid smart staffing status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = perform(http.MethodPut, "/smart-staffing", []byte(`{"posts":`), func(r *gin.Engine) {
		r.PUT("/smart-staffing", h.UpdateSmartStaffing)
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed smart staffing body status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = perform(http.MethodPut, "/engine-settings", []byte(`{"runtime":{"pathBuilder":{"modelId":999,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"path"},"titleBuilder":{"modelId":999,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"title"},"healthChecker":{"modelId":999,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"health"},"guard":{"auditLevel":"full","fallbackAssignee":0}}}`), func(r *gin.Engine) {
		r.PUT("/engine-settings", h.UpdateEngineSettings)
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid engine settings status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = perform(http.MethodPut, "/engine-settings", []byte(`{"runtime":`), func(r *gin.Engine) {
		r.PUT("/engine-settings", h.UpdateEngineSettings)
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed engine settings body status=%d body=%s", rec.Code, rec.Body.String())
	}

	staffingBody := []byte(`{"posts":{"intake":{"agentId":` + strconv.FormatUint(uint64(intakeAgent.ID), 10) + `},"decision":{"agentId":` + strconv.FormatUint(uint64(decisionAgent.ID), 10) + `,"mode":"direct_first"},"slaAssurance":{"agentId":` + strconv.FormatUint(uint64(slaAgent.ID), 10) + `}}}`)
	rec = perform(http.MethodPut, "/smart-staffing", staffingBody, func(r *gin.Engine) {
		r.PUT("/smart-staffing", h.UpdateSmartStaffing)
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("valid smart staffing status=%d body=%s", rec.Code, rec.Body.String())
	}

	engineBody := []byte(`{"runtime":{"pathBuilder":{"modelId":` + strconv.FormatUint(uint64(model.ID), 10) + `,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"path"},"titleBuilder":{"modelId":` + strconv.FormatUint(uint64(model.ID), 10) + `,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"title"},"healthChecker":{"modelId":` + strconv.FormatUint(uint64(model.ID), 10) + `,"temperature":0.2,"maxRetries":1,"timeoutSeconds":30,"systemPrompt":"health"},"guard":{"auditLevel":"summary","fallbackAssignee":` + strconv.FormatUint(uint64(fallback.ID), 10) + `}}}`)
	rec = perform(http.MethodPut, "/engine-settings", engineBody, func(r *gin.Engine) {
		r.PUT("/engine-settings", h.UpdateEngineSettings)
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("valid engine settings status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = perform(http.MethodGet, "/smart-staffing", nil, func(r *gin.Engine) {
		r.GET("/smart-staffing", h.GetSmartStaffing)
	})
	var resp handler.R
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode smart staffing response: %v", err)
	}
	if rec.Code != http.StatusOK || resp.Code != 0 {
		t.Fatalf("GetSmartStaffing failed status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = perform(http.MethodGet, "/engine-settings", nil, func(r *gin.Engine) {
		r.GET("/engine-settings", h.GetEngineSettings)
	})
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode engine settings response: %v", err)
	}
	if rec.Code != http.StatusOK || resp.Code != 0 {
		t.Fatalf("GetEngineSettings failed status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestEngineConfigServiceHealthHelpersAndRuntimeFailures(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	t.Run("guard health reports warn fail and pass", func(t *testing.T) {
		item := svc.guardHealth(EngineGuardConfig{AuditLevel: "full", FallbackAssignee: 0})
		if item.Status != "warn" || item.Message == "" {
			t.Fatalf("expected warn guard health, got %+v", item)
		}

		item = svc.guardHealth(EngineGuardConfig{AuditLevel: "summary", FallbackAssignee: 999999})
		if item.Status != "fail" || item.Message == "" {
			t.Fatalf("expected fail guard health, got %+v", item)
		}

		fallback := coremodel.User{Username: "guard-fallback", IsActive: true}
		if err := db.Create(&fallback).Error; err != nil {
			t.Fatalf("create fallback user: %v", err)
		}
		item = svc.guardHealth(EngineGuardConfig{AuditLevel: "summary", FallbackAssignee: fallback.ID})
		if item.Status != "pass" {
			t.Fatalf("expected pass guard health, got %+v", item)
		}
	})

	t.Run("path title and health helpers reject invalid runtime params", func(t *testing.T) {
		pathItem := svc.pathHealth(EnginePathConfig{
			EngineModelConfig: EngineModelConfig{ModelID: 1},
			MaxRetries:        -1,
			TimeoutSeconds:    30,
			SystemPrompt:      "path",
		})
		if pathItem.Status != "fail" || pathItem.Message == "" {
			t.Fatalf("expected fail path health, got %+v", pathItem)
		}

		titleItem := svc.titleBuilderHealth(EnginePathConfig{
			EngineModelConfig: EngineModelConfig{ModelID: 1},
			MaxRetries:        0,
			TimeoutSeconds:    0,
			SystemPrompt:      "title",
		})
		if titleItem.Status != "fail" || titleItem.Message == "" {
			t.Fatalf("expected fail title health, got %+v", titleItem)
		}

		healthItem := svc.healthCheckerHealth(EnginePathConfig{
			EngineModelConfig: EngineModelConfig{ModelID: 1},
			MaxRetries:        0,
			TimeoutSeconds:    30,
			SystemPrompt:      "",
		})
		if healthItem.Status != "fail" || healthItem.Message == "" {
			t.Fatalf("expected fail health checker health, got %+v", healthItem)
		}
	})

	t.Run("runtime config methods reject zero timeout and negative retries", func(t *testing.T) {
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPathSystemPromptKey).Update("value", "path prompt").Error; err != nil {
			t.Fatalf("set path prompt: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPathTimeoutKey).Update("value", "0").Error; err != nil {
			t.Fatalf("set path timeout: %v", err)
		}
		if _, err := svc.PathBuilderRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected path timeout configuration error, got %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPathTimeoutKey).Update("value", "30").Error; err != nil {
			t.Fatalf("reset path timeout: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPathMaxRetriesKey).Update("value", "-1").Error; err != nil {
			t.Fatalf("set path retries: %v", err)
		}
		if _, err := svc.PathBuilderRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected path retries configuration error, got %v", err)
		}

		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketSessionTitlePromptKey).Update("value", "title prompt").Error; err != nil {
			t.Fatalf("set title prompt: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketSessionTitleTimeoutKey).Update("value", "0").Error; err != nil {
			t.Fatalf("set title timeout: %v", err)
		}
		if _, err := svc.SessionTitleRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected title timeout configuration error, got %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketSessionTitleTimeoutKey).Update("value", "30").Error; err != nil {
			t.Fatalf("reset title timeout: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketSessionTitleMaxRetriesKey).Update("value", "-1").Error; err != nil {
			t.Fatalf("set title retries: %v", err)
		}
		if _, err := svc.SessionTitleRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected title retries configuration error, got %v", err)
		}

		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPublishHealthPromptKey).Update("value", "health prompt").Error; err != nil {
			t.Fatalf("set health prompt: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPublishHealthTimeoutKey).Update("value", "0").Error; err != nil {
			t.Fatalf("set health timeout: %v", err)
		}
		if _, err := svc.HealthCheckRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected health timeout configuration error, got %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPublishHealthTimeoutKey).Update("value", "45").Error; err != nil {
			t.Fatalf("reset health timeout: %v", err)
		}
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", SmartTicketPublishHealthMaxRetriesKey).Update("value", "-1").Error; err != nil {
			t.Fatalf("set health retries: %v", err)
		}
		if _, err := svc.HealthCheckRuntimeConfig(); err == nil || !errors.Is(err, ErrEngineNotConfigured) {
			t.Fatalf("expected health retries configuration error, got %v", err)
		}
	})

	t.Run("build llm runtime config rejects missing model inactive provider and decrypt failure", func(t *testing.T) {
		if _, err := svc.buildLLMRuntimeConfig("测试引擎", 0, 0.2, 128, 1, 30); err == nil || err.Error() != "测试引擎未配置模型" {
			t.Fatalf("expected missing model error, got %v", err)
		}

		inactiveProvider := aiapp.Provider{
			Name:     "Inactive Provider",
			Type:     aiapp.ProviderTypeOpenAI,
			Protocol: "openai",
			BaseURL:  "https://inactive.example.test",
			Status:   aiapp.ProviderStatusInactive,
		}
		if err := db.Create(&inactiveProvider).Error; err != nil {
			t.Fatalf("create inactive provider: %v", err)
		}
		inactiveModel := aiapp.AIModel{
			ProviderID:  inactiveProvider.ID,
			ModelID:     "inactive-model",
			DisplayName: "Inactive Model",
			Type:        aiapp.ModelTypeLLM,
			Status:      aiapp.ModelStatusActive,
		}
		if err := db.Create(&inactiveModel).Error; err != nil {
			t.Fatalf("create inactive model: %v", err)
		}
		if _, err := svc.buildLLMRuntimeConfig("测试引擎", inactiveModel.ID, 0.2, 128, 1, 30); !errors.Is(err, ErrModelNotFound) {
			t.Fatalf("expected ErrModelNotFound for inactive provider, got %v", err)
		}

		badKeyProvider := aiapp.Provider{
			Name:            "Bad Key Provider",
			Type:            aiapp.ProviderTypeOpenAI,
			Protocol:        "openai",
			BaseURL:         "https://badkey.example.test",
			APIKeyEncrypted: []byte("not-valid-base64"),
			Status:          aiapp.ProviderStatusActive,
		}
		if err := db.Create(&badKeyProvider).Error; err != nil {
			t.Fatalf("create bad-key provider: %v", err)
		}
		badKeyModel := aiapp.AIModel{
			ProviderID:  badKeyProvider.ID,
			ModelID:     "bad-key-model",
			DisplayName: "Bad Key Model",
			Type:        aiapp.ModelTypeLLM,
			Status:      aiapp.ModelStatusActive,
		}
		if err := db.Create(&badKeyModel).Error; err != nil {
			t.Fatalf("create bad-key model: %v", err)
		}
		if _, err := svc.buildLLMRuntimeConfig("测试引擎", badKeyModel.ID, 0.2, 128, 1, 30); err == nil || !strings.Contains(err.Error(), "API Key 解密失败") {
			t.Fatalf("expected api key decrypt failure, got %v", err)
		}
	})
}

func TestEngineConfigHandlerGettersExposeConfiguredRuntimeContracts(t *testing.T) {
	svc, db := newEngineConfigTestService(t)

	provider := aiapp.Provider{
		Name:     "Getter OpenAI",
		Type:     aiapp.ProviderTypeOpenAI,
		Protocol: "openai",
		BaseURL:  "https://getter.example.test",
		Status:   aiapp.ProviderStatusActive,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	model := aiapp.AIModel{
		ProviderID:  provider.ID,
		ModelID:     "gpt-getter",
		DisplayName: "GPT Getter",
		Type:        aiapp.ModelTypeLLM,
		Status:      aiapp.ModelStatusActive,
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	fallback := coremodel.User{Username: "getter-fallback", IsActive: true}
	if err := db.Create(&fallback).Error; err != nil {
		t.Fatalf("create fallback user: %v", err)
	}

	var intakeAgent aiapp.Agent
	var decisionAgent aiapp.Agent
	var slaAgent aiapp.Agent
	if err := db.Where("code = ?", "itsm.servicedesk").First(&intakeAgent).Error; err != nil {
		t.Fatalf("load intake agent: %v", err)
	}
	if err := db.Where("code = ?", "itsm.decision").First(&decisionAgent).Error; err != nil {
		t.Fatalf("load decision agent: %v", err)
	}
	if err := db.Where("code = ?", "itsm.sla_assurance").First(&slaAgent).Error; err != nil {
		t.Fatalf("load SLA assurance agent: %v", err)
	}
	if err := db.Model(&aiapp.Agent{}).Where("id IN ?", []uint{intakeAgent.ID, decisionAgent.ID, slaAgent.ID}).Update("model_id", model.ID).Error; err != nil {
		t.Fatalf("bind agent models: %v", err)
	}
	if err := db.Table("ai_tools").Where("name = ?", "itsm.service_match").Update("runtime_config",
		`{"modelId":`+strconv.FormatUint(uint64(model.ID), 10)+`,"temperature":0.2,"maxTokens":1024,"timeoutSeconds":30}`).Error; err != nil {
		t.Fatalf("configure service match tool runtime: %v", err)
	}

	var staffingReq UpdateSmartStaffingRequest
	staffingReq.Posts.Intake.AgentID = intakeAgent.ID
	staffingReq.Posts.Decision.AgentID = decisionAgent.ID
	staffingReq.Posts.Decision.Mode = "ai_only"
	staffingReq.Posts.SLAAssurance.AgentID = slaAgent.ID
	if err := svc.UpdateSmartStaffingConfig(&staffingReq); err != nil {
		t.Fatalf("update smart staffing config: %v", err)
	}

	var engineReq UpdateEngineSettingsRequest
	engineReq.Runtime.PathBuilder.ModelID = model.ID
	engineReq.Runtime.PathBuilder.Temperature = 0.35
	engineReq.Runtime.PathBuilder.MaxRetries = 4
	engineReq.Runtime.PathBuilder.TimeoutSeconds = 90
	engineReq.Runtime.PathBuilder.SystemPrompt = "getter path prompt"
	engineReq.Runtime.TitleBuilder.ModelID = model.ID
	engineReq.Runtime.TitleBuilder.Temperature = 0.15
	engineReq.Runtime.TitleBuilder.MaxRetries = 2
	engineReq.Runtime.TitleBuilder.TimeoutSeconds = 40
	engineReq.Runtime.TitleBuilder.SystemPrompt = "getter title prompt"
	engineReq.Runtime.HealthChecker.ModelID = model.ID
	engineReq.Runtime.HealthChecker.Temperature = 0.2
	engineReq.Runtime.HealthChecker.MaxRetries = 1
	engineReq.Runtime.HealthChecker.TimeoutSeconds = 55
	engineReq.Runtime.HealthChecker.SystemPrompt = "getter health prompt"
	engineReq.Runtime.Guard.AuditLevel = "summary"
	engineReq.Runtime.Guard.FallbackAssignee = fallback.ID
	if err := svc.UpdateEngineSettingsConfig(&engineReq); err != nil {
		t.Fatalf("update engine settings config: %v", err)
	}

	injector := do.New()
	do.ProvideValue(injector, svc)
	h, err := NewEngineConfigHandler(injector)
	if err != nil {
		t.Fatalf("NewEngineConfigHandler: %v", err)
	}

	perform := func(path string, register func(*gin.Engine)) *httptest.ResponseRecorder {
		t.Helper()
		gin.SetMode(gin.TestMode)
		r := gin.New()
		register(r)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	t.Run("smart staffing getter returns configured posts", func(t *testing.T) {
		rec := perform("/smart-staffing", func(r *gin.Engine) {
			r.GET("/smart-staffing", h.GetSmartStaffing)
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("GetSmartStaffing status=%d body=%s", rec.Code, rec.Body.String())
		}

		var resp handler.R
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode smart staffing response: %v", err)
		}
		if resp.Code != 0 {
			t.Fatalf("unexpected smart staffing response: %+v", resp)
		}
		raw, err := json.Marshal(resp.Data)
		if err != nil {
			t.Fatalf("marshal smart staffing data: %v", err)
		}
		var got SmartStaffingConfig
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode smart staffing payload: %v", err)
		}
		if got.Posts.Intake.AgentID != intakeAgent.ID || got.Posts.Decision.AgentID != decisionAgent.ID || got.Posts.Decision.Mode != "ai_only" || got.Posts.SLAAssurance.AgentID != slaAgent.ID {
			t.Fatalf("unexpected smart staffing payload: %+v", got)
		}
	})

	t.Run("engine settings getter returns configured runtime", func(t *testing.T) {
		rec := perform("/engine-settings", func(r *gin.Engine) {
			r.GET("/engine-settings", h.GetEngineSettings)
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("GetEngineSettings status=%d body=%s", rec.Code, rec.Body.String())
		}

		var resp handler.R
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode engine settings response: %v", err)
		}
		if resp.Code != 0 {
			t.Fatalf("unexpected engine settings response: %+v", resp)
		}
		raw, err := json.Marshal(resp.Data)
		if err != nil {
			t.Fatalf("marshal engine settings data: %v", err)
		}
		var got EngineSettingsConfig
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode engine settings payload: %v", err)
		}
		if got.Runtime.PathBuilder.ModelID != model.ID || got.Runtime.PathBuilder.ProviderID != provider.ID || got.Runtime.PathBuilder.SystemPrompt != "getter path prompt" {
			t.Fatalf("unexpected path builder payload: %+v", got.Runtime.PathBuilder)
		}
		if got.Runtime.TitleBuilder.ModelID != model.ID || got.Runtime.TitleBuilder.SystemPrompt != "getter title prompt" {
			t.Fatalf("unexpected title builder payload: %+v", got.Runtime.TitleBuilder)
		}
		if got.Runtime.HealthChecker.ModelID != model.ID || got.Runtime.HealthChecker.SystemPrompt != "getter health prompt" {
			t.Fatalf("unexpected health checker payload: %+v", got.Runtime.HealthChecker)
		}
		if got.Runtime.Guard.AuditLevel != "summary" || got.Runtime.Guard.FallbackAssignee != fallback.ID {
			t.Fatalf("unexpected guard payload: %+v", got.Runtime.Guard)
		}
	})
}
