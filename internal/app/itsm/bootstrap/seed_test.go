package bootstrap

import (
	"encoding/json"
	"metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	itsmtools "metis/internal/app/itsm/tools"
	"strconv"
	"strings"
	"testing"

	"metis/internal/model"
)

func TestSeedCatalogs_CreatesExpectedRootsAndChildren(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}

	var count int64
	if err := db.Model(&ServiceCatalog{}).Count(&count).Error; err != nil {
		t.Fatalf("count catalogs: %v", err)
	}
	if count != 24 {
		t.Fatalf("expected 24 catalogs, got %d", count)
	}

	var roots int64
	if err := db.Model(&ServiceCatalog{}).Where("parent_id IS NULL").Count(&roots).Error; err != nil {
		t.Fatalf("count roots: %v", err)
	}
	if roots != 6 {
		t.Fatalf("expected 6 roots, got %d", roots)
	}
}

func TestSeedPriorities_RestoresSoftDeletedAndRepairsDriftedDefaults(t *testing.T) {
	db := newTestDB(t)

	softDeleted := Priority{
		Name:        "旧紧急",
		Code:        "P0",
		Value:       99,
		Color:       "#000000",
		Description: "旧数据",
		IsActive:    false,
	}
	if err := db.Create(&softDeleted).Error; err != nil {
		t.Fatalf("create soft-deleted priority seed: %v", err)
	}
	if err := db.Delete(&softDeleted).Error; err != nil {
		t.Fatalf("soft delete priority seed: %v", err)
	}

	drifted := Priority{
		Name:        "漂移中优先级",
		Code:        "P2",
		Value:       33,
		Color:       "#123456",
		Description: "错误描述",
		IsActive:    false,
	}
	if err := db.Create(&drifted).Error; err != nil {
		t.Fatalf("create drifted priority seed: %v", err)
	}

	if err := seedPriorities(db); err != nil {
		t.Fatalf("seed priorities: %v", err)
	}

	var p0 Priority
	if err := db.Unscoped().Where("code = ?", "P0").First(&p0).Error; err != nil {
		t.Fatalf("load restored P0: %v", err)
	}
	if p0.DeletedAt.Valid || p0.Name != "紧急" || p0.Value != 1 || p0.Color != "#FF0000" || !p0.IsActive {
		t.Fatalf("expected restored canonical P0, got %+v", p0)
	}

	var p2 Priority
	if err := db.Where("code = ?", "P2").First(&p2).Error; err != nil {
		t.Fatalf("load repaired P2: %v", err)
	}
	if p2.Name != "中" || p2.Value != 3 || p2.Color != "#FFAA00" || p2.Description != "中等优先级" || !p2.IsActive {
		t.Fatalf("expected repaired canonical P2, got %+v", p2)
	}
}

func TestSeedSLATemplates_RestoresSoftDeletedAndRepairsDriftedDefaults(t *testing.T) {
	db := newTestDB(t)

	softDeleted := SLATemplate{
		Name:              "旧标准",
		Code:              "standard",
		Description:       "旧模板",
		ResponseMinutes:   999,
		ResolutionMinutes: 9999,
		IsActive:          false,
	}
	if err := db.Create(&softDeleted).Error; err != nil {
		t.Fatalf("create soft-deleted sla seed: %v", err)
	}
	if err := db.Delete(&softDeleted).Error; err != nil {
		t.Fatalf("soft delete sla seed: %v", err)
	}

	drifted := SLATemplate{
		Name:              "漂移模板",
		Code:              "urgent",
		Description:       "错误配置",
		ResponseMinutes:   123,
		ResolutionMinutes: 456,
		IsActive:          false,
	}
	if err := db.Create(&drifted).Error; err != nil {
		t.Fatalf("create drifted sla seed: %v", err)
	}

	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}

	var standard SLATemplate
	if err := db.Unscoped().Where("code = ?", "standard").First(&standard).Error; err != nil {
		t.Fatalf("load restored standard sla: %v", err)
	}
	if standard.DeletedAt.Valid || standard.Name != "标准" || standard.ResponseMinutes != 240 || standard.ResolutionMinutes != 1440 || !standard.IsActive {
		t.Fatalf("expected restored canonical standard SLA, got %+v", standard)
	}

	var urgent SLATemplate
	if err := db.Where("code = ?", "urgent").First(&urgent).Error; err != nil {
		t.Fatalf("load repaired urgent sla: %v", err)
	}
	if urgent.Name != "紧急" || urgent.ResponseMinutes != 30 || urgent.ResolutionMinutes != 240 || urgent.Description != "紧急 SLA，响应 30 分钟，解决 4 小时" || !urgent.IsActive {
		t.Fatalf("expected repaired canonical urgent SLA, got %+v", urgent)
	}
}

func TestSeedPriorityAndSLATemplateSeeds_CreateMissingAndStayIdempotent(t *testing.T) {
	db := newTestDB(t)

	if err := seedPriorities(db); err != nil {
		t.Fatalf("seed priorities first run: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed sla templates first run: %v", err)
	}
	if err := seedPriorities(db); err != nil {
		t.Fatalf("seed priorities second run: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed sla templates second run: %v", err)
	}

	var priorityCount int64
	if err := db.Model(&Priority{}).Count(&priorityCount).Error; err != nil {
		t.Fatalf("count priorities: %v", err)
	}
	if priorityCount != 5 {
		t.Fatalf("expected 5 seeded priorities, got %d", priorityCount)
	}

	var slaCount int64
	if err := db.Model(&SLATemplate{}).Count(&slaCount).Error; err != nil {
		t.Fatalf("count sla templates: %v", err)
	}
	if slaCount != 5 {
		t.Fatalf("expected 5 seeded sla templates, got %d", slaCount)
	}

	var p3 Priority
	if err := db.Where("code = ?", "P3").First(&p3).Error; err != nil {
		t.Fatalf("load P3: %v", err)
	}
	if p3.Name != "低" || p3.Value != 4 || p3.Color != "#00AA00" || !p3.IsActive {
		t.Fatalf("unexpected seeded P3: %+v", p3)
	}

	var infraChange SLATemplate
	if err := db.Where("code = ?", "infra-change").First(&infraChange).Error; err != nil {
		t.Fatalf("load infra-change SLA: %v", err)
	}
	if infraChange.Name != "基础设施变更" || infraChange.ResponseMinutes != 60 || infraChange.ResolutionMinutes != 480 || !infraChange.IsActive {
		t.Fatalf("unexpected seeded infra-change SLA: %+v", infraChange)
	}
}

func TestMigrateServiceRuntimeVersions_BackfillsServicesAndLegacyTickets(t *testing.T) {
	db := newTestDB(t)
	catalog := ServiceCatalog{Name: "Root", Code: "runtime-root", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "Runtime Service",
		Code:              "runtime-service",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "initial spec",
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	ticket := Ticket{
		Code:        "TICK-LEGACY-RUNTIME",
		Title:       "legacy ticket",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  1,
		RequesterID: 1,
	}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create legacy ticket: %v", err)
	}

	if err := migrateServiceRuntimeVersions(db); err != nil {
		t.Fatalf("migrate service runtime versions: %v", err)
	}
	if err := migrateServiceRuntimeVersions(db); err != nil {
		t.Fatalf("migrate service runtime versions second run: %v", err)
	}

	var versions []ServiceDefinitionVersion
	if err := db.Where("service_id = ?", service.ID).Find(&versions).Error; err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one idempotent runtime version, got %+v", versions)
	}
	var updated Ticket
	if err := db.First(&updated, ticket.ID).Error; err != nil {
		t.Fatalf("load ticket: %v", err)
	}
	if updated.ServiceVersionID == nil || *updated.ServiceVersionID != versions[0].ID {
		t.Fatalf("expected legacy ticket backfilled with version %d, got %v", versions[0].ID, updated.ServiceVersionID)
	}
}

func TestMigrateServiceRuntimeVersions_ReusesExistingVersionAndPreservesBoundTickets(t *testing.T) {
	db := newTestDB(t)

	catalog := ServiceCatalog{Name: "Runtime Root", Code: "runtime-root-preserve", IsActive: true}
	if err := db.Create(&catalog).Error; err != nil {
		t.Fatalf("create catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "Runtime Service Preserve",
		Code:              "runtime-service-preserve",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "spec",
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	existingVersion, err := definition.GetOrCreateServiceRuntimeVersion(db, service.ID)
	if err != nil {
		t.Fatalf("create existing runtime version: %v", err)
	}

	legacyTicket := Ticket{
		Code:        "TICK-RUNTIME-BACKFILL",
		Title:       "legacy runtime backfill",
		ServiceID:   service.ID,
		EngineType:  "smart",
		Status:      TicketStatusDecisioning,
		PriorityID:  1,
		RequesterID: 1,
	}
	if err := db.Create(&legacyTicket).Error; err != nil {
		t.Fatalf("create legacy ticket: %v", err)
	}

	alreadyBoundID := uint(999)
	boundTicket := Ticket{
		Code:             "TICK-RUNTIME-BOUND",
		Title:            "bound runtime ticket",
		ServiceID:        service.ID,
		ServiceVersionID: &alreadyBoundID,
		EngineType:       "smart",
		Status:           TicketStatusDecisioning,
		PriorityID:       1,
		RequesterID:      1,
	}
	if err := db.Create(&boundTicket).Error; err != nil {
		t.Fatalf("create bound ticket: %v", err)
	}

	if err := migrateServiceRuntimeVersions(db); err != nil {
		t.Fatalf("migrateServiceRuntimeVersions: %v", err)
	}

	var versions []ServiceDefinitionVersion
	if err := db.Where("service_id = ?", service.ID).Order("id ASC").Find(&versions).Error; err != nil {
		t.Fatalf("list service versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected existing runtime version to be reused, got %+v", versions)
	}
	if versions[0].ID != existingVersion.ID {
		t.Fatalf("expected runtime version %d to be reused, got %d", existingVersion.ID, versions[0].ID)
	}

	var refreshedLegacy Ticket
	if err := db.First(&refreshedLegacy, legacyTicket.ID).Error; err != nil {
		t.Fatalf("reload legacy ticket: %v", err)
	}
	if refreshedLegacy.ServiceVersionID == nil || *refreshedLegacy.ServiceVersionID != existingVersion.ID {
		t.Fatalf("expected legacy ticket backfilled to existing version %d, got %v", existingVersion.ID, refreshedLegacy.ServiceVersionID)
	}

	var refreshedBound Ticket
	if err := db.First(&refreshedBound, boundTicket.ID).Error; err != nil {
		t.Fatalf("reload bound ticket: %v", err)
	}
	if refreshedBound.ServiceVersionID == nil || *refreshedBound.ServiceVersionID != alreadyBoundID {
		t.Fatalf("expected existing service_version_id %d to remain untouched, got %v", alreadyBoundID, refreshedBound.ServiceVersionID)
	}
}

func TestSeedServiceDefinitions_ServerAccessUsesNaturalSpecAndPreservesStructuredContract(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}
	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var service ServiceDefinition
	if err := db.Where("code = ?", "prod-server-temporary-access").First(&service).Error; err != nil {
		t.Fatalf("find server access service: %v", err)
	}

	for _, forbidden := range []string{
		"target_servers",
		"access_window",
		"operation_purpose",
		"access_reason",
		"form.access_reason",
		"position_department",
		"department_code",
		"position_code",
		"ops_admin",
		"network_admin",
		"security_admin",
	} {
		if strings.Contains(service.CollaborationSpec, forbidden) {
			t.Fatalf("server access collaboration spec should be natural text, found machine token %q in %q", forbidden, service.CollaborationSpec)
		}
	}

	var schema struct {
		Fields []struct {
			Key  string `json:"key"`
			Type string `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(service.IntakeFormSchema), &schema); err != nil {
		t.Fatalf("unmarshal intake form schema: %v", err)
	}
	got := make([]string, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		got = append(got, field.Key)
		if field.Key == "access_reason" && field.Type != "textarea" {
			t.Fatalf("expected access_reason to remain free text textarea, got %q", field.Type)
		}
	}
	want := []string{"target_servers", "access_window", "operation_purpose", "access_reason"}
	if len(got) != len(want) {
		t.Fatalf("expected field keys %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected field keys %v, got %v", want, got)
		}
	}

	operator := itsmtools.NewOperator(db, nil, nil, nil, nil, nil)
	detail, err := operator.LoadService(service.ID)
	if err != nil {
		t.Fatalf("load service through operator: %v", err)
	}
	if len(service.WorkflowJSON) == 0 {
		t.Fatal("expected seeded server access workflow json")
	}
	if detail.FormSchema == nil {
		t.Fatal("expected operator detail to include form schema")
	}
	if len(detail.FormFields) != len(want) {
		t.Fatalf("expected %d operator form fields, got %d", len(want), len(detail.FormFields))
	}
	for i, key := range want {
		if detail.FormFields[i].Key != key {
			t.Fatalf("expected operator field keys %v, got %+v", want, detail.FormFields)
		}
	}
	if detail.RoutingFieldHint != nil {
		t.Fatalf("expected textarea routing field to be ignored, got %+v", detail.RoutingFieldHint)
	}

	workflow := string(service.WorkflowJSON)
	for _, required := range []string{"form.access_reason", "it", "ops_admin", "network_admin", "security_admin"} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("expected server access workflow to preserve structured token %q, got %s", required, workflow)
		}
	}
}

func TestSeedServiceDefinitions_DBBackupUsesNaturalSpecAndPreservesStructuredContract(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}
	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var service ServiceDefinition
	if err := db.Where("code = ?", "db-backup-whitelist-action-flow").First(&service).Error; err != nil {
		t.Fatalf("find db backup service: %v", err)
	}

	for _, forbidden := range []string{
		"database_name",
		"source_ip",
		"whitelist_window",
		"access_reason",
		"position_department",
		"department_code",
		"position_code",
		"db_admin",
		"decision.execute_action",
		"db_backup_whitelist_precheck",
		"db_backup_whitelist_apply",
		"backup_whitelist_precheck",
		"backup_whitelist_apply",
	} {
		if strings.Contains(service.CollaborationSpec, forbidden) {
			t.Fatalf("db backup collaboration spec should be natural text, found machine token %q in %q", forbidden, service.CollaborationSpec)
		}
	}

	var schema struct {
		Fields []struct {
			Key  string `json:"key"`
			Type string `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(service.IntakeFormSchema), &schema); err != nil {
		t.Fatalf("unmarshal db backup intake form schema: %v", err)
	}
	want := []struct {
		key string
		typ string
	}{
		{"database_name", "text"},
		{"source_ip", "text"},
		{"whitelist_window", "text"},
		{"access_reason", "textarea"},
	}
	if len(schema.Fields) != len(want) {
		t.Fatalf("expected field keys %v, got %+v", want, schema.Fields)
	}
	for i, field := range schema.Fields {
		if field.Key != want[i].key || field.Type != want[i].typ {
			t.Fatalf("expected field %d to be %s/%s, got %s/%s", i, want[i].key, want[i].typ, field.Key, field.Type)
		}
	}

	var actions []ServiceAction
	if err := db.Where("service_id = ?", service.ID).Order("code ASC").Find(&actions).Error; err != nil {
		t.Fatalf("load db backup actions: %v", err)
	}
	actionIDsByCode := map[string]uint{}
	for _, action := range actions {
		actionIDsByCode[action.Code] = action.ID
	}
	for _, code := range []string{"db_backup_whitelist_precheck", "db_backup_whitelist_apply"} {
		if actionIDsByCode[code] == 0 {
			t.Fatalf("expected seeded db backup action %q, got %#v", code, actionIDsByCode)
		}
	}

	var workflow struct {
		Nodes []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Label    string `json:"label"`
				ActionID uint   `json:"action_id"`
			} `json:"data"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Data   struct {
				Outcome string `json:"outcome"`
			} `json:"data"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(service.WorkflowJSON), &workflow); err != nil {
		t.Fatalf("unmarshal db backup workflow json: %v", err)
	}
	actionNodeIDs := map[uint]string{}
	for _, node := range workflow.Nodes {
		if node.Type == "action" {
			actionNodeIDs[node.Data.ActionID] = node.ID
			if node.Data.ActionID == actionIDsByCode["db_backup_whitelist_precheck"] && !strings.Contains(node.Data.Label, "预检") {
				t.Fatalf("expected precheck action node label to mention precheck, got %q", node.Data.Label)
			}
			if node.Data.ActionID == actionIDsByCode["db_backup_whitelist_apply"] && !strings.Contains(node.Data.Label, "放行") {
				t.Fatalf("expected apply action node label to mention release, got %q", node.Data.Label)
			}
		}
	}
	applyNodeID := actionNodeIDs[actionIDsByCode["db_backup_whitelist_apply"]]
	if actionNodeIDs[actionIDsByCode["db_backup_whitelist_precheck"]] == "" || applyNodeID == "" {
		t.Fatalf("expected workflow action nodes bound to real action ids, got %#v workflow=%s", actionNodeIDs, service.WorkflowJSON)
	}
	for _, edge := range workflow.Edges {
		if edge.Source == "db_process" && edge.Data.Outcome == "rejected" && edge.Target == applyNodeID {
			t.Fatalf("db backup rejected edge must not pass through apply action: %s", service.WorkflowJSON)
		}
	}
}

func TestSeedServiceDefinitions_BossUsesNaturalSpecAndPreservesStructuredContract(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}
	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var service ServiceDefinition
	if err := db.Where("code = ?", "boss-serial-change-request").First(&service).Error; err != nil {
		t.Fatalf("find boss service: %v", err)
	}

	for _, forbidden := range []string{
		"subject",
		"request_category",
		"prod_change",
		"risk_level",
		"rollback_required",
		"impact_modules",
		"gateway",
		"change_items",
		"position_department",
		"department_code",
		"position_code",
		"headquarters",
		"serial_reviewer",
		"ops_admin",
	} {
		if strings.Contains(service.CollaborationSpec, forbidden) {
			t.Fatalf("boss collaboration spec should be natural text, found machine token %q in %q", forbidden, service.CollaborationSpec)
		}
	}
	if strings.Contains(service.CollaborationSpec, "\n\n") {
		t.Fatalf("boss collaboration spec should use single line breaks, got %q", service.CollaborationSpec)
	}

	var schema struct {
		Fields []struct {
			Key     string `json:"key"`
			Type    string `json:"type"`
			Options []struct {
				Value string `json:"value"`
			} `json:"options"`
			Props struct {
				Columns []struct {
					Key     string `json:"key"`
					Type    string `json:"type"`
					Options []struct {
						Value string `json:"value"`
					} `json:"options"`
				} `json:"columns"`
			} `json:"props"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(service.IntakeFormSchema), &schema); err != nil {
		t.Fatalf("unmarshal boss intake form schema: %v", err)
	}

	wantFields := []struct {
		key string
		typ string
	}{
		{"subject", "text"},
		{"request_category", "select"},
		{"risk_level", "radio"},
		{"expected_finish_time", "datetime"},
		{"change_window", "date_range"},
		{"impact_scope", "textarea"},
		{"rollback_required", "select"},
		{"impact_modules", "multi_select"},
		{"change_items", "table"},
	}
	if len(schema.Fields) != len(wantFields) {
		t.Fatalf("expected boss field keys %v, got %+v", wantFields, schema.Fields)
	}
	fieldByKey := map[string]int{}
	for i, field := range schema.Fields {
		if field.Key != wantFields[i].key || field.Type != wantFields[i].typ {
			t.Fatalf("expected field %d to be %s/%s, got %s/%s", i, wantFields[i].key, wantFields[i].typ, field.Key, field.Type)
		}
		fieldByKey[field.Key] = i
	}
	assertOptionValues := func(key string, want []string) {
		t.Helper()
		field := schema.Fields[fieldByKey[key]]
		got := make([]string, 0, len(field.Options))
		for _, opt := range field.Options {
			got = append(got, opt.Value)
		}
		if len(got) != len(want) {
			t.Fatalf("expected %s options %v, got %v", key, want, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %s options %v, got %v", key, want, got)
			}
		}
	}
	assertOptionValues("request_category", []string{"prod_change", "access_grant", "emergency_support"})
	assertOptionValues("risk_level", []string{"low", "medium", "high"})
	assertOptionValues("rollback_required", []string{"required", "not_required"})
	assertOptionValues("impact_modules", []string{"gateway", "payment", "monitoring", "order"})

	changeItems := schema.Fields[fieldByKey["change_items"]]
	wantColumns := []string{"system", "resource", "permission_level", "effective_range", "reason"}
	if len(changeItems.Props.Columns) != len(wantColumns) {
		t.Fatalf("expected change_items columns %v, got %+v", wantColumns, changeItems.Props.Columns)
	}
	for i, column := range changeItems.Props.Columns {
		if column.Key != wantColumns[i] {
			t.Fatalf("expected change_items columns %v, got %+v", wantColumns, changeItems.Props.Columns)
		}
		if column.Key == "permission_level" {
			got := make([]string, 0, len(column.Options))
			for _, opt := range column.Options {
				got = append(got, opt.Value)
			}
			want := []string{"read", "read_write"}
			if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("expected permission_level options %v, got %v", want, got)
			}
		}
	}
}

func TestSeedServiceDefinitions_DBBackupMigratesLegacyActionCodes(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}

	var catalog ServiceCatalog
	if err := db.Where("code = ?", "application-platform:database").First(&catalog).Error; err != nil {
		t.Fatalf("find catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "生产数据库备份白名单临时放行申请",
		Code:              "db-backup-whitelist-action-flow",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "旧协作规范",
		IsActive:          true,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create legacy service: %v", err)
	}
	legacyConfig := JSONField(`{"url":"/custom-precheck","method":"POST"}`)
	if err := db.Create(&ServiceAction{
		Name:       "旧预检",
		Code:       "backup_whitelist_precheck",
		ActionType: "http",
		ConfigJSON: legacyConfig,
		ServiceID:  service.ID,
		IsActive:   true,
	}).Error; err != nil {
		t.Fatalf("create legacy action: %v", err)
	}

	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var migrated ServiceAction
	if err := db.Where("service_id = ? AND code = ?", service.ID, "db_backup_whitelist_precheck").First(&migrated).Error; err != nil {
		t.Fatalf("expected legacy precheck action to migrate to canonical code: %v", err)
	}
	if string(migrated.ConfigJSON) != string(legacyConfig) {
		t.Fatalf("expected migration to preserve action config, got %s", migrated.ConfigJSON)
	}
	var legacyCount int64
	if err := db.Model(&ServiceAction{}).Where("service_id = ? AND code = ?", service.ID, "backup_whitelist_precheck").Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy action: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy action code to be migrated, still found %d", legacyCount)
	}
}

func TestSeedServiceDefinitions_RestoresSoftDeletedBuiltinServiceInPlace(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}

	var catalog ServiceCatalog
	if err := db.Where("code = ?", "infra-network:network").First(&catalog).Error; err != nil {
		t.Fatalf("find catalog: %v", err)
	}
	var sla SLATemplate
	if err := db.Where("code = ?", "urgent").First(&sla).Error; err != nil {
		t.Fatalf("find drift sla: %v", err)
	}

	service := ServiceDefinition{
		Name:              "旧 VPN 服务",
		Code:              "vpn-access-request",
		Description:       "旧描述",
		CatalogID:         catalog.ID,
		EngineType:        "classic",
		SLAID:             &sla.ID,
		CollaborationSpec: "旧协作规范",
		IsActive:          false,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create drifted service: %v", err)
	}
	originalID := service.ID
	if err := db.Delete(&service).Error; err != nil {
		t.Fatalf("soft delete service: %v", err)
	}

	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var restored ServiceDefinition
	if err := db.Unscoped().Where("code = ?", "vpn-access-request").First(&restored).Error; err != nil {
		t.Fatalf("reload restored service: %v", err)
	}
	if restored.ID != originalID {
		t.Fatalf("expected restored service to reuse id %d, got %d", originalID, restored.ID)
	}
	if restored.DeletedAt.Valid || !restored.IsActive {
		t.Fatalf("expected restored service to be active and undeleted, got %+v", restored)
	}
	if restored.Name != "VPN 开通申请" || restored.EngineType != "smart" {
		t.Fatalf("expected canonical service identity, got %+v", restored)
	}
	if !strings.Contains(restored.CollaborationSpec, "访问原因包括线上支持") {
		t.Fatalf("expected canonical collaboration spec, got %q", restored.CollaborationSpec)
	}
}

func TestSeedServiceDefinitions_RestoresAndRepairsCanonicalActionsInPlace(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}

	var catalog ServiceCatalog
	if err := db.Where("code = ?", "application-platform:database").First(&catalog).Error; err != nil {
		t.Fatalf("find catalog: %v", err)
	}
	service := ServiceDefinition{
		Name:              "生产数据库备份白名单临时放行申请",
		Code:              "db-backup-whitelist-action-flow",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		CollaborationSpec: "漂移协作规范",
		IsActive:          false,
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	driftedApply := ServiceAction{
		Name:       "错误放行",
		Code:       "db_backup_whitelist_apply",
		Description:"错误描述",
		ActionType: "script",
		ConfigJSON: JSONField(`{"url":"/wrong","method":"GET"}`),
		ServiceID:  service.ID,
		IsActive:   false,
	}
	if err := db.Create(&driftedApply).Error; err != nil {
		t.Fatalf("create drifted apply action: %v", err)
	}

	softDeletedPrecheck := ServiceAction{
		Name:       "错误预检",
		Code:       "db_backup_whitelist_precheck",
		Description:"错误预检描述",
		ActionType: "script",
		ConfigJSON: JSONField(`{"url":"/stale","method":"GET"}`),
		ServiceID:  service.ID,
		IsActive:   false,
	}
	if err := db.Create(&softDeletedPrecheck).Error; err != nil {
		t.Fatalf("create soft-deleted precheck action: %v", err)
	}
	precheckID := softDeletedPrecheck.ID
	if err := db.Delete(&softDeletedPrecheck).Error; err != nil {
		t.Fatalf("soft delete precheck action: %v", err)
	}

	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var precheck ServiceAction
	if err := db.Unscoped().Where("service_id = ? AND code = ?", service.ID, "db_backup_whitelist_precheck").First(&precheck).Error; err != nil {
		t.Fatalf("reload precheck action: %v", err)
	}
	if precheck.ID != precheckID || precheck.DeletedAt.Valid || !precheck.IsActive {
		t.Fatalf("expected precheck action restored in place, got %+v", precheck)
	}
	if precheck.ActionType != "http" || !strings.Contains(string(precheck.ConfigJSON), "/precheck") {
		t.Fatalf("expected canonical precheck config, got %+v", precheck)
	}

	var apply ServiceAction
	if err := db.Where("service_id = ? AND code = ?", service.ID, "db_backup_whitelist_apply").First(&apply).Error; err != nil {
		t.Fatalf("reload apply action: %v", err)
	}
	if !apply.IsActive || apply.ActionType != "http" || apply.Name != "执行备份白名单放行" || !strings.Contains(string(apply.ConfigJSON), "/apply") {
		t.Fatalf("expected apply action repaired to canonical seed, got %+v", apply)
	}

	var count int64
	if err := db.Unscoped().Model(&ServiceAction{}).Where("service_id = ? AND code IN ?", service.ID, []string{"db_backup_whitelist_precheck", "db_backup_whitelist_apply"}).Count(&count).Error; err != nil {
		t.Fatalf("count canonical actions: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected exactly two canonical actions, got %d", count)
	}

	var updated ServiceDefinition
	if err := db.First(&updated, service.ID).Error; err != nil {
		t.Fatalf("reload service: %v", err)
	}
	workflow := string(updated.WorkflowJSON)
	if !strings.Contains(workflow, `"action_id":`+strconv.FormatUint(uint64(precheck.ID), 10)) || !strings.Contains(workflow, `"action_id":`+strconv.FormatUint(uint64(apply.ID), 10)) {
		t.Fatalf("expected workflow to bind restored canonical action ids, got %s", workflow)
	}
}

func TestSeedDBBackupWhitelistWorkflow_SkipsWhenActionsAreIncomplete(t *testing.T) {
	db := newTestDB(t)

	service := ServiceDefinition{
		Name:       "DB Backup Workflow",
		Code:       "db-backup-seed-skip",
		EngineType: "smart",
		IsActive:   true,
		WorkflowJSON: JSONField(`{"nodes":[{"id":"legacy"}],"edges":[]}`),
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}
	if err := db.Create(&ServiceAction{
		Name:       "仅有预检",
		Code:       "db_backup_whitelist_precheck",
		ActionType: "http",
		ServiceID:  service.ID,
		IsActive:   true,
	}).Error; err != nil {
		t.Fatalf("create precheck action: %v", err)
	}

	seedDBBackupWhitelistWorkflow(db, service.ID)

	var updated ServiceDefinition
	if err := db.First(&updated, service.ID).Error; err != nil {
		t.Fatalf("reload service: %v", err)
	}
	if string(updated.WorkflowJSON) != `{"nodes":[{"id":"legacy"}],"edges":[]}` {
		t.Fatalf("expected workflow json to remain unchanged when actions missing, got %s", updated.WorkflowJSON)
	}
}

func TestSeedDBBackupWhitelistWorkflow_UsesActiveCanonicalActionIDs(t *testing.T) {
	db := newTestDB(t)

	service := ServiceDefinition{
		Name:       "DB Backup Workflow",
		Code:       "db-backup-seed-apply",
		EngineType: "smart",
		IsActive:   true,
		WorkflowJSON: JSONField(`{"nodes":[{"id":"legacy"}],"edges":[]}`),
	}
	if err := db.Create(&service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	precheck := ServiceAction{
		Name:       "预检",
		Code:       "db_backup_whitelist_precheck",
		ActionType: "http",
		ServiceID:  service.ID,
		IsActive:   true,
	}
	if err := db.Create(&precheck).Error; err != nil {
		t.Fatalf("create precheck action: %v", err)
	}
	apply := ServiceAction{
		Name:       "放行",
		Code:       "db_backup_whitelist_apply",
		ActionType: "http",
		ServiceID:  service.ID,
		IsActive:   true,
	}
	if err := db.Create(&apply).Error; err != nil {
		t.Fatalf("create apply action: %v", err)
	}
	legacyAlias := ServiceAction{
		Name:       "旧放行别名",
		Code:       "backup_whitelist_apply",
		ActionType: "http",
		ServiceID:  service.ID,
		IsActive:   true,
	}
	if err := db.Create(&legacyAlias).Error; err != nil {
		t.Fatalf("create legacy alias action: %v", err)
	}

	seedDBBackupWhitelistWorkflow(db, service.ID)

	var updated ServiceDefinition
	if err := db.First(&updated, service.ID).Error; err != nil {
		t.Fatalf("reload service: %v", err)
	}
	workflow := string(updated.WorkflowJSON)
	if !strings.Contains(workflow, `"db_precheck_action"`) || !strings.Contains(workflow, `"db_apply_action"`) {
		t.Fatalf("expected canonical db backup action nodes, got %s", workflow)
	}
	if !strings.Contains(workflow, `"action_id":`+strconv.FormatUint(uint64(precheck.ID), 10)) {
		t.Fatalf("expected workflow to include precheck action id %d, got %s", precheck.ID, workflow)
	}
	if !strings.Contains(workflow, `"action_id":`+strconv.FormatUint(uint64(apply.ID), 10)) {
		t.Fatalf("expected workflow to include apply action id %d, got %s", apply.ID, workflow)
	}
	if strings.Contains(workflow, `"action_id":`+strconv.FormatUint(uint64(legacyAlias.ID), 10)) {
		t.Fatalf("expected legacy alias action id %d to be ignored, got %s", legacyAlias.ID, workflow)
	}
}

func TestSeedServiceDefinitions_VPNUsesNaturalSpecAndPreservesStructuredContract(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}
	if err := seedSLATemplates(db); err != nil {
		t.Fatalf("seed SLA templates: %v", err)
	}
	if err := seedServiceDefinitions(db); err != nil {
		t.Fatalf("seed service definitions: %v", err)
	}

	var service ServiceDefinition
	if err := db.Where("code = ?", "vpn-access-request").First(&service).Error; err != nil {
		t.Fatalf("find vpn service: %v", err)
	}

	for _, forbidden := range []string{
		"vpn_account",
		"device_usage",
		"request_kind",
		"form.request_kind",
		"position_department",
		"department_code",
		"position_code",
		"network_admin",
		"security_admin",
		"online_support",
		"troubleshooting",
		"production_emergency",
		"network_access_issue",
		"external_collaboration",
		"long_term_remote_work",
		"cross_border_access",
		"security_compliance",
	} {
		if strings.Contains(service.CollaborationSpec, forbidden) {
			t.Fatalf("vpn collaboration spec should be natural text, found machine token %q in %q", forbidden, service.CollaborationSpec)
		}
	}

	var schema struct {
		Fields []struct {
			Key     string `json:"key"`
			Type    string `json:"type"`
			Options []struct {
				Value string `json:"value"`
			} `json:"options"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(service.IntakeFormSchema), &schema); err != nil {
		t.Fatalf("unmarshal vpn intake form schema: %v", err)
	}
	fieldTypes := map[string]string{}
	optionValues := map[string]bool{}
	for _, field := range schema.Fields {
		fieldTypes[field.Key] = field.Type
		for _, option := range field.Options {
			optionValues[option.Value] = true
		}
	}
	expectedFields := map[string]string{
		"vpn_account":  "text",
		"device_usage": "textarea",
		"request_kind": "select",
	}
	for key, typ := range expectedFields {
		if fieldTypes[key] != typ {
			t.Fatalf("expected vpn field %s type %s, got fields=%v", key, typ, fieldTypes)
		}
	}
	for _, value := range []string{"online_support", "troubleshooting", "production_emergency", "network_access_issue", "external_collaboration", "long_term_remote_work", "cross_border_access", "security_compliance"} {
		if !optionValues[value] {
			t.Fatalf("expected request_kind option %q, got %#v", value, optionValues)
		}
	}

	workflow := string(service.WorkflowJSON)
	for _, required := range []string{"form.request_kind", "it", "network_admin", "security_admin"} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("expected vpn workflow json to preserve %q, got %s", required, workflow)
		}
	}
}

func TestSeedCatalogs_IsIdempotentByCode(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs first run: %v", err)
	}
	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs second run: %v", err)
	}

	var count int64
	if err := db.Model(&ServiceCatalog{}).Count(&count).Error; err != nil {
		t.Fatalf("count catalogs: %v", err)
	}
	if count != 24 {
		t.Fatalf("expected 24 catalogs after rerun, got %d", count)
	}
}

func TestSeedCatalogs_RecreatesSoftDeletedCatalog(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}

	var catalog ServiceCatalog
	if err := db.Where("code = ?", "account-access:provisioning").First(&catalog).Error; err != nil {
		t.Fatalf("find seeded catalog: %v", err)
	}
	originalID := catalog.ID
	if err := db.Delete(&catalog).Error; err != nil {
		t.Fatalf("soft delete catalog: %v", err)
	}

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs rerun: %v", err)
	}

	var restored ServiceCatalog
	if err := db.Where("code = ?", "account-access:provisioning").First(&restored).Error; err != nil {
		t.Fatalf("find restored catalog: %v", err)
	}
	if restored.ID != originalID {
		t.Fatalf("expected soft-deleted catalog to be restored in place, got original=%d restored=%d", originalID, restored.ID)
	}

	var visibleCount int64
	if err := db.Model(&ServiceCatalog{}).Where("code = ?", "account-access:provisioning").Count(&visibleCount).Error; err != nil {
		t.Fatalf("count restored catalog: %v", err)
	}
	if visibleCount != 1 {
		t.Fatalf("expected restored catalog to be visible once, got %d", visibleCount)
	}
}

func TestSeedCatalogs_RestoresSoftDeletedRootAndChildInPlace(t *testing.T) {
	db := newTestDB(t)

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs: %v", err)
	}

	var root ServiceCatalog
	if err := db.Where("code = ?", "application-platform").First(&root).Error; err != nil {
		t.Fatalf("find root catalog: %v", err)
	}
	rootID := root.ID
	if err := db.Model(&root).Updates(map[string]any{
		"name":        "旧应用平台",
		"description": "legacy root",
		"icon":        "LegacyRoot",
		"sort_order":  999,
		"is_active":   false,
	}).Error; err != nil {
		t.Fatalf("drift root catalog: %v", err)
	}
	if err := db.Delete(&root).Error; err != nil {
		t.Fatalf("soft delete root catalog: %v", err)
	}

	var child ServiceCatalog
	if err := db.Where("code = ?", "application-platform:database").First(&child).Error; err != nil {
		t.Fatalf("find child catalog: %v", err)
	}
	childID := child.ID
	if err := db.Model(&child).Updates(map[string]any{
		"name":        "旧数据库支持",
		"description": "legacy child",
		"icon":        "LegacyChild",
		"sort_order":  777,
		"parent_id":   nil,
		"is_active":   false,
	}).Error; err != nil {
		t.Fatalf("drift child catalog: %v", err)
	}
	if err := db.Delete(&child).Error; err != nil {
		t.Fatalf("soft delete child catalog: %v", err)
	}

	if err := seedCatalogs(db); err != nil {
		t.Fatalf("seed catalogs rerun: %v", err)
	}

	var restoredRoot ServiceCatalog
	if err := db.Where("code = ?", "application-platform").First(&restoredRoot).Error; err != nil {
		t.Fatalf("find restored root catalog: %v", err)
	}
	if restoredRoot.ID != rootID {
		t.Fatalf("expected root restored in place, got original=%d restored=%d", rootID, restoredRoot.ID)
	}
	if restoredRoot.Name != "应用与平台支持" || restoredRoot.Icon != "Container" || restoredRoot.SortOrder != 40 || !restoredRoot.IsActive || restoredRoot.ParentID != nil {
		t.Fatalf("unexpected restored root catalog: %+v", restoredRoot)
	}

	var restoredChild ServiceCatalog
	if err := db.Where("code = ?", "application-platform:database").First(&restoredChild).Error; err != nil {
		t.Fatalf("find restored child catalog: %v", err)
	}
	if restoredChild.ID != childID {
		t.Fatalf("expected child restored in place, got original=%d restored=%d", childID, restoredChild.ID)
	}
	if restoredChild.ParentID == nil || *restoredChild.ParentID != restoredRoot.ID {
		t.Fatalf("expected child parent %d, got %v", restoredRoot.ID, restoredChild.ParentID)
	}
	if restoredChild.Name != "数据库支持" || restoredChild.Icon != "Database" || restoredChild.SortOrder != 3 || !restoredChild.IsActive {
		t.Fatalf("unexpected restored child catalog: %+v", restoredChild)
	}
}

func TestSeedMenus_RestoresSoftDeletedApprovalPendingMenu(t *testing.T) {
	db := newTestDB(t)

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	var menu model.Menu
	if err := db.Where("permission = ?", "itsm:ticket:approval:pending").First(&menu).Error; err != nil {
		t.Fatalf("find approval pending menu: %v", err)
	}
	originalID := menu.ID
	if err := db.Delete(&menu).Error; err != nil {
		t.Fatalf("soft delete approval pending menu: %v", err)
	}

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus rerun: %v", err)
	}

	var restored model.Menu
	if err := db.Where("permission = ?", "itsm:ticket:approval:pending").First(&restored).Error; err != nil {
		t.Fatalf("find restored approval pending menu: %v", err)
	}
	if restored.ID != originalID {
		t.Fatalf("expected approval pending menu to be restored in place, got original=%d restored=%d", originalID, restored.ID)
	}
	if restored.Name != "我的待办" {
		t.Fatalf("expected restored menu name 我的待办, got %s", restored.Name)
	}
	if restored.Path != "/itsm/tickets/approvals/pending" {
		t.Fatalf("expected restored menu path /itsm/tickets/approvals/pending, got %s", restored.Path)
	}
	if restored.Sort != 2 {
		t.Fatalf("expected restored menu sort 2, got %d", restored.Sort)
	}

	var visibleCount int64
	if err := db.Model(&model.Menu{}).Where("permission = ?", "itsm:ticket:approval:pending").Count(&visibleCount).Error; err != nil {
		t.Fatalf("count visible approval pending menu: %v", err)
	}
	if visibleCount != 1 {
		t.Fatalf("expected restored approval pending menu to be visible once, got %d", visibleCount)
	}

	var totalCount int64
	if err := db.Unscoped().Model(&model.Menu{}).Where("permission = ?", "itsm:ticket:approval:pending").Count(&totalCount).Error; err != nil {
		t.Fatalf("count all approval pending menu rows: %v", err)
	}
	if totalCount != 1 {
		t.Fatalf("expected one approval pending menu row including soft-deleted records, got %d", totalCount)
	}
}

func TestSeedMenus_MigratesLegacyDirectoriesAndCatalogMenu(t *testing.T) {
	db := newTestDB(t)

	legacyTicketDir := model.Menu{
		Name:       "工单管理",
		Type:       model.MenuTypeDirectory,
		Permission: "itsm:ticket",
		Sort:       1,
	}
	if err := db.Create(&legacyTicketDir).Error; err != nil {
		t.Fatalf("create legacy ticket dir: %v", err)
	}
	legacyChild := model.Menu{
		ParentID:   &legacyTicketDir.ID,
		Name:       "旧子菜单",
		Type:       model.MenuTypeMenu,
		Path:       "/itsm/legacy-child",
		Permission: "itsm:ticket:legacy-child",
		Sort:       0,
	}
	if err := db.Create(&legacyChild).Error; err != nil {
		t.Fatalf("create legacy child menu: %v", err)
	}
	oldCatalogMenu := model.Menu{
		Name:       "旧服务目录",
		Type:       model.MenuTypeMenu,
		Path:       "/itsm/catalogs",
		Permission: "itsm:catalog:list",
		Sort:       9,
	}
	if err := db.Create(&oldCatalogMenu).Error; err != nil {
		t.Fatalf("create old catalog menu: %v", err)
	}
	oldCatalogButton := model.Menu{
		ParentID:   &oldCatalogMenu.ID,
		Name:       "旧分类按钮",
		Type:       model.MenuTypeButton,
		Permission: "itsm:catalog:legacy-button",
		Sort:       0,
	}
	if err := db.Create(&oldCatalogButton).Error; err != nil {
		t.Fatalf("create old catalog button: %v", err)
	}
	oldServiceMenu := model.Menu{
		Name:       "服务定义",
		Type:       model.MenuTypeMenu,
		Path:       "/legacy/services",
		Permission: "itsm:service:list",
		Sort:       99,
	}
	if err := db.Create(&oldServiceMenu).Error; err != nil {
		t.Fatalf("create old service menu: %v", err)
	}

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	var itsmDir model.Menu
	if err := db.Where("permission = ?", "itsm").First(&itsmDir).Error; err != nil {
		t.Fatalf("find itsm dir: %v", err)
	}

	var movedChild model.Menu
	if err := db.Where("permission = ?", "itsm:ticket:legacy-child").First(&movedChild).Error; err != nil {
		t.Fatalf("find moved child menu: %v", err)
	}
	if movedChild.ParentID == nil || *movedChild.ParentID != itsmDir.ID {
		t.Fatalf("expected moved child parent %d, got %v", itsmDir.ID, movedChild.ParentID)
	}

	var ticketDir model.Menu
	if err := db.Unscoped().Where("permission = ?", "itsm:ticket").First(&ticketDir).Error; err != nil {
		t.Fatalf("find legacy ticket dir: %v", err)
	}
	if !ticketDir.DeletedAt.Valid {
		t.Fatalf("expected legacy ticket dir to be soft deleted")
	}

	var deletedCatalog model.Menu
	if err := db.Unscoped().Where("permission = ?", "itsm:catalog:list").First(&deletedCatalog).Error; err != nil {
		t.Fatalf("find old catalog menu: %v", err)
	}
	if !deletedCatalog.DeletedAt.Valid {
		t.Fatalf("expected old catalog menu to be soft deleted")
	}

	var deletedCatalogButton model.Menu
	if err := db.Unscoped().Where("permission = ?", "itsm:catalog:legacy-button").First(&deletedCatalogButton).Error; err != nil {
		t.Fatalf("find old catalog button: %v", err)
	}
	if !deletedCatalogButton.DeletedAt.Valid {
		t.Fatalf("expected old catalog button to be soft deleted")
	}

	var serviceMenu model.Menu
	if err := db.Where("permission = ?", "itsm:service:list").First(&serviceMenu).Error; err != nil {
		t.Fatalf("find service menu: %v", err)
	}
	if serviceMenu.Name != "服务目录" {
		t.Fatalf("expected service menu renamed to 服务目录, got %s", serviceMenu.Name)
	}
}

func TestSeedMenus_RestoresAndRepairsButtonsInPlace(t *testing.T) {
	db := newTestDB(t)

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus first run: %v", err)
	}

	var serviceMenu model.Menu
	if err := db.Where("permission = ?", "itsm:service:list").First(&serviceMenu).Error; err != nil {
		t.Fatalf("find service menu: %v", err)
	}

	var createButton model.Menu
	if err := db.Where("permission = ?", "itsm:catalog:create").First(&createButton).Error; err != nil {
		t.Fatalf("find create catalog button: %v", err)
	}
	createButtonID := createButton.ID
	if err := db.Delete(&createButton).Error; err != nil {
		t.Fatalf("soft delete create catalog button: %v", err)
	}

	var updateButton model.Menu
	if err := db.Where("permission = ?", "itsm:catalog:update").First(&updateButton).Error; err != nil {
		t.Fatalf("find update catalog button: %v", err)
	}
	if err := db.Model(&model.Menu{}).Where("id = ?", updateButton.ID).Updates(map[string]any{
		"name":      "错误名称",
		"type":      model.MenuTypeMenu,
		"sort":      999,
		"parent_id": nil,
	}).Error; err != nil {
		t.Fatalf("drift update catalog button: %v", err)
	}

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus second run: %v", err)
	}

	var restoredCreate model.Menu
	if err := db.Where("permission = ?", "itsm:catalog:create").First(&restoredCreate).Error; err != nil {
		t.Fatalf("find restored create catalog button: %v", err)
	}
	if restoredCreate.ID != createButtonID {
		t.Fatalf("expected create catalog button restored in place, got original=%d restored=%d", createButtonID, restoredCreate.ID)
	}
	if restoredCreate.ParentID == nil || *restoredCreate.ParentID != serviceMenu.ID {
		t.Fatalf("expected restored create button parent %d, got %v", serviceMenu.ID, restoredCreate.ParentID)
	}
	if restoredCreate.Name != "新增分类" || restoredCreate.Type != model.MenuTypeButton || restoredCreate.Sort != 3 {
		t.Fatalf("unexpected restored create button shape: %+v", restoredCreate)
	}

	var repairedUpdate model.Menu
	if err := db.Where("permission = ?", "itsm:catalog:update").First(&repairedUpdate).Error; err != nil {
		t.Fatalf("find repaired update catalog button: %v", err)
	}
	if repairedUpdate.ParentID == nil || *repairedUpdate.ParentID != serviceMenu.ID {
		t.Fatalf("expected repaired update button parent %d, got %v", serviceMenu.ID, repairedUpdate.ParentID)
	}
	if repairedUpdate.Name != "编辑分类" || repairedUpdate.Type != model.MenuTypeButton || repairedUpdate.Sort != 4 {
		t.Fatalf("unexpected repaired update button shape: %+v", repairedUpdate)
	}

	var totalCount int64
	if err := db.Unscoped().Model(&model.Menu{}).Where("permission = ?", "itsm:catalog:create").Count(&totalCount).Error; err != nil {
		t.Fatalf("count create catalog button rows: %v", err)
	}
	if totalCount != 1 {
		t.Fatalf("expected one create catalog button row including soft-deleted records, got %d", totalCount)
	}
}

func TestSeedMenus_RestoresServiceMenuAndRemovesObsoleteMenus(t *testing.T) {
	db := newTestDB(t)

	wrongParent := model.Menu{
		Name:       "错误父目录",
		Type:       model.MenuTypeDirectory,
		Permission: "itsm:wrong-parent",
	}
	if err := db.Create(&wrongParent).Error; err != nil {
		t.Fatalf("create wrong parent menu: %v", err)
	}

	serviceMenu := model.Menu{
		ParentID:   &wrongParent.ID,
		Name:       "旧服务定义",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/services",
		Icon:       "Legacy",
		Permission: "itsm:service:list",
		Sort:       99,
	}
	if err := db.Create(&serviceMenu).Error; err != nil {
		t.Fatalf("create drifted service menu: %v", err)
	}
	serviceMenuID := serviceMenu.ID
	if err := db.Delete(&serviceMenu).Error; err != nil {
		t.Fatalf("soft delete drifted service menu: %v", err)
	}

	obsoleteMenus := []model.Menu{
		{Name: "旧历史", Type: model.MenuTypeMenu, Permission: "itsm:ticket:history"},
		{Name: "旧待办目录", Type: model.MenuTypeDirectory, Permission: "itsm:ticket:todo"},
		{Name: "旧审批目录", Type: model.MenuTypeDirectory, Permission: "itsm:ticket:approvals"},
		{Name: "表单管理", Type: model.MenuTypeMenu, Permission: "itsm:form:list"},
	}
	for _, menu := range obsoleteMenus {
		menu := menu
		if err := db.Create(&menu).Error; err != nil {
			t.Fatalf("create obsolete menu %s: %v", menu.Permission, err)
		}
		if menu.Permission == "itsm:form:list" {
			btn := model.Menu{
				ParentID:   &menu.ID,
				Name:       "旧表单按钮",
				Type:       model.MenuTypeButton,
				Permission: "itsm:form:legacy-button",
			}
			if err := db.Create(&btn).Error; err != nil {
				t.Fatalf("create obsolete form button: %v", err)
			}
		}
	}

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	var itsmDir model.Menu
	if err := db.Where("permission = ?", "itsm").First(&itsmDir).Error; err != nil {
		t.Fatalf("find itsm dir: %v", err)
	}

	var restoredService model.Menu
	if err := db.Where("permission = ?", "itsm:service:list").First(&restoredService).Error; err != nil {
		t.Fatalf("find restored service menu: %v", err)
	}
	if restoredService.ID != serviceMenuID {
		t.Fatalf("expected restored service menu in place, got original=%d restored=%d", serviceMenuID, restoredService.ID)
	}
	if restoredService.ParentID == nil || *restoredService.ParentID != itsmDir.ID {
		t.Fatalf("expected restored service menu parent %d, got %v", itsmDir.ID, restoredService.ParentID)
	}
	if restoredService.Name != "服务目录" || restoredService.Type != model.MenuTypeMenu || restoredService.Path != "/itsm/services" || restoredService.Icon != "Cog" || restoredService.Sort != 4 {
		t.Fatalf("unexpected restored service menu: %+v", restoredService)
	}

	for _, permission := range []string{"itsm:ticket:history", "itsm:ticket:todo", "itsm:ticket:approvals", "itsm:form:list", "itsm:form:legacy-button"} {
		var menu model.Menu
		if err := db.Unscoped().Where("permission = ?", permission).First(&menu).Error; err != nil {
			t.Fatalf("find obsolete menu %s: %v", permission, err)
		}
		if !menu.DeletedAt.Valid {
			t.Fatalf("expected obsolete menu %s to be soft deleted", permission)
		}
	}
}
