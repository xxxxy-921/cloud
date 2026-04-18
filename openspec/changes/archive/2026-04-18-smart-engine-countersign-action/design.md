## Context

Smart Engine 当前决策循环是 `agenticDecision() → DecisionPlan → executeDecisionPlan()` 单步推进模式。每次决策创建一个（或多个顺序覆盖的）activity，等待人工处理后触发下一轮。

关键限制：
1. `executeDecisionPlan` 对 `plan.Activities` 做 `for range`，每次用新 activity ID 覆盖 `ticket.current_activity_id`——无并行执行概念
2. Action 执行路径是 `AI 输出 → scheduler 异步 webhook → 无自动 re-trigger`——Agent 看不到 action 结果
3. `TicketActivity` 有 `ExecutionMode` 字段但 Smart Engine 从未使用

参考实现：bklite-cloud Python 版本的并签使用 Classic Engine + 固定工作流 (`execution_mode: "parallel"`)，action 触发使用 `trigger_actions_task` 同步调用。Metis 需要在 Smart Engine 中以 Agentic 方式实现等价能力。

## Goals / Non-Goals

**Goals:**
- Smart Engine 支持真正的多角色并签：AI 输出 `execution_mode: "parallel"` → 创建 activity group → Progress 汇聚检查
- Action 作为 Agent 的 tool call：`decision.execute_action` 在 ReAct loop 中同步执行 webhook，Agent 看到结果继续推理
- LLM 驱动的 BDD 端到端验证并签和 action 元调用两个场景
- 现有 BDD（server_access、vpn_smart、vpn_deterministic 等）不受影响

**Non-Goals:**
- Classic Engine 的并行网关逻辑不改动
- 不涉及前端 UI 变更
- 不实现并签中的"一票否决"（rejection 语义）——本次只覆盖全部 approve 后汇聚
- 不实现 action 失败后的自动重试策略（Agent 可以看到失败结果自行决策）

## Decisions

### D1: 并签用 activity_group_id 而非复用 Classic Engine 的 token 机制

**选择**：`TicketActivity` 新增 `ActivityGroupID string` 字段，并签 activities 共享同一个 UUID。

**替代方案**：复用 Classic Engine 的 `ExecutionToken` parallel fork/join 机制。

**理由**：Smart Engine 无 token 概念，引入 token 会破坏其"每步 AI 决策"的架构。activity_group_id 是一个轻量级关联，只影响 Smart Engine 的 `executeDecisionPlan` 和 `Progress`，对 Classic Engine 透明。

### D2: Action 执行从 scheduler 异步迁移到 ReAct tool call

**选择**：新增 `decision.execute_action` 决策工具，Agent 在 ReAct loop 中同步调用。

**替代方案 A**：保留 scheduler 异步模式 + `HandleActionExecute` 完成后自动提交 `itsm-smart-progress`。

**替代方案 B**：混合模式——快 action 用 tool call，慢 action 走 scheduler。

**理由**：替代方案 A 仍然是断裂的——Agent 无法在同一推理链中看到 action 结果，需要多轮 LLM 调用。替代方案 B 增加分支复杂度。真正的 Agentic 设计应该让 Agent 完全掌控 action 执行时机和结果感知。工具内置超时（复用 ActionConfig.Timeout）防止阻塞。

### D3: DecisionPlan 输出格式向后兼容

**选择**：`execution_mode` 字段可选，默认空值等同 `"single"`。`type: "action"` 在 activities 中仍然合法（Classic Engine 仍用），但 Smart Engine 的 prompt 引导 Agent 使用 tool call 而非 action activity。

**理由**：渐进迁移。现有确定性 BDD（`vpn_smart_engine_deterministic.feature`）使用手工构造的 `DecisionPlan` 可能包含 action type，不应被破坏。

### D4: Progress 汇聚逻辑放在 SmartEngine.Progress 而非单独的 converge 方法

**选择**：在 `Progress()` 标记 activity completed 之后、触发下一轮决策之前，插入 group 汇聚检查。

**理由**：汇聚检查是 Progress 的天然扩展点。单独方法会增加调用方复杂度，且需要知道何时调用。放在 Progress 内部，调用方（BDD 步骤、scheduler task handler）无需感知并签存在。

### D5: BDD 测试策略——LLM 驱动 + 依赖环境变量 gating

**选择**：并签和 action 元调用 BDD 均为 LLM 驱动（`@llm` tag 或 `hasLLMConfig()` gating），不提供确定性替代。

**理由**：用户明确要求 LLM 驱动验证——"否则场景验证不出来，上线就会翻车"。确定性测试只能验证引擎齿轮，无法验证 AI 是否理解协作规范、是否正确输出 `execution_mode: "parallel"`、是否在正确时机调用 `decision.execute_action`。

## Risks / Trade-offs

**[Risk] `decision.execute_action` 在 DB 事务内执行 HTTP 调用** → 在 `HandleSmartProgress` (scheduler task) 中，`runDecisionCycle` 可能运行在 `db.Transaction` 内。HTTP webhook 阻塞会延长事务持有时间。Mitigation：tool handler 内部使用独立 DB 连接执行 action 记录写入，不依赖外层事务。BDD 测试中 `bc.db` 不在事务内，无此问题。

**[Risk] LLM 输出不稳定导致 BDD flaky** → AI 可能偶尔不输出 `execution_mode: "parallel"` 或不调用 `execute_action`。Mitigation：协作规范中明确标注 "execution_mode 必须为 parallel"、"必须使用 decision.execute_action 执行动作"；BDD 工作流生成增加 retry（已有 3 次重试模式）。

**[Risk] 并签 activity_group_id 增加 TicketActivity 表查询** → `Progress` 每次完成 activity 时多一次 COUNT 查询。Mitigation：查询条件走 `activity_group_id` 索引，且只在字段非空时触发，对单签活动无影响。

**[Risk] `type: "action"` 在 DecisionPlan 中保留但 Smart Engine prompt 不再引导使用** → 可能出现 AI 偶尔输出 action activity 而非使用 tool call。Mitigation：validateDecisionPlan 中可以 warn（非 block），`executeDecisionPlan` 保留对 action activity 的处理逻辑作为 fallback。
