## Why

ITSM 智能模式需要调用 AI 能力完成两件事：**将协作规范解析为可视化工作流**（设计时）和**运行时工单决策**。当前这些 AI 配置信息（Provider、Model、Agent）散落在 AI 模块中，业务管理员配置 ITSM 服务时无从得知使用哪个 LLM、如何调整参数。需要在 ITSM 模块内提供专属的引擎配置面板，屏蔽 AI 模块底层细节，让管理员用 ITSM 领域语言完成配置。同时，服务定义页面需要新增「解析工作流」能力——用户编写完协作规范后一键生成 ReactFlow 可视化工作流。

## What Changes

- **新增 ITSM 引擎配置页面**：ITSM 模块侧边栏新增「引擎配置」菜单，页面分三个区块：工作流解析引擎、运行时决策引擎、通用设置
- **扩展 Agent 类型**：AI Agent 模型新增 `internal` 类型，标记模块内部使用的 Agent，不在 AI 模块 Agent 管理列表中展示
- **ITSM Seed 内置 Agent**：Seed 阶段自动创建 `itsm.generator`（工作流解析）和 `itsm.runtime`（运行时决策）两个 internal Agent，携带内置 system_prompt
- **引擎配置聚合 API**：`GET/PUT /api/v1/itsm/engine/config` 一个接口聚合读写 Agent 配置（Provider + Model + Temperature）和 SystemConfig（decision_mode、max_retries 等）
- **工作流解析 API**：`POST /api/v1/itsm/workflows/generate` 接收协作规范 + 可用动作，读取 `itsm.generator` Agent 配置调用 LLM，返回 workflow_json
- **服务定义 UI 增加解析能力**：基础信息 Tab 的协作规范区域新增「解析工作流」按钮；工作流 Tab 从只读升级为可交互（支持查看 LLM 生成结果、手动微调节点配置）

## Capabilities

### New Capabilities
- `itsm-engine-config`: ITSM 智能引擎配置管理——引擎配置页面 UI + 聚合 API + Seed 默认值
- `itsm-workflow-generate`: 协作规范解析为工作流——后端 LLM 调用 + 拓扑校验 + 前端解析按钮与结果渲染

### Modified Capabilities
- `ai-agent`: Agent 模型新增 `internal` 类型，Agent 列表 API 默认过滤 internal 类型
- `itsm-service-definition-ui`: 基础信息 Tab 新增协作规范解析按钮；工作流 Tab 从只读升级为支持查看生成结果与节点配置

## Impact

- **后端**：`internal/app/itsm/` 新增 engine config handler/service；`internal/app/ai/` Agent 模型新增 type 字段
- **前端**：`web/src/apps/itsm/` 新增 engine-config 页面；修改 service-definition 详情页
- **API**：新增 2 个 ITSM 端点（engine config + workflow generate）；修改 AI Agent list API（过滤 internal）
- **Seed**：ITSM seed 新增 2 个 internal Agent 记录 + SystemConfig 默认值
- **菜单**：ITSM 侧边栏新增「引擎配置」菜单项
