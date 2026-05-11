package itsm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	appcore "metis/internal/app"
	aiapp "metis/internal/app/ai/runtime"
	itsmbootstrap "metis/internal/app/itsm/bootstrap"
	"metis/internal/app/itsm/catalog"
	"metis/internal/app/itsm/config"
	"metis/internal/app/itsm/definition"
	itsmdesk "metis/internal/app/itsm/desk"
	"metis/internal/app/itsm/engine"
	"metis/internal/app/itsm/sla"
	"metis/internal/app/itsm/testutil"
	itsmticket "metis/internal/app/itsm/ticket"
	"metis/internal/app/itsm/tools"
	"metis/internal/channel"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/crypto"
	"metis/internal/repository"
	"metis/internal/scheduler"
	"metis/internal/service"
)

type stubChannelDriver struct {
	payload channel.Payload
}

func (d *stubChannelDriver) Send(_ map[string]any, payload channel.Payload) error {
	d.payload = payload
	return nil
}

func (d *stubChannelDriver) Test(map[string]any) error { return nil }

type fakeKnowledgeSearcher struct{}

func (fakeKnowledgeSearcher) SearchKnowledge(kbIDs []uint, query string, limit int) ([]appcore.AIKnowledgeResult, error) {
	return []appcore.AIKnowledgeResult{{
		Title:   query,
		Content: "matched",
		Score:   float64(len(kbIDs) + limit),
	}}, nil
}

type noopAdapter struct{}

func (noopAdapter) LoadPolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) SavePolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) AddPolicy(string, string, []string) error                  { return nil }
func (noopAdapter) RemovePolicy(string, string, []string) error               { return nil }
func (noopAdapter) RemoveFilteredPolicy(string, string, int, ...string) error { return nil }

var _ persist.Adapter = (*noopAdapter)(nil)

func newMessageChannelServiceForITSMAppTest(t *testing.T, driver channel.Driver) (*service.MessageChannelService, *gorm.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	if err := db.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate message channel: %v", err)
	}
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewMessageChannel)
	do.Provide(injector, service.NewMessageChannel)
	svc := do.MustInvoke[*service.MessageChannelService](injector)
	svc.DriverResolver = func(string) (channel.Driver, error) { return driver, nil }
	return svc, db
}

func newUserServiceForITSMAppTest(t *testing.T) (*service.UserService, *database.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewUser)
	return do.MustInvoke[*service.UserService](injector), &database.DB{DB: db}
}

func TestITSMAppMetadataAndRegistryProvider(t *testing.T) {
	injector := do.New()
	registry := &tools.Registry{}
	do.ProvideValue(injector, registry)

	app := &ITSMApp{injector: injector}
	if app.Name() != "itsm" {
		t.Fatalf("Name=%q, want itsm", app.Name())
	}
	if len(app.Models()) != 18 {
		t.Fatalf("Models len=%d, want 18", len(app.Models()))
	}
	if got := app.GetToolRegistry(); got != registry {
		t.Fatalf("GetToolRegistry=%v, want registry", got)
	}
}

func newEngineConfigServiceForITSMAppTest(t *testing.T, db *gorm.DB) *config.EngineConfigService {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, aiapp.NewAgentRepo)
	do.Provide(injector, aiapp.NewAgentService)
	do.Provide(injector, aiapp.NewModelRepo)
	do.Provide(injector, aiapp.NewProviderRepo)
	do.ProvideValue(injector, crypto.EncryptionKey(crypto.DeriveKey("test-secret")))
	do.Provide(injector, aiapp.NewProviderService)
	do.Provide(injector, config.NewEngineConfigService)
	return do.MustInvoke[*config.EngineConfigService](injector)
}

func TestITSMAppBuildRuntimeContextAndTaskMetadata(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := db.AutoMigrate(&aiapp.Agent{}, &aiapp.AgentSession{}); err != nil {
		t.Fatalf("migrate ai tables: %v", err)
	}
	configSvc := newEngineConfigServiceForITSMAppTest(t, db)
	if err := db.Save(&model.SystemConfig{Key: config.SmartTicketIntakeAgentKey, Value: "9"}).Error; err != nil {
		t.Fatalf("save intake config: %v", err)
	}
	agent := aiapp.Agent{BaseModel: model.BaseModel{ID: 9}, Name: "服务台受理岗", Type: aiapp.AgentTypeAssistant, IsActive: true}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create intake agent: %v", err)
	}
	session := aiapp.AgentSession{BaseModel: model.BaseModel{ID: 21}, AgentID: agent.ID, UserID: 7, Status: aiapp.SessionStatusRunning}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	store := tools.NewSessionStateStore(db)
	if err := store.SaveState(session.ID, &tools.ServiceDeskState{
		Stage:                 "awaiting_confirmation",
		CandidateServiceIDs:   []uint{2, 3},
		TopMatchServiceID:     2,
		ConfirmedServiceID:    2,
		ConfirmationRequired:  true,
		LoadedServiceID:       2,
		RequestText:           "申请 VPN",
		DraftSummary:          "VPN 开通申请",
		DraftFormData:         map[string]any{"region": "gz"},
		MissingFields:         []string{"reason"},
		MinDecisionReady:      true,
		ConfirmedDraftVersion: 1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.ProvideValue(injector, configSvc)
	do.ProvideValue(injector, store)
	app := &ITSMApp{injector: injector}

	got, err := app.BuildAgentRuntimeContext(context.Background(), "", session.ID, 7)
	if err != nil {
		t.Fatalf("BuildAgentRuntimeContext: %v", err)
	}
	if !strings.Contains(got, "ITSM Service Desk Runtime Context") || !strings.Contains(got, `"next_expected_action"`) || !strings.Contains(got, `"draft_summary": "VPN 开通申请"`) {
		t.Fatalf("unexpected runtime context: %s", got)
	}

	empty, err := app.BuildAgentRuntimeContext(context.Background(), "", session.ID, 999)
	if err != nil {
		t.Fatalf("BuildAgentRuntimeContext wrong user: %v", err)
	}
	if empty != "" {
		t.Fatalf("expected empty context for wrong user, got %q", empty)
	}

	zeroSession, err := app.BuildAgentRuntimeContext(context.Background(), "", 0, 7)
	if err != nil {
		t.Fatalf("BuildAgentRuntimeContext zero session: %v", err)
	}
	if zeroSession != "" {
		t.Fatalf("expected empty context for zero session, got %q", zeroSession)
	}

	if err := store.SaveState(session.ID, &tools.ServiceDeskState{Stage: "idle"}); err != nil {
		t.Fatalf("save idle state: %v", err)
	}
	idleCtx, err := app.BuildAgentRuntimeContext(context.Background(), "", session.ID, 7)
	if err != nil {
		t.Fatalf("BuildAgentRuntimeContext idle state: %v", err)
	}
	if idleCtx != "" {
		t.Fatalf("expected empty context for idle state, got %q", idleCtx)
	}

	noConfigDB := testutil.NewTestDB(t)
	noConfigInjector := do.New()
	noConfigSvc := newEngineConfigServiceForITSMAppTest(t, noConfigDB)
	do.ProvideValue(noConfigInjector, &database.DB{DB: noConfigDB})
	do.ProvideValue(noConfigInjector, noConfigSvc)
	do.ProvideValue(noConfigInjector, tools.NewSessionStateStore(noConfigDB))
	noConfigApp := &ITSMApp{injector: noConfigInjector}
	noConfigCtx, err := noConfigApp.BuildAgentRuntimeContext(context.Background(), "", session.ID, 7)
	if err != nil {
		t.Fatalf("BuildAgentRuntimeContext without intake config: %v", err)
	}
	if noConfigCtx != "" {
		t.Fatalf("expected empty context without intake config, got %q", noConfigCtx)
	}

	taskInjector := do.New()
	do.ProvideValue(taskInjector, &database.DB{DB: db})
	do.ProvideValue(taskInjector, engine.NewClassicEngine(engine.NewParticipantResolver(nil), nil, nil))
	do.ProvideValue(taskInjector, engine.NewSmartEngine(nil, nil, nil, engine.NewParticipantResolver(nil), nil, nil))
	do.ProvideValue(taskInjector, &config.EngineConfigService{})
	do.ProvideValue(taskInjector, engine.NewParticipantResolver(nil))
	do.ProvideValue(taskInjector, &definition.KnowledgeDocService{})
	taskApp := &ITSMApp{injector: taskInjector}
	tasks := taskApp.Tasks()
	if len(tasks) != 7 {
		t.Fatalf("Tasks len=%d, want 7", len(tasks))
	}
	if tasks[0].Name != "itsm-action-execute" || tasks[5].Name != "itsm-sla-check" || tasks[6].Name != "itsm-smart-recovery" {
		t.Fatalf("unexpected task defs: %+v", tasks)
	}
}

func TestITSMAppProvidersRegistersConstructors(t *testing.T) {
	injector := do.New()
	app := &ITSMApp{}
	app.Providers(injector)
	if app.injector != injector {
		t.Fatalf("expected injector to be stored")
	}
}

func TestITSMAppProvidersResolveEngineAssemblies(t *testing.T) {
	db := testutil.NewTestDB(t)
	userSvc, _ := newUserServiceForITSMAppTest(t)
	messageChannelSvc, _ := newMessageChannelServiceForITSMAppTest(t, &stubChannelDriver{})

	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.ProvideValue(injector, userSvc)
	do.ProvideValue(injector, messageChannelSvc)
	do.ProvideValue(injector, appcore.AIKnowledgeSearcher(fakeKnowledgeSearcher{}))
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, aiapp.NewAgentRepo)
	do.Provide(injector, aiapp.NewAgentService)
	do.Provide(injector, aiapp.NewModelRepo)
	do.Provide(injector, aiapp.NewProviderRepo)
	do.ProvideValue(injector, crypto.EncryptionKey(crypto.DeriveKey("test-secret")))
	do.Provide(injector, aiapp.NewProviderService)

	app := &ITSMApp{}
	app.Providers(injector)

	resolver := do.MustInvoke[*engine.ParticipantResolver](injector)
	if resolver == nil {
		t.Fatal("expected participant resolver to resolve")
	}

	classicEngine := do.MustInvoke[*engine.ClassicEngine](injector)
	if classicEngine == nil {
		t.Fatal("expected classic engine to resolve")
	}

	smartEngine := do.MustInvoke[*engine.SmartEngine](injector)
	if smartEngine == nil {
		t.Fatal("expected smart engine to resolve")
	}
}

func TestITSMAppGenerateSessionTitleAndSeed(t *testing.T) {
	titleInjector := do.New()
	do.ProvideValue(titleInjector, &itsmdesk.SessionTitleService{})
	app := &ITSMApp{injector: titleInjector}
	title, handled, err := app.GenerateSessionTitle(context.Background(), 1, 2, 3, "申请 VPN")
	if err != nil || handled || title != "" {
		t.Fatalf("unexpected GenerateSessionTitle result: title=%q handled=%v err=%v", title, handled, err)
	}

	db := testutil.NewTestDB(t)
	if err := db.AutoMigrate(&model.Role{}); err != nil {
		t.Fatalf("migrate roles: %v", err)
	}
	if err := db.Create(&model.Role{Name: "Admin", Code: model.RoleAdmin}).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := db.Create(&model.Role{Name: "User", Code: model.RoleUser}).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}
	if err := db.Create(&model.User{Username: "admin", IsActive: true, RoleID: 1}).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	modelText, err := casbinmodel.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(modelText, &noopAdapter{})
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	seedApp := &ITSMApp{}
	if err := seedApp.Seed(db, enforcer, true); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	var cfg model.SystemConfig
	if err := db.Where("\"key\" = ?", config.SmartTicketDecisionModeKey).First(&cfg).Error; err != nil {
		t.Fatalf("expected engine config after seed: %v", err)
	}
	if cfg.Value == "" {
		t.Fatalf("expected non-empty decision mode config after seed")
	}
	if err := itsmbootstrap.SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed idempotency: %v", err)
	}
}

func TestITSMAppRoutesRegistersKeyEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	injector := do.New()
	do.ProvideValue(injector, &catalog.CatalogHandler{})
	do.ProvideValue(injector, &definition.ServiceDefHandler{})
	do.ProvideValue(injector, &definition.ServiceActionHandler{})
	do.ProvideValue(injector, &sla.PriorityHandler{})
	do.ProvideValue(injector, &sla.SLATemplateHandler{})
	do.ProvideValue(injector, &sla.EscalationRuleHandler{})
	do.ProvideValue(injector, &itsmticket.TicketHandler{})
	do.ProvideValue(injector, &definition.KnowledgeDocHandler{})
	do.ProvideValue(injector, &config.EngineConfigHandler{})
	do.ProvideValue(injector, &definition.WorkflowGenerateHandler{})
	do.ProvideValue(injector, &itsmticket.VariableHandler{})
	do.ProvideValue(injector, &itsmticket.TokenHandler{})
	do.ProvideValue(injector, &itsmdesk.ServiceDeskHandler{})

	app := &ITSMApp{injector: injector}
	engine := gin.New()
	api := engine.Group("/api")
	app.Routes(api)

	routes := engine.Routes()
	want := map[string]bool{
		"POST /api/itsm/catalogs":                         false,
		"GET /api/itsm/catalogs/tree":                     false,
		"POST /api/itsm/services":                         false,
		"GET /api/itsm/services/:id/health":               false,
		"POST /api/itsm/services/:id/actions":             false,
		"POST /api/itsm/services/:id/knowledge-documents": false,
		"GET /api/itsm/smart-staffing/config":             false,
		"POST /api/itsm/workflows/generate":               false,
		"POST /api/itsm/tickets":                          false,
		"GET /api/itsm/tickets/monitor":                   false,
		"POST /api/itsm/tickets/:id/override/jump":        false,
		"POST /api/itsm/tickets/:id/recovery":             false,
		"PUT /api/itsm/tickets/:id/sla/pause":             false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, seen := range want {
		if !seen {
			t.Fatalf("missing registered route %s", route)
		}
	}
	if len(routes) < len(want) {
		t.Fatalf("registered routes=%d, want at least %d", len(routes), len(want))
	}
}

func TestSchedulerSubmitterPersistsTaskExecutions(t *testing.T) {
	db := testutil.NewTestDB(t)
	submitter := &schedulerSubmitter{db: db}
	payload, _ := json.Marshal(map[string]any{"ticketId": 1})

	if err := submitter.SubmitTask("itsm-smart-progress", payload); err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return submitter.SubmitTaskTx(tx, "itsm-sla-check", payload)
	}); err != nil {
		t.Fatalf("SubmitTaskTx: %v", err)
	}

	var executions []model.TaskExecution
	if err := db.Order("id asc").Find(&executions).Error; err != nil {
		t.Fatalf("list task executions: %v", err)
	}
	if len(executions) != 2 {
		t.Fatalf("expected 2 task executions, got %d", len(executions))
	}
	if executions[0].TaskName != "itsm-smart-progress" || executions[0].Trigger != scheduler.TriggerAPI || executions[0].Status != scheduler.ExecPending {
		t.Fatalf("unexpected first execution: %+v", executions[0])
	}
	if executions[1].TaskName != "itsm-sla-check" {
		t.Fatalf("unexpected second execution: %+v", executions[1])
	}
}

func TestNotificationAdapterValidatesRecipientsAndEmails(t *testing.T) {
	driver := &stubChannelDriver{}
	svc, db := newMessageChannelServiceForITSMAppTest(t, driver)
	if err := db.Create(&model.MessageChannel{Name: "mail", Type: "email", Config: `{}`, Enabled: true}).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 7}, Username: "no-email", IsActive: true}).Error; err != nil {
		t.Fatalf("create user without email: %v", err)
	}
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 8}, Username: "with-email", Email: "ops@example.com", IsActive: true}).Error; err != nil {
		t.Fatalf("create user with email: %v", err)
	}

	adapter := &notificationAdapter{svc: svc, db: db}
	if err := adapter.Send(context.Background(), 1, "subject", "body", nil); err != engine.ErrNotificationNoRecipients {
		t.Fatalf("expected ErrNotificationNoRecipients, got %v", err)
	}
	if err := adapter.Send(context.Background(), 1, "subject", "body", []uint{7}); err != engine.ErrNotificationNoEmail {
		t.Fatalf("expected ErrNotificationNoEmail, got %v", err)
	}
	if err := adapter.Send(context.Background(), 1, "subject", "body", []uint{8}); err != nil {
		t.Fatalf("Send success path: %v", err)
	}
	if len(driver.payload.To) != 1 || driver.payload.To[0] != "ops@example.com" || driver.payload.Subject != "subject" {
		t.Fatalf("unexpected delivered payload: %+v", driver.payload)
	}
}

func TestAIKnowledgeAdapterMapsSearchResults(t *testing.T) {
	adapter := &aiKnowledgeAdapter{searcher: fakeKnowledgeSearcher{}}
	results, err := adapter.Search([]uint{1, 2}, "VPN", 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Title != "VPN" || results[0].Content != "matched" || results[0].Score != 5 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestUserProviderAdapterListsOnlyActiveUsers(t *testing.T) {
	userSvc, wrapped := newUserServiceForITSMAppTest(t)
	if err := wrapped.DB.Create(&model.User{BaseModel: model.BaseModel{ID: 1}, Username: "active-user", IsActive: true}).Error; err != nil {
		t.Fatalf("create active user: %v", err)
	}
	if err := wrapped.DB.Create(&model.User{BaseModel: model.BaseModel{ID: 2}, Username: "inactive-user"}).Error; err != nil {
		t.Fatalf("create inactive user: %v", err)
	}
	if err := wrapped.DB.Model(&model.User{}).Where("id = ?", 2).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable inactive user: %v", err)
	}

	adapter := &userProviderAdapter{userSvc: userSvc}
	users, err := adapter.ListActiveUsers()
	if err != nil {
		t.Fatalf("ListActiveUsers: %v", err)
	}
	if len(users) != 1 || users[0].UserID != 1 || users[0].Name != "active-user" {
		t.Fatalf("unexpected active users: %+v", users)
	}
}
