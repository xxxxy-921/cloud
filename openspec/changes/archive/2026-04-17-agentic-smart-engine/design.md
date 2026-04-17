## Context

当前 SmartEngine（`engine/smart.go`，1035 行）通过单次 `llm.Client.Chat()` 调用实现流程决策。全量上下文（工单快照 + 所有活跃用户列表 + 动作列表）一次性灌入 prompt，LLM 输出 `DecisionPlan` JSON。这在小规模环境下可工作，但存在三个结构性问题：

1. **全量用户灌入不可扩展** — `compilePolicy()` 调 `ListActiveUsers()` 返回所有活跃用户，500+ 用户就逼近 token 上限
2. **无法利用知识库** — `callAgent()` 有 TODO 注释但未实现知识注入，Agent 无法查阅服务处理规范
3. **单轮无法分步推理** — 复杂场景需要先查组织架构确定审批链，再查历史了解处理模式，最后综合决策

现有 `internal/llm` 包已完整支持 tool calling（`ChatRequest.Tools` / `ChatResponse.ToolCalls`），OpenAI 和 Anthropic client 均已实现，无需任何改动。

## Goals / Non-Goals

**Goals:**

- 将 SmartEngine 的决策调用从单次 structured output 升级为多轮 ReAct 工具调用循环
- 提供 7 个决策域工具，让 Agent 按需查询上下文而非全量灌入
- 完全复用现有 AI 基础设施（llm.Client、AgentProvider、KnowledgeSearcher、ai_agents 表配置）
- 保持 WorkflowEngine 接口不变，对 TicketService 调用方透明
- 保持置信度门控、人工确认、失败熔断等外层机制不变

**Non-Goals:**

- 不改 Classic Engine 任何代码
- 不改 Agent Runtime（Gateway/ReactExecutor/Session）— 那是交互式 Chat 场景的基础设施
- 不改 `internal/llm` 包
- 不做前端改动
- 不实现 embedding 语义搜索（similar_history 用 service_id + 关键字匹配）
- 不实现 action dry-run 预检（留给后续迭代）

## Decisions

### D1: 在 SmartEngine 内建轻量 ReAct 循环（方案 B）

**选择**：SmartEngine 内部实现 ~50 行的 for 循环，直接调用 `llm.Client.Chat()` 带 `Tools` 参数。

**替代方案**：
- 方案 A — 复用 AgentGateway.Run()：Gateway 耦合 Session 管理 + SSE 流式 + Message 持久化，解耦成本高，且决策引擎不需要这些能力
- 方案 C — 提取 ReAct Core 为共享库：架构最干净，但重构量大，当前只有 SmartEngine 一个非交互式消费者，过早抽象

**理由**：决策循环与交互式 Chat 的差异决定了不值得复用上层框架：
- 不需要 SSE streaming（后台异步任务）
- 不需要 Session 持久化（决策是瞬时过程）
- 不需要 Memory 提取/注入
- 工具集固定（7 个），不需要动态 Registry
- MaxTurns 小（8 轮），不需要复杂的终止策略

复用边界：`llm.Client` + `AgentProvider.GetAgentConfig()` + `KnowledgeSearcher` 完全复用，只是绕过 Gateway/Executor 层。

### D2: 决策域工具设计为只读查询工具

**选择**：7 个工具全部是只读查询，Agent 不能通过工具执行副作用操作。

**理由**：
- Agent 的职责是"决策"不是"执行"，执行由 `executeDecisionPlan()` / `pendApprovalDecisionPlan()` 负责
- 只读工具消除了工具调用失败导致数据不一致的风险
- 与现有的置信度门控机制兼容 — Agent 输出 DecisionPlan，引擎校验后才执行

### D3: 工具通过 tx *gorm.DB 直接查询，不经过 Service 层

**选择**：决策工具直接用事务中的 `tx` 查询数据库表，只有 `resolve_participant` 通过已有的 `ParticipantResolver` 走 `OrgService` 接口。

**替代方案**：每个工具调用对应的 Service 层方法。

**理由**：
- 决策工具的查询都是简单的 SELECT，不涉及业务逻辑
- 事务内查询保证数据一致性（决策期间数据不会变）
- 避免为 7 个工具在 Service 层新增 7 个方法的膨胀
- `ParticipantResolver` 是例外，因为它封装了 Org App 的组织架构查询逻辑，复用价值高

### D4: 初始 seed 精简 + 工具按需获取的信息分配

**选择**：

| 信息 | 初始 seed（prompt 直接携带） | 工具按需查询 |
|------|-----|------|
| 工单基本字段（code/title/status/priority） | ✅ | |
| 服务名称 + 引擎类型 | ✅ | |
| allowed_step_types + 当前状态 | ✅ | |
| 协作规范（collaboration_spec） | ✅ 在 system prompt | |
| 完整表单数据 | | `decision.ticket_context` |
| SLA 剩余时间 | | `decision.sla_status` |
| 活动历史 | | `decision.ticket_context` |
| 知识库内容 | | `decision.knowledge_search` |
| 具体参与人解析 | | `decision.resolve_participant` |
| 人员负载/可用性 | | `decision.user_workload` |
| 历史类似工单 | | `decision.similar_history` |
| 可用动作列表 | | `decision.list_actions` |

**理由**：Agent 一定需要看工单基本信息和规则约束才能开始推理，这些放在 seed 里。其余信息 Agent 根据推理需要按需查询，避免 token 浪费。

### D5: 决策工具硬编码绑定，不通过 ai_agent_tools 表

**选择**：工具定义和执行函数在 `smart_tools.go` 中硬编码注册，不走 `ai_agent_tools` 数据库表的动态绑定。

**理由**：
- 决策工具是引擎内部能力，不是可配置的外挂工具
- 管理员不应能在后台禁用 `resolve_participant`（否则引擎瘫痪）
- 与 Classic Engine 的节点类型同理 — 没人会让管理员在后台禁用 exclusive gateway
- `tools/provider.go` 中 seed 的决策智能体仍引用这些工具名用于 system prompt 提示，但执行时不经过 ToolHandlerRegistry

### D6: ReAct 循环最多 8 轮，最终输出必须是 DecisionPlan JSON

**选择**：
- `decisionMaxTurns = 8`
- 循环终止条件：LLM 返回无 tool_calls 时，解析 content 为 DecisionPlan
- 如果 8 轮后仍在调用工具，视为决策失败

**理由**：
- 典型决策路径：查上下文(1轮) → 查参与人(1轮) → 查知识(1轮) → 输出决策(1轮) = 4 轮
- 8 轮给复杂场景（需要多次查询不同类型参与人、检查多个候选人负载）留够余量
- 强制终止防止无限循环消耗 token

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 多轮调用增加延迟（4-8 次 LLM 调用 vs 原来 1 次） | 决策耗时从 ~3s 增至 ~10-20s | 决策是异步任务（itsm-smart-progress），不阻塞 HTTP；超时仍由 `decision_timeout_seconds` 控制 |
| 工具调用失败可能中断决策 | Agent 可能因工具错误卡住 | 工具返回结构化错误信息让 Agent 继续推理；整体仍有 3 次失败熔断 |
| 初始 seed 过于精简，Agent 不知道该查什么 | 决策质量下降 | system prompt 中包含明确的工具使用指引和推理步骤建议 |
| DecisionPlan 校验逻辑变化 | 不再校验 participant 在候选列表中 | 改为校验 participant 是否为活跃用户（直接查 DB） |
| `decision_timeout_seconds` 默认 30s 可能不够多轮调用 | 复杂决策超时 | 建议将默认值调整为 60s，或在文档中提示管理员按需调高 |
