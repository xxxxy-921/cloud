## Context

ITSM 智能模式已有 Smart Engine spec（`itsm-smart-engine`），定义了基于 AI Agent 的工单决策循环。但当前缺少两个关键环节：

1. **引擎配置管理**：Smart Engine 依赖 AI 模块的 Provider + Model 来调用 LLM，但没有统一的配置入口。管理员需要在 AI 模块和 ITSM 模块之间来回跳转。
2. **工作流解析**：协作规范（CollaborationSpec）到工作流（workflowJSON）的转换能力未实现。当前服务定义 UI 的工作流 Tab 是只读的，无法生成工作流。

AI 模块已有完整的 Provider（连接管理 + 密钥加密）和 Model（模型定义）体系。Agent 模型当前支持 `assistant` 和 `coding` 两种类型，均面向用户交互场景。

## Goals / Non-Goals

**Goals:**
- 在 ITSM 模块内提供引擎配置页面，管理员用 ITSM 业务语言配置 AI 能力
- 通过 AI Agent 实体承载 LLM 配置（Provider + Model + Temperature + SystemPrompt），与 AI 模块体系联动
- 支持工作流解析引擎和运行时决策引擎独立配置（可用不同模型）
- 实现协作规范到工作流的 LLM 解析能力，前端一键触发
- Seed 自动创建 ITSM 内部 Agent 和默认配置

**Non-Goals:**
- 不在 ITSM 引擎配置页面暴露 system_prompt 编辑（跟 Agent 走）
- 不实现工作流的拖拽编辑器（本次只做解析生成 + 节点配置查看）
- 不改变 Smart Engine 的运行时决策循环逻辑（只提供配置入口）
- 不支持多租户引擎配置（全局配置，非服务级别）

## Decisions

### Decision 1: 用 Agent 实体承载 LLM 配置，新增 `internal` 类型

**选择**：扩展 Agent 模型新增 `internal` 类型，ITSM 创建 `itsm.generator` 和 `itsm.runtime` 两个 internal Agent。

**替代方案**：
- A) 在 SystemConfig K/V 中存储 provider_id + model（简单但脱离 AI 模块体系）
- B) 创建 ITSM 专属配置模型（独立但重复建设）

**理由**：Agent 已有 provider_id（通过 model_id → AIModel → Provider 关联）、temperature、system_prompt 等字段，完美匹配 ITSM 的需求。新增 `internal` 类型让这些 Agent 不出现在 AI 模块的用户 Agent 列表中，保持界面干净。其他模块（如未来的自动化引擎）也可以复用此模式。

### Decision 2: 引擎配置存储分层——Agent 字段 + SystemConfig

**选择**：LLM 相关配置（Provider、Model、Temperature）存在 Agent 实体上；ITSM 特有运维参数（decision_mode、max_retries、timeout、reasoning_log）存在 SystemConfig K/V 表。

**理由**：Agent 已有的字段涵盖 LLM 配置，无需重复存储。SystemConfig 适合存少量运维参数，且 Seed 机制已支持。聚合 API 在 handler 层将两者合并为统一的响应结构。

### Decision 3: 聚合 API 而非组合调用

**选择**：ITSM 提供 `GET/PUT /api/v1/itsm/engine/config` 聚合 API，内部同时读写 Agent 和 SystemConfig。

**替代方案**：前端分别调用 AI Agent API + SystemConfig API。

**理由**：聚合 API 让前端只需一次请求，且隐藏了存储细节。如果将来存储方式变化（比如 Agent 换成专属模型），前端无需改动。

### Decision 4: 工作流解析独立于 Agent Session

**选择**：工作流解析是一次性 LLM 调用（`POST /api/v1/itsm/workflows/generate`），直接用 `llm.Client` 完成，不创建 AgentSession。

**理由**：解析不需要对话上下文、工具调用、Memory 等 Agent 运行时特性。直接构建 LLM Client 更轻量、更快。Agent 实体只用来存储配置，不用来驱动执行。

### Decision 5: 工作流 Tab 升级为可交互模式

**选择**：Smart 类型服务的工作流 Tab 从只读升级为可交互——支持查看 LLM 生成的节点详情（activity_kind、参与人、表单字段），但仍不支持拖拽编排。

**理由**：生成的工作流需要人工确认节点配置是否正确（比如参与人是否对、表单字段是否齐全）。纯只读无法满足这个需求。但完整的拖拽编辑器超出本次范围。

## Risks / Trade-offs

**[Agent 模型耦合]** → ITSM 依赖 AI Agent 模型的字段结构。如果 Agent 模型重构（如移除 temperature 字段），ITSM 配置会受影响。Mitigation：internal Agent 的使用模式简单（只读/写固定字段），变更风险低。

**[内部 Agent 的 model_id 关联]** → Agent 当前通过 model_id FK 关联 AIModel（非直接关联 Provider）。ITSM 引擎配置页面需要展示 Provider 下拉 → Model 下拉的联动，但实际存储的是 model_id。Mitigation：前端通过 Provider list API + Model list API（按 provider 过滤）实现联动，保存时只需存 model_id。

**[Seed 时 Provider 未创建]** → 首次安装时 ITSM Seed 先于用户配置 AI Provider 运行，internal Agent 的 model_id 为空。Mitigation：允许 model_id 为空的 internal Agent；引擎配置页面检测到未配置时展示引导提示。

**[单次 LLM 调用的稳定性]** → 工作流解析依赖 LLM 生成正确的 JSON 结构。Mitigation：后端增加重试（max_retries 配置）+ 结构校验 + 拓扑校验；前端展示解析失败时的友好提示。

## Open Questions

- Smart Engine 运行时决策（`itsm-smart-progress` 任务）当前如何获取 Agent 配置？是否需要同步改造为读取 `itsm.runtime` Agent？（建议：是，保持一致）
