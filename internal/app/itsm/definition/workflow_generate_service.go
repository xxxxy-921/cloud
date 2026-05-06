package definition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	. "metis/internal/app/itsm/config"
	. "metis/internal/app/itsm/domain"
	"metis/internal/app/itsm/engine"
	"metis/internal/app/itsm/form"
	"metis/internal/llm"
	"strings"
	"time"

	"github.com/samber/do/v2"
)

var (
	ErrPathEngineNotConfigured    = errors.New("path builder is not configured")
	ErrCollaborationSpecEmpty     = errors.New("collaboration spec is required")
	ErrWorkflowGeneration         = errors.New("reference path generation failed")
	ErrPathEngineUpstream         = errors.New("path builder upstream call failed")
	ErrGeneratedReferenceContract = errors.New("generated reference path does not satisfy the service contract")
)

const bossSerialChangeServiceCode = "boss-serial-change-request"

var bossRequiredFieldKeys = []string{
	"subject",
	"request_category",
	"risk_level",
	"change_window",
	"impact_scope",
	"rollback_required",
	"impact_modules",
	"change_items",
}

type pathEngineConfigProvider interface {
	PathBuilderRuntimeConfig() (LLMEngineRuntimeConfig, error)
}

type workflowLLMClientFactory func(protocol, baseURL, apiKey string) (llm.Client, error)

// WorkflowGenerateService handles path engine calls that turn collaboration specs into workflow JSON.
type WorkflowGenerateService struct {
	engineConfigSvc  pathEngineConfigProvider
	actionRepo       *ServiceActionRepo
	serviceDefSvc    *ServiceDefService
	llmClientFactory workflowLLMClientFactory
}

func NewWorkflowGenerateService(i do.Injector) (*WorkflowGenerateService, error) {
	return &WorkflowGenerateService{
		engineConfigSvc:  do.MustInvoke[*EngineConfigService](i),
		actionRepo:       do.MustInvoke[*ServiceActionRepo](i),
		serviceDefSvc:    do.MustInvoke[*ServiceDefService](i),
		llmClientFactory: llm.NewClient,
	}, nil
}

type GenerateRequest struct {
	ServiceID         uint   `json:"serviceId"`
	CollaborationSpec string `json:"collaborationSpec"`
}

type GenerateResponse struct {
	WorkflowJSON json.RawMessage            `json:"workflowJson"`
	Retries      int                        `json:"retries"`
	Errors       []engine.ValidationError   `json:"errors,omitempty"`
	Saved        bool                       `json:"saved"`
	Service      *ServiceDefinitionResponse `json:"service,omitempty"`
	HealthCheck  *ServiceHealthCheck        `json:"healthCheck,omitempty"`
}

func (s *WorkflowGenerateService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if strings.TrimSpace(req.CollaborationSpec) == "" {
		return nil, ErrCollaborationSpecEmpty
	}
	if s.engineConfigSvc == nil {
		return nil, ErrPathEngineNotConfigured
	}

	engineCfg, err := s.engineConfigSvc.PathBuilderRuntimeConfig()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPathEngineNotConfigured, err)
	}

	clientFactory := s.llmClientFactory
	if clientFactory == nil {
		clientFactory = llm.NewClient
	}
	client, err := clientFactory(engineCfg.Protocol, engineCfg.BaseURL, engineCfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create llm client: %v", ErrPathEngineNotConfigured, err)
	}

	var actionsContext string
	if req.ServiceID > 0 && s.actionRepo != nil {
		actions, listErr := s.actionRepo.ListByService(req.ServiceID)
		if listErr == nil && len(actions) > 0 {
			actionsContext = s.buildActionsContext(actions)
		}
	}

	var contractContext string
	if req.ServiceID > 0 && s.serviceDefSvc != nil {
		if svc, getErr := s.serviceDefSvc.Get(req.ServiceID); getErr == nil {
			contractContext = buildWorkflowGenerationContractContext(svc)
		}
	}

	maxRetries := engineCfg.MaxRetries
	temp := float32(engineCfg.Temperature)
	systemPrompt := strings.TrimSpace(engineCfg.SystemPrompt)

	var lastWorkflowJSON json.RawMessage
	var lastErrors []engine.ValidationError

	for attempt := 0; attempt <= maxRetries; attempt++ {
		userMsg := s.buildUserMessage(req.CollaborationSpec, actionsContext+contractContext, lastErrors)
		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: systemPrompt},
			{Role: llm.RoleUser, Content: userMsg},
		}

		timeoutSec := engineCfg.TimeoutSeconds
		callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		resp, callErr := client.Chat(callCtx, llm.ChatRequest{
			Model:          engineCfg.Model,
			Messages:       messages,
			Temperature:    &temp,
			MaxTokens:      engineCfg.MaxTokens,
			ResponseFormat: &llm.ResponseFormat{Type: "json_object"},
		})
		cancel()
		if callErr != nil {
			slog.Warn("workflow generate: llm call failed", "attempt", attempt+1, "error", callErr)
			if attempt < maxRetries && isRetriablePathEngineError(callErr) {
				lastErrors = []engine.ValidationError{{
					Level:   "blocking",
					Message: fmt.Sprintf("path builder upstream attempt %d failed: %v", attempt+1, callErr),
				}}
				continue
			}
			return nil, fmt.Errorf("%w: %s", ErrPathEngineUpstream, normalizePathEngineUpstreamMessage(callErr))
		}

		workflowJSON, extractErr := extractJSON(resp.Content)
		if extractErr != nil {
			slog.Warn("workflow generate: json extraction failed", "attempt", attempt+1, "error", extractErr)
			lastErrors = []engine.ValidationError{{
				Level:   "blocking",
				Message: fmt.Sprintf("failed to parse generated JSON: %v", extractErr),
			}}
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("%w: generated content is not valid JSON", ErrWorkflowGeneration)
		}

		validationErrors := engine.ValidateWorkflow(workflowJSON)
		intakeFormSchema, formErrors := extractGeneratedIntakeFormSchema(workflowJSON)
		validationErrors = append(validationErrors, formErrors...)
		validationErrors = append(validationErrors, s.validateGeneratedContract(req.ServiceID, workflowJSON, intakeFormSchema)...)
		lastWorkflowJSON = workflowJSON

		if len(validationErrors) == 0 || !hasBlockingErrors(validationErrors) {
			return s.buildGenerateResponse(req, workflowJSON, intakeFormSchema, attempt, validationErrors)
		}

		slog.Warn(
			"workflow generate: validation failed",
			"attempt", attempt+1,
			"maxRetries", maxRetries,
			"retrying", attempt < maxRetries,
			"errorCount", len(validationErrors),
			"validationErrors", workflowValidationErrorsLogValue(validationErrors),
		)
		lastErrors = validationErrors

		if attempt < maxRetries {
			continue
		}

		if len(formErrors) > 0 {
			return nil, fmt.Errorf("%w: generated workflow is missing a valid form schema", ErrWorkflowGeneration)
		}
		return s.buildGenerateResponse(req, lastWorkflowJSON, intakeFormSchema, attempt, validationErrors)
	}

	return nil, ErrWorkflowGeneration
}

func (s *WorkflowGenerateService) validateGeneratedContract(serviceID uint, workflowJSON json.RawMessage, intakeFormSchema json.RawMessage) []engine.ValidationError {
	serviceCode := ""
	if serviceID > 0 && s.serviceDefSvc != nil {
		if svc, err := s.serviceDefSvc.Get(serviceID); err == nil {
			serviceCode = svc.Code
		}
	}
	errs := validateGeneratedServiceContract(serviceCode, workflowJSON, intakeFormSchema)
	if serviceID > 0 && s.serviceDefSvc != nil {
		if def, err := engine.ParseWorkflowDef(workflowJSON); err == nil {
			if issue := s.serviceDefSvc.checkWorkflowParticipantAvailability(def); issue != nil {
				errs = append(errs, engine.ValidationError{
					Level:   "blocking",
					Message: issue.Message,
				})
			}
		}
	}
	return errs
}

func validateGeneratedServiceContract(serviceCode string, workflowJSON json.RawMessage, intakeFormSchema json.RawMessage) []engine.ValidationError {
	var errs []engine.ValidationError
	def, parseErr := engine.ParseWorkflowDef(workflowJSON)
	if parseErr != nil {
		errs = append(errs, engine.ValidationError{
			Level:   "blocking",
			Message: fmt.Sprintf("generated workflow cannot be parsed: %v", parseErr),
		})
		return errs
	}

	switch serviceCode {
	case bossSerialChangeServiceCode:
		if len(intakeFormSchema) == 0 || string(intakeFormSchema) == "null" {
			return []engine.ValidationError{{Level: "blocking", Message: "BOSS contract requires an intake form schema"}}
		}
		fieldKeys, fieldErr := generatedFormFieldKeys(intakeFormSchema)
		if fieldErr != nil {
			return []engine.ValidationError{{Level: "blocking", Message: fieldErr.Error()}}
		}
		if !workflowHasFormNode(def) {
			errs = append(errs, engine.ValidationError{Level: "blocking", Message: "BOSS contract requires a form node"})
		}
		if !workflowHasParticipantDrivenNode(def) {
			errs = append(errs, engine.ValidationError{Level: "blocking", Message: "BOSS contract requires participant-driven approval nodes"})
		}
		errs = append(errs, validateBossGeneratedContract(def, fieldKeys)...)
	}
	return errs
}

func validateBossGeneratedContract(def *engine.WorkflowDef, fieldKeys map[string]struct{}) []engine.ValidationError {
	var errs []engine.ValidationError
	for _, key := range bossRequiredFieldKeys {
		if _, ok := fieldKeys[key]; !ok {
			errs = append(errs, engine.ValidationError{
				Level:   "blocking",
				Message: fmt.Sprintf("BOSS contract requires intake form field %q", key),
			})
		}
	}

	if !workflowHasPositionDepartment(def, "headquarters", "serial_reviewer") {
		errs = append(errs, engine.ValidationError{
			Level:   "blocking",
			Message: "BOSS contract requires a headquarters.serial_reviewer approval node",
		})
	}
	if !workflowHasPositionDepartment(def, "it", "ops_admin") {
		errs = append(errs, engine.ValidationError{
			Level:   "blocking",
			Message: "BOSS contract requires an it.ops_admin approval node",
		})
	}
	return errs
}

func generatedFormFieldKeys(intakeFormSchema json.RawMessage) (map[string]struct{}, error) {
	var schema struct {
		Fields []struct {
			Key string `json:"key"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(intakeFormSchema, &schema); err != nil {
		return nil, fmt.Errorf("generated intake form schema is invalid: %w", err)
	}
	if len(schema.Fields) == 0 {
		return nil, errors.New("generated intake form schema must define at least one field")
	}
	keys := make(map[string]struct{}, len(schema.Fields))
	for _, field := range schema.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return nil, errors.New("generated intake form schema contains an empty field key")
		}
		keys[key] = struct{}{}
	}
	return keys, nil
}

func workflowHasFormNode(def *engine.WorkflowDef) bool {
	if def == nil {
		return false
	}
	for _, node := range def.Nodes {
		if node.Type == engine.NodeForm {
			return true
		}
	}
	return false
}

func workflowHasParticipantDrivenNode(def *engine.WorkflowDef) bool {
	if def == nil {
		return false
	}
	for _, node := range def.Nodes {
		if node.Type != engine.NodeForm && node.Type != engine.NodeApprove && node.Type != engine.NodeProcess {
			continue
		}
		data, err := engine.ParseNodeData(node.Data)
		if err == nil && len(data.Participants) > 0 {
			return true
		}
	}
	return false
}

func workflowHasPositionDepartment(def *engine.WorkflowDef, departmentCode string, positionCode string) bool {
	if def == nil {
		return false
	}
	for _, node := range def.Nodes {
		if node.Type != engine.NodeApprove && node.Type != engine.NodeProcess {
			continue
		}
		data, err := engine.ParseNodeData(node.Data)
		if err != nil {
			continue
		}
		for _, participant := range data.Participants {
			if participant.Type != "position_department" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(participant.DepartmentCode), departmentCode) &&
				strings.EqualFold(strings.TrimSpace(participant.PositionCode), positionCode) {
				return true
			}
		}
	}
	return false
}

func workflowValidationErrorsLogValue(validationErrors []engine.ValidationError) string {
	const maxLoggedErrors = 5
	if len(validationErrors) == 0 {
		return ""
	}

	limit := len(validationErrors)
	if limit > maxLoggedErrors {
		limit = maxLoggedErrors
	}

	var sb strings.Builder
	for i := 0; i < limit; i++ {
		if i > 0 {
			sb.WriteString("; ")
		}
		validationErr := validationErrors[i]
		level := validationErr.Level
		if level == "" {
			level = "blocking"
		}
		sb.WriteString("[")
		sb.WriteString(level)
		sb.WriteString("]")
		if validationErr.NodeID != "" {
			sb.WriteString(" node=")
			sb.WriteString(validationErr.NodeID)
		}
		if validationErr.EdgeID != "" {
			sb.WriteString(" edge=")
			sb.WriteString(validationErr.EdgeID)
		}
		if validationErr.Message != "" {
			sb.WriteString(" ")
			sb.WriteString(validationErr.Message)
		}
	}
	if len(validationErrors) > limit {
		sb.WriteString(fmt.Sprintf("; ... %d more", len(validationErrors)-limit))
	}
	return sb.String()
}

func (s *WorkflowGenerateService) buildGenerateResponse(req *GenerateRequest, workflowJSON json.RawMessage, intakeFormSchema json.RawMessage, retries int, validationErrors []engine.ValidationError) (*GenerateResponse, error) {
	resp := &GenerateResponse{
		WorkflowJSON: workflowJSON,
		Retries:      retries,
		Errors:       validationErrors,
	}

	if req.ServiceID == 0 || s.serviceDefSvc == nil {
		resp.Saved = true
		return resp, nil
	}

	if len(intakeFormSchema) == 0 {
		var formErrors []engine.ValidationError
		intakeFormSchema, formErrors = extractGeneratedIntakeFormSchema(workflowJSON)
		if len(formErrors) > 0 {
			return nil, fmt.Errorf("%w: generated workflow is missing a valid form schema", ErrGeneratedReferenceContract)
		}
	}

	updated, err := s.serviceDefSvc.Update(req.ServiceID, map[string]any{
		"workflow_json":      JSONField(workflowJSON),
		"collaboration_spec": req.CollaborationSpec,
		"intake_form_schema": JSONField(intakeFormSchema),
	})
	if err != nil {
		return nil, err
	}
	health, err := s.serviceDefSvc.RefreshPublishHealthCheck(req.ServiceID)
	if err != nil {
		return nil, err
	}
	updated, err = s.serviceDefSvc.Get(updated.ID)
	if err != nil {
		return nil, err
	}
	serviceResp := updated.ToResponse()
	resp.Service = &serviceResp
	resp.HealthCheck = health
	resp.Saved = true
	return resp, nil
}

func extractGeneratedIntakeFormSchema(workflowJSON json.RawMessage) (json.RawMessage, []engine.ValidationError) {
	var workflow struct {
		Nodes []struct {
			ID   string          `json:"id"`
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			Message: fmt.Sprintf("generated workflow json is invalid: %v", err),
		}}
	}

	for _, node := range workflow.Nodes {
		if node.Type != "form" {
			continue
		}
		var data struct {
			FormSchema json.RawMessage `json:"formSchema"`
		}
		if err := json.Unmarshal(node.Data, &data); err != nil {
			return nil, []engine.ValidationError{{
				Level:   "blocking",
				NodeID:  node.ID,
				Message: fmt.Sprintf("form node %s contains invalid data: %v", node.ID, err),
			}}
		}
		if len(data.FormSchema) == 0 || string(data.FormSchema) == "null" {
			continue
		}
		return normalizeGeneratedFormSchema(node.ID, data.FormSchema)
	}

	return nil, []engine.ValidationError{{
		Level:   "blocking",
		Message: "generated workflow must include a form node with formSchema",
	}}
}

func normalizeGeneratedFormSchema(nodeID string, raw json.RawMessage) (json.RawMessage, []engine.ValidationError) {
	var schemaMap map[string]any
	if err := json.Unmarshal(raw, &schemaMap); err != nil {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			NodeID:  nodeID,
			Message: fmt.Sprintf("formSchema in node %s is not valid JSON: %v", nodeID, err),
		}}
	}
	if schemaMap["version"] == nil {
		schemaMap["version"] = float64(1)
	}
	fields, ok := schemaMap["fields"].([]any)
	if !ok || len(fields) == 0 {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			NodeID:  nodeID,
			Message: "formSchema.fields must contain at least one field",
		}}
	}
	for _, rawField := range fields {
		field, ok := rawField.(map[string]any)
		if !ok {
			continue
		}
		if required, ok := field["required"].(bool); !ok || !required {
			field["required"] = true
		}
		if normalized := normalizeGeneratedOptions(field["options"]); normalized != nil {
			field["options"] = normalized
		}
	}

	normalized, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			NodeID:  nodeID,
			Message: fmt.Sprintf("failed to normalize formSchema in node %s: %v", nodeID, err),
		}}
	}
	var schema form.FormSchema
	if err := json.Unmarshal(normalized, &schema); err != nil {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			NodeID:  nodeID,
			Message: fmt.Sprintf("normalized formSchema in node %s is invalid: %v", nodeID, err),
		}}
	}
	if errs := form.ValidateSchema(schema); len(errs) > 0 {
		validationErrors := make([]engine.ValidationError, 0, len(errs))
		for _, err := range errs {
			validationErrors = append(validationErrors, engine.ValidationError{
				Level:   "blocking",
				NodeID:  nodeID,
				Message: "formSchema " + err.Error(),
			})
		}
		return nil, validationErrors
	}
	canonical, err := json.Marshal(schema)
	if err != nil {
		return nil, []engine.ValidationError{{
			Level:   "blocking",
			NodeID:  nodeID,
			Message: fmt.Sprintf("failed to marshal canonical formSchema in node %s: %v", nodeID, err),
		}}
	}
	return canonical, nil
}

func normalizeGeneratedOptions(raw any) any {
	options, ok := raw.([]any)
	if !ok || len(options) == 0 {
		return raw
	}
	normalized := make([]any, 0, len(options))
	for _, option := range options {
		switch value := option.(type) {
		case string:
			normalized = append(normalized, map[string]any{"label": value, "value": value})
		case map[string]any:
			if value["value"] == nil && value["label"] != nil {
				value["value"] = value["label"]
			}
			if value["label"] == nil && value["value"] != nil {
				value["label"] = fmt.Sprintf("%v", value["value"])
			}
			normalized = append(normalized, value)
		default:
			label := fmt.Sprintf("%v", value)
			normalized = append(normalized, map[string]any{"label": label, "value": label})
		}
	}
	return normalized
}

func hasBlockingErrors(errs []engine.ValidationError) bool {
	for _, e := range errs {
		if !e.IsWarning() {
			return true
		}
	}
	return false
}

func (s *WorkflowGenerateService) buildActionsContext(actions []ServiceAction) string {
	var sb strings.Builder
	sb.WriteString("\n\n## Available service actions\n")
	sb.WriteString("Only reference action nodes that are backed by one of the actions below.\n\n")
	for _, a := range actions {
		sb.WriteString(fmt.Sprintf("- **%s** (id: `%d`, code: `%s`)", a.Name, a.ID, a.Code))
		if a.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", a.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func buildWorkflowGenerationContractContext(svc *ServiceDefinition) string {
	if svc == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Service contract\n\n")
	if code := strings.TrimSpace(svc.Code); code != "" {
		sb.WriteString("- Service code: `")
		sb.WriteString(code)
		sb.WriteString("`\n")
	}
	sb.WriteString("- The generated workflow must include at least one `type=\"form\"` node.\n")
	sb.WriteString("- The form node must embed a non-null `data.formSchema` object.\n")
	sb.WriteString("- Reuse the existing intake form contract instead of inventing different field keys.\n")

	if len(svc.IntakeFormSchema) > 0 {
		sb.WriteString("\n### Existing intake form schema\n\n")
		sb.WriteString(string(svc.IntakeFormSchema))
		sb.WriteString("\n")
	}

	if len(svc.WorkflowJSON) > 0 {
		sb.WriteString("\n### Existing reference path\n\n")
		sb.WriteString(string(svc.WorkflowJSON))
		sb.WriteString("\n")
	}

	if svc.Code == bossSerialChangeServiceCode {
		sb.WriteString("\n### BOSS contract rules\n\n")
		sb.WriteString("- Keep the BOSS intake field keys exactly as defined in the intake form schema.\n")
		for _, key := range bossRequiredFieldKeys {
			sb.WriteString("- Required intake field key: `")
			sb.WriteString(key)
			sb.WriteString("`\n")
		}
		sb.WriteString("- Include a participant-driven node assigned to `headquarters.serial_reviewer`.\n")
		sb.WriteString("- Include a participant-driven node assigned to `it.ops_admin`.\n")
		sb.WriteString("- The BOSS flow is serial: requester form -> headquarters.serial_reviewer -> it.ops_admin -> end.\n")
	}

	return sb.String()
}

func (s *WorkflowGenerateService) buildUserMessage(spec string, actionsCtx string, prevErrors []engine.ValidationError) string {
	var sb strings.Builder
	sb.WriteString("Generate a workflow JSON draft from the collaboration spec below.\n\n")
	sb.WriteString("## Collaboration Spec\n\n")
	sb.WriteString(spec)

	if actionsCtx != "" {
		sb.WriteString(actionsCtx)
	}

	if len(prevErrors) > 0 {
		sb.WriteString("\n\n## Fix the following validation issues before regenerating\n\n")
		for _, e := range prevErrors {
			prefix := ""
			if e.NodeID != "" {
				prefix = fmt.Sprintf("[node %s] ", e.NodeID)
			} else if e.EdgeID != "" {
				prefix = fmt.Sprintf("[edge %s] ", e.EdgeID)
			}
			sb.WriteString(fmt.Sprintf("- %s%s\n", prefix, e.Message))
		}
		if validationErrorsRequireParticipantRepair(prevErrors) {
			sb.WriteString("\n## Participant repair rules\n\n")
			sb.WriteString("- Every form/process/approve node must use `data.participants`.\n")
			sb.WriteString("- Use canonical participant fields only: `type`, `value`, `position_code`, `department_code`.\n")
			sb.WriteString("- The requester node must look exactly like this: ")
			sb.WriteString(`"participants":[{"type":"requester"}]`)
			sb.WriteString("\n")
			sb.WriteString("- Position+department assignments must look exactly like this: ")
			sb.WriteString(`"participants":[{"type":"position_department","department_code":"it","position_code":"network_admin"}]`)
			sb.WriteString("\n")
		}
		if validationErrorsRequireActionRepair(prevErrors) {
			sb.WriteString("\n## Action repair rules\n\n")
			sb.WriteString("- Every `type=\"action\"` node must bind a real `action_id` from the available service actions list.\n")
			sb.WriteString("- Do not put executable service actions into process/notify/end nodes.\n")
		}
	}

	sb.WriteString("\n\nReturn JSON only. Do not wrap the response in markdown.")
	return sb.String()
}

func (s *WorkflowGenerateService) BuildUserMessage(spec string, actionsCtx string, prevErrors []engine.ValidationError) string {
	return s.buildUserMessage(spec, actionsCtx, prevErrors)
}

func validationErrorsRequireParticipantRepair(validationErrors []engine.ValidationError) bool {
	for _, validationErr := range validationErrors {
		msg := strings.ToLower(validationErr.Message)
		if strings.Contains(msg, "participant") ||
			strings.Contains(msg, "participants") ||
			strings.Contains(msg, "position_code") ||
			strings.Contains(msg, "department_code") {
			return true
		}
	}
	return false
}

func validationErrorsRequireActionRepair(validationErrors []engine.ValidationError) bool {
	for _, validationErr := range validationErrors {
		msg := strings.ToLower(validationErr.Message)
		if strings.Contains(msg, "action_id") || strings.Contains(msg, "action") {
			return true
		}
	}
	return false
}

func extractJSON(content string) (json.RawMessage, error) {
	content = strings.TrimSpace(content)
	if json.Valid([]byte(content)) {
		return json.RawMessage(content), nil
	}

	if idx := strings.Index(content, "```"); idx >= 0 {
		start := idx + 3
		if nl := strings.Index(content[start:], "\n"); nl >= 0 {
			start += nl + 1
		}
		if end := strings.Index(content[start:], "```"); end >= 0 {
			candidate := strings.TrimSpace(content[start : start+end])
			if json.Valid([]byte(candidate)) {
				return json.RawMessage(candidate), nil
			}
		}
	}

	first := strings.Index(content, "{")
	last := strings.LastIndex(content, "}")
	if first >= 0 && last > first {
		candidate := content[first : last+1]
		if json.Valid([]byte(candidate)) {
			return json.RawMessage(candidate), nil
		}
	}

	return nil, fmt.Errorf("no valid JSON object found in generated content")
}

func ExtractJSON(content string) (json.RawMessage, error) {
	return extractJSON(content)
}

func isRetriablePathEngineError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(msg, "status code: 429") ||
		strings.Contains(msg, "status code: 500") ||
		strings.Contains(msg, "status code: 502") ||
		strings.Contains(msg, "status code: 503") ||
		strings.Contains(msg, "status code: 504") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporar")
}

func normalizePathEngineUpstreamMessage(err error) string {
	if err == nil {
		return "upstream model call failed"
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "status code: 502") && strings.Contains(lower, "unexpected end of json input") {
		return "upstream gateway returned an empty or invalid 502 response body"
	}
	if strings.Contains(lower, "status code: 502") {
		return "upstream gateway returned 502 Bad Gateway"
	}
	if strings.Contains(lower, "status code: 503") {
		return "upstream model service is temporarily unavailable"
	}
	if strings.Contains(lower, "status code: 504") || strings.Contains(lower, "timeout") {
		return "upstream model service timed out"
	}
	if strings.Contains(lower, "status code: 429") {
		return "upstream model service is rate limited"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "path builder request timed out"
	}
	return msg
}
