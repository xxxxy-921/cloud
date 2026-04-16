## Why

当前 ITSM 服务台智能体只有 6 个基础 CRUD 工具（search_services、create_ticket、query_ticket 等），无法支撑完整的"意图识别 → 服务匹配 → 信息收集 → 草稿确认 → 工单创建"闭环。决策智能体也没有组织架构上下文。参考 bklite-cloud 的成熟实现，需要补齐服务台的完整工具链、通用上下文工具，并复刻经过验证的 agent system prompt。

## What Changes

- **删除** 现有 3 个 ITSM 预置智能体（IT 服务台、ITSM 流程决策、ITSM 处理协助）
- **新增** 2 个智能体，复刻 bklite 的 system prompt：
  - `IT 服务台智能体`（public, react, temp 0.3）— 用户交互、提单引导
  - `流程决策智能体`（private, react, temp 0.2）— SmartEngine 内部使用
- **替换/新增** ITSM 工具（从 6 个扩展到 10 个）：
  - 新增：`itsm.service_match`、`itsm.service_confirm`、`itsm.service_load`、`itsm.new_request`、`itsm.draft_prepare`、`itsm.draft_confirm`、`itsm.validate_participants`、`itsm.ticket_withdraw`
  - 增强：`itsm.ticket_create`（加入前置条件检查）
  - 保留：`itsm.my_tickets`
  - **删除**：`itsm.search_services`、`itsm.query_ticket`、`itsm.cancel_ticket`、`itsm.add_comment`
- **新增** 3 个通用工具（AI App seed，toolkit: "general"）：
  - `general.current_time` — 多时区当前时间
  - `system.current_user_profile` — 当前用户资料+组织归属
  - `organization.org_context` — 组织架构查询
- **新增** 服务台会话状态管理（利用 session state 存储草稿生命周期）

## Capabilities

### New Capabilities

- `itsm-service-desk-toolkit`: 服务台智能体完整工具链 — 服务匹配、服务加载、草稿生命周期、参与人校验、工单创建闭环
- `itsm-general-tools`: 通用上下文工具 — 时间、用户档案、组织架构查询，供所有 ITSM 智能体使用

### Modified Capabilities

- `itsm-agent-tools`: 重写工具定义和 seed 绑定，删除旧工具、新增完整工具链、更新智能体配置和 prompt

## Impact

- `internal/app/itsm/tools/provider.go` — 重写工具定义 + seed 逻辑
- `internal/app/itsm/tools/handlers.go` — 重写 handler 接口和实现
- `internal/app/ai/seed.go` — 新增 3 个通用工具 seed
- `internal/app/ai/tool_model.go` — 可能需要扩展 Tool 模型（toolkit 分类）
- AI Agent 执行层需要能调度 ITSM 工具 handler
- 依赖 Org App 的 service/repository（优雅降级：Org App 未安装时通用工具返回空结果）
