package engine

import (
	"context"
	"encoding/json"
	"errors"
	appcore "metis/internal/app"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestBuildInitialSeedIncludesDecisionTrigger(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		code text,
		title text,
		description text,
		status text,
		outcome text,
		source text,
		priority_id integer,
		form_data text
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
		t.Fatalf("create priorities: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '紧急')`).Error; err != nil {
		t.Fatalf("insert priority: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, code, title, description, status, outcome, source, priority_id) VALUES (42, 'TICK-42', 'VPN', '线上支持', 'decisioning', '', 'agent', 1)`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}

	completedActivityID := uint(9)
	engine := &SmartEngine{}
	systemMsg, userMsg, err := engine.buildInitialSeed(db, 42, &serviceModel{
		ID:                7,
		Name:              "VPN 开通申请",
		Description:       "VPN service",
		CollaborationSpec: "处理完成后结束流程。",
	}, "direct_first", &completedActivityID, "activity_completed")
	if err != nil {
		t.Fatalf("build initial seed: %v", err)
	}
	if !strings.Contains(systemMsg, "## 服务处理规范") {
		t.Fatalf("expected system prompt to include service spec")
	}
	for _, needle := range []string{`"trigger_reason": "activity_completed"`, `"completed_activity_id": 9`, `"decision_mode": "direct_first"`, `"code": "TICK-42"`} {
		if !strings.Contains(userMsg, needle) {
			t.Fatalf("expected user seed to contain %s, got %s", needle, userMsg)
		}
	}
}

func TestAgenticSystemPromptGuardsServerAccessLexicalRouting(t *testing.T) {
	spec := testServerAccessRoutingSpec

	prompt := buildAgenticSystemPrompt(spec, "ai_only", "")

	for _, want := range []string{
		"结构化路由判定守卫",
		"安全窗口",
		"不是 security_admin 分支证据",
		`"position_code":"ops_admin"`,
		`"position_code":"network_admin"`,
		`"position_code":"security_admin"`,
		"decision.resolve_participant 的 department_code/position_code 必须与最终输出活动的业务分支一致",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestLooksLikeDBBackupWhitelistSpecMatchesNaturalSpec(t *testing.T) {
	spec := `员工在 IT 服务台申请生产数据库备份白名单临时放行时，服务台需要确认目标数据库、发起备份访问的来源 IP、白名单放行时间窗，以及这次临时放行的申请原因。
申请资料收齐后，系统会先做一次白名单参数预检，确认数据库、来源 IP、放行窗口和申请原因满足放行前置条件。预检通过后，交给信息部数据库管理员处理。
数据库管理员完成处理后，系统执行备份白名单放行；放行成功后流程结束。驳回时不进入补充或返工，流程按驳回结果结束。`

	if !looksLikeDBBackupWhitelistSpec(spec) {
		t.Fatalf("expected natural db backup whitelist spec to enable deterministic guard")
	}
}

func TestLooksLikeBossSerialChangeSpecMatchesNaturalSpec(t *testing.T) {
	spec := `员工在 IT 服务台提交高风险变更协同申请时，服务台需要确认申请主题、申请类别、风险等级、期望完成时间、变更窗口、影响范围、回滚要求、影响模块，以及每一项变更明细。
申请类别包括生产变更、访问授权和应急支持；风险等级包括低、中、高；回滚要求包括需要和不需要；影响模块可选择网关、支付、监控和订单。变更明细需要说明系统、资源、权限级别、生效时段和变更理由，权限级别包括只读和读写。
申请提交后，先交给总部处理人处理；总部处理人完成后，再交给信息部运维管理员处理。运维管理员完成处理后流程结束。`

	if !looksLikeBossSerialChangeSpec(spec) {
		t.Fatalf("expected natural boss serial change spec to enable deterministic guard")
	}
}

func TestTicketActionSucceededAcceptsLegacyDBBackupActionCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&serviceActionModel{}, &actionExecutionModel{}); err != nil {
		t.Fatalf("migrate action tables: %v", err)
	}
	action := serviceActionModel{ID: 7, Name: "旧预检", Code: "backup_whitelist_precheck", ServiceID: 3, IsActive: true}
	if err := db.Create(&action).Error; err != nil {
		t.Fatalf("create legacy action: %v", err)
	}
	if err := db.Create(&actionExecutionModel{TicketID: 42, ServiceActionID: action.ID, Status: "success"}).Error; err != nil {
		t.Fatalf("create action execution: %v", err)
	}

	ok, err := ticketActionSucceeded(db, 42, "db_backup_whitelist_precheck")
	if err != nil {
		t.Fatalf("ticketActionSucceeded: %v", err)
	}
	if !ok {
		t.Fatal("expected canonical db backup action lookup to match legacy successful execution")
	}
}

func TestBuildInitialSeedIncludesRejectedActivityPolicy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		code text,
		title text,
		description text,
		status text,
		outcome text,
		source text,
		priority_id integer,
		form_data text
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
		t.Fatalf("create priorities: %v", err)
	}
	if err := db.AutoMigrate(&activityModel{}); err != nil {
		t.Fatalf("migrate activities: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
		t.Fatalf("insert priority: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, code, title, description, status, outcome, source, priority_id) VALUES (42, 'TICK-42', 'VPN', '线上支持', 'rejected_decisioning', '', 'agent', 1)`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	completed := activityModel{
		ID:                9,
		TicketID:          42,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "不符合申请要求",
	}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed activity: %v", err)
	}

	completedActivityID := uint(9)
	engine := &SmartEngine{}
	_, userMsg, err := engine.buildInitialSeed(db, 42, &serviceModel{
		ID:                7,
		Name:              "VPN 开通申请",
		Description:       "VPN service",
		CollaborationSpec: "处理完成后结束流程。",
		WorkflowJSON:      vpnWorkflowContextFixture,
	}, "direct_first", &completedActivityID, "activity_completed")
	if err != nil {
		t.Fatalf("build initial seed: %v", err)
	}
	for _, needle := range []string{
		`"rejected_activity_policy"`,
		`"must_explain_rejection": true`,
		`"operator_opinion": "不符合申请要求"`,
		`不得在没有新证据的情况下重复创建刚被驳回的同一人工处理任务`,
		`协作规范未显式定义补充信息或返工路径时，不得创建申请人补充表单`,
		`"workflow_context"`,
	} {
		if !strings.Contains(userMsg, needle) {
			t.Fatalf("expected rejected seed to contain %s, got %s", needle, userMsg)
		}
	}
	if strings.Contains(userMsg, "退回申请人补充\"") {
		t.Fatalf("rejected fallback must not default to requester supplement, got %s", userMsg)
	}
}

func TestBuildInitialSeedIncludesApprovedNextStepAnchor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		code text,
		title text,
		description text,
		status text,
		outcome text,
		source text,
		priority_id integer,
		form_data text
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
		t.Fatalf("create priorities: %v", err)
	}
	if err := db.AutoMigrate(&activityModel{}); err != nil {
		t.Fatalf("migrate activities: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
		t.Fatalf("insert priority: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, code, title, description, status, outcome, source, priority_id) VALUES (43, 'TICK-43', 'VPN', '线上支持', 'approved_decisioning', '', 'agent', 1)`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	completed := activityModel{
		ID:                10,
		TicketID:          43,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		NodeID:            "network_process",
		Status:            ActivityCompleted,
		TransitionOutcome: TicketOutcomeApproved,
		DecisionReasoning: "条件满足继续推进",
	}
	if err := db.Create(&completed).Error; err != nil {
		t.Fatalf("create completed activity: %v", err)
	}

	completedActivityID := uint(10)
	engine := &SmartEngine{}
	_, userMsg, err := engine.buildInitialSeed(db, 43, &serviceModel{
		ID:                7,
		Name:              "VPN 开通申请",
		Description:       "VPN service",
		CollaborationSpec: "处理完成后继续推进。",
		WorkflowJSON:      vpnWorkflowContextFixture,
	}, "direct_first", &completedActivityID, "activity_completed")
	if err != nil {
		t.Fatalf("build initial seed: %v", err)
	}
	for _, needle := range []string{
		`"approved_next_step"`,
		`"target_node_id": "end"`,
		`"target_node_type": "end"`,
		`应遵循此路径继续推进，不应偏离`,
	} {
		if !strings.Contains(userMsg, needle) {
			t.Fatalf("expected approved seed to contain %s, got %s", needle, userMsg)
		}
	}
}

func TestBuildInitialSeedGracefullyDegradesWithoutWorkflowOrPriority(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		code text,
		title text,
		description text,
		status text,
		outcome text,
		source text,
		priority_id integer,
		form_data text
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
		t.Fatalf("create priorities: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, code, title, description, status, outcome, source, priority_id, form_data) VALUES (44, 'TICK-44', 'VPN', '简化上下文', 'decisioning', '', 'portal', 999, '{bad-json}')`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}

	engine := &SmartEngine{}
	_, userMsg, err := engine.buildInitialSeed(db, 44, &serviceModel{
		ID:                8,
		Name:              "轻量智能服务",
		Description:       "no workflow",
		CollaborationSpec: "按协作规范推进。",
	}, "  ", nil, TriggerReasonInitialDecision)
	if err != nil {
		t.Fatalf("build initial seed: %v", err)
	}
	for _, needle := range []string{
		`"trigger_reason": "initial_decision"`,
		`"decision_mode": "direct_first"`,
		`"priority": ""`,
		`"status": "decisioning"`,
	} {
		if !strings.Contains(userMsg, needle) {
			t.Fatalf("expected degraded seed to contain %s, got %s", needle, userMsg)
		}
	}
	for _, forbidden := range []string{`"workflow_context"`, `"approved_next_step"`, `"rejected_activity_policy"`} {
		if strings.Contains(userMsg, forbidden) {
			t.Fatalf("did not expect %s in degraded seed, got %s", forbidden, userMsg)
		}
	}
}

type capturingDecisionExecutor struct {
	resp     *appcore.AIDecisionResponse
	err      error
	calls    int
	lastReq  appcore.AIDecisionRequest
}

func (e *capturingDecisionExecutor) Execute(_ context.Context, _ uint, req appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	e.calls++
	e.lastReq = req
	if e.err != nil {
		return nil, e.err
	}
	return e.resp, nil
}

type toolExercisingDecisionExecutor struct {
	calls          int
	toolResult     json.RawMessage
	toolErr        error
	lastReq        appcore.AIDecisionRequest
}

func (e *toolExercisingDecisionExecutor) Execute(_ context.Context, _ uint, req appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	e.calls++
	e.lastReq = req
	if req.ToolHandler == nil {
		return nil, errors.New("missing tool handler")
	}
	e.toolResult, e.toolErr = req.ToolHandler("decision.unknown_tool", json.RawMessage(`{}`))
	return &appcore.AIDecisionResponse{
		Content: `{"next_step_type":"complete","confidence":0.99,"reasoning":"未知工具已被安全收口"}`,
	}, nil
}

type knowledgeSearchExercisingDecisionExecutor struct {
	calls          int
	toolResult     json.RawMessage
	toolErr        error
	lastReq        appcore.AIDecisionRequest
}

func (e *knowledgeSearchExercisingDecisionExecutor) Execute(_ context.Context, _ uint, req appcore.AIDecisionRequest) (*appcore.AIDecisionResponse, error) {
	e.calls++
	e.lastReq = req
	if req.ToolHandler == nil {
		return nil, errors.New("missing tool handler")
	}
	e.toolResult, e.toolErr = req.ToolHandler("decision.knowledge_search", json.RawMessage(`{"query":"vpn 预检","limit":2}`))
	return &appcore.AIDecisionResponse{
		Content: `{"next_step_type":"complete","confidence":0.93,"reasoning":"知识搜索已提供上下文"}`,
	}, nil
}

func TestAgenticDecisionGuardAndToolContextContracts(t *testing.T) {
	t.Run("missing decision agent id is rejected before executor call", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		engine := NewSmartEngine(&capturingDecisionExecutor{}, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 0})

		_, err := engine.agenticDecision(context.Background(), db, 1, nil, &serviceModel{Name: "智能服务"}, TriggerReasonInitialDecision)
		if err == nil || !strings.Contains(err.Error(), "流程决策岗未上岗") {
			t.Fatalf("expected missing decision agent error, got %v", err)
		}
	})

	t.Run("malformed knowledge base ids do not block executor request", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
			t.Fatalf("create priorities: %v", err)
		}
		if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
			t.Fatalf("insert priority: %v", err)
		}
		ticket := ticketModel{
			Code:       "TICK-AGENTIC-1",
			Title:      "VPN",
			Status:     TicketStatusDecisioning,
			PriorityID: 1,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if err := db.Exec(`UPDATE itsm_tickets SET description = ?, source = ? WHERE id = ?`, "线上支持", "agent", ticket.ID).Error; err != nil {
			t.Fatalf("update ticket payload columns: %v", err)
		}

		executor := &capturingDecisionExecutor{
			resp: &appcore.AIDecisionResponse{
				Content: `{"next_step_type":"complete","confidence":0.99,"reasoning":"资料齐备"}`,
			},
		}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 21})

		plan, err := engine.agenticDecision(context.Background(), db, ticket.ID, nil, &serviceModel{
			ID:               8,
			Name:             "智能服务",
			Description:      "svc",
			KnowledgeBaseIDs: `not-json`,
		}, TriggerReasonInitialDecision)
		if err != nil {
			t.Fatalf("agenticDecision: %v", err)
		}
		if executor.calls != 1 {
			t.Fatalf("expected one executor call, got %d", executor.calls)
		}
		if plan.NextStepType != "complete" || plan.Confidence != 0.99 {
			t.Fatalf("unexpected parsed plan: %+v", plan)
		}
		if len(executor.lastReq.Tools) == 0 || executor.lastReq.ToolHandler == nil {
			t.Fatalf("expected tools to be attached to executor request, got %+v", executor.lastReq)
		}
		if executor.lastReq.Metadata["ticketID"] != ticket.ID {
			t.Fatalf("expected metadata ticketID=%d, got %+v", ticket.ID, executor.lastReq.Metadata)
		}
		if !strings.Contains(executor.lastReq.UserMessage, `"decision_cycle"`) {
			t.Fatalf("expected user seed in executor request, got %s", executor.lastReq.UserMessage)
		}
	})

	t.Run("valid knowledge base ids flow into knowledge search tool context", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
			t.Fatalf("create priorities: %v", err)
		}
		if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
			t.Fatalf("insert priority: %v", err)
		}
		ticket := ticketModel{
			Code:       "TICK-AGENTIC-KB",
			Title:      "VPN",
			Status:     TicketStatusDecisioning,
			PriorityID: 1,
			EngineType: "smart",
		}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if err := db.Exec(`UPDATE itsm_tickets SET description = ?, source = ? WHERE id = ?`, "线上支持", "agent", ticket.ID).Error; err != nil {
			t.Fatalf("update ticket payload columns: %v", err)
		}

		searcher := &fakeKnowledgeSearcher{}
		executor := &knowledgeSearchExercisingDecisionExecutor{}
		engine := NewSmartEngine(executor, searcher, nil, nil, nil, decisionAgentConfigProvider{agentID: 26})

		plan, err := engine.agenticDecision(context.Background(), db, ticket.ID, nil, &serviceModel{
			ID:               13,
			Name:             "智能服务",
			Description:      "svc",
			KnowledgeBaseIDs: `[11,22]`,
		}, TriggerReasonInitialDecision)
		if err != nil {
			t.Fatalf("agenticDecision: %v", err)
		}
		if executor.calls != 1 {
			t.Fatalf("expected one executor call, got %d", executor.calls)
		}
		if executor.toolErr != nil {
			t.Fatalf("expected knowledge search tool to succeed, got %v", executor.toolErr)
		}
		if plan.NextStepType != "complete" || plan.Confidence != 0.93 {
			t.Fatalf("unexpected parsed plan: %+v", plan)
		}
		if !reflect.DeepEqual(searcher.kbIDs, []uint{11, 22}) || searcher.query != "vpn 预检" || searcher.limit != 2 {
			t.Fatalf("expected parsed knowledge base ids to reach searcher, got kbIDs=%v query=%q limit=%d", searcher.kbIDs, searcher.query, searcher.limit)
		}
		if !strings.Contains(string(executor.toolResult), "VPN 规范") {
			t.Fatalf("expected tool result payload to include knowledge result, got %s", executor.toolResult)
		}
	})

	t.Run("executor failure bubbles out to caller", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
			t.Fatalf("create priorities: %v", err)
		}
		if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
			t.Fatalf("insert priority: %v", err)
		}
		ticket := ticketModel{Code: "TICK-AGENTIC-2", Title: "VPN", Status: TicketStatusDecisioning, PriorityID: 1, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if err := db.Exec(`UPDATE itsm_tickets SET description = ?, source = ? WHERE id = ?`, "线上支持", "agent", ticket.ID).Error; err != nil {
			t.Fatalf("update ticket payload columns: %v", err)
		}

		executor := &capturingDecisionExecutor{err: errors.New("llm unavailable")}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 22})

		_, err := engine.agenticDecision(context.Background(), db, ticket.ID, nil, &serviceModel{ID: 9, Name: "智能服务"}, TriggerReasonInitialDecision)
		if err == nil || !strings.Contains(err.Error(), "llm unavailable") {
			t.Fatalf("expected executor failure to bubble, got %v", err)
		}
		if executor.calls != 1 {
			t.Fatalf("expected one executor call, got %d", executor.calls)
		}
	})

	t.Run("missing ticket surfaces build initial seed error before executor call", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		executor := &capturingDecisionExecutor{
			resp: &appcore.AIDecisionResponse{
				Content: `{"next_step_type":"complete","confidence":0.99}`,
			},
		}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 24})

		_, err := engine.agenticDecision(context.Background(), db, 9999, nil, &serviceModel{ID: 11, Name: "智能服务"}, TriggerReasonInitialDecision)
		if err == nil || !strings.Contains(err.Error(), "build initial seed") || !strings.Contains(err.Error(), "ticket not found") {
			t.Fatalf("expected build initial seed ticket-not-found error, got %v", err)
		}
		if executor.calls != 0 {
			t.Fatalf("expected executor to be skipped when seed build fails, got %d calls", executor.calls)
		}
	})

	t.Run("unknown tool is folded into business tool error payload", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
			t.Fatalf("create priorities: %v", err)
		}
		if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
			t.Fatalf("insert priority: %v", err)
		}
		ticket := ticketModel{Code: "TICK-AGENTIC-3", Title: "VPN", Status: TicketStatusDecisioning, PriorityID: 1, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if err := db.Exec(`UPDATE itsm_tickets SET description = ?, source = ? WHERE id = ?`, "线上支持", "agent", ticket.ID).Error; err != nil {
			t.Fatalf("update ticket payload columns: %v", err)
		}

		executor := &toolExercisingDecisionExecutor{}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 23})

		plan, err := engine.agenticDecision(context.Background(), db, ticket.ID, nil, &serviceModel{ID: 10, Name: "智能服务"}, TriggerReasonInitialDecision)
		if err != nil {
			t.Fatalf("agenticDecision: %v", err)
		}
		if plan.NextStepType != "complete" {
			t.Fatalf("unexpected parsed plan after tool exercise: %+v", plan)
		}
		if executor.toolErr != nil {
			t.Fatalf("expected unknown tool to return business payload instead of error, got %v", executor.toolErr)
		}
		if !strings.Contains(string(executor.toolResult), "未知工具") {
			t.Fatalf("expected unknown tool payload to mention unknown tool, got %s", executor.toolResult)
		}
	})

	t.Run("invalid model output bubbles parse error", func(t *testing.T) {
		db := newSmartRunCycleDB(t)
		if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
			t.Fatalf("create priorities: %v", err)
		}
		if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
			t.Fatalf("insert priority: %v", err)
		}
		ticket := ticketModel{Code: "TICK-AGENTIC-4", Title: "VPN", Status: TicketStatusDecisioning, PriorityID: 1, EngineType: "smart"}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		if err := db.Exec(`UPDATE itsm_tickets SET description = ?, source = ? WHERE id = ?`, "线上支持", "agent", ticket.ID).Error; err != nil {
			t.Fatalf("update ticket payload columns: %v", err)
		}

		executor := &capturingDecisionExecutor{
			resp: &appcore.AIDecisionResponse{Content: `not-json-at-all`},
		}
		engine := NewSmartEngine(executor, nil, nil, nil, nil, decisionAgentConfigProvider{agentID: 25})

		_, err := engine.agenticDecision(context.Background(), db, ticket.ID, nil, &serviceModel{ID: 12, Name: "智能服务"}, TriggerReasonInitialDecision)
		if err == nil || !strings.Contains(err.Error(), "JSON 解析失败") {
			t.Fatalf("expected parse error from malformed model output, got %v", err)
		}
		if executor.calls != 1 {
			t.Fatalf("expected one executor call, got %d", executor.calls)
		}
	})
}

func TestBuildInitialSeedIgnoresStaleCompletedActivityButKeepsBranchInsights(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_tickets (
		id integer primary key,
		code text,
		title text,
		description text,
		status text,
		outcome text,
		source text,
		priority_id integer,
		form_data text
	)`).Error; err != nil {
		t.Fatalf("create tickets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE itsm_priorities (id integer primary key, name text)`).Error; err != nil {
		t.Fatalf("create priorities: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_priorities (id, name) VALUES (1, '普通')`).Error; err != nil {
		t.Fatalf("insert priority: %v", err)
	}
	if err := db.Exec(`INSERT INTO itsm_tickets (id, code, title, description, status, outcome, source, priority_id, form_data) VALUES (45, 'TICK-45', 'VPN', '安全合规访问', 'decisioning', '', 'agent', 1, '{"request_kind":"security_compliance"}')`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}

	completedActivityID := uint(999)
	engine := &SmartEngine{}
	_, userMsg, err := engine.buildInitialSeed(db, 45, &serviceModel{
		ID:                9,
		Name:              "VPN 开通申请",
		Description:       "VPN service",
		CollaborationSpec: "处理任务完成后直接结束流程。",
		WorkflowJSON:      branchContractWorkflowFixture,
	}, "direct_first", &completedActivityID, TriggerReasonActivityDone)
	if err != nil {
		t.Fatalf("build initial seed: %v", err)
	}
	for _, needle := range []string{
		`"trigger_reason": "activity_completed"`,
		`"selected_branch"`,
		`"branch_node_id": "security_process"`,
		`"allowed_next_branch_nodes"`,
	} {
		if !strings.Contains(userMsg, needle) {
			t.Fatalf("expected stale-completed seed to retain branch insight %s, got %s", needle, userMsg)
		}
	}
	for _, forbidden := range []string{`"completed_activity"`, `"approved_next_step"`, `"rejected_activity_policy"`} {
		if strings.Contains(userMsg, forbidden) {
			t.Fatalf("did not expect %s when completed activity lookup misses, got %s", forbidden, userMsg)
		}
	}
}

type fakeDecisionDataProvider struct {
	ticket                *DecisionTicketData
	history               []activityModel
	activityByID          map[uint]activityModel
	assignments           map[uint][]ActivityAssignmentInfo
	current               []CurrentActivityInfo
	executed              []ExecutedActionInfo
	totalActions          int64
	assignment            *CurrentAssignmentInfo
	groups                []ParallelGroupInfo
	pendingNames          []string
	users                 map[uint]*UserBasicInfo
	userErrors            map[uint]error
	pendingCounts         map[uint]int64
	similarHistory        []TicketHistoryRow
	completedTicketCount  int64
	ticketActivityCounts  map[uint]int64
	slaData               *SLATicketData
	serviceActions        []ServiceActionRow
	serviceActionByID     map[uint]*ServiceActionRow
	resolveUserIDs        []uint
	resolveErr            error
}

func (f fakeDecisionDataProvider) GetTicketContext(uint) (*DecisionTicketData, error) {
	return f.ticket, nil
}
func (f fakeDecisionDataProvider) GetDecisionHistory(uint) ([]activityModel, error) {
	return f.history, nil
}
func (f fakeDecisionDataProvider) GetActivityByID(_ uint, activityID uint) (*activityModel, error) {
	activity := f.activityByID[activityID]
	return &activity, nil
}
func (f fakeDecisionDataProvider) GetActivityAssignments(activityID uint) ([]ActivityAssignmentInfo, error) {
	return f.assignments[activityID], nil
}
func (f fakeDecisionDataProvider) GetCurrentActivities(uint) ([]CurrentActivityInfo, error) {
	return f.current, nil
}
func (f fakeDecisionDataProvider) GetExecutedActions(uint) ([]ExecutedActionInfo, error) {
	return f.executed, nil
}
func (f fakeDecisionDataProvider) CountActiveServiceActions(uint, uint) (int64, error) {
	return f.totalActions, nil
}
func (f fakeDecisionDataProvider) GetCurrentAssignment(uint) (*CurrentAssignmentInfo, error) {
	return f.assignment, nil
}
func (f fakeDecisionDataProvider) GetParallelGroups(uint) ([]ParallelGroupInfo, error) {
	return f.groups, nil
}
func (f fakeDecisionDataProvider) GetPendingActivityNames(uint, string) ([]string, error) {
	return f.pendingNames, nil
}
func (f fakeDecisionDataProvider) GetUserBasicInfo(userID uint) (*UserBasicInfo, error) {
	if err, ok := f.userErrors[userID]; ok {
		return nil, err
	}
	if user, ok := f.users[userID]; ok {
		return user, nil
	}
	return &UserBasicInfo{ID: 1, Username: "admin", IsActive: true}, nil
}
func (f fakeDecisionDataProvider) CountUserPendingActivities(userID uint) (int64, error) {
	if count, ok := f.pendingCounts[userID]; ok {
		return count, nil
	}
	return 0, nil
}
func (f fakeDecisionDataProvider) GetSimilarHistory(uint, uint, int) ([]TicketHistoryRow, error) {
	return f.similarHistory, nil
}
func (f fakeDecisionDataProvider) CountCompletedTickets(uint) (int64, error) {
	return f.completedTicketCount, nil
}
func (f fakeDecisionDataProvider) CountTicketActivities(ticketID uint) (int64, error) {
	if count, ok := f.ticketActivityCounts[ticketID]; ok {
		return count, nil
	}
	return 0, nil
}
func (f fakeDecisionDataProvider) GetSLAData(uint) (*SLATicketData, error) {
	return f.slaData, nil
}
func (f fakeDecisionDataProvider) ListActiveServiceActions(uint, uint) ([]ServiceActionRow, error) {
	return f.serviceActions, nil
}
func (f fakeDecisionDataProvider) GetServiceAction(_ uint, actionID uint, _ uint) (*ServiceActionRow, error) {
	if action, ok := f.serviceActionByID[actionID]; ok {
		return action, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f fakeDecisionDataProvider) ResolveForTool(*ParticipantResolver, uint, json.RawMessage) ([]uint, error) {
	return f.resolveUserIDs, f.resolveErr
}

func TestDecisionTicketContextReturnsStableDecisionAnchors(t *testing.T) {
	now := time.Now()
	def := toolTicketContext()
	raw, err := def.Handler(&decisionToolContext{
		ticketID:            42,
		serviceID:           7,
		completedActivityID: uintPtrIf(9),
		data: fakeDecisionDataProvider{
			ticket: &DecisionTicketData{
				Code:                  "TICK-42",
				Title:                 "VPN",
				Description:           "线上支持",
				Status:                "in_progress",
				Source:                "agent",
				FormData:              `{"vpn_account":"wenhaowu@dev.com"}`,
				SLAResponseDeadline:   &now,
				SLAResolutionDeadline: &now,
			},
			history: []activityModel{
				{ID: 9, Name: "处理", ActivityType: "process", Status: ActivityApproved, TransitionOutcome: "completed", FinishedAt: &now},
			},
			activityByID: map[uint]activityModel{
				9: {ID: 9, Name: "处理", ActivityType: "process", Status: ActivityApproved, TransitionOutcome: "completed", FinishedAt: &now},
			},
			assignments: map[uint][]ActivityAssignmentInfo{
				9: {{ParticipantType: "user", UserID: uintPtrIf(1), AssigneeID: uintPtrIf(1), Status: "completed", FinishedAt: &now}},
			},
			current: []CurrentActivityInfo{
				{Name: "处理中", ActivityType: "process", Status: ActivityPending},
			},
			executed: []ExecutedActionInfo{
				{ActionName: "预检", ActionCode: "precheck", Status: "success"},
				{ActionName: "放行", ActionCode: "apply", Status: "success"},
			},
			totalActions: 2,
			assignment:   &CurrentAssignmentInfo{AssigneeID: 1, AssigneeName: "admin"},
			groups:       []ParallelGroupInfo{{ActivityGroupID: "group-1", Total: 2, Completed: 1}},
			pendingNames: []string{"安全处理"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("ticket context: %v", err)
	}

	var resp struct {
		IsTerminal        bool `json:"is_terminal"`
		CurrentActivities []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"current_activities"`
		ActionProgress struct {
			Total        int  `json:"total"`
			Executed     int  `json:"executed"`
			AllCompleted bool `json:"all_completed"`
		} `json:"action_progress"`
		ParallelGroups []struct {
			GroupID           string   `json:"group_id"`
			PendingActivities []string `json:"pending_activities"`
		} `json:"parallel_groups"`
		CompletedActivity struct {
			ID           uint `json:"id"`
			Participants []struct {
				UserID uint `json:"user_id"`
			} `json:"participants"`
		} `json:"completed_activity"`
		CompletedRequirements []struct {
			Type      string `json:"type"`
			Satisfied bool   `json:"satisfied"`
		} `json:"completed_requirements"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	if resp.IsTerminal {
		t.Fatalf("expected active ticket")
	}
	if len(resp.CurrentActivities) != 1 || resp.CurrentActivities[0].Name != "处理中" {
		t.Fatalf("expected pending current activity, got %+v", resp.CurrentActivities)
	}
	if resp.ActionProgress.Total != 2 || resp.ActionProgress.Executed != 2 || !resp.ActionProgress.AllCompleted {
		t.Fatalf("expected complete action progress, got %+v", resp.ActionProgress)
	}
	if len(resp.ParallelGroups) != 1 || resp.ParallelGroups[0].GroupID != "group-1" || len(resp.ParallelGroups[0].PendingActivities) != 1 {
		t.Fatalf("expected parallel group progress, got %+v", resp.ParallelGroups)
	}
	if resp.CompletedActivity.ID != 9 || len(resp.CompletedActivity.Participants) != 1 || resp.CompletedActivity.Participants[0].UserID != 1 {
		t.Fatalf("expected completed activity participant facts, got %+v", resp.CompletedActivity)
	}
	if len(resp.CompletedRequirements) != 1 || resp.CompletedRequirements[0].Type != "process" || !resp.CompletedRequirements[0].Satisfied {
		t.Fatalf("expected completed requirements, got %+v", resp.CompletedRequirements)
	}
}

func TestDecisionTicketContextMarksRejectedActivityForRecovery(t *testing.T) {
	now := time.Now()
	def := toolTicketContext()
	raw, err := def.Handler(&decisionToolContext{
		ticketID:            42,
		serviceID:           7,
		workflowJSON:        vpnWorkflowContextFixture,
		collaborationSpec:   "处理任务完成后直接结束流程。",
		completedActivityID: uintPtrIf(9),
		data: fakeDecisionDataProvider{
			ticket: &DecisionTicketData{
				Code:        "TICK-42",
				Title:       "VPN",
				Description: "线上支持",
				Status:      "in_progress",
				Source:      "agent",
				FormData:    `{"vpn_account":"demo@qq.com","request_kind":"online_support"}`,
			},
			history: []activityModel{
				{ID: 9, Name: "网络管理员处理", ActivityType: NodeProcess, Status: ActivityCompleted, NodeID: "network_process", TransitionOutcome: "rejected", DecisionReasoning: "不符合申请要求", FinishedAt: &now},
			},
			activityByID: map[uint]activityModel{
				9: {ID: 9, Name: "网络管理员处理", ActivityType: NodeProcess, Status: ActivityCompleted, NodeID: "network_process", TransitionOutcome: "rejected", DecisionReasoning: "不符合申请要求", FinishedAt: &now},
			},
			assignments: map[uint][]ActivityAssignmentInfo{
				9: {{ParticipantType: "position_department", PositionID: uintPtrIf(11), DepartmentID: uintPtrIf(22), AssigneeID: uintPtrIf(1), Status: "completed", FinishedAt: &now}},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("ticket context: %v", err)
	}

	var resp struct {
		CompletedActivity struct {
			Outcome                  string `json:"outcome"`
			OperatorOpinion          string `json:"operator_opinion"`
			Satisfied                bool   `json:"satisfied"`
			RequiresRecoveryDecision bool   `json:"requires_recovery_decision"`
		} `json:"completed_activity"`
		CompletedRequirements []struct {
			Outcome                  string `json:"outcome"`
			OperatorOpinion          string `json:"operator_opinion"`
			Satisfied                bool   `json:"satisfied"`
			RequiresRecoveryDecision bool   `json:"requires_recovery_decision"`
		} `json:"completed_requirements"`
		WorkflowContext struct {
			Kind        string `json:"kind"`
			RelatedStep struct {
				ID            string `json:"id"`
				OutgoingEdges []struct {
					Target string `json:"target"`
				} `json:"outgoing_edges"`
			} `json:"related_step"`
			HumanSteps []struct {
				ID string `json:"id"`
			} `json:"human_steps"`
		} `json:"workflow_context"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	if resp.CompletedActivity.Outcome != "rejected" || resp.CompletedActivity.OperatorOpinion != "不符合申请要求" || resp.CompletedActivity.Satisfied || !resp.CompletedActivity.RequiresRecoveryDecision {
		t.Fatalf("expected rejected completed activity recovery facts, got %+v", resp.CompletedActivity)
	}
	if len(resp.CompletedRequirements) != 1 || resp.CompletedRequirements[0].Satisfied || !resp.CompletedRequirements[0].RequiresRecoveryDecision {
		t.Fatalf("expected rejected completed requirement facts, got %+v", resp.CompletedRequirements)
	}
	if resp.WorkflowContext.Kind != "ai_generated_workflow_blueprint" || resp.WorkflowContext.RelatedStep.ID != "network_process" || len(resp.WorkflowContext.RelatedStep.OutgoingEdges) != 1 {
		t.Fatalf("expected workflow context anchored to rejected activity, got %+v", resp.WorkflowContext)
	}
}

func TestDecisionTicketContextExposesSelectedVPNBranchContract(t *testing.T) {
	now := time.Now()
	def := toolTicketContext()
	raw, err := def.Handler(&decisionToolContext{
		ticketID:            42,
		serviceID:           7,
		workflowJSON:        branchContractWorkflowFixture,
		collaborationSpec:   "处理任务完成后直接结束流程。",
		completedActivityID: uintPtrIf(9),
		data: fakeDecisionDataProvider{
			ticket: &DecisionTicketData{
				Code:        "TICK-42",
				Title:       "VPN",
				Description: "安全合规访问",
				Status:      "rejected_decisioning",
				Source:      "agent",
				FormData:    `{"request_kind":"security_compliance"}`,
			},
			history: []activityModel{
				{ID: 9, Name: "信息安全管理员处理", ActivityType: NodeProcess, Status: ActivityCompleted, NodeID: "security_process", TransitionOutcome: "rejected", DecisionReasoning: "安全条件不满足", FinishedAt: &now},
			},
			activityByID: map[uint]activityModel{
				9: {ID: 9, Name: "信息安全管理员处理", ActivityType: NodeProcess, Status: ActivityCompleted, NodeID: "security_process", TransitionOutcome: "rejected", DecisionReasoning: "安全条件不满足", FinishedAt: &now},
			},
			assignments: map[uint][]ActivityAssignmentInfo{
				9: {{ParticipantType: "position_department", PositionID: uintPtrIf(11), DepartmentID: uintPtrIf(22), AssigneeID: uintPtrIf(1), Status: "completed", FinishedAt: &now}},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("ticket context: %v", err)
	}

	var resp struct {
		SelectedBranch struct {
			BranchNodeID               string `json:"branch_node_id"`
			BranchLabel                string `json:"branch_label"`
			BranchRejectedTerminal     bool   `json:"branch_rejected_terminal"`
			BranchTerminalOnCompletion bool   `json:"branch_terminal_on_completion"`
		} `json:"selected_branch"`
		ActiveBranchContract struct {
			BranchNodeID string `json:"branch_node_id"`
		} `json:"active_branch_contract"`
		AllowedNextBranchNodes []string `json:"allowed_next_branch_nodes"`
		CompletionContract     struct {
			RejectedTargetNodeID      string `json:"rejected_target_node_id"`
			CanCompleteAfterRejection bool   `json:"can_complete_after_rejection"`
		} `json:"completion_contract"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	if resp.SelectedBranch.BranchNodeID != "security_process" || resp.ActiveBranchContract.BranchNodeID != "security_process" {
		t.Fatalf("expected security branch contract, got %+v / %+v", resp.SelectedBranch, resp.ActiveBranchContract)
	}
	if !resp.SelectedBranch.BranchRejectedTerminal || !resp.SelectedBranch.BranchTerminalOnCompletion {
		t.Fatalf("expected terminal branch contract, got %+v", resp.SelectedBranch)
	}
	if len(resp.AllowedNextBranchNodes) != 1 || resp.AllowedNextBranchNodes[0] != "end_reject_sec" {
		t.Fatalf("expected rejected continuation to stay on branch terminal node, got %+v", resp.AllowedNextBranchNodes)
	}
	if resp.CompletionContract.RejectedTargetNodeID != "end_reject_sec" || !resp.CompletionContract.CanCompleteAfterRejection {
		t.Fatalf("expected rejected completion contract, got %+v", resp.CompletionContract)
	}
}

func TestDecisionTicketContextExposesSelectedServerAccessBranchFromCurrentActivity(t *testing.T) {
	def := toolTicketContext()
	raw, err := def.Handler(&decisionToolContext{
		ticketID:          52,
		serviceID:         8,
		workflowJSON:      branchContractWorkflowFixture,
		collaborationSpec: "处理任务完成后直接结束流程。",
		data: fakeDecisionDataProvider{
			ticket: &DecisionTicketData{
				Code:        "TICK-52",
				Title:       "Server Access",
				Description: "高敏访问",
				Status:      "waiting_human",
				Source:      "agent",
				FormData:    `{"request_kind":"security_compliance"}`,
			},
			current: []CurrentActivityInfo{
				{ID: 12, Name: "信息安全管理员处理", ActivityType: NodeProcess, NodeID: "security_process", Status: ActivityPending},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("ticket context: %v", err)
	}

	var resp struct {
		SelectedBranch struct {
			BranchNodeID string `json:"branch_node_id"`
			BranchLabel  string `json:"branch_label"`
		} `json:"selected_branch"`
		CurrentBranchNodeID string `json:"current_branch_node_id"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	if resp.SelectedBranch.BranchNodeID != "security_process" || resp.CurrentBranchNodeID != "security_process" {
		t.Fatalf("expected current security branch to be exposed, got %+v", resp)
	}
}

func TestValidateDecisionPlanRejectsDuplicateCompletedHumanActivity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, is_active) VALUES (1, true)`).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "处理",
		ActivityType:      NodeProcess,
		Status:            ActivityApproved,
		TransitionOutcome: "completed",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if err := db.Create(&assignmentModel{
		TicketID:        ticket.ID,
		ActivityID:      activity.ID,
		ParticipantType: "user",
		UserID:          uintPtrIf(1),
		AssigneeID:      uintPtrIf(1),
		Status:          "completed",
	}).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	eng := &SmartEngine{}
	err = eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:          NodeProcess,
			ParticipantID: uintPtrIf(1),
			Instructions:  "再次处理",
		}},
		Confidence: 0.95,
	}, &serviceModel{ID: 1}, nil)
	if err == nil || !strings.Contains(err.Error(), "重复创建已完成的人工活动") {
		t.Fatalf("expected duplicate human activity validation error, got %v", err)
	}
}

func TestValidateDecisionPlanGuardContracts(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, ticketModel) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}, &serviceActionModel{}); err != nil {
			t.Fatalf("migrate db: %v", err)
		}
		if err := db.Exec(`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`).Error; err != nil {
			t.Fatalf("create users: %v", err)
		}
		if err := db.Exec(`INSERT INTO users (id, is_active) VALUES (1, true), (2, false)`).Error; err != nil {
			t.Fatalf("insert users: %v", err)
		}
		ticket := ticketModel{Status: TicketStatusDecisioning, EngineType: "smart", ServiceID: 1}
		if err := db.Create(&ticket).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		return db, ticket
	}

	eng := &SmartEngine{}

	t.Run("nil plan rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, nil, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "decision plan is nil") {
			t.Fatalf("expected nil plan error, got %v", err)
		}
	})

	t.Run("invalid next step type rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: "bogus_step",
			Confidence:   0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "next_step_type") {
			t.Fatalf("expected invalid next_step_type error, got %v", err)
		}
	})

	t.Run("complete plan cannot include activities", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: "complete",
			Activities: []DecisionActivity{{
				Type: NodeProcess,
			}},
			Confidence: 0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "complete 决策不能包含活动") {
			t.Fatalf("expected complete-with-activities error, got %v", err)
		}
	})

	t.Run("non complete plan requires activity", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeProcess,
			Confidence:   0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "必须包含至少一个活动") {
			t.Fatalf("expected empty-activities error, got %v", err)
		}
	})

	t.Run("confidence out of range rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				ParticipantID: uintPtrIf(1),
			}},
			Confidence: 1.2,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "confidence") {
			t.Fatalf("expected bad confidence error, got %v", err)
		}
	})

	t.Run("missing participant user rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				ParticipantID: uintPtrIf(99),
			}},
			Confidence: 0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "用户不存在") {
			t.Fatalf("expected missing user error, got %v", err)
		}
	})

	t.Run("inactive participant user rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				ParticipantID: uintPtrIf(2),
			}},
			Confidence: 0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "用户未激活") {
			t.Fatalf("expected inactive user error, got %v", err)
		}
	})

	t.Run("human activity without resolvable participant rejected", func(t *testing.T) {
		db, ticket := setup(t)
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type: NodeProcess,
			}},
			Confidence: 0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "缺少可解析的处理人") {
			t.Fatalf("expected missing participant resolution error, got %v", err)
		}
	})

	t.Run("inactive action rejected", func(t *testing.T) {
		db, ticket := setup(t)
		if err := db.Create(&serviceActionModel{
			ID:         7,
			ServiceID:  1,
			Name:       "禁用动作",
			Code:       "disabled_action",
			ActionType: "http",
			ConfigJSON: `{"url":"https://example.com"}`,
			IsActive:   false,
		}).Error; err != nil {
			t.Fatalf("create disabled action: %v", err)
		}
		err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
			NextStepType: NodeAction,
			Activities: []DecisionActivity{{
				Type:     NodeAction,
				ActionID: uintPtrIf(7),
			}},
			Confidence: 0.3,
		}, &serviceModel{ID: 1}, nil)
		if err == nil || !strings.Contains(err.Error(), "不在服务可用动作列表中") {
			t.Fatalf("expected inactive action error, got %v", err)
		}
	})
}

func TestValidateDecisionPlanNormalizesVPNRouteFromCollaborationSpec(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"request_kind":"network_access_issue"}`)

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "security_admin",
		}},
		Confidence: 0.95,
	}
	err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, CollaborationSpec: testVPNRoutingSpec}, nil)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := plan.Activities[0].PositionCode; got != "network_admin" {
		t.Fatalf("expected participant normalized to network_admin, got %q", got)
	}
	if got := plan.Activities[0].DepartmentCode; got != "it" {
		t.Fatalf("expected department to remain it, got %q", got)
	}
}

func TestValidateDecisionPlanRejectsMissingVPNRouteField(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"vpn_account":"demo@example.com"}`)

	eng := &SmartEngine{}
	err := eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "security_admin",
		}},
		Confidence: 0.95,
	}, &serviceModel{ID: 1, CollaborationSpec: testVPNRoutingSpec}, nil)
	if err == nil || !strings.Contains(err.Error(), "request_kind") {
		t.Fatalf("expected missing request_kind validation error, got %v", err)
	}
}

func TestValidateDecisionPlanNormalizesServerAccessSecurityBoundary(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"access_reason":"结合异常访问核查、日志固定和证据保全判断是否需要进一步安全处置。"}`)

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "ops_admin",
		}},
		Confidence: 0.95,
	}
	err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, CollaborationSpec: testServerAccessRoutingSpec}, nil)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := plan.Activities[0].PositionCode; got != "security_admin" {
		t.Fatalf("expected participant normalized to security_admin, got %q", got)
	}
}

func TestValidateDecisionPlanDoesNotTreatProductionAppHostAsOpsRoute(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"operation_purpose":"登录生产应用主机核查审计痕迹并做取证分析。","access_reason":"核查安全审计痕迹并完成取证分析，确认是否存在异常访问。"}`)

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "ops_admin",
		}},
		Confidence: 0.95,
	}
	err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, CollaborationSpec: testServerAccessRoutingSpec}, nil)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := plan.Activities[0].PositionCode; got != "security_admin" {
		t.Fatalf("expected participant normalized to security_admin, got %q", got)
	}
}

func TestValidateDecisionPlanNormalizesServerAccessNetworkRoute(t *testing.T) {
	db, ticket := setupStructuredRoutingValidationDB(t, `{"operation_purpose":"配合抓包和链路诊断，核对负载均衡后的网络访问路径。"}`)

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "position_department",
			DepartmentCode:  "it",
			PositionCode:    "security_admin",
		}},
		Confidence: 0.95,
	}
	err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, CollaborationSpec: testServerAccessRoutingSpec}, nil)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := plan.Activities[0].PositionCode; got != "network_admin" {
		t.Fatalf("expected participant normalized to network_admin, got %q", got)
	}
}

const testVPNRoutingSpec = `流程通过 form.request_kind 进入排他网关：线上支持(online_support)、故障排查(troubleshooting)、生产应急(production_emergency)、网络接入问题(network_access_issue)进入网络管理员处理，岗位编码 network_admin；外部协作(external_collaboration)、长期远程办公(long_term_remote_work)、跨境访问(cross_border_access)、安全合规事项(security_compliance)进入信息安全管理员处理，岗位编码 security_admin。`

const testServerAccessRoutingSpec = `员工在 IT 服务台申请生产服务器临时访问时，服务台需要确认要访问的服务器或资源范围、访问时段、本次操作目的，以及为什么需要临时进入生产环境。

访问原因通常分为三类：应用发布、进程排障、日志排查、磁盘清理、主机巡检、生产运维操作偏主机和应用运维，交给信息部运维管理员处理；网络抓包、连通性诊断、ACL 调整、负载均衡变更、防火墙策略调整偏网络诊断与策略处理，交给信息部网络管理员处理；安全审计、入侵排查、漏洞修复验证、取证分析、合规检查偏安全与合规风险，交给信息部信息安全管理员处理。

处理人完成处理后流程结束。`

func setupStructuredRoutingValidationDB(t *testing.T, formData string) (*gorm.DB, ticketModel) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`,
		`CREATE TABLE positions (id integer primary key, code text)`,
		`CREATE TABLE departments (id integer primary key, code text)`,
		`CREATE TABLE user_positions (id integer primary key, user_id integer, position_id integer, department_id integer, deleted_at datetime)`,
		`INSERT INTO users (id, is_active) VALUES (1, true), (2, true), (3, true)`,
		`INSERT INTO positions (id, code) VALUES (10, 'ops_admin'), (11, 'network_admin'), (12, 'security_admin')`,
		`INSERT INTO departments (id, code) VALUES (21, 'it')`,
		`INSERT INTO user_positions (id, user_id, position_id, department_id) VALUES (1, 1, 11, 21), (2, 2, 12, 21), (3, 3, 10, 21)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}

	ticket := ticketModel{Status: "decisioning", EngineType: "smart", FormData: formData}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	return db, ticket
}

func TestValidateDecisionPlanRejectsRepeatedActivityAfterRejectedCompletion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, is_active) VALUES (1, true)`).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "不符合申请要求",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if err := db.Create(&assignmentModel{
		TicketID:        ticket.ID,
		ActivityID:      activity.ID,
		ParticipantType: "user",
		UserID:          uintPtrIf(1),
		AssigneeID:      uintPtrIf(1),
		Status:          "completed",
	}).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	eng := &SmartEngine{}
	err = eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:          NodeProcess,
			ParticipantID: uintPtrIf(1),
			Instructions:  "处理 VPN 开通申请",
		}},
		Confidence: 0.95,
	}, &serviceModel{ID: 1}, &activity.ID)
	if err == nil || !strings.Contains(err.Error(), "刚被驳回") {
		t.Fatalf("expected rejected duplicate validation error, got %v", err)
	}

	err = eng.validateDecisionPlan(db, ticket.ID, &DecisionPlan{
		NextStepType:  "complete",
		ExecutionMode: "single",
		Confidence:    0.95,
	}, &serviceModel{ID: 1}, &activity.ID)
	if err != nil {
		t.Fatalf("expected complete decision to remain allowed after rejection context, got %v", err)
	}
}

func TestValidateDecisionPlanRejectsRequesterSupplementWithoutSpec(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "不符合申请要求",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeForm,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeForm,
			ParticipantType: "requester",
			Instructions:    "退回申请人补充 VPN 申请信息",
		}},
		Confidence: 0.9,
	}
	err = eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{
		ID:                1,
		CollaborationSpec: "处理完成后直接结束流程。",
	}, &activity.ID)
	if err == nil || !strings.Contains(err.Error(), "协作规范未显式定义补充信息或返工路径") {
		t.Fatalf("expected requester supplement to be rejected without explicit spec, got %v", err)
	}
}

func TestValidateDecisionPlanRejectsRequesterProcessAfterRejectedWithoutSpec(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "不符合申请要求",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeProcess,
			ParticipantType: "requester",
			Instructions:    "请申请人补充 VPN 申请理由",
		}},
		Confidence: 0.9,
	}
	err = eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{
		ID:                1,
		CollaborationSpec: "处理完成后直接结束流程。",
	}, &activity.ID)
	if err == nil || !strings.Contains(err.Error(), "申请人补充/返工活动") {
		t.Fatalf("expected requester process recovery to be rejected without explicit spec, got %v", err)
	}
}

func TestValidateDecisionPlanAllowsRequesterSupplementWhenSpecExplicit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "资料不足",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeForm,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:            NodeForm,
			ParticipantType: "requester",
			Instructions:    "退回申请人补充 VPN 申请信息",
		}},
		Confidence: 0.9,
	}
	err = eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{
		ID:                1,
		CollaborationSpec: "处理人驳回后，流程退回申请人补充信息，申请人补充后重新提交。",
	}, &activity.ID)
	if err != nil {
		t.Fatalf("expected requester supplement to be allowed when spec is explicit, got %v", err)
	}
}

func TestValidateDecisionPlanAllowsExplicitRecoveryIntentForSameRejectedParticipant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, is_active) VALUES (1, true)`).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	activity := activityModel{
		TicketID:          ticket.ID,
		Name:              "网络管理员处理",
		ActivityType:      NodeProcess,
		Status:            ActivityCompleted,
		TransitionOutcome: "rejected",
		DecisionReasoning: "需补充新的抓包证据",
	}
	if err := db.Create(&activity).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if err := db.Create(&assignmentModel{
		TicketID:        ticket.ID,
		ActivityID:      activity.ID,
		ParticipantType: "user",
		UserID:          uintPtrIf(1),
		AssigneeID:      uintPtrIf(1),
		Status:          "completed",
	}).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	eng := &SmartEngine{}
	plan := &DecisionPlan{
		NextStepType:  NodeProcess,
		ExecutionMode: "single",
		Activities: []DecisionActivity{{
			Type:          NodeProcess,
			ParticipantID: uintPtrIf(1),
			Instructions:  "基于新抓包证据重新评估并升级处理",
		}},
		Confidence: 0.95,
	}
	err = eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{
		ID:                1,
		CollaborationSpec: "处理人驳回后可基于补充的新证据继续恢复处理。",
	}, &activity.ID)
	if err != nil {
		t.Fatalf("expected explicit recovery intent to be allowed, got %v", err)
	}
}

const vpnWorkflowContextFixture = `{
  "nodes": [
    {"id": "start", "type": "start", "data": {"label": "开始"}},
    {"id": "network_process", "type": "process", "data": {"label": "网络管理员处理", "participants": [{"type": "position_department", "department_code": "it", "position_code": "network_admin"}]}},
    {"id": "end", "type": "end", "data": {"label": "结束"}}
  ],
  "edges": [
    {"id": "e1", "source": "start", "target": "network_process"},
    {"id": "e2", "source": "network_process", "target": "end", "data": {"outcome": "approved"}}
  ]
}`

const branchContractWorkflowFixture = `{
  "nodes": [
    {"id": "start", "type": "start", "data": {"label": "开始"}},
    {"id": "gateway_route", "type": "exclusive", "data": {"label": "访问原因路由"}},
    {"id": "network_process", "type": "process", "data": {"label": "网络管理员处理", "participants": [{"type": "position_department", "department_code": "it", "position_code": "network_admin"}]}},
    {"id": "security_process", "type": "process", "data": {"label": "信息安全管理员处理", "participants": [{"type": "position_department", "department_code": "it", "position_code": "security_admin"}]}},
    {"id": "end_ok_net", "type": "end", "data": {"label": "网络分支结束"}},
    {"id": "end_reject_net", "type": "end", "data": {"label": "网络分支驳回结束"}},
    {"id": "end_ok_sec", "type": "end", "data": {"label": "安全分支结束"}},
    {"id": "end_reject_sec", "type": "end", "data": {"label": "安全分支驳回结束"}}
  ],
  "edges": [
    {"id": "e1", "source": "start", "target": "gateway_route"},
    {"id": "e2", "source": "gateway_route", "target": "network_process", "data": {"condition": {"field": "form.request_kind", "operator": "contains_any", "value": ["online_support", "troubleshooting"]}}},
    {"id": "e3", "source": "gateway_route", "target": "security_process", "data": {"condition": {"field": "form.request_kind", "operator": "contains_any", "value": ["security_compliance", "external_collaboration"]}}},
    {"id": "e4", "source": "network_process", "target": "end_ok_net", "data": {"outcome": "approved"}},
    {"id": "e5", "source": "network_process", "target": "end_reject_net", "data": {"outcome": "rejected"}},
    {"id": "e6", "source": "security_process", "target": "end_ok_sec", "data": {"outcome": "approved"}},
    {"id": "e7", "source": "security_process", "target": "end_reject_sec", "data": {"outcome": "rejected"}}
  ]
}`

const nodeIDValidationWorkflowFixture = `{
  "nodes": [
    {"id": "start", "type": "start", "data": {"label": "开始"}},
    {"id": "node_form", "type": "form", "data": {"label": "申请表单", "participants": [{"type": "requester"}]}},
    {"id": "node_process", "type": "process", "data": {"label": "IT审批", "participants": [{"type": "position", "value": "it_mgr"}]}},
    {"id": "end_ok", "type": "end", "data": {"label": "结束"}},
    {"id": "end_reject", "type": "end", "data": {"label": "驳回结束"}}
  ],
  "edges": [
    {"id": "e1", "source": "start", "target": "node_form"},
    {"id": "e2", "source": "node_form", "target": "node_process", "data": {"outcome": "submitted"}},
    {"id": "e3", "source": "node_process", "target": "end_ok", "data": {"outcome": "approved"}},
    {"id": "e4", "source": "node_process", "target": "end_reject", "data": {"outcome": "rejected"}}
  ]
}`

func TestValidateDecisionPlanNodeID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&ticketModel{}, &activityModel{}, &assignmentModel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key, is_active boolean, deleted_at datetime)`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, is_active) VALUES (1, true)`).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ticket := ticketModel{Status: "in_progress", EngineType: "smart"}
	db.Create(&ticket)

	eng := &SmartEngine{}

	t.Run("valid node_id preserved", func(t *testing.T) {
		plan := &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				NodeID:        "node_process",
				ParticipantID: uintPtrIf(1),
			}},
			Confidence: 0.9,
		}
		err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, WorkflowJSON: nodeIDValidationWorkflowFixture}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Activities[0].NodeID != "node_process" {
			t.Fatalf("expected node_id to be preserved, got %q", plan.Activities[0].NodeID)
		}
	})

	t.Run("nonexistent node_id cleared", func(t *testing.T) {
		plan := &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				NodeID:        "node_nonexistent",
				ParticipantID: uintPtrIf(1),
			}},
			Confidence: 0.9,
		}
		err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, WorkflowJSON: nodeIDValidationWorkflowFixture}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Activities[0].NodeID != "" {
			t.Fatalf("expected node_id to be cleared, got %q", plan.Activities[0].NodeID)
		}
	})

	t.Run("type mismatch node_id cleared", func(t *testing.T) {
		plan := &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				NodeID:        "node_form", // form node, not process
				ParticipantID: uintPtrIf(1),
			}},
			Confidence: 0.9,
		}
		err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1, WorkflowJSON: nodeIDValidationWorkflowFixture}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Activities[0].NodeID != "" {
			t.Fatalf("expected node_id to be cleared for type mismatch, got %q", plan.Activities[0].NodeID)
		}
	})

	t.Run("no workflow_json skips check", func(t *testing.T) {
		plan := &DecisionPlan{
			NextStepType: NodeProcess,
			Activities: []DecisionActivity{{
				Type:          NodeProcess,
				NodeID:        "anything",
				ParticipantID: uintPtrIf(1),
			}},
			Confidence: 0.9,
		}
		err := eng.validateDecisionPlan(db, ticket.ID, plan, &serviceModel{ID: 1}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Activities[0].NodeID != "anything" {
			t.Fatalf("expected node_id to be preserved when no workflow_json, got %q", plan.Activities[0].NodeID)
		}
	})
}

func TestBuildWorkflowContextApprovedEdgeTarget(t *testing.T) {
	ctx := buildWorkflowContext(nodeIDValidationWorkflowFixture, "", nil, "", "", &activityModel{
		ID:                1,
		ActivityType:      NodeProcess,
		Name:              "IT审批",
		NodeID:            "node_process",
		TransitionOutcome: "approved",
		Status:            ActivityCompleted,
	})
	if ctx == nil {
		t.Fatal("expected non-nil workflow context")
	}
	relatedStep, ok := ctx["related_step"].(map[string]any)
	if !ok {
		t.Fatal("expected related_step in workflow context")
	}
	approvedTarget, ok := relatedStep["approved_edge_target"].(map[string]any)
	if !ok {
		t.Fatal("expected approved_edge_target in related_step")
	}
	if approvedTarget["node_id"] != "end_ok" {
		t.Fatalf("expected approved target node_id=end_ok, got %v", approvedTarget["node_id"])
	}
	if _, exists := relatedStep["rejected_edge_target"]; exists {
		t.Fatal("approved path should not have rejected_edge_target")
	}
}

func TestBuildWorkflowContextRejectedEdgeTarget(t *testing.T) {
	ctx := buildWorkflowContext(nodeIDValidationWorkflowFixture, "", nil, "", "", &activityModel{
		ID:                2,
		ActivityType:      NodeProcess,
		Name:              "IT审批",
		NodeID:            "node_process",
		TransitionOutcome: "rejected",
		Status:            ActivityCompleted,
	})
	if ctx == nil {
		t.Fatal("expected non-nil workflow context")
	}
	relatedStep, ok := ctx["related_step"].(map[string]any)
	if !ok {
		t.Fatal("expected related_step in workflow context")
	}
	rejectedTarget, ok := relatedStep["rejected_edge_target"].(map[string]any)
	if !ok {
		t.Fatal("expected rejected_edge_target in related_step")
	}
	if rejectedTarget["node_id"] != "end_reject" {
		t.Fatalf("expected rejected target node_id=end_reject, got %v", rejectedTarget["node_id"])
	}
	if _, exists := relatedStep["approved_edge_target"]; exists {
		t.Fatal("rejected path should not have approved_edge_target")
	}
}

func TestBuildWorkflowContextEmptyNodeIDFallback(t *testing.T) {
	ctx := buildWorkflowContext(nodeIDValidationWorkflowFixture, "", nil, "", "", &activityModel{
		ID:                3,
		ActivityType:      NodeProcess,
		Name:              "IT审批",
		NodeID:            "", // empty — should trigger fallback note
		TransitionOutcome: "approved",
		Status:            ActivityCompleted,
	})
	if ctx == nil {
		t.Fatal("expected non-nil workflow context")
	}
	if _, ok := ctx["related_step"]; ok {
		t.Fatal("expected no related_step when NodeID is empty")
	}
	if _, ok := ctx["related_step_note"]; !ok {
		t.Fatal("expected related_step_note when NodeID is empty")
	}
}

func TestActivityFactMapFormData(t *testing.T) {
	t.Run("with form_data", func(t *testing.T) {
		a := &activityModel{
			ID:           1,
			ActivityType: "form",
			Name:         "申请表单",
			Status:       ActivityCompleted,
			FormData:     `{"name":"张三","reason":"VPN申请"}`,
		}
		result := activityFactMap(a, nil)
		fd, ok := result["form_data"]
		if !ok {
			t.Fatal("expected form_data in activityFactMap result")
		}
		raw, ok := fd.(json.RawMessage)
		if !ok {
			t.Fatalf("expected json.RawMessage, got %T", fd)
		}
		if string(raw) != `{"name":"张三","reason":"VPN申请"}` {
			t.Fatalf("unexpected form_data: %s", raw)
		}
	})

	t.Run("without form_data", func(t *testing.T) {
		a := &activityModel{
			ID:           2,
			ActivityType: "process",
			Name:         "处理",
			Status:       ActivityCompleted,
		}
		result := activityFactMap(a, nil)
		if _, ok := result["form_data"]; ok {
			t.Fatal("expected no form_data when FormData is empty")
		}
	})
}
