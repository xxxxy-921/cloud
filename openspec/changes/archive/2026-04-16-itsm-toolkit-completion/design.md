## Context

Metis ITSM 模块有一个 AI 驱动的服务台智能体和一个流程决策智能体，但当前服务台智能体仅有 6 个基础 CRUD 工具，无法支撑完整的"意图识别 → 服务匹配 → 信息收集 → 草稿确认 → 工单创建"提单闭环。参考 bklite-cloud 的成熟实现（经过生产验证的 16 个工具 + 4 个 Toolset + 会话状态机），需要将缺失的工具链补齐。

当前状态：
- `internal/app/itsm/tools/provider.go` — 6 个简单工具定义 + 3 个预置智能体 seed
- `internal/app/itsm/tools/handlers.go` — `TicketQuerier` 接口 + `Registry` 执行器
- `internal/app/ai/seed.go` — AI App 的 builtin 工具（search_knowledge 等）
- 决策智能体（SmartEngine）已实现完整的 prompt→JSON 决策模式，不需要改动

Metis Org App 结构完整（Department/Position/UserPosition），所有查询能力已具备，无需改动 Org App 代码。

## Goals / Non-Goals

**Goals:**
- 将服务台智能体工具从 6 个扩展到 10 个，覆盖完整提单闭环
- 新增 3 个通用上下文工具（时间、用户档案、组织架构）
- 复刻 bklite 经过验证的 agent system prompt（服务台 19 条约束版、决策 4 条原则版）
- 删除旧的 3 个智能体，替换为 2 个高质量智能体
- 实现服务台会话状态管理（草稿生命周期）

**Non-Goals:**
- 不改动决策智能体的执行模式（SmartEngine 保持 prompt→JSON，不加 tool-use）
- 不改动 Org App 的模型或服务层
- 不实现 LLM 级别的智能服务匹配（service_match 使用关键词+权重匹配，不调 LLM）
- 不实现处理协助智能体（后续单独做）
- 不改动前端代码

## Decisions

### D1: 工具执行层复用现有 Registry 模式

**决定**: 扩展现有 `tools/handlers.go` 的 `Registry` + `ToolHandler` 模式，不引入新的执行框架。

**理由**: 现有模式简单直接，`ToolHandler func(ctx, userID, args) (json.RawMessage, error)` 签名已够用。bklite 用 Python 的 pydantic-ai 运行时，但 Metis 的 Go 实现不需要那么重的框架。

**替代方案**: 引入专门的 ToolExecutor 接口层 → 过度设计，当前场景不需要。

### D2: 会话状态存储在 SessionMessage 中

**决定**: 服务台状态机（候选服务列表、已选服务、草稿版本等）存储在 `AgentSession` 的一个新 `State` JSON 字段中。

**理由**:
- 状态与会话生命周期绑定，会话结束状态自然失效
- bklite 用 Django session 存储，等价做法
- 不需要新建表

**替代方案**: 存在独立的 state 表 → 增加复杂度，且生命周期管理更麻烦。

### D3: 通用工具放 AI App seed，ITSM 工具放 ITSM seed

**决定**: `general.current_time`、`system.current_user_profile`、`organization.org_context` 三个通用工具在 AI App 的 `seed.go` 中注册（toolkit: "general"）。ITSM 工具在 ITSM 的 `tools/provider.go` 中注册（toolkit: "itsm"）。

**理由**: 通用工具不仅 ITSM 用，其他 App 的 Agent 也可能需要。放 AI App 层更合理。

### D4: TicketQuerier 接口拆分为多个专用接口

**决定**: 将现有的单一 `TicketQuerier` 接口拆分为：
- `ServiceDeskOperator` — 服务台提单流程（match、load、draft、create 等）
- `ContextProvider` — 通用上下文（time、user_profile、org_context）

**理由**: 职责分离。`ServiceDeskOperator` 由 ITSM App 实现，`ContextProvider` 由 AI App 桥接 Org App 实现。避免一个接口过于臃肿。

### D5: service_match 使用关键词权重匹配而非 LLM

**决定**: `itsm.service_match` 使用数据库查询 + 关键词匹配 + 置信度评分，不调用 LLM。

**理由**:
- bklite 的 LLM 匹配虽精确但增加延迟和 token 成本
- Go 端发起 LLM 调用会增加复杂度（需要拿 provider 配置）
- Agent 本身就是 LLM，它可以基于 match 结果做判断
- 后续可升级为 LLM 匹配

### D6: AgentSession 增加 State 字段

**决定**: 在 `AgentSession` 模型上增加 `State` 字段（JSON text），用于存储服务台状态机。

**理由**: 最小侵入性改动，一个字段解决问题。ReactExecutor 在执行工具时可读写此字段。

## Risks / Trade-offs

- **[关键词匹配精度不如 LLM]** → 可接受，Agent 层会做二次判断；后续可升级
- **[Org App 未安装时通用工具不可用]** → `current_user_profile` 和 `org_context` 返回降级结果（仅基础用户信息），不阻断流程
- **[Session State 并发写入]** → 服务台场景是单用户单会话，不存在并发问题
- **[旧工具数据残留]** → seed 逻辑需要清理旧工具记录和旧智能体绑定
