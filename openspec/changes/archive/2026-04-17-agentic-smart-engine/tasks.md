## 1. 类型与接口调整

- [x] 1.1 `engine/engine.go`: 新增 `DecisionToolMaxTurns = 8` 常量；新增 `DefaultDecisionTimeoutSeconds` 默认值从 30 改为 60
- [x] 1.2 `engine/engine.go`: 从 `TicketPolicySnapshot` struct 中移除 `ParticipantCandidates` 和 `AvailableActions` 字段
- [x] 1.3 `engine/resolver.go`: 新增 `ResolveForTool(tx *gorm.DB, ticketID uint, toolArgs json.RawMessage) ([]ParticipantCandidate, error)` 方法，封装 JSON 参数解析后调用现有 `Resolve()`

## 2. 决策域工具实现

- [x] 2.1 `engine/smart_tools.go`: 定义 `decisionToolDef` struct（Def llm.ToolDef + Handler func）和 `allDecisionTools()` 注册函数
- [x] 2.2 `engine/smart_tools.go`: 实现 `decision.ticket_context` — 查询完整工单上下文（表单、SLA、活动历史、当前指派）
- [x] 2.3 `engine/smart_tools.go`: 实现 `decision.knowledge_search` — 调用 `KnowledgeSearcher.Search()` 查询服务关联知识库
- [x] 2.4 `engine/smart_tools.go`: 实现 `decision.resolve_participant` — 调用 `ParticipantResolver.ResolveForTool()` 按类型解析参与人
- [x] 2.5 `engine/smart_tools.go`: 实现 `decision.user_workload` — 查询用户待处理活动数和活跃状态
- [x] 2.6 `engine/smart_tools.go`: 实现 `decision.similar_history` — 查询同服务已完成工单摘要和聚合统计
- [x] 2.7 `engine/smart_tools.go`: 实现 `decision.sla_status` — 计算 SLA 剩余时间和紧急程度
- [x] 2.8 `engine/smart_tools.go`: 实现 `decision.list_actions` — 查询服务可用 ServiceAction 列表

## 3. ReAct 循环实现

- [x] 3.1 `engine/smart_react.go`: 实现 `agenticDecision(ctx, tx, ticketID, svc) (*DecisionPlan, error)` — ReAct 主循环（messages + tools → llm.Chat → tool dispatch → loop）
- [x] 3.2 `engine/smart_react.go`: 实现 `executeDecisionTool(ctx, tx, ticketID, tc llm.ToolCall) string` — map 查找 + 调用 + 错误包装
- [x] 3.3 `engine/smart_react.go`: 实现 `buildDecisionToolDefs() []llm.ToolDef` — 从 allDecisionTools() 提取 ToolDef 列表
- [x] 3.4 `engine/smart_react.go`: 实现 `buildInitialSeed(tx, ticketID, svc) (systemMsg, userMsg string)` — 构建精简初始 seed 和策略约束

## 4. SmartEngine 核心改造

- [x] 4.1 `engine/smart.go`: 扩展 `SmartEngine` struct 新增 `resolver *ParticipantResolver` 字段；更新 `NewSmartEngine()` 签名
- [x] 4.2 `engine/smart.go`: 重写 `runDecisionCycle()` — 调用 `agenticDecision()` 替代 `callAgent()`，移除 `buildTicketCase()`/`compilePolicy()` 的全量构建
- [x] 4.3 `engine/smart.go`: 删除 `callAgent()` 和 `callAgentWithCorrection()` 两个方法
- [x] 4.4 `engine/smart.go`: 简化 `buildTicketCase()` 为内部辅助函数（仅供 `decision.ticket_context` 工具复用），或直接内联到工具实现中
- [x] 4.5 `engine/smart.go`: 简化 `compilePolicy()` — 移除 `ListActiveUsers()` 调用和 `available_actions` 查询
- [x] 4.6 `engine/smart.go`: 调整 `validateDecisionPlan()` — 校验 participant 改为直接查 DB `is_active`，校验 action 改为直接查 `itsm_service_actions`
- [x] 4.7 `engine/smart.go`: 更新 `buildSystemPrompt()` — 追加工具使用指引段落和终止输出格式说明

## 5. 上层集成

- [x] 5.1 `app.go`: 更新 `NewSmartEngine()` 调用，传入已有的 `ParticipantResolver` 实例
- [x] 5.2 `tools/provider.go`: 更新决策智能体 seed — `MaxTurns: 8`，SystemPrompt 增加工具使用指引，ToolNames 列出 decision.* 工具名（仅用于 prompt 提示，不影响执行）

## 6. 验证

- [x] 6.1 `go build -tags dev ./cmd/server/` 编译通过
- [x] 6.2 `go test ./internal/app/itsm/...` 现有测试通过
- [ ] 6.3 手动验证：智能工单创建后，决策循环正确执行 ReAct 多轮调用，Timeline 记录工具调用过程
