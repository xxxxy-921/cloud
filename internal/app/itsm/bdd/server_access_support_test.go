package bdd

// server_access_support_test.go — Server access workflow LLM generation and service publish helpers for BDD tests.
//
// Uses the LLM (gated by LLM_TEST_* env vars) to generate the server access workflow
// from the collaboration spec, matching the VPN BDD approach.

import (
	"context"
	"encoding/json"
	"fmt"
	. "metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	"time"

	ai "metis/internal/app/ai/runtime"
	"metis/internal/app/itsm/engine"
	"metis/internal/llm"
)

// serverAccessCollaborationSpec is the collaboration spec for the production server temporary access service.
const serverAccessCollaborationSpec = `这是一个生产服务器临时访问申请服务。
用户来找服务台时，先把访问账号、目标主机、来源 IP、访问时段和访问目的这些信息问清楚，再整理成可以确认的申请摘要。
常见的应用排障、主机巡检、日志查看、进程处理、磁盘清理和一般生产运维访问，交给信息部的运维管理员岗位处理，处理参与者类型必须使用 position_department，部门编码使用 it，岗位编码使用 ops_admin。
网络抓包、链路诊断、ACL 调整、负载均衡检查、防火墙策略核对和其他网络侧访问，交给信息部的网络管理员岗位处理，处理参与者类型必须使用 position_department，部门编码使用 it，岗位编码使用 network_admin。
安全审计、取证分析、漏洞修复验证、入侵排查、合规核查和其他高敏访问，交给信息部的安全管理员岗位处理，处理参与者类型必须使用 position_department，部门编码使用 it，岗位编码使用 security_admin。
不要让申请人在表单里自己选择处理类别，流程决策智能体应根据访问目的和访问原因在运行时判断应该流转到哪个处理岗位。
处理完成后直接结束流程，不需要额外生成取消分支。`

const serverAccessWorkflowJSONBDD = `{"nodes":[{"id":"start","type":"start","position":{"x":400,"y":50},"data":{"label":"开始","nodeType":"start"}},{"id":"request","type":"form","position":{"x":400,"y":200},"data":{"label":"填写服务器临时访问申请","nodeType":"form","participants":[{"type":"requester"}],"formSchema":{"fields":[{"key":"target_servers","type":"textarea","label":"访问服务器"},{"key":"access_window","type":"date_range","label":"访问时段"},{"key":"operation_purpose","type":"textarea","label":"操作目的"},{"key":"access_reason","type":"textarea","label":"访问原因"}]}}},{"id":"route","type":"exclusive","position":{"x":400,"y":380},"data":{"label":"访问原因智能参考路由","nodeType":"exclusive"}},{"id":"ops_process","type":"process","position":{"x":120,"y":580},"data":{"label":"运维管理员处理","nodeType":"process","participants":[{"type":"position_department","department_code":"it","position_code":"ops_admin"}]}},{"id":"network_process","type":"process","position":{"x":400,"y":580},"data":{"label":"网络管理员处理","nodeType":"process","participants":[{"type":"position_department","department_code":"it","position_code":"network_admin"}]}},{"id":"security_process","type":"process","position":{"x":680,"y":580},"data":{"label":"安全管理员处理","nodeType":"process","participants":[{"type":"position_department","department_code":"it","position_code":"security_admin"}]}},{"id":"end","type":"end","position":{"x":400,"y":820},"data":{"label":"结束","nodeType":"end"}}],"edges":[{"id":"edge_start_request","source":"start","target":"request"},{"id":"edge_request_route","source":"request","target":"route"},{"id":"edge_route_ops","source":"route","target":"ops_process","data":{"condition":{"field":"form.access_reason","operator":"contains_any","value":["应用发布","进程排障","进程排查","日志排查","日志查看","日志排障","磁盘清理","清理磁盘","主机巡检","生产运维操作"],"edge_id":"edge_route_ops"}}},{"id":"edge_route_network","source":"route","target":"network_process","data":{"condition":{"field":"form.access_reason","operator":"contains_any","value":["网络抓包","抓包分析","连通性诊断","链路诊断","ACL调整","访问控制列表调整","负载均衡变更","LB变更","防火墙策略调整","防火墙规则调整"],"edge_id":"edge_route_network"}}},{"id":"edge_route_security","source":"route","target":"security_process","data":{"condition":{"field":"form.access_reason","operator":"contains_any","value":["安全审计","安全核查","入侵排查","异常访问核查","漏洞修复验证","漏洞复测","取证分析","证据保全","合规检查","合规审查"],"edge_id":"edge_route_security"}}},{"id":"edge_route_default","source":"route","target":"security_process","data":{"default":true}},{"id":"edge_ops_end","source":"ops_process","target":"end","data":{"outcome":"approved"}},{"id":"edge_ops_rejected","source":"ops_process","target":"request","data":{"outcome":"rejected"}},{"id":"edge_network_end","source":"network_process","target":"end","data":{"outcome":"approved"}},{"id":"edge_network_rejected","source":"network_process","target":"request","data":{"outcome":"rejected"}},{"id":"edge_security_end","source":"security_process","target":"end","data":{"outcome":"approved"}},{"id":"edge_security_rejected","source":"security_process","target":"request","data":{"outcome":"rejected"}}]}`

// serverAccessCasePayload defines test data for a single server access BDD scenario.
type serverAccessCasePayload struct {
	Summary          string
	OpenMessage      string
	FormData         map[string]any
	ExpectedPosition string // expected position code for routing assertion
}

// serverAccessCasePayloads covers the baseline 3-way routing and the free-text variants
// we need to keep stable for production server access requests.
var serverAccessCasePayloads = map[string]serverAccessCasePayload{
	"ops": {
		Summary:     "生产服务器临时访问申请：需要登录生产机排查应用进程异常。",
		OpenMessage: "生产环境一台应用主机进程异常，我需要临时上去看日志并处理。",
		FormData: map[string]any{
			"access_account": "ops.reader",
			"target_host":    "prod-app-02",
			"source_ip":      "10.20.30.41",
			"access_window":  "今晚 20:00 到 21:00",
			"access_purpose": "排查生产应用进程异常，确认日志和运行状态。",
			"access_reason":  "生产发布后需要进程排障并查看日志。",
		},
		ExpectedPosition: "ops_admin",
	},
	"network": {
		Summary:     "生产服务器临时访问申请：需要登录生产机配合网络链路诊断。",
		OpenMessage: "我们要抓包核对生产链路连通性，请先帮我整理访问申请。",
		FormData: map[string]any{
			"access_account": "net.trace",
			"target_host":    "prod-gateway-01",
			"source_ip":      "10.20.30.42",
			"access_window":  "今晚 21:00 到 22:30",
			"access_purpose": "配合抓包和链路诊断，核对负载均衡后的网络访问路径。",
			"access_reason":  "需要网络抓包和连通性诊断，确认链路路径。",
		},
		ExpectedPosition: "network_admin",
	},
	"security": {
		Summary:     "生产服务器临时访问申请：需要进入生产机做安全审计取证分析。",
		OpenMessage: "安全这边要上生产机核查审计痕迹并做取证分析，先帮我整理申请。",
		FormData: map[string]any{
			"access_account": "sec.audit",
			"target_host":    "prod-app-03",
			"source_ip":      "10.20.30.43",
			"access_window":  "今晚 23:00 到 23:45",
			"access_purpose": "核查安全审计痕迹并完成取证分析，确认是否存在异常访问。",
			"access_reason":  "需要安全审计和取证分析，确认是否存在异常访问。",
		},
		ExpectedPosition: "security_admin",
	},
	"boundary_security": {
		Summary:     "生产服务器临时访问申请：需要在异常访问核查过程中进入生产机保全证据。",
		OpenMessage: "这次不是单纯排障，我需要上生产机先核对异常访问痕迹并保全证据。",
		FormData: map[string]any{
			"access_account": "sec.boundary",
			"target_host":    "prod-app-04",
			"source_ip":      "10.20.30.44",
			"access_window":  "今晚 19:30 到 20:30",
			"access_purpose": "结合异常访问核查、日志固定和证据保全判断是否需要进一步安全处置。",
			"access_reason":  "异常访问核查并做取证分析，同时保全相关证据。",
		},
		ExpectedPosition: "security_admin",
	},
	"ops_synonym": {
		Summary:     "生产服务器临时访问申请：需要登录生产机做进程排查。",
		OpenMessage: "需要临时登录生产主机做进程排查。",
		FormData: map[string]any{
			"access_account": "ops.synonym",
			"target_host":    "prod-app-05",
			"source_ip":      "10.20.30.45",
			"access_window":  "今晚 20:00 到 21:00",
			"access_purpose": "排查服务运行状态。",
			"access_reason":  "进程排查",
		},
		ExpectedPosition: "ops_admin",
	},
	"ops_reordered": {
		Summary:     "生产服务器临时访问申请：需要登录生产机清理磁盘空间。",
		OpenMessage: "需要临时登录生产主机清理磁盘空间。",
		FormData: map[string]any{
			"access_account": "ops.disk",
			"target_host":    "prod-app-06",
			"source_ip":      "10.20.30.46",
			"access_window":  "今晚 21:00 到 22:00",
			"access_purpose": "释放磁盘空间。",
			"access_reason":  "清理磁盘",
		},
		ExpectedPosition: "ops_admin",
	},
	"ops_long_sentence": {
		Summary:     "生产服务器临时访问申请：发布后检查进程和日志。",
		OpenMessage: "需要在发布后检查进程和日志。",
		FormData: map[string]any{
			"access_account": "ops.long",
			"target_host":    "prod-app-07",
			"source_ip":      "10.20.30.47",
			"access_window":  "今晚 22:00 到 23:00",
			"access_purpose": "检查发布后运行状态。",
			"access_reason":  "生产发布后需要进程排查并查看日志",
		},
		ExpectedPosition: "ops_admin",
	},
	"network_combined": {
		Summary:     "生产服务器临时访问申请：网络联合诊断。",
		OpenMessage: "需要抓包并确认链路连通性。",
		FormData: map[string]any{
			"access_account": "net.combo",
			"target_host":    "prod-gateway-02",
			"source_ip":      "10.20.30.48",
			"access_window":  "今晚 23:00 到 23:30",
			"access_purpose": "定位网络访问异常。",
			"access_reason":  "网络抓包和连通性诊断",
		},
		ExpectedPosition: "network_admin",
	},
	"security_combined": {
		Summary:     "生产服务器临时访问申请：异常访问安全核查。",
		OpenMessage: "需要核查异常访问并保全证据。",
		FormData: map[string]any{
			"access_account": "sec.combo",
			"target_host":    "prod-app-08",
			"source_ip":      "10.20.30.49",
			"access_window":  "今晚 23:30 到 23:59",
			"access_purpose": "判断是否需要进一步安全处置。",
			"access_reason":  "异常访问核查并做取证分析",
		},
		ExpectedPosition: "security_admin",
	},
	"default_security": {
		Summary:     "生产服务器临时访问申请：未分类临时事项。",
		OpenMessage: "需要处理一个未分类临时事项。",
		FormData: map[string]any{
			"access_account": "sec.default",
			"target_host":    "prod-app-09",
			"source_ip":      "10.20.30.50",
			"access_window":  "明晚 20:00 到 20:30",
			"access_purpose": "处理临时事项。",
			"access_reason":  "临时上机处理一个未分类事项",
		},
		ExpectedPosition: "security_admin",
	},
}

// generateServerAccessWorkflow calls the LLM to generate a server access workflow JSON
// from the collaboration spec. Same pattern as generateVPNWorkflow.
func generateServerAccessWorkflow(cfg llmConfig) (json.RawMessage, error) {
	client, err := llm.NewClient(llm.ProtocolOpenAI, cfg.baseURL, cfg.apiKey)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	svc := &WorkflowGenerateService{}
	maxRetries := 3

	var lastErrors []engine.ValidationError

	for attempt := 0; attempt <= maxRetries; attempt++ {
		userMsg := svc.BuildUserMessage(serverAccessCollaborationSpec, "", lastErrors)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		resp, err := client.Chat(ctx, llm.ChatRequest{
			Model: cfg.model,
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: PathBuilderSystemPrompt},
				{Role: llm.RoleUser, Content: userMsg},
			},
			MaxTokens: 4096,
		})
		cancel()

		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("LLM call failed after %d attempts: %w", attempt+1, err)
		}

		workflowJSON, extractErr := ExtractJSON(resp.Content)
		if extractErr != nil {
			lastErrors = []engine.ValidationError{
				{Message: fmt.Sprintf("输出不是有效 JSON: %v", extractErr)},
			}
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("JSON extraction failed after %d attempts: %w", attempt+1, extractErr)
		}

		validationErrors := engine.ValidateWorkflow(workflowJSON)
		var blockingErrors []engine.ValidationError
		for _, e := range validationErrors {
			if !e.IsWarning() {
				blockingErrors = append(blockingErrors, e)
			}
		}

		if len(blockingErrors) == 0 {
			return workflowJSON, nil
		}

		lastErrors = blockingErrors
		if attempt < maxRetries {
			continue
		}

		return nil, fmt.Errorf("workflow validation failed after %d attempts: %v", attempt+1, blockingErrors)
	}

	return nil, fmt.Errorf("workflow generation failed")
}

// publishServerAccessSmartService creates the full service configuration for server access BDD tests.
// We publish a fixed workflow here so routing regression cases stay deterministic while
// the smart engine still uses the decision agent at runtime.
func publishServerAccessSmartService(bc *bddContext) error {
	workflowJSON := json.RawMessage(serverAccessWorkflowJSONBDD)

	// 2. ServiceCatalog
	catalog := &ServiceCatalog{
		Name:     "基础设施服务",
		Code:     "infra-compute",
		IsActive: true,
	}
	if err := bc.db.Create(catalog).Error; err != nil {
		return fmt.Errorf("create service catalog: %w", err)
	}

	// 3. Priority
	priority := &Priority{
		Name:     "紧急",
		Code:     "urgent-sa",
		Value:    2,
		Color:    "#f5222d",
		IsActive: true,
	}
	if err := bc.db.Create(priority).Error; err != nil {
		return fmt.Errorf("create priority: %w", err)
	}
	bc.priority = priority

	// 4. Seed Agent record (process decision agent)
	agent := &ai.Agent{
		Name:         "流程决策智能体",
		Type:         "assistant",
		IsActive:     true,
		Visibility:   "private",
		Strategy:     "react",
		SystemPrompt: decisionAgentSystemPrompt,
		MaxTokens:    2048,
		MaxTurns:     1,
		CreatedBy:    1,
	}
	if err := bc.db.Create(agent).Error; err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	// Override GORM default temperature to 0 (don't send temperature to models that reject it)
	bc.db.Model(agent).Update("temperature", 0)

	// 5. ServiceDefinition with engine_type=smart
	svc := &ServiceDefinition{
		Name:              "生产服务器临时访问申请",
		Code:              "server-access-request",
		CatalogID:         catalog.ID,
		EngineType:        "smart",
		WorkflowJSON:      JSONField(workflowJSON),
		CollaborationSpec: serverAccessCollaborationSpec,
		AgentID:           &agent.ID,
		IsActive:          true,
	}
	if err := bc.db.Create(svc).Error; err != nil {
		return fmt.Errorf("create service definition: %w", err)
	}
	bc.service = svc

	return nil
}
