package bootstrap

import (
	"encoding/json"
	. "metis/internal/app/itsm/config"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/prompts"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	aiapp "metis/internal/app/ai/runtime"
	orgapp "metis/internal/app/org"
	org "metis/internal/app/org/domain"
	coremodel "metis/internal/model"
)

type noopAdapter struct{}

func (noopAdapter) LoadPolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) SavePolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) AddPolicy(string, string, []string) error                  { return nil }
func (noopAdapter) RemovePolicy(string, string, []string) error               { return nil }
func (noopAdapter) RemoveFilteredPolicy(string, string, int, ...string) error { return nil }

var _ persist.Adapter = (*noopAdapter)(nil)

func newSeedAlignmentDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&ServiceCatalog{}, &ServiceDefinition{}, &ServiceDefinitionVersion{}, &ServiceAction{}, &Priority{}, &SLATemplate{}, &EscalationRule{},
		&org.Department{}, &org.Position{}, &org.DepartmentPosition{}, &org.UserPosition{},
		&coremodel.User{}, &coremodel.Role{}, &coremodel.Menu{}, &coremodel.SystemConfig{},
		&aiapp.Provider{}, &aiapp.AIModel{}, &aiapp.Agent{}, &aiapp.Tool{}, &aiapp.AgentTool{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	for _, name := range []string{
		"general.current_time",
		"system.current_user_profile",
	} {
		tool := aiapp.Tool{
			Toolkit:     "test",
			Name:        name,
			DisplayName: name,
			Description: "test seed tool",
			IsActive:    true,
		}
		if err := db.Where(aiapp.Tool{Name: name}).FirstOrCreate(&tool).Error; err != nil {
			t.Fatalf("seed tool %s: %v", name, err)
		}
	}
	return db
}

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(`[request_definition]
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
		t.Fatalf("create casbin model: %v", err)
	}
	e, err := casbin.NewEnforcer(m, &noopAdapter{})
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	return e
}

func TestBuiltInSmartSeedsAlignParticipantsAndInstallAdminIdentity(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	adminRole := coremodel.Role{Name: "Admin", Code: coremodel.RoleAdmin}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	userRole := coremodel.Role{Name: "User", Code: coremodel.RoleUser}
	if err := db.Create(&userRole).Error; err != nil {
		t.Fatalf("create user role: %v", err)
	}
	admin := coremodel.User{Username: "admin", IsActive: true, RoleID: adminRole.ID}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	var orgApp orgapp.OrgApp
	if err := orgApp.Seed(db, enforcer, true); err != nil {
		t.Fatalf("seed org: %v", err)
	}
	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm: %v", err)
	}

	t.Run("user role can access service desk self-service", func(t *testing.T) {
		for _, tc := range []struct {
			obj string
			act string
		}{
			{obj: "itsm", act: "read"},
			{obj: "itsm:service-desk:use", act: "read"},
			{obj: "itsm:ticket:mine", act: "read"},
			{obj: "itsm:ticket:approval:pending", act: "read"},
			{obj: "itsm:ticket:approval:history", act: "read"},
			{obj: "/api/v1/itsm/service-desk/sessions/:sid/state", act: "GET"},
			{obj: "/api/v1/itsm/service-desk/sessions/:sid/draft/submit", act: "POST"},
			{obj: "/api/v1/itsm/tickets/mine", act: "GET"},
			{obj: "/api/v1/itsm/tickets", act: "POST"},
			{obj: "/api/v1/itsm/tickets/approvals/pending", act: "GET"},
			{obj: "/api/v1/itsm/tickets/approvals/history", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id/timeline", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id/activities", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id/tokens", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id/variables", act: "GET"},
			{obj: "/api/v1/itsm/tickets/:id/progress", act: "POST"},
			{obj: "/api/v1/itsm/tickets/:id/claim", act: "POST"},
		} {
			allowed, err := enforcer.Enforce(coremodel.RoleUser, tc.obj, tc.act)
			if err != nil {
				t.Fatalf("enforce %s %s: %v", tc.obj, tc.act, err)
			}
			if !allowed {
				t.Fatalf("expected user role to access %s %s", tc.act, tc.obj)
			}
		}
	})

	var dept org.Department
	if err := db.Where("code = ?", "it").First(&dept).Error; err != nil {
		t.Fatalf("load it dept: %v", err)
	}
	var pos org.Position
	if err := db.Where("code = ?", "it_admin").First(&pos).Error; err != nil {
		t.Fatalf("load it_admin: %v", err)
	}

	t.Run("org positions include required built-ins", func(t *testing.T) {
		for _, code := range []string{"it_admin", "db_admin", "network_admin", "security_admin", "ops_admin", "serial_reviewer"} {
			var count int64
			if err := db.Model(&org.Position{}).Where("code = ?", code).Count(&count).Error; err != nil {
				t.Fatalf("count position %s: %v", code, err)
			}
			if count != 1 {
				t.Fatalf("expected position %s to exist once, got %d", code, count)
			}
		}
	})

	t.Run("it department allows ops admin", func(t *testing.T) {
		var dept org.Department
		if err := db.Where("code = ?", "it").First(&dept).Error; err != nil {
			t.Fatalf("load it dept: %v", err)
		}
		var pos org.Position
		if err := db.Where("code = ?", "ops_admin").First(&pos).Error; err != nil {
			t.Fatalf("load ops_admin: %v", err)
		}
		var count int64
		if err := db.Model(&org.DepartmentPosition{}).Where("department_id = ? AND position_id = ?", dept.ID, pos.ID).Count(&count).Error; err != nil {
			t.Fatalf("count dept-position: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected ops_admin to be allowed in it, got %d", count)
		}
	})

	t.Run("built-in smart services reference aligned participant codes", func(t *testing.T) {
		var services []ServiceDefinition
		if err := db.Where("engine_type = ?", "smart").Find(&services).Error; err != nil {
			t.Fatalf("load smart services: %v", err)
		}
		wanted := map[string][]string{
			"copilot-account-request": {"IT管理员"},
		}
		workflowWanted := map[string][]string{
			"boss-serial-change-request":      {"subject", "request_category", "prod_change", "risk_level", "rollback_required", "impact_modules", "gateway", "change_items", "permission_level", "read_write"},
			"db-backup-whitelist-action-flow": {"database_name", "source_ip", "whitelist_window", "access_reason", `"type":"action"`, "db_precheck_action", "db_apply_action", "db_backup_whitelist_precheck", "db_backup_whitelist_apply"},
			"prod-server-temporary-access":    {"target_servers", "access_window", "operation_purpose", "access_reason", "ops_admin", "network_admin", "security_admin"},
			"vpn-access-request":              {"vpn_account", "device_usage", "request_kind", "network_admin", "security_admin"},
		}
		for _, svc := range services {
			if needles, ok := wanted[svc.Code]; ok {
				for _, needle := range needles {
					if !strings.Contains(svc.CollaborationSpec, needle) {
						t.Fatalf("service %s missing participant marker %q in collaboration spec", svc.Code, needle)
					}
				}
			}
			if needles, ok := workflowWanted[svc.Code]; ok {
				var actions []ServiceAction
				if err := db.Where("service_id = ?", svc.ID).Find(&actions).Error; err != nil {
					t.Fatalf("load service %s actions: %v", svc.Code, err)
				}
				structured := string(svc.IntakeFormSchema) + string(svc.WorkflowJSON)
				for _, action := range actions {
					structured += action.Code
				}
				for _, needle := range needles {
					if !strings.Contains(structured, needle) {
						t.Fatalf("service %s missing structured marker %q in schema/workflow", svc.Code, needle)
					}
				}
			}
			if strings.Contains(svc.CollaborationSpec, "dba_admin") {
				t.Fatalf("service %s should not reference legacy dba_admin code", svc.Code)
			}
			if svc.Code == "boss-serial-change-request" && strings.Contains(svc.CollaborationSpec, "serial-reviewer") {
				t.Fatalf("service %s should not reference fixed serial-reviewer user", svc.Code)
			}
			if svc.Code == "boss-serial-change-request" {
				if len(svc.WorkflowJSON) == 0 {
					t.Fatalf("service %s should seed a golden workflow json", svc.Code)
				}
				var workflow struct {
					Nodes []struct {
						Type string `json:"type"`
						Data struct {
							Participants []struct {
								Type           string `json:"type"`
								DepartmentCode string `json:"department_code"`
								PositionCode   string `json:"position_code"`
							} `json:"participants"`
							FormSchema json.RawMessage `json:"formSchema"`
						} `json:"data"`
					} `json:"nodes"`
				}
				if err := json.Unmarshal([]byte(svc.WorkflowJSON), &workflow); err != nil {
					t.Fatalf("unmarshal boss workflow: %v", err)
				}
				hasFormSchema := false
				hasSerialReviewer := false
				hasOpsAdmin := false
				for _, node := range workflow.Nodes {
					if node.Type == "form" && len(node.Data.FormSchema) > 0 && string(node.Data.FormSchema) != "null" {
						hasFormSchema = true
					}
					for _, participant := range node.Data.Participants {
						if participant.Type == "position_department" && participant.DepartmentCode == "headquarters" && participant.PositionCode == "serial_reviewer" {
							hasSerialReviewer = true
						}
						if participant.Type == "position_department" && participant.DepartmentCode == "it" && participant.PositionCode == "ops_admin" {
							hasOpsAdmin = true
						}
					}
				}
				if !hasFormSchema || !hasSerialReviewer || !hasOpsAdmin {
					t.Fatalf("service %s seeded workflow should contain form schema and serial participants", svc.Code)
				}
			}
		}
	})

	t.Run("decision agent gets required tool bindings", func(t *testing.T) {
		var agent aiapp.Agent
		if err := db.Where("name = ?", "流程决策智能体").First(&agent).Error; err != nil {
			t.Fatalf("load decision agent: %v", err)
		}

		var tools []aiapp.Tool
		if err := db.Table("ai_tools").
			Joins("JOIN ai_agent_tools ON ai_agent_tools.tool_id = ai_tools.id").
			Where("ai_agent_tools.agent_id = ?", agent.ID).
			Find(&tools).Error; err != nil {
			t.Fatalf("load decision agent tools: %v", err)
		}

		have := map[string]bool{}
		for _, tool := range tools {
			have[tool.Name] = true
		}
		for _, name := range []string{
			"decision.ticket_context",
			"decision.knowledge_search",
			"decision.resolve_participant",
			"decision.user_workload",
			"decision.similar_history",
			"decision.sla_status",
			"decision.list_actions",
			"decision.execute_action",
		} {
			if !have[name] {
				t.Fatalf("expected decision agent to bind tool %s", name)
			}
		}
	})

	t.Run("install admin gets it_admin identity", func(t *testing.T) {
		var count int64
		if err := db.Table("user_positions").Where("user_id = ? AND department_id = ? AND position_id = ? AND is_primary = ?", admin.ID, dept.ID, pos.ID, true).Count(&count).Error; err != nil {
			t.Fatalf("count admin user position: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected admin to have primary it/it_admin identity, got %d", count)
		}
	})

	t.Run("install admin gets all built-in ITSM test identities", func(t *testing.T) {
		expected := []struct {
			DeptCode string
			PosCode  string
			Primary  bool
		}{
			{DeptCode: "it", PosCode: "it_admin", Primary: true},
			{DeptCode: "it", PosCode: "db_admin"},
			{DeptCode: "it", PosCode: "network_admin"},
			{DeptCode: "it", PosCode: "security_admin"},
			{DeptCode: "it", PosCode: "ops_admin"},
			{DeptCode: "headquarters", PosCode: "serial_reviewer"},
		}
		for _, item := range expected {
			var count int64
			if err := db.Table("user_positions AS up").
				Joins("JOIN departments AS d ON d.id = up.department_id").
				Joins("JOIN positions AS p ON p.id = up.position_id").
				Where("up.user_id = ? AND d.code = ? AND p.code = ? AND up.is_primary = ?", admin.ID, item.DeptCode, item.PosCode, item.Primary).
				Count(&count).Error; err != nil {
				t.Fatalf("count admin identity %s/%s: %v", item.DeptCode, item.PosCode, err)
			}
			if count != 1 {
				t.Fatalf("expected admin identity %s/%s primary=%v once, got %d", item.DeptCode, item.PosCode, item.Primary, count)
			}
		}
	})

	t.Run("repeated full seed keeps admin identities idempotent", func(t *testing.T) {
		if err := orgApp.Seed(db, enforcer, true); err != nil {
			t.Fatalf("repeat seed org: %v", err)
		}
		if err := SeedITSM(db, enforcer); err != nil {
			t.Fatalf("repeat seed itsm: %v", err)
		}
		var count int64
		if err := db.Table("user_positions").Where("user_id = ?", admin.ID).Count(&count).Error; err != nil {
			t.Fatalf("count admin identities after repeat seed: %v", err)
		}
		if count != 6 {
			t.Fatalf("expected 6 admin identities after repeat seed, got %d", count)
		}
	})
}

func TestSeedITSM_AllowsNilEnforcerWhileStillSeedingCoreArtifacts(t *testing.T) {
	db := newSeedAlignmentDB(t)

	if err := SeedITSM(db, nil); err != nil {
		t.Fatalf("SeedITSM with nil enforcer: %v", err)
	}

	var itsmRoot coremodel.Menu
	if err := db.Where("permission = ?", "itsm").First(&itsmRoot).Error; err != nil {
		t.Fatalf("find itsm root menu: %v", err)
	}
	if itsmRoot.Type != coremodel.MenuTypeDirectory {
		t.Fatalf("expected itsm root directory, got %+v", itsmRoot)
	}

	var priorityCount int64
	if err := db.Model(&Priority{}).Count(&priorityCount).Error; err != nil {
		t.Fatalf("count priorities: %v", err)
	}
	if priorityCount == 0 {
		t.Fatal("expected priorities to be seeded even without enforcer")
	}

	var serviceCount int64
	if err := db.Model(&ServiceDefinition{}).Count(&serviceCount).Error; err != nil {
		t.Fatalf("count service definitions: %v", err)
	}
	if serviceCount == 0 {
		t.Fatal("expected service definitions to be seeded even without enforcer")
	}
}

func TestSeedITSM_RestoresDriftedPriorityAndSLATemplateDefaults(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	driftedPriority := Priority{
		Name:        "漂移紧急",
		Code:        "P0",
		Value:       99,
		Color:       "#000000",
		Description: "错误优先级",
		IsActive:    false,
	}
	if err := db.Create(&driftedPriority).Error; err != nil {
		t.Fatalf("create drifted priority: %v", err)
	}
	if err := db.Delete(&driftedPriority).Error; err != nil {
		t.Fatalf("soft delete drifted priority: %v", err)
	}

	driftedSLA := SLATemplate{
		Name:              "漂移标准",
		Code:              "standard",
		Description:       "错误模板",
		ResponseMinutes:   999,
		ResolutionMinutes: 9999,
		IsActive:          false,
	}
	if err := db.Create(&driftedSLA).Error; err != nil {
		t.Fatalf("create drifted sla: %v", err)
	}
	if err := db.Delete(&driftedSLA).Error; err != nil {
		t.Fatalf("soft delete drifted sla: %v", err)
	}

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("SeedITSM restore drifted defaults: %v", err)
	}

	var restoredPriority Priority
	if err := db.Unscoped().Where("code = ?", "P0").First(&restoredPriority).Error; err != nil {
		t.Fatalf("load restored priority: %v", err)
	}
	if restoredPriority.DeletedAt.Valid || restoredPriority.Name != "紧急" || restoredPriority.Value != 1 || restoredPriority.Color != "#FF0000" || !restoredPriority.IsActive {
		t.Fatalf("expected restored canonical priority, got %+v", restoredPriority)
	}

	var restoredSLA SLATemplate
	if err := db.Unscoped().Where("code = ?", "standard").First(&restoredSLA).Error; err != nil {
		t.Fatalf("load restored sla template: %v", err)
	}
	if restoredSLA.DeletedAt.Valid || restoredSLA.Name != "标准" || restoredSLA.ResponseMinutes != 240 || restoredSLA.ResolutionMinutes != 1440 || !restoredSLA.IsActive {
		t.Fatalf("expected restored canonical sla template, got %+v", restoredSLA)
	}
}

func TestSeedITSMGrantsUserServiceDeskSessionPolicies(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm: %v", err)
	}

	required := [][]string{
		{coremodel.RoleUser, "/api/v1/itsm/smart-staffing/config", "GET"},
		{coremodel.RoleUser, "/api/v1/ai/sessions", "GET"},
		{coremodel.RoleUser, "/api/v1/ai/sessions", "POST"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid", "GET"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid", "DELETE"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/chat", "POST"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/stream", "GET"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/cancel", "POST"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/images", "POST"},
	}
	for _, policy := range required {
		ok, err := enforcer.HasPolicy(policy)
		if err != nil {
			t.Fatalf("check policy %v: %v", policy, err)
		}
		if !ok {
			t.Fatalf("expected user service desk policy %v", policy)
		}
	}

	forbidden := [][]string{
		{coremodel.RoleUser, "/api/v1/itsm/smart-staffing/config", "PUT"},
		{coremodel.RoleUser, "/api/v1/ai/agents", "GET"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/messages/:mid", "PUT"},
		{coremodel.RoleUser, "/api/v1/ai/sessions/:sid/continue", "POST"},
	}
	for _, policy := range forbidden {
		ok, err := enforcer.HasPolicy(policy)
		if err != nil {
			t.Fatalf("check forbidden policy %v: %v", policy, err)
		}
		if ok {
			t.Fatalf("user should not receive privileged policy %v", policy)
		}
	}
}

func TestSeedITSM_RebuildsLegacySubmissionIndexAndBackfillsRuntimeVersions(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	if err := db.AutoMigrate(&ServiceDeskSubmission{}, &Ticket{}); err != nil {
		t.Fatalf("migrate runtime version prerequisites: %v", err)
	}
	if err := db.Migrator().DropIndex(&ServiceDeskSubmission{}, "idx_itsm_submission_draft"); err != nil {
		t.Fatalf("drop modern draft index: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_itsm_submission_draft ON itsm_service_desk_submissions(session_id, draft_version, fields_hash)`).Error; err != nil {
		t.Fatalf("create legacy draft index: %v", err)
	}

	catalog := ServiceCatalog{Name: "Legacy Root", Code: "legacy-runtime-root", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create custom catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "Legacy Runtime Service",
		Code:              "legacy-runtime-service",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "legacy runtime contract",
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create custom service: %v", err)
	}
	ticket := Ticket{
		Code:        "LEGACY-RUNTIME-1",
		Title:       "legacy runtime ticket",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  1,
		RequesterID: 1,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create legacy ticket: %v", err)
	}

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm: %v", err)
	}

	legacyA := ServiceDeskSubmission{SessionID: 7, DraftVersion: 1, FieldsHash: "same", RequestHash: "r1", Status: "submitted"}
	if err := db.Create(&legacyA).Error; err != nil {
		t.Fatalf("insert first rebuilt submission: %v", err)
	}
	legacyB := ServiceDeskSubmission{SessionID: 7, DraftVersion: 1, FieldsHash: "same", RequestHash: "r2", Status: "submitted"}
	if err := db.Create(&legacyB).Error; err != nil {
		t.Fatalf("expected rebuilt index to allow distinct request_hash, got %v", err)
	}

	var versions []ServiceDefinitionVersion
	if err := db.Where("service_id = ?", service.ID).Find(&versions).Error; err != nil {
		t.Fatalf("list runtime versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one runtime version after seed, got %+v", versions)
	}

	var updated Ticket
	if err := db.First(&updated, ticket.ID).Error; err != nil {
		t.Fatalf("reload legacy ticket: %v", err)
	}
	if updated.ServiceVersionID == nil || *updated.ServiceVersionID != versions[0].ID {
		t.Fatalf("expected legacy ticket backfilled to version %d, got %v", versions[0].ID, updated.ServiceVersionID)
	}

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm second run: %v", err)
	}
	if err := db.Where("service_id = ?", service.ID).Find(&versions).Error; err != nil {
		t.Fatalf("list runtime versions after second seed: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected idempotent runtime version count 1 after second seed, got %+v", versions)
	}
}

func TestSeedITSM_LeavesModernSubmissionIndexAndBoundRuntimeVersionUntouched(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	if err := db.AutoMigrate(&ServiceDeskSubmission{}, &Ticket{}); err != nil {
		t.Fatalf("migrate runtime version prerequisites: %v", err)
	}

	catalog := ServiceCatalog{Name: "Modern Root", Code: "modern-runtime-root", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "Modern Runtime Service",
		Code:              "modern-runtime-service",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "modern runtime contract",
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	version := ServiceDefinitionVersion{
		ServiceID:   service.ID,
		Version:     3,
		ContentHash: "existing-hash",
		EngineType:  "smart",
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create bound runtime version: %v", err)
	}
	ticket := Ticket{
		Code:             "MODERN-RUNTIME-1",
		Title:            "modern runtime ticket",
		ServiceID:        service.ID,
		ServiceVersionID: &version.ID,
		EngineType:       "smart",
		Status:           TicketStatusDecisioning,
		PriorityID:       1,
		RequesterID:      1,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm: %v", err)
	}

	legacy, err := hasLegacyServiceDeskSubmissionIndex(db)
	if err != nil {
		t.Fatalf("hasLegacyServiceDeskSubmissionIndex: %v", err)
	}
	if legacy {
		t.Fatal("expected modern submission index to remain modern")
	}

	var versions []ServiceDefinitionVersion
	if err := db.Where("service_id = ?", service.ID).Order("id ASC").Find(&versions).Error; err != nil {
		t.Fatalf("list service versions: %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("expected runtime versions to remain available after seed")
	}
	foundOriginal := false
	for _, item := range versions {
		if item.ID == version.ID {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Fatalf("expected pre-bound runtime version %d to remain, got %+v", version.ID, versions)
	}

	var updated Ticket
	if err := db.First(&updated, ticket.ID).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if updated.ServiceVersionID == nil || *updated.ServiceVersionID != version.ID {
		t.Fatalf("expected bound ticket to keep runtime version %d, got %v", version.ID, updated.ServiceVersionID)
	}
}

func TestSeedITSM_ReplacesLegacyEngineConfigPolicies(t *testing.T) {
	db := newSeedAlignmentDB(t)
	enforcer := newTestEnforcer(t)

	if _, err := enforcer.AddPolicy("admin", "/api/v1/itsm/engine/config", "GET"); err != nil {
		t.Fatalf("seed legacy GET policy: %v", err)
	}
	if _, err := enforcer.AddPolicy("admin", "/api/v1/itsm/engine/config", "PUT"); err != nil {
		t.Fatalf("seed legacy PUT policy: %v", err)
	}
	if _, err := enforcer.AddPolicy("admin", "itsm:engine:config", "read"); err != nil {
		t.Fatalf("seed legacy menu policy: %v", err)
	}

	if err := SeedITSM(db, enforcer); err != nil {
		t.Fatalf("seed itsm: %v", err)
	}

	for _, policy := range [][]string{
		{"admin", "/api/v1/itsm/engine/config", "GET"},
		{"admin", "/api/v1/itsm/engine/config", "PUT"},
		{"admin", "itsm:engine:config", "read"},
	} {
		ok, err := enforcer.HasPolicy(policy)
		if err != nil {
			t.Fatalf("check removed policy %v: %v", policy, err)
		}
		if ok {
			t.Fatalf("expected legacy engine-config policy removed: %v", policy)
		}
	}

	for _, policy := range [][]string{
		{"admin", "/api/v1/itsm/engine-settings/config", "GET"},
		{"admin", "/api/v1/itsm/engine-settings/config", "PUT"},
		{"admin", "itsm:engine-settings:config", "read"},
	} {
		ok, err := enforcer.HasPolicy(policy)
		if err != nil {
			t.Fatalf("check new policy %v: %v", policy, err)
		}
		if !ok {
			t.Fatalf("expected replacement policy seeded: %v", policy)
		}
	}
}

func TestMigratePriorityCommitmentColumnsDropsLegacyColumns(t *testing.T) {
	db := newSeedAlignmentDB(t)

	for _, column := range []string{"default_response_minutes", "default_resolution_minutes"} {
		if err := db.Exec("ALTER TABLE itsm_priorities ADD COLUMN " + column + " INTEGER DEFAULT 0").Error; err != nil {
			t.Fatalf("add legacy column %s: %v", column, err)
		}
		if !db.Migrator().HasColumn("itsm_priorities", column) {
			t.Fatalf("expected legacy column %s to exist before migration", column)
		}
	}

	migratePriorityCommitmentColumns(db)

	for _, column := range []string{"default_response_minutes", "default_resolution_minutes"} {
		if db.Migrator().HasColumn("itsm_priorities", column) {
			t.Fatalf("expected legacy column %s to be dropped", column)
		}
	}
}

func TestSeedEngineConfigMigratesAndDeletesExistingPathEngineAgent(t *testing.T) {
	db := newSeedAlignmentDB(t)
	code := "itsm.path_builder"
	modelID := uint(42)
	agent := aiapp.Agent{
		Name:         "旧路径引擎",
		Code:         &code,
		Type:         aiapp.AgentTypeInternal,
		Visibility:   aiapp.AgentVisibilityTeam,
		SystemPrompt: "stale prompt",
		Temperature:  0.9,
		ModelID:      &modelID,
		IsActive:     false,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create stale path builder agent: %v", err)
	}

	if err := SeedEngineConfig(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	var modelConfig coremodel.SystemConfig
	if err := db.Where("\"key\" = ?", SmartTicketPathModelKey).First(&modelConfig).Error; err != nil {
		t.Fatalf("load path model config: %v", err)
	}
	if modelConfig.Value != "42" {
		t.Fatalf("expected path model config to be migrated, got %s", modelConfig.Value)
	}
	var tempConfig coremodel.SystemConfig
	if err := db.Where("\"key\" = ?", SmartTicketPathTemperatureKey).First(&tempConfig).Error; err != nil {
		t.Fatalf("load path temperature config: %v", err)
	}
	if tempConfig.Value != "0.9" {
		t.Fatalf("expected path temperature config to be migrated, got %s", tempConfig.Value)
	}
	var count int64
	if err := db.Model(&aiapp.Agent{}).Where("code = ?", code).Count(&count).Error; err != nil {
		t.Fatalf("count legacy path agent: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected path builder agent to be deleted")
	}
}

func TestSeedEngineConfigMigratesLegacySmartTicketConfig(t *testing.T) {
	db := newSeedAlignmentDB(t)
	legacyCode := "itsm.generator"
	legacyAgent := aiapp.Agent{
		Name:         "旧工作流解析",
		Code:         &legacyCode,
		Type:         aiapp.AgentTypeInternal,
		Visibility:   aiapp.AgentVisibilityTeam,
		SystemPrompt: "stale prompt",
		Temperature:  0.9,
		IsActive:     false,
	}
	if err := db.Create(&legacyAgent).Error; err != nil {
		t.Fatalf("create legacy path agent: %v", err)
	}
	legacyConfigs := []coremodel.SystemConfig{
		{Key: "itsm.engine.servicedesk.agent_id", Value: "11"},
		{Key: "itsm.engine.decision.agent_id", Value: "12"},
		{Key: "itsm.engine.sla_assurance.agent_id", Value: "14"},
		{Key: "itsm.engine.decision.decision_mode", Value: "ai_only"},
		{Key: "itsm.engine.general.max_retries", Value: "4"},
		{Key: "itsm.engine.general.timeout_seconds", Value: "90"},
		{Key: "itsm.engine.general.reasoning_log", Value: "summary"},
		{Key: "itsm.engine.general.fallback_assignee", Value: "13"},
	}
	for _, cfg := range legacyConfigs {
		if err := db.Create(&cfg).Error; err != nil {
			t.Fatalf("create legacy config %s: %v", cfg.Key, err)
		}
	}

	if err := SeedEngineConfig(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	expected := map[string]string{
		SmartTicketIntakeAgentKey:       "11",
		SmartTicketDecisionAgentKey:     "12",
		SmartTicketSLAAssuranceAgentKey: "14",
		SmartTicketDecisionModeKey:      "ai_only",
		SmartTicketPathMaxRetriesKey:    "4",
		SmartTicketPathTimeoutKey:       "90",
		SmartTicketGuardAuditLevelKey:   "summary",
		SmartTicketGuardFallbackKey:     "13",
	}
	for key, value := range expected {
		var got coremodel.SystemConfig
		if err := db.Where("\"key\" = ?", key).First(&got).Error; err != nil {
			t.Fatalf("load migrated config %s: %v", key, err)
		}
		if got.Value != value {
			t.Fatalf("expected %s=%s, got %s", key, value, got.Value)
		}
	}
	for _, cfg := range legacyConfigs {
		var count int64
		if err := db.Model(&coremodel.SystemConfig{}).Where("\"key\" = ?", cfg.Key).Count(&count).Error; err != nil {
			t.Fatalf("count legacy config %s: %v", cfg.Key, err)
		}
		if count != 0 {
			t.Fatalf("expected legacy config %s to be deleted", cfg.Key)
		}
	}
	var legacyCount int64
	if err := db.Model(&aiapp.Agent{}).Where("code = ?", legacyCode).Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy path agent: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy path agent code to be removed")
	}
}

func TestSeedEngineConfigCreatesPromptAndTitleBuilderDefaults(t *testing.T) {
	db := newSeedAlignmentDB(t)
	if err := SeedEngineConfig(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	requiredKeys := []string{
		SmartTicketPathSystemPromptKey,
		SmartTicketSessionTitleModelKey,
		SmartTicketSessionTitleTemperatureKey,
		SmartTicketSessionTitleMaxRetriesKey,
		SmartTicketSessionTitleTimeoutKey,
		SmartTicketSessionTitlePromptKey,
		SmartTicketPublishHealthModelKey,
		SmartTicketPublishHealthTemperatureKey,
		SmartTicketPublishHealthMaxRetriesKey,
		SmartTicketPublishHealthTimeoutKey,
		SmartTicketPublishHealthPromptKey,
	}
	for _, key := range requiredKeys {
		var cfg coremodel.SystemConfig
		if err := db.Where("\"key\" = ?", key).First(&cfg).Error; err != nil {
			t.Fatalf("required config %s missing: %v", key, err)
		}
		if strings.TrimSpace(cfg.Value) == "" {
			t.Fatalf("required config %s has empty value", key)
		}
	}
}

func TestSeedEngineConfigSyncsPathBuilderPromptDefault(t *testing.T) {
	db := newSeedAlignmentDB(t)
	if err := db.Create(&coremodel.SystemConfig{
		Key:   SmartTicketPathSystemPromptKey,
		Value: "stale prompt",
	}).Error; err != nil {
		t.Fatalf("create stale path prompt config: %v", err)
	}

	if err := SeedEngineConfig(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	var cfg coremodel.SystemConfig
	if err := db.Where("\"key\" = ?", SmartTicketPathSystemPromptKey).First(&cfg).Error; err != nil {
		t.Fatalf("load path prompt config: %v", err)
	}
	if cfg.Value != prompts.PathBuilderSystemPromptDefault {
		t.Fatalf("expected path prompt default to be synced")
	}
	for _, snippet := range []string{
		"access_reason：访问原因",
		"必须基于 form.access_reason 路由",
		"rejected 出边都不可省略",
	} {
		if !strings.Contains(cfg.Value, snippet) {
			t.Fatalf("synced path prompt missing expected guidance: %s", snippet)
		}
	}
}

func TestSeedEngineConfigSyncsPublishHealthPromptDefault(t *testing.T) {
	db := newSeedAlignmentDB(t)
	if err := db.Create(&coremodel.SystemConfig{
		Key:   SmartTicketPublishHealthPromptKey,
		Value: "stale health prompt",
	}).Error; err != nil {
		t.Fatalf("create stale publish health prompt config: %v", err)
	}

	if err := SeedEngineConfig(db); err != nil {
		t.Fatalf("seed engine config: %v", err)
	}

	var cfg coremodel.SystemConfig
	if err := db.Where("\"key\" = ?", SmartTicketPublishHealthPromptKey).First(&cfg).Error; err != nil {
		t.Fatalf("load publish health prompt config: %v", err)
	}
	if cfg.Value != prompts.PublishHealthSystemPromptDefault {
		t.Fatalf("expected publish health prompt default to be synced")
	}
	for _, snippet := range []string{
		"不要输出 runtime_config 类问题",
		"不要检查审计日志存储",
		"不要因为输入里未出现校验代码、校验逻辑、用户表数据、存储路径或基础设施说明",
	} {
		if !strings.Contains(cfg.Value, snippet) {
			t.Fatalf("synced publish health prompt missing expected guidance: %s", snippet)
		}
	}
}
