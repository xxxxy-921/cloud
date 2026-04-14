## Why

ITSM 双引擎项目 Phase 3。Phase 1 建立了 App 骨架、全量数据模型和手动工单流转，Phase 2 实现了经典 BPMN 工作流引擎（ReactFlow 编辑器 + 确定性状态机）。本期实现智能工作流引擎——完全由 AI Agent 驱动的工单流转，以及 ITSM Agent 工具集（对话式提单、工单查询等）。

完成后的完整链路：用户与 IT 服务台 Agent 对话提工单 → SmartEngine 调用流程决策 Agent 决策流转 → 处理人借助 AI Copilot 处理 → Agent 验证完结。同时经典引擎继续独立运行，两套引擎共享统一的 Ticket/Activity/Assignment 数据层。

## What Changes

### 智能工作流引擎（SmartEngine）
- SmartEngine 实现 WorkflowEngine 接口（Start/Progress/Cancel），作为 ClassicEngine 的对等实现
- 决策循环核心：TicketCase 快照构建 → TicketPolicySnapshot 编译 → Agent Decision → Validate → Progress
- 新增数据结构：TicketCase（工单快照）、TicketPolicySnapshot（策略快照）、TicketDecisionPlan（决策计划）
- Agent 调用：通过 AI App 的 LLM Client，将 Collaboration Spec 作为首要上下文 + 知识库作为补充上下文
- 信心机制：confidence 阈值（服务级可配），高信心自动执行，低信心等待人工确认
- 人工覆盖：强制跳转、改派、驳回，记录 overridden_by 和覆盖原因
- 决策超时：context.WithTimeout，可配（默认 30s），超时转人工队列
- Fallback 降级：AI 不可用时转人工队列，连续 3 次失败自动停用 AI 决策
- 运行时动态流程图：基于 TicketActivity 历史动态生成路径可视化，展示 AI 决策推理和人工覆盖标记

### ITSM Agent 工具集
- 6 个 Builtin Tool 注册到 AI App 的 ToolRegistry：
  - `itsm.search_services` — 搜索可用服务
  - `itsm.create_ticket` — 创建工单（source=agent，关联 agent_session_id）
  - `itsm.query_ticket` — 查询工单详情（支持 ID 和编号查询，权限校验）
  - `itsm.list_my_tickets` — 查询我的工单（状态筛选、分页）
  - `itsm.cancel_ticket` — 取消工单（权限校验）
  - `itsm.add_comment` — 添加评论
- 工具通过 IOC 注入 AI App 的 ToolRegistry，AI App 不存在时静默跳过

### 预置 Agent（Seed）
- IT 服务台 Agent（public，绑定全部 6 个 ITSM 工具）— 用户侧对话提单
- 流程决策 Agent（private，temperature 0.2）— 系统侧工单流转决策
- 处理协助 Agent（team，绑定知识库）— 处理人侧 AI Copilot

### 前端
- 智能服务配置面板：Collaboration Spec 编辑器（Markdown）、Agent 下拉选择、知识库多选绑定、信心阈值滑块
- 人工覆盖操作面板：确认/拒绝 AI 决策、强制跳转、改派
- 动态流程图组件：基于 Activity 历史的路径渲染 + AI 推理展示 + 覆盖标记
- AI Copilot 入口：工单详情中打开 Agent Session，自动注入工单上下文

### Scheduler
- 注册 `itsm-smart-progress` 异步任务：执行智能决策循环

## Capabilities

### New Capabilities
- `itsm-smart-engine`: 智能工作流引擎——决策循环 + 信心机制 + 人工覆盖 + Fallback + 运行时可视化
- `itsm-agent-tools`: Agent 工具集（6 个工具）+ 3 个预置 Agent + 对话式提单 + AI Copilot

### Modified Capabilities
（无）

## Impact

- **后端新增**：`internal/app/itsm/engine/smart.go`（SmartEngine）、`engine/snapshot.go`（TicketCase 构建）、`engine/policy.go`（Policy 编译）、`engine/planner.go`（Agent 调用）、`engine/validator_smart.go`（决策校验）、`tools/`（ITSM 工具注册和实现）
- **后端修改**：`TicketService.Create()` 在 `engine_type="smart"` 时调用 `SmartEngine.Start()`；`app.go` 的 `Providers()` 新增 IOC 注入 AI App 服务
- **前端新增**：智能服务配置面板组件、人工覆盖操作面板组件、动态流程图组件、AI Copilot 入口
- **前端修改**：服务定义编辑器接入智能引擎配置、工单详情页接入人工覆盖和 AI Copilot
- **AI App 集成**：IOC 注入 AgentService / LLM Client / KnowledgeService / ToolRegistry
- **Scheduler**：注册 `itsm-smart-progress` 异步任务
- **Seed**：3 个预置 Agent 定义（IT 服务台 / 流程决策 / 处理协助）+ Agent 的 ITSM 工具绑定
- **Casbin 策略**：为人工覆盖 API 端点添加策略
