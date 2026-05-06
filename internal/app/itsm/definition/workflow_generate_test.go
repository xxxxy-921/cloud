package definition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	. "metis/internal/app/itsm/config"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	"metis/internal/llm"
	"metis/internal/model"
)

type fakePathEngineConfigProvider struct {
	cfg LLMEngineRuntimeConfig
	err error
}

func (p fakePathEngineConfigProvider) PathBuilderRuntimeConfig() (LLMEngineRuntimeConfig, error) {
	return p.cfg, p.err
}

type fakeWorkflowLLMClient struct {
	responses []llm.ChatResponse
	errs      []error
	calls     int
	requests  []llm.ChatRequest
}

func (c *fakeWorkflowLLMClient) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	c.calls++
	c.requests = append(c.requests, req)
	idx := c.calls - 1
	if idx < len(c.errs) && c.errs[idx] != nil {
		return nil, c.errs[idx]
	}
	if idx < len(c.responses) {
		resp := c.responses[idx]
		return &resp, nil
	}
	return &llm.ChatResponse{}, nil
}

func (c *fakeWorkflowLLMClient) ChatStream(context.Context, llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	return nil, llm.ErrNotSupported
}

func (c *fakeWorkflowLLMClient) Embedding(context.Context, llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	return nil, llm.ErrNotSupported
}

func newWorkflowGenerateServiceForRetryTest(client *fakeWorkflowLLMClient, maxRetries int) *WorkflowGenerateService {
	return &WorkflowGenerateService{
		engineConfigSvc: fakePathEngineConfigProvider{cfg: LLMEngineRuntimeConfig{
			Model:          "gpt-test",
			Protocol:       llm.ProtocolOpenAI,
			BaseURL:        "https://example.test/v1",
			APIKey:         "test-key",
			Temperature:    0.3,
			MaxTokens:      1024,
			MaxRetries:     maxRetries,
			TimeoutSeconds: 30,
			SystemPrompt:   "configured prompt",
		}},
		llmClientFactory: func(string, string, string) (llm.Client, error) {
			return client, nil
		},
	}
}

func validWorkflowJSONForGenerateTest() string {
	return `{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"start"}},
			{"id":"request","type":"form","data":{"label":"request form","participants":[{"type":"requester"}],"formSchema":{"version":1,"fields":[{"key":"summary","type":"textarea","label":"Summary","required":true}]}}},
			{"id":"end","type":"end","data":{"label":"end"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"request","data":{}},
			{"id":"e2","source":"request","target":"end","data":{"outcome":"submitted"}}
		]
	}`
}

func workflowWithBlockingIssueForGenerateTest(userID uint) string {
	return fmt.Sprintf(`{
		"nodes": [
			{"id":"start","type":"start","data":{"label":"start"}},
			{"id":"request","type":"form","data":{"label":"request form","participants":[{"type":"requester"}],"formSchema":{"version":1,"fields":[{"key":"summary","type":"textarea","label":"Summary","required":true}]}}},
			{"id":"process","type":"process","data":{"label":"process","participants":[{"type":"user","value":"%d"}]}},
			{"id":"end","type":"end","data":{"label":"end"}}
		],
		"edges": [
			{"id":"e1","source":"start","target":"request","data":{}},
			{"id":"e2","source":"request","target":"process","data":{"outcome":"submitted"}},
			{"id":"e3","source":"process","target":"end","data":{"outcome":"approved"}}
		]
	}`, userID)
}

func validWorkflowDraftWithFormSchema() json.RawMessage {
	return json.RawMessage(validWorkflowJSONForGenerateTest())
}

func bossContractFormSchemaForTest() json.RawMessage {
	return json.RawMessage(`{
		"version":1,
		"fields":[
			{"key":"subject","type":"text","label":"Subject","required":true},
			{"key":"request_category","type":"select","label":"Category","required":true,"options":[{"label":"prod_change","value":"prod_change"}]},
			{"key":"risk_level","type":"radio","label":"Risk","required":true,"options":[{"label":"high","value":"high"}]},
			{"key":"change_window","type":"date_range","label":"Window","required":true},
			{"key":"impact_scope","type":"textarea","label":"Impact","required":true},
			{"key":"rollback_required","type":"select","label":"Rollback","required":true,"options":[{"label":"required","value":"required"}]},
			{"key":"impact_modules","type":"multi_select","label":"Modules","required":true,"options":[{"label":"gateway","value":"gateway"}]},
			{"key":"change_items","type":"table","label":"Items","required":true,"props":{"columns":[{"key":"system","type":"text","label":"System","required":true}]}}
		]
	}`)
}

func bossContractWorkflowForTest() json.RawMessage {
	return json.RawMessage(`{
		"nodes":[
			{"id":"start","type":"start","data":{"label":"start"}},
			{"id":"request","type":"form","data":{"label":"request","participants":[{"type":"requester"}],"formSchema":{"version":1,"fields":[{"key":"subject","type":"text","label":"Subject","required":true}]}}},
			{"id":"serial","type":"process","data":{"label":"serial","participants":[{"type":"position_department","department_code":"headquarters","position_code":"serial_reviewer"}]}},
			{"id":"ops","type":"process","data":{"label":"ops","participants":[{"type":"position_department","department_code":"it","position_code":"ops_admin"}]}},
			{"id":"end","type":"end","data":{"label":"end"}}
		],
		"edges":[
			{"id":"e1","source":"start","target":"request","data":{}},
			{"id":"e2","source":"request","target":"serial","data":{"outcome":"submitted"}},
			{"id":"e3","source":"serial","target":"ops","data":{"outcome":"approved"}},
			{"id":"e4","source":"serial","target":"end","data":{"outcome":"rejected"}},
			{"id":"e5","source":"ops","target":"end","data":{"outcome":"approved"}},
			{"id":"e6","source":"ops","target":"end","data":{"outcome":"rejected"}}
		]
	}`)
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "bare json", input: `{"nodes":[],"edges":[]}`},
		{name: "markdown block", input: "```json\n{\"nodes\":[],\"edges\":[]}\n```"},
		{name: "wrapped text", input: `workflow: {"nodes":[],"edges":[]}`},
		{name: "invalid", input: "not json", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !json.Valid(got) {
				t.Fatalf("expected valid json, got %s", got)
			}
		})
	}
}

func TestExtractGeneratedIntakeFormSchema_NormalizesOptions(t *testing.T) {
	workflow := json.RawMessage(`{"nodes":[{"id":"form1","type":"form","data":{"formSchema":{"fields":[{"key":"reason","type":"select","label":"Reason","options":["ops","security"]}]}}}],"edges":[]}`)
	schemaJSON, errs := extractGeneratedIntakeFormSchema(workflow)
	if len(errs) > 0 {
		t.Fatalf("expected schema extraction to pass, got %+v", errs)
	}
	var schema struct {
		Fields []struct {
			Key      string `json:"key"`
			Required bool   `json:"required"`
			Options  []struct {
				Label string `json:"label"`
				Value string `json:"value"`
			} `json:"options"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if len(schema.Fields) != 1 || schema.Fields[0].Key != "reason" || !schema.Fields[0].Required {
		t.Fatalf("unexpected normalized schema: %s", schemaJSON)
	}
	if len(schema.Fields[0].Options) != 2 || schema.Fields[0].Options[0].Value != "ops" {
		t.Fatalf("expected string options to be normalized, got %s", schemaJSON)
	}
}

func TestBuildUserMessage_ContainsRepairGuidance(t *testing.T) {
	svc := &WorkflowGenerateService{}
	actionsCtx := svc.buildActionsContext([]ServiceAction{
		{BaseModel: model.BaseModel{ID: 7}, Name: "Send Email", Code: "send-email", Description: "notify reviewers"},
	})
	msg := svc.buildUserMessage("Route the request for approval", actionsCtx, []engine.ValidationError{
		{NodeID: "node-1", Level: "blocking", Message: "participants are required"},
		{EdgeID: "edge-2", Level: "blocking", Message: "action_id is missing"},
	})

	for _, snippet := range []string{
		"Route the request for approval",
		"Available service actions",
		"send-email",
		"Fix the following validation issues before regenerating",
		"[node node-1]",
		"[edge edge-2]",
		`"participants":[{"type":"requester"}]`,
		`"participants":[{"type":"position_department","department_code":"it","position_code":"network_admin"}]`,
	} {
		if !strings.Contains(msg, snippet) {
			t.Fatalf("expected message to contain %q, got %q", snippet, msg)
		}
	}
}

func TestBuildWorkflowGenerationContractContext_InjectsIntakeSchemaAndBOSSRules(t *testing.T) {
	ctx := buildWorkflowGenerationContractContext(&ServiceDefinition{
		Code:             bossSerialChangeServiceCode,
		IntakeFormSchema: JSONField(bossContractFormSchemaForTest()),
	})

	for _, snippet := range []string{
		"## Service contract",
		"Existing intake form schema",
		"subject",
		"request_category",
		"headquarters.serial_reviewer",
		"it.ops_admin",
		"requester form -> headquarters.serial_reviewer -> it.ops_admin -> end",
	} {
		if !strings.Contains(ctx, snippet) {
			t.Fatalf("expected contract context to contain %q, got %q", snippet, ctx)
		}
	}
}

func TestBuildActionsContext(t *testing.T) {
	svc := &WorkflowGenerateService{}
	result := svc.buildActionsContext([]ServiceAction{
		{BaseModel: model.BaseModel{ID: 1}, Name: "Email", Code: "send-email", Description: "send a notification"},
		{BaseModel: model.BaseModel{ID: 2}, Name: "Ticket", Code: "create-ticket"},
	})
	for _, snippet := range []string{"Available service actions", "send-email", "create-ticket", "id: `1`", "send a notification"} {
		if !strings.Contains(result, snippet) {
			t.Fatalf("expected actions context to contain %q, got %q", snippet, result)
		}
	}
}

func TestGenerate_UsesConfiguredSystemPrompt(t *testing.T) {
	client := &fakeWorkflowLLMClient{
		responses: []llm.ChatResponse{{Content: validWorkflowJSONForGenerateTest()}},
	}
	svc := newWorkflowGenerateServiceForRetryTest(client, 0)
	_, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(client.requests) == 0 || client.requests[0].Messages[0].Content != "configured prompt" {
		t.Fatalf("expected configured system prompt, got %+v", client.requests)
	}
}

func TestGenerate_FailsFastOnLLMUpstreamError(t *testing.T) {
	client := &fakeWorkflowLLMClient{errs: []error{
		context.DeadlineExceeded,
		context.DeadlineExceeded,
		context.DeadlineExceeded,
		context.DeadlineExceeded,
	}}
	svc := newWorkflowGenerateServiceForRetryTest(client, 3)
	_, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if !errors.Is(err, ErrPathEngineUpstream) {
		t.Fatalf("expected ErrPathEngineUpstream, got %v", err)
	}
	if client.calls != 4 {
		t.Fatalf("expected llm to retry upstream failures, got %d calls", client.calls)
	}
}

func TestWorkflowGenerateHandlerReturnsBadGatewayForLLMUpstreamError(t *testing.T) {
	client := &fakeWorkflowLLMClient{errs: []error{
		context.DeadlineExceeded,
		context.DeadlineExceeded,
		context.DeadlineExceeded,
		context.DeadlineExceeded,
	}}
	h := &WorkflowGenerateHandler{svc: newWorkflowGenerateServiceForRetryTest(client, 3)}
	c, rec := newGinContext(http.MethodPost, "/api/v1/itsm/workflows/generate")
	c.Request.Body = io.NopCloser(bytes.NewBufferString(`{"collaborationSpec":"generate a VPN request flow"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Generate(c)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNormalizePathEngineUpstreamMessage_Empty502Body(t *testing.T) {
	msg := normalizePathEngineUpstreamMessage(errors.New("error, status code: 502, status: 502 Bad Gateway, message: unexpected end of JSON input, body: "))
	if !strings.Contains(msg, "empty or invalid 502 response body") {
		t.Fatalf("expected normalized 502 message, got %q", msg)
	}
}

func TestGenerateRejectsEmptyCollaborationSpec(t *testing.T) {
	svc := newWorkflowGenerateServiceForRetryTest(&fakeWorkflowLLMClient{}, 1)
	_, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "   "})
	if !errors.Is(err, ErrCollaborationSpecEmpty) {
		t.Fatalf("expected ErrCollaborationSpecEmpty, got %v", err)
	}
}

func TestGenerateFailsWhenPathEngineConfigMissing(t *testing.T) {
	svc := &WorkflowGenerateService{engineConfigSvc: fakePathEngineConfigProvider{err: errors.New("model missing")}}
	_, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if !errors.Is(err, ErrPathEngineNotConfigured) {
		t.Fatalf("expected ErrPathEngineNotConfigured, got %v", err)
	}
}

func TestGenerate_RetriesJSONExtractionFailure(t *testing.T) {
	client := &fakeWorkflowLLMClient{
		responses: []llm.ChatResponse{
			{Content: "not json"},
			{Content: validWorkflowJSONForGenerateTest()},
		},
	}
	svc := newWorkflowGenerateServiceForRetryTest(client, 1)
	resp, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if err != nil {
		t.Fatalf("generate workflow: %v", err)
	}
	if client.calls != 2 || resp.Retries != 1 {
		t.Fatalf("expected one retry, got calls=%d retries=%d", client.calls, resp.Retries)
	}
}

func TestGenerate_ReturnsErrorWhenJSONExtractionNeverSucceeds(t *testing.T) {
	client := &fakeWorkflowLLMClient{responses: []llm.ChatResponse{{Content: "not json"}}}
	svc := newWorkflowGenerateServiceForRetryTest(client, 0)
	_, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if !errors.Is(err, ErrWorkflowGeneration) {
		t.Fatalf("expected ErrWorkflowGeneration, got %v", err)
	}
}

func TestGenerate_RetriesValidationFailure(t *testing.T) {
	client := &fakeWorkflowLLMClient{
		responses: []llm.ChatResponse{
			{Content: `{"nodes":[],"edges":[]}`},
			{Content: validWorkflowJSONForGenerateTest()},
		},
	}
	svc := newWorkflowGenerateServiceForRetryTest(client, 1)
	resp, err := svc.Generate(context.Background(), &GenerateRequest{CollaborationSpec: "generate a VPN request flow"})
	if err != nil {
		t.Fatalf("generate workflow: %v", err)
	}
	if client.calls != 2 || resp.Retries != 1 {
		t.Fatalf("expected one retry, got calls=%d retries=%d", client.calls, resp.Retries)
	}
}

func TestWorkflowGenerateHandlerReturnsOKForParsableWorkflowWithBlockingIssues(t *testing.T) {
	client := &fakeWorkflowLLMClient{responses: []llm.ChatResponse{{Content: workflowWithBlockingIssueForGenerateTest(42)}}}
	h := &WorkflowGenerateHandler{svc: newWorkflowGenerateServiceForRetryTest(client, 0)}
	c, rec := newGinContext(http.MethodPost, "/api/v1/itsm/workflows/generate")
	c.Request.Body = io.NopCloser(bytes.NewBufferString(`{"collaborationSpec":"generate a VPN request flow"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Generate(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Code int              `json:"code"`
		Data GenerateResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != 0 || len(got.Data.Errors) == 0 || !got.Data.Saved {
		t.Fatalf("expected parsable draft with errors to be returned, got %+v", got)
	}
}

func TestWorkflowValidationErrorsLogValue(t *testing.T) {
	got := workflowValidationErrorsLogValue([]engine.ValidationError{
		{NodeID: "gateway-1", EdgeID: "edge-2", Level: "blocking", Message: "first error"},
		{NodeID: "approve-1", Message: "second error"},
	})
	if !strings.Contains(got, "[blocking] node=gateway-1 edge=edge-2 first error") {
		t.Fatalf("expected first validation error details, got %q", got)
	}
	if !strings.Contains(got, "[blocking] node=approve-1 second error") {
		t.Fatalf("expected second validation error details, got %q", got)
	}
}

func TestWorkflowValidationErrorsLogValueTruncatesLongLists(t *testing.T) {
	got := workflowValidationErrorsLogValue([]engine.ValidationError{
		{Message: "err-1"},
		{Message: "err-2"},
		{Message: "err-3"},
		{Message: "err-4"},
		{Message: "err-5"},
		{Message: "err-6"},
	})
	if strings.Contains(got, "err-6") || !strings.Contains(got, "... 1 more") {
		t.Fatalf("expected truncation, got %q", got)
	}
}

func TestBuildGenerateResponse_PersistsWorkflowAndHealthSnapshot(t *testing.T) {
	db := newTestDB(t)
	serviceDefs := newServiceDefServiceForTest(t, db)
	catSvc := newCatalogServiceForTest(t, db)

	root, _ := catSvc.Create("Root", "root", "", "", nil, 10)
	user := createServiceHealthUser(t, db, "operator", true)
	serviceAgent := createServiceHealthAgent(t, db, "service-agent", true)
	decisionAgent := createServiceHealthAgent(t, db, "decision-agent", true)
	setServiceHealthDecisionAgent(t, db, decisionAgent.ID)
	seedServiceHealthPathEngine(t, db)
	service, err := serviceDefs.Create(&ServiceDefinition{
		Name:              "Smart",
		Code:              "smart-generate-response",
		CatalogID:         root.ID,
		EngineType:        "smart",
		IntakeFormSchema:  serviceHealthIntakeFormSchema(),
		CollaborationSpec: "collect request details and route them",
		AgentID:           &serviceAgent.ID,
		WorkflowJSON:      JSONField(validServiceHealthWorkflow(user.ID)),
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	svc := &WorkflowGenerateService{serviceDefSvc: serviceDefs}
	workflowJSON := validWorkflowDraftWithFormSchema()
	intakeFormSchema := json.RawMessage(`{"version":1,"fields":[{"key":"summary","type":"textarea","label":"Summary","required":true}]}`)
	resp, err := svc.buildGenerateResponse(&GenerateRequest{
		ServiceID:         service.ID,
		CollaborationSpec: "generate request path",
	}, workflowJSON, intakeFormSchema, 0, nil)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}
	if resp.Service == nil || resp.HealthCheck == nil || resp.Service.PublishHealthCheck == nil {
		t.Fatalf("expected service and health snapshot, got %+v", resp)
	}
	if string(resp.Service.WorkflowJSON) != string(workflowJSON) {
		t.Fatalf("expected workflow json to be saved, got %s", resp.Service.WorkflowJSON)
	}
}

func TestBuildGenerateResponse_PersistsBlockingDraftAndHealthFailure(t *testing.T) {
	db := newTestDB(t)
	serviceDefs := newServiceDefServiceForTest(t, db)
	catSvc := newCatalogServiceForTest(t, db)

	root, _ := catSvc.Create("Root", "root", "", "", nil, 10)
	user := createServiceHealthUser(t, db, "operator", true)
	serviceAgent := createServiceHealthAgent(t, db, "service-agent", true)
	decisionAgent := createServiceHealthAgent(t, db, "decision-agent", true)
	setServiceHealthDecisionAgent(t, db, decisionAgent.ID)
	seedServiceHealthPathEngine(t, db)
	service, err := serviceDefs.Create(&ServiceDefinition{
		Name:              "Smart",
		Code:              "smart-blocking-draft",
		CatalogID:         root.ID,
		EngineType:        "smart",
		CollaborationSpec: "route the request",
		AgentID:           &serviceAgent.ID,
		IntakeFormSchema:  serviceHealthIntakeFormSchema(),
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	workflowJSON := json.RawMessage(workflowWithBlockingIssueForGenerateTest(user.ID))
	validationErrors := engine.ValidateWorkflow(workflowJSON)
	svc := &WorkflowGenerateService{serviceDefSvc: serviceDefs}
	resp, err := svc.buildGenerateResponse(&GenerateRequest{
		ServiceID:         service.ID,
		CollaborationSpec: "generate request path",
	}, workflowJSON, json.RawMessage(`{"version":1,"fields":[{"key":"summary","type":"textarea","label":"Summary","required":true}]}`), 0, validationErrors)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}
	if !resp.Saved || resp.Service == nil || resp.HealthCheck == nil {
		t.Fatalf("expected saved blocking draft, got %+v", resp)
	}
	if resp.HealthCheck.Status != "fail" {
		t.Fatalf("expected fail health check, got %+v", resp.HealthCheck)
	}
}

func TestValidateGeneratedServiceContract_BOSSRequiresGoldenContract(t *testing.T) {
	errs := validateGeneratedServiceContract(bossSerialChangeServiceCode, bossContractWorkflowForTest(), bossContractFormSchemaForTest())
	if len(errs) != 0 {
		t.Fatalf("expected boss contract to pass, got %+v", errs)
	}
}

func TestValidateGeneratedServiceContract_BOSSRejectsMissingSerialReviewer(t *testing.T) {
	errs := validateGeneratedServiceContract(bossSerialChangeServiceCode, validWorkflowDraftWithFormSchema(), bossContractFormSchemaForTest())
	if len(errs) == 0 {
		t.Fatal("expected boss contract errors")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err.Message, "serial_reviewer") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected serial reviewer contract error, got %+v", errs)
	}
}
