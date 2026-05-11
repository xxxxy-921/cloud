package definition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	appcore "metis/internal/app"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	. "metis/internal/app/itsm/config"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	"metis/internal/llm"
)

type workflowGenerateOrgContextResolverStub struct {
	ctx *appcore.OrgContextResult
	err error
}

func (s workflowGenerateOrgContextResolverStub) GetUserDeptScope(uint, bool) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) GetUserPositionIDs(uint) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) GetUserDepartmentIDs(uint) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) GetUserPositions(uint) ([]appcore.OrgPosition, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) GetUserDepartment(uint) (*appcore.OrgDepartment, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) QueryContext(string, string, string, bool) (*appcore.OrgContextResult, error) {
	return s.ctx, s.err
}
func (s workflowGenerateOrgContextResolverStub) FindUsersByPositionCode(string) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) FindUsersByDepartmentCode(string) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) FindUsersByPositionAndDepartment(string, string) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) FindUsersByPositionID(uint) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) FindUsersByDepartmentID(uint) ([]uint, error) {
	return nil, nil
}
func (s workflowGenerateOrgContextResolverStub) FindManagerByUserID(uint) (uint, error) {
	return 0, nil
}

type workflowGenerateOrgStructureResolverStub struct {
	searchResult  *appcore.OrgStructureSearchResult
	searchErr     error
	resolveResult *appcore.OrgParticipantResolveResult
	resolveErr    error
}

func (s workflowGenerateOrgStructureResolverStub) SearchOrgStructure(query string, kinds []string, limit int) (*appcore.OrgStructureSearchResult, error) {
	return s.searchResult, s.searchErr
}

func (s workflowGenerateOrgStructureResolverStub) ResolveOrgParticipant(departmentHint, positionHint string, limit int) (*appcore.OrgParticipantResolveResult, error) {
	return s.resolveResult, s.resolveErr
}

func TestWorkflowGenerateHandlerCapabilitiesReturnsEngineContract(t *testing.T) {
	h := &WorkflowGenerateHandler{}
	gin.SetMode(gin.TestMode)

	rec := performJSONRequest(t, func(r *gin.Engine) {
		r.GET("/workflow-capabilities", h.Capabilities)
	}, http.MethodGet, "/workflow-capabilities", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code int                       `json:"code"`
		Data engine.WorkflowCapability `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode capabilities response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: %+v", resp)
	}
	want := engine.WorkflowCapabilities()
	if resp.Data.Version != want.Version {
		t.Fatalf("capabilities version = %s, want %s", resp.Data.Version, want.Version)
	}
	if got, ok := resp.Data.NodeTypes[engine.NodeAction]; !ok || len(got.RequiredFields) == 0 || got.RequiredFields[0] != want.NodeTypes[engine.NodeAction].RequiredFields[0] {
		t.Fatalf("expected action node capability contract, got %+v want %+v", got, want.NodeTypes[engine.NodeAction])
	}
	if got, ok := resp.Data.NodeTypes[engine.NodeScript]; !ok || len(got.RequiredFields) == 0 || got.RequiredFields[0] != want.NodeTypes[engine.NodeScript].RequiredFields[0] {
		t.Fatalf("expected script node capability contract, got %+v want %+v", got, want.NodeTypes[engine.NodeScript])
	}
	if got, ok := resp.Data.NodeTypes[engine.NodeBError]; !ok || got.Executable != want.NodeTypes[engine.NodeBError].Executable {
		t.Fatalf("expected boundary error node contract, got %+v want %+v", got, want.NodeTypes[engine.NodeBError])
	}
}

func TestHandleDocParseContracts(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	docSvc := newKnowledgeDocServiceForTest(t, db, serviceDefs)

	root, err := catSvc.Create("Root", "root-doc-task", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc-task", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	doc, err := docSvc.Upload(service.ID, "guide.txt", int64(len("hello parser task")), bytes.NewReader([]byte("hello parser task")))
	if err != nil {
		t.Fatalf("upload doc: %v", err)
	}

	parseTask := HandleDocParse(docSvc)
	if err := parseTask(context.Background(), json.RawMessage(`{"document_id":`+strconv.FormatUint(uint64(doc.ID), 10)+`}`)); err != nil {
		t.Fatalf("parse task success: %v", err)
	}

	reloaded, err := docSvc.repo.GetByID(doc.ID)
	if err != nil {
		t.Fatalf("reload parsed doc: %v", err)
	}
	if reloaded.ParseStatus != "completed" || reloaded.ParsedText == "" {
		t.Fatalf("expected completed parsed doc, got %+v", reloaded)
	}

	if err := parseTask(context.Background(), json.RawMessage(`{}`)); err == nil || err.Error() != "document_id is required" {
		t.Fatalf("missing document id error = %v", err)
	}
	if err := parseTask(context.Background(), json.RawMessage(`not-json`)); err == nil {
		t.Fatal("expected invalid payload to fail")
	}
}

func TestWorkflowGenerateHandlerMapsRequestAndEngineErrors(t *testing.T) {
	run := func(t *testing.T, handler *WorkflowGenerateHandler, body string) *httptest.ResponseRecorder {
		t.Helper()
		c, rec := newGinContext(http.MethodPost, "/api/v1/itsm/workflows/generate")
		c.Request.Body = io.NopCloser(bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		handler.Generate(c)
		return rec
	}

	t.Run("invalid json request returns 400", func(t *testing.T) {
		rec := run(t, &WorkflowGenerateHandler{svc: newWorkflowGenerateServiceForRetryTest(&fakeWorkflowLLMClient{}, 0)}, `{"collaborationSpec":`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request for malformed json, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("empty collaboration spec returns 400", func(t *testing.T) {
		rec := run(t, &WorkflowGenerateHandler{svc: newWorkflowGenerateServiceForRetryTest(&fakeWorkflowLLMClient{}, 0)}, `{"collaborationSpec":"   "}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request for empty collaboration spec, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing engine config returns 400", func(t *testing.T) {
		h := &WorkflowGenerateHandler{svc: &WorkflowGenerateService{
			engineConfigSvc: fakePathEngineConfigProvider{err: errors.New("missing model")},
		}}
		rec := run(t, h, `{"collaborationSpec":"用户提交 VPN 申请后经理审批"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected missing config to return 400, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unparseable workflow response returns 422", func(t *testing.T) {
		h := &WorkflowGenerateHandler{svc: newWorkflowGenerateServiceForRetryTest(&fakeWorkflowLLMClient{
			responses: []llm.ChatResponse{{Content: "not json"}},
		}, 0)}
		rec := run(t, h, `{"collaborationSpec":"用户提交 VPN 申请后经理审批"}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected invalid generated workflow to return 422, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestValidateGeneratedServiceContractForBossFlow(t *testing.T) {
	validSchema := json.RawMessage(`{"version":1,"fields":[
		{"key":"subject","type":"text","label":"申请主题"},
		{"key":"request_category","type":"select","label":"申请类别","options":[{"label":"变更","value":"prod_change"}]},
		{"key":"risk_level","type":"radio","label":"风险等级","options":[{"label":"高","value":"high"}]},
		{"key":"change_window","type":"date_range","label":"变更窗口"},
		{"key":"impact_scope","type":"textarea","label":"影响范围"},
		{"key":"rollback_required","type":"select","label":"回滚要求","options":[{"label":"需要","value":"required"}]},
		{"key":"impact_modules","type":"multi_select","label":"影响模块","options":[{"label":"网关","value":"gateway"}]},
		{"key":"change_items","type":"table","label":"变更明细","props":{"columns":[{"key":"system","type":"text","label":"系统"}]}}
	]}`)

	t.Run("non boss service skips strict contract", func(t *testing.T) {
		errs := validateGeneratedServiceContract("generic-service", json.RawMessage(validWorkflowJSONForGenerateTest()), nil)
		if len(errs) != 0 {
			t.Fatalf("expected non-boss service to skip contract checks, got %+v", errs)
		}
	})

	t.Run("boss contract rejects missing schema", func(t *testing.T) {
		errs := validateGeneratedServiceContract(bossSerialChangeServiceCode, json.RawMessage(validWorkflowJSONForGenerateTest()), nil)
		if len(errs) == 0 {
			t.Fatal("expected missing intake schema to fail")
		}
		if len(errs) != 1 || !strings.Contains(errs[0].Message, "intake form schema") {
			t.Fatalf("expected missing intake schema error, got %+v", errs)
		}
	})

	t.Run("boss contract rejects missing required field and participant nodes", func(t *testing.T) {
		badSchema := json.RawMessage(`{"version":1,"fields":[
			{"key":"subject","type":"text","label":"申请主题"},
			{"key":"request_category","type":"select","label":"申请类别","options":[{"label":"变更","value":"prod_change"}]},
			{"key":"risk_level","type":"radio","label":"风险等级","options":[{"label":"高","value":"high"}]},
			{"key":"change_window","type":"date_range","label":"变更窗口"},
			{"key":"impact_scope","type":"textarea","label":"影响范围"},
			{"key":"rollback_required","type":"select","label":"回滚要求","options":[{"label":"需要","value":"required"}]},
			{"key":"impact_modules","type":"multi_select","label":"影响模块","options":[{"label":"网关","value":"gateway"}]}
		]}`)
		errs := validateGeneratedServiceContract(bossSerialChangeServiceCode, json.RawMessage(validVPNWorkflowJSONForGenerateTest()), badSchema)
		if len(errs) == 0 {
			t.Fatal("expected missing boss field and participant contract to fail")
		}
		joined := make([]string, 0, len(errs))
		for _, err := range errs {
			joined = append(joined, err.Message)
		}
		msg := strings.Join(joined, " | ")
		if !strings.Contains(msg, `field "change_items"`) || !strings.Contains(msg, "headquarters.serial_reviewer") || !strings.Contains(msg, "it.ops_admin") {
			t.Fatalf("expected missing field and required positions, got %+v", errs)
		}
	})

	t.Run("boss contract accepts canonical workflow", func(t *testing.T) {
		errs := validateGeneratedServiceContract(bossSerialChangeServiceCode, json.RawMessage(validBossWorkflowJSONForGenerateTest()), validSchema)
		if len(errs) != 0 {
			t.Fatalf("expected valid boss workflow contract, got %+v", errs)
		}
	})
}

func TestGeneratedFormSchemaNormalizationAndHealthLocations(t *testing.T) {
	t.Run("generated form field keys reject invalid schema", func(t *testing.T) {
		if _, err := generatedFormFieldKeys(json.RawMessage(`{"fields":[]}`)); err == nil || !strings.Contains(err.Error(), "at least one field") {
			t.Fatalf("expected empty fields schema to fail, got %v", err)
		}
		if _, err := generatedFormFieldKeys(json.RawMessage(`{"fields":[{"key":" "}]}`)); err == nil || !strings.Contains(err.Error(), "empty field key") {
			t.Fatalf("expected empty field key to fail, got %v", err)
		}
	})

	t.Run("normalize options canonicalizes string map and scalar values", func(t *testing.T) {
		got := normalizeGeneratedOptions([]any{
			"ops",
			map[string]any{"label": "安全"},
			42,
		})
		options, ok := got.([]any)
		if !ok || len(options) != 3 {
			t.Fatalf("unexpected normalized options: %#v", got)
		}
		first := options[0].(map[string]any)
		second := options[1].(map[string]any)
		third := options[2].(map[string]any)
		if first["label"] != "ops" || first["value"] != "ops" {
			t.Fatalf("unexpected first option: %#v", first)
		}
		if second["label"] != "安全" || second["value"] != "安全" {
			t.Fatalf("expected missing value to copy label, got %#v", second)
		}
		if third["label"] != "42" || third["value"] != "42" {
			t.Fatalf("expected scalar option to stringify, got %#v", third)
		}
	})

	t.Run("normalize table columns makes required and options explicit", func(t *testing.T) {
		field := map[string]any{
			"type": "table",
			"props": map[string]any{
				"columns": []any{
					map[string]any{"key": "kind", "type": "select", "label": "类型", "options": []any{"ops"}},
					map[string]any{"key": "owner", "type": "text", "label": "负责人"},
				},
			},
		}
		normalizeGeneratedTableColumns(field)
		columns := field["props"].(map[string]any)["columns"].([]any)
		first := columns[0].(map[string]any)
		second := columns[1].(map[string]any)
		if first["required"] != true || second["required"] != true {
			t.Fatalf("expected table columns to default required=true, got %#v", columns)
		}
		firstOptions := first["options"].([]any)
		firstOption := firstOptions[0].(map[string]any)
		if firstOption["label"] != "ops" || firstOption["value"] != "ops" {
			t.Fatalf("expected column options to be normalized, got %#v", firstOptions)
		}
	})

	t.Run("publish health location prefers edge then node then collaboration spec", func(t *testing.T) {
		edgeLoc := workflowValidationHealthLocation(engine.ValidationError{EdgeID: "edge-1", NodeID: "node-1", Message: "broken edge"})
		if edgeLoc.Kind != "workflow_edge" || edgeLoc.RefID != "edge-1" || edgeLoc.Path != "service.workflowJson.edges[id=edge-1]" {
			t.Fatalf("unexpected edge location: %+v", edgeLoc)
		}
		nodeLoc := workflowValidationHealthLocation(engine.ValidationError{NodeID: "node-2", Message: "broken node"})
		if nodeLoc.Kind != "workflow_node" || nodeLoc.RefID != "node-2" || nodeLoc.Path != "service.workflowJson.nodes[id=node-2]" {
			t.Fatalf("unexpected node location: %+v", nodeLoc)
		}
		specLoc := workflowValidationHealthLocation(engine.ValidationError{Message: "spec issue"})
		if specLoc.Kind != "collaboration_spec" || specLoc.Path != "service.collaborationSpec" {
			t.Fatalf("unexpected fallback location: %+v", specLoc)
		}
	})

	t.Run("publish health payload keeps blocking evidence and recommendations", func(t *testing.T) {
		health := publishHealthCheckFromValidationErrors(99, []engine.ValidationError{
			{Level: "blocking", EdgeID: "edge-a", Message: "缺少默认分支"},
			{Level: "", NodeID: "node-b", Message: ""},
		})
		if health.ServiceID != 99 || health.Status != "fail" || len(health.Items) != 2 {
			t.Fatalf("unexpected publish health payload: %+v", health)
		}
		if health.Items[0].Location.Kind != "workflow_edge" || !strings.Contains(health.Items[0].Evidence, "缺少默认分支") {
			t.Fatalf("unexpected first health item: %+v", health.Items[0])
		}
		if health.Items[1].Location.Kind != "workflow_node" || health.Items[1].Message != "参考路径存在结构性阻塞项" {
			t.Fatalf("expected empty message to use blocking fallback, got %+v", health.Items[1])
		}
		if !strings.Contains(health.Items[1].Recommendation, "重新生成") {
			t.Fatalf("expected actionable recommendation, got %+v", health.Items[1])
		}
	})
}

func TestWorkflowGenerateOrgContextAndToolCollectionContracts(t *testing.T) {
	t.Run("build org context keeps only active coded departments and positions", func(t *testing.T) {
		svc := &WorkflowGenerateService{
			orgResolver: workflowGenerateOrgContextResolverStub{ctx: &appcore.OrgContextResult{
				Departments: []appcore.OrgContextDepartment{
					{Code: "it", Name: "信息部", IsActive: true},
					{Code: "", Name: "无编码部门", IsActive: true},
					{Code: "legacy", Name: "停用部门", IsActive: false},
				},
				Positions: []appcore.OrgContextPosition{
					{Code: "network_admin", Name: "网络管理员", IsActive: true},
					{Code: "", Name: "无编码岗位", IsActive: true},
					{Code: "legacy_pos", Name: "停用岗位", IsActive: false},
				},
			}},
		}
		ctx := svc.buildOrgContext()
		if !strings.Contains(ctx, "信息部") || !strings.Contains(ctx, "network_admin") {
			t.Fatalf("expected active coded org entries in context, got %q", ctx)
		}
		if strings.Contains(ctx, "无编码部门") || strings.Contains(ctx, "停用岗位") {
			t.Fatalf("expected inactive or code-less entries to be filtered, got %q", ctx)
		}
	})

	t.Run("build org context returns empty when resolver unavailable or empty", func(t *testing.T) {
		if got := (&WorkflowGenerateService{}).buildOrgContext(); got != "" {
			t.Fatalf("expected empty context without resolver, got %q", got)
		}
		svc := &WorkflowGenerateService{orgResolver: workflowGenerateOrgContextResolverStub{err: errors.New("boom")}}
		if got := svc.buildOrgContext(); got != "" {
			t.Fatalf("expected empty context on resolver error, got %q", got)
		}
	})

	t.Run("collect org context from tool calls", func(t *testing.T) {
		client := &fakeWorkflowLLMClient{
			responses: []llm.ChatResponse{
				{
					ToolCalls: []llm.ToolCall{
						{ID: "call-search", Name: "workflow.org_search_structure", Arguments: `{"query":"信息部","kinds":["department","position"],"limit":5}`},
						{ID: "call-resolve", Name: "workflow.org_resolve_participant", Arguments: `{"department_hint":"信息部","position_hint":"网络管理员","limit":3}`},
					},
				},
				{Content: "无需继续查询"},
			},
		}
		svc := &WorkflowGenerateService{
			orgStructureResolver: workflowGenerateOrgStructureResolverStub{
				searchResult: &appcore.OrgStructureSearchResult{
					Departments: []appcore.OrgContextDepartment{{Code: "it", Name: "信息部", IsActive: true}},
					Positions:   []appcore.OrgContextPosition{{Code: "network_admin", Name: "网络管理员", IsActive: true}},
				},
				resolveResult: &appcore.OrgParticipantResolveResult{
					Candidates: []appcore.OrgParticipantCandidate{{
						Type:           "position_department",
						DepartmentCode: "it",
						DepartmentName: "信息部",
						PositionCode:   "network_admin",
						PositionName:   "网络管理员",
						CandidateCount: 2,
					}},
				},
			},
		}

		ctx, err := svc.collectOrgContextWithTools(context.Background(), client, LLMEngineRuntimeConfig{
			Model:          "test-model",
			TimeoutSeconds: 5,
			MaxTokens:      256,
		}, "原始 system prompt", 0.2, "员工申请 VPN 开通", workflowPromptContext{ActionsContext: "动作上下文"})
		if err != nil {
			t.Fatalf("collectOrgContextWithTools: %v", err)
		}
		if !strings.Contains(ctx, "组织搜索") || !strings.Contains(ctx, "query=`信息部`") || !strings.Contains(ctx, "position_department") || !strings.Contains(ctx, "network_admin") {
			t.Fatalf("expected collected org tool context, got %q", ctx)
		}
		if len(client.requests) == 0 || !strings.Contains(client.requests[0].Messages[0].Content, "组织上下文查询助手") || !strings.Contains(client.requests[0].Messages[1].Content, "员工申请 VPN 开通") {
			t.Fatalf("expected org preflight prompts to be sent, got %+v", client.requests)
		}
	})

	t.Run("collect org context surfaces upstream llm failure", func(t *testing.T) {
		client := &fakeWorkflowLLMClient{errs: []error{errors.New("llm down")}}
		svc := &WorkflowGenerateService{
			orgStructureResolver: workflowGenerateOrgStructureResolverStub{},
		}
		_, err := svc.collectOrgContextWithTools(context.Background(), client, LLMEngineRuntimeConfig{
			Model:          "test-model",
			TimeoutSeconds: 5,
			MaxTokens:      256,
		}, "原始 system prompt", 0.2, "员工申请 VPN 开通", workflowPromptContext{})
		if err == nil || !strings.Contains(err.Error(), "org preflight llm call") {
			t.Fatalf("expected upstream llm failure to surface, got %v", err)
		}
	})

	t.Run("execute org tool handles invalid payload missing resolver and unknown tool", func(t *testing.T) {
		svc := &WorkflowGenerateService{}
		msg, result := svc.executeWorkflowOrgTool(llm.ToolCall{Name: "workflow.org_search_structure", Arguments: `not-json`})
		if result != nil || !strings.Contains(msg, `"ok":false`) || !strings.Contains(msg, "组织结构解析器不可用") {
			t.Fatalf("expected missing resolver error payload, got msg=%q result=%+v", msg, result)
		}

		svc.orgStructureResolver = workflowGenerateOrgStructureResolverStub{}
		msg, result = svc.executeWorkflowOrgTool(llm.ToolCall{Name: "workflow.org_search_structure", Arguments: `not-json`})
		if result != nil || !strings.Contains(msg, "参数不是有效 JSON") {
			t.Fatalf("expected invalid payload error, got msg=%q result=%+v", msg, result)
		}

		msg, result = svc.executeWorkflowOrgTool(llm.ToolCall{Name: "workflow.unknown", Arguments: `{}`})
		if result != nil || !strings.Contains(msg, "未知组织上下文工具") {
			t.Fatalf("expected unknown tool error, got msg=%q result=%+v", msg, result)
		}
	})
}
