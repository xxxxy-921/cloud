## Why

当前 SmartEngine 的"AI 决策"本质是一次性 Structured Output 调用：将全量上下文（含所有用户列表）灌入 prompt，LLM 单次输出 JSON。这不是 Agent，无法分步推理、按需查询、利用知识库。面对 500+ 用户企业、模糊协作规范、需要历史模式参考的场景，单次调用模型不够用。需要将决策引擎从"AI 辅助路由"升级为"带工具的 Agentic 决策循环"。

## What Changes

- **SmartEngine 核心决策流程重写**：`callAgent()` 从单次 `llm.Chat()` 改为带工具的 ReAct 多轮循环，复用现有 `internal/llm` 的 tool calling 基础设施（ChatRequest.Tools / ChatResponse.ToolCalls）
- **新增决策域工具集**（7 个只读工具）：`decision.ticket_context`、`decision.knowledge_search`、`decision.resolve_participant`、`decision.user_workload`、`decision.similar_history`、`decision.sla_status`、`decision.list_actions`，让 Agent 按需查询而非全量灌入
- **简化初始上下文**：TicketCase 快照精简为 seed 信息（工单基本字段 + 服务名称），细节由工具按需获取；TicketPolicySnapshot 移除 `participant_candidates` 全量用户列表，只保留 `allowed_step_types` 和状态约束
- **删除 `callAgentWithCorrection()` 格式纠正重试**：ReAct 循环天然支持多轮纠错，不再需要专门的重试函数
- **更新决策智能体 seed 配置**：`MaxTurns` 从 1 提升到 8，绑定决策域工具，SystemPrompt 增加工具使用指引

## Capabilities

### New Capabilities
- `itsm-decision-tools`: 决策域工具集的定义、参数 schema、执行实现，及其与引擎的集成机制
- `itsm-smart-react`: SmartEngine 内置轻量 ReAct 循环，基于 `llm.Client.Chat()` 的多轮 tool-use 调用

### Modified Capabilities
- `itsm-smart-engine`: 决策循环核心流程变更 — 从单次 LLM 调用改为 ReAct 多轮工具调用；TicketCase/PolicySnapshot 结构简化；校验逻辑调整（不再校验候选人列表）

## Impact

- **后端 engine/ 目录**：`smart.go` 重写核心方法，新增 `smart_react.go` 和 `smart_tools.go` 两个文件，`engine.go` 和 `resolver.go` 小改
- **后端 tools/ 目录**：`provider.go` 更新决策智能体 seed
- **后端 app.go**：`NewSmartEngine` 构造函数参数扩展（增加 resolver）
- **LLM 层零改动**：完全复用 `internal/llm` 的 ChatRequest{Tools}/ChatResponse{ToolCalls} 和 OpenAI/Anthropic client 实现
- **AI 基础配置完全复用**：AgentProvider.GetAgentConfig()（模型/协议/密钥/温度/提示词）、KnowledgeSearcher、ai_agents 表管理后台配置均不变
- **前端零改动**：本次仅改后端引擎逻辑
- **Classic Engine 零影响**：不涉及经典引擎任何代码
