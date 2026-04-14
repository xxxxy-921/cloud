## Context

ITSM 双引擎项目 Phase 3。总体架构设计见 `openspec/changes/itsm-dual-engine/design.md`，Phase 1（骨架 + 模型 + 手动工单）和 Phase 2（经典 BPMN 引擎）已完成。

本期实现第二个引擎——SmartEngine。它与 ClassicEngine 实现相同的 `WorkflowEngine` 接口，但核心区别在于：ClassicEngine 遍历预定义的 ReactFlow 图结构，SmartEngine 在每一步转换时请求 AI Agent 决定下一步操作。

**"灵魂与躯壳"架构**：ITSM（躯壳）引用 AI App 的 Agent（灵魂）。智能服务通过 `ServiceDefinition.AgentID` 绑定到一个 Agent，Agent 的 system_prompt + 知识库 + 工具定义了服务的智能。ITSM 只负责组装工单上下文并传给 Agent，不在内部重建 AI 基础设施。

项目已有的关键依赖：AI App（Agent/Knowledge/LLM Client/ToolRegistry）、Scheduler（异步任务引擎）、Org App（部门/岗位用于派单候选人）。Phase 1 已预留全部数据模型字段（engine_type、collaboration_spec、agent_id、agent_config、ai_decision、ai_reasoning、confidence 等），Phase 3 只加逻辑不加表。

## Goals / Non-Goals

**Goals:**

- SmartEngine 实现 WorkflowEngine 接口，Agent 驱动的决策循环（每步新鲜决策，不依赖预定义流程图）
- 信心机制：渐进式信任建立，confidence 阈值对比决定自动执行还是等待人工确认
- 人工覆盖：每个 AI 决策点均可被授权用户覆盖（确认/拒绝/强制跳转/改派/驳回）
- Fallback：AI 不可用或连续失败时自动降级到人工队列
- 6 个 ITSM 工具注册到 AI App 的 ToolRegistry，支撑对话式提单和 Agent 操作工单
- 3 个预置 Agent 通过 Seed 创建（IT 服务台 / 流程决策 / 处理协助）
- 对话式提单：用户在 AI Chat 中与 IT 服务台 Agent 对话创建工单（标准 Agent + Tool 模式，无需专用前端）
- AI Copilot：处理人在工单详情中打开 Agent Session 获取处理协助（复用现有 Agent Session API）
- 运行时动态流程图：基于 Activity 历史生成路径可视化
- 智能服务配置 UI：Collaboration Spec 编辑、Agent 选择、知识库绑定、信心阈值设置

**Non-Goals:**

- 经典 BPMN 引擎（Phase 2 已完成）
- SLA 检查定时任务和升级执行（Phase 4）
- 工单报表和仪表盘（Phase 4）
- 故障管理（Phase 4）
- 自定义 Agent 创建 UI（使用 AI App 现有 Agent 管理界面）
- MCP Server 集成（使用 AI App 现有能力）

## Decisions

### D1: SmartEngine 直接调用 LLM Client 做工作流决策，而非通过 Agent Session

**选择**: SmartEngine 调用 AI App 的 `llm.Client`（或 AgentService 的内部方法），传入构建好的消息序列，获取结构化 JSON 输出。不创建 Agent Session、不走 SSE 流式响应。

**替代方案**: 为每次决策创建一个 Agent Session，通过 Agent 运行时执行。

**理由**: 工作流决策是后端纯计算过程，不需要 UI 交互、不需要流式输出、不需要多轮对话。直接调用 LLM Client 更简洁，避免了 Session 管理的开销。Agent 的配置（模型、temperature、system_prompt）仍然通过 AgentService 获取，只是调用方式更直接。

### D2: Collaboration Spec 注入 system prompt 最高优先级，Agent 自身 system_prompt 次之

**选择**: 构建 system prompt 时，Collaboration Spec 全文放在最前面（作为 "服务处理规范"），Agent 自身的 system_prompt 放在其后（作为 "角色定义"）。

**理由**: Collaboration Spec 是服务特定的处理规范（流程约束、质量要求、分级规则），应当覆盖 Agent 的通用行为。Agent 的 system_prompt 定义角色能力，Spec 定义具体服务的规则。

### D3: TicketDecisionPlan 是结构化 JSON 输出，Agent 返回 JSON，引擎解析并校验

**选择**: Agent 的输出格式要求为 JSON，包含 `next_step_type`、`activities`、`reasoning`、`confidence` 字段。引擎解析 JSON 后校验合法性（next_step_type 是否在 Policy 允许列表中、参与人是否在候选列表中等）。

**替代方案**: Agent 返回自然语言，引擎再用另一次 LLM 调用解析。

**理由**: 现代 LLM 支持 JSON mode / structured output，一次调用即可获得结构化输出。减少调用次数，降低延迟和成本。校验失败时重试一次（附带格式纠正提示），仍失败则转人工。

### D4: 信心阈值是服务级可配置项，存储在 agent_config.confidence_threshold

**选择**: `ServiceDefinition.AgentConfig` 是一个 JSON 字段，包含 `confidence_threshold`（float, 默认 0.8）、`decision_timeout_seconds`（int, 默认 30）、`fallback_strategy`（string, 默认 "manual_queue"）等配置。

**理由**: 不同服务对自动化的信任度不同。密码重置服务可以设低阈值（0.5），系统权限变更服务应设高阈值（0.95）。服务级配置让管理员精细控制。

### D5: 工具注册通过 IOC 可选注入——ITSM 获取 ToolRegistry（若存在），注册工具，不存在时静默跳过

**选择**: ITSM App 在 `Providers()` 中通过 `do.InvokeAs[ToolRegistry](injector)` 尝试获取 AI App 的 ToolRegistry。成功则注册 6 个 ITSM Builtin Tool，失败则跳过（仅 info 日志）。

**替代方案**: 在 AI App 中硬编码 ITSM 工具定义。

**理由**: 保持 App 间松耦合。ITSM App 拥有自己的工具实现和 schema 定义，AI App 不需要知道 ITSM 的存在。Edition 不含 AI App 时 ITSM 经典功能完全不受影响。

### D6: 对话式提单是标准的 Agent + Tool 模式，无需专用前端

**选择**: 用户在 AI Chat 中选择"IT 服务台" Agent 进行对话。Agent 根据对话内容调用 `itsm.search_services` 搜索匹配服务，确认后调用 `itsm.create_ticket` 创建工单。标准 AI Chat UI 即可完成。

**替代方案**: 在 Agent Chat 中嵌入工单创建表单组件。

**理由**: Tool 调用是 Agent 的原生能力。Agent 可以在对话中灵活决定何时搜索服务、如何引导用户补充信息、何时创建工单。工单创建后 `agent_session_id` 关联对话，处理人可回溯完整上下文。不需要任何前端定制。

### D7: AI Copilot 是打开一个带有工单上下文的 Agent Session，复用现有 API

**选择**: 工单详情页提供"AI 协助"按钮，点击后调用 `POST /api/v1/ai/sessions`（传入 agent_id=处理协助 Agent ID），并在 initial message 中注入当前工单的摘要信息。后续交互复用 `/api/v1/ai/sessions/:sid/messages` 和 `/api/v1/ai/sessions/:sid/stream`。

**理由**: 复用 AI App 现有的 Session 管理和 SSE 流式响应能力。前端只需在工单详情页嵌入一个 Chat 面板（或跳转到 AI Chat 页面）。

### D8: 决策超时使用 context.WithTimeout，可配，默认 30 秒

**选择**: SmartEngine 在调用 LLM Client 时使用 `context.WithTimeout(ctx, timeout)`。超时时间从 `agent_config.decision_timeout_seconds` 读取，缺省 30 秒。超时后将工单放入人工决策队列。

**理由**: LLM 调用可能因网络或模型负载导致长时间等待。30 秒是大多数场景的合理上限。对于复杂决策（如需要知识检索）的服务，管理员可调高到 60 秒。

### D9: Fallback 策略——连续 3 次 AI 失败后自动转人工队列

**选择**: 每个工单维护一个 `ai_failure_count`（在 TicketActivity 或工单维度）。Agent 调用失败（超时/解析错误/校验不通过）时 +1，成功时归零。连续达到 3 次后，自动将工单转入人工决策队列，不再尝试 AI 决策，直到管理员手动选择"重新尝试 AI"。

**理由**: 避免在 AI 服务故障期间反复调用浪费资源。3 次是一个平衡点——给 AI 重试机会，又不过度延迟。管理员可以在故障恢复后手动恢复 AI 决策。

## Risks / Trade-offs

- **[风险] 智能引擎对 AI App 的强依赖** → 缓解：SmartEngine 通过 IOC 延迟解析 AI App 服务（AgentService、LLM Client、KnowledgeService），AI App 不存在时智能服务创建被禁用（UI 灰掉 engine_type="smart" 选项），经典服务完全不受影响
- **[风险] Agent 幻觉导致非法决策** → 缓解：严格的 DecisionPlan 校验（next_step_type 必须在 Policy 允许列表内、参与人必须在候选列表内、action_id 必须存在）；校验失败重试一次，仍失败转人工
- **[风险] 决策延迟影响用户体验** → 缓解：决策异步执行（Scheduler async task `itsm-smart-progress`），前端通过轮询/SSE 获取进展；决策超时（默认 30s）确保不会无限等待
- **[风险] Collaboration Spec 编写质量参差** → 缓解：提供模板库；可选的 AI 辅助 Spec 生成（后续迭代）；空 Spec 时 Agent 仅依据自身 system_prompt 和工单上下文做决策（降级但不阻塞）
- **[权衡] 工具注册的跨 App 耦合** → 接受：ITSM 通过 IOC 可选注入 AI App ToolRegistry，编译期无直接依赖。AI App 不存在时工具不注册，ITSM 经典功能不受影响
- **[权衡] 预置 Agent 的 Seed 依赖 AI App** → 接受：Seed 中检测 AI App 可用性，不可用时跳过 Agent 创建。管理员安装 AI App 后重启即可补充创建
