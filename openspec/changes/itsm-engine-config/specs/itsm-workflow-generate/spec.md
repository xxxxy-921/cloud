## ADDED Requirements

### Requirement: 协作规范解析 API

系统 SHALL 提供 `POST /api/v1/itsm/workflows/generate` API，接收协作规范和上下文信息，调用 LLM 解析生成 ReactFlow 格式的工作流 JSON。API 受 JWT + Casbin 权限保护。

请求结构：
```json
{
  "service_id": 1,
  "collaboration_spec": "用户提交申请，收集数据库名...",
  "available_actions": [
    {"code": "precheck", "name": "预检查", "description": "检查数据库连接"}
  ]
}
```

响应结构：
```json
{
  "workflow_json": {
    "nodes": [...],
    "edges": [...]
  }
}
```

#### Scenario: 成功解析协作规范
- **WHEN** 管理员调用解析 API 并传入有效的协作规范
- **THEN** 系统 SHALL 读取 `itsm.generator` Agent 配置构建 LLM Client，将协作规范 + 内置约束提示词 + 可用动作信息组合为 prompt，调用 LLM 获取结果，解析为 workflow_json 返回

#### Scenario: 引擎未配置
- **WHEN** 调用解析 API 但 `itsm.generator` Agent 的 model_id 为空
- **THEN** 系统 SHALL 返回 400 错误 "工作流解析引擎未配置，请前往引擎配置页面设置"

#### Scenario: 协作规范为空
- **WHEN** 调用解析 API 但 collaboration_spec 为空字符串
- **THEN** 系统 SHALL 返回 400 错误 "协作规范不能为空"

#### Scenario: LLM 调用失败
- **WHEN** LLM 调用返回错误或超时
- **THEN** 系统 SHALL 按 `itsm.engine.general.max_retries` 配置重试，全部失败后返回 500 错误，包含错误摘要信息

#### Scenario: LLM 返回无效 JSON
- **WHEN** LLM 返回的内容无法解析为合法的 workflow_json 结构
- **THEN** 系统 SHALL 触发重试（计入重试次数），全部失败后返回 500 错误 "工作流解析失败，请检查协作规范描述是否清晰"

### Requirement: 工作流 JSON 结构校验

系统 SHALL 对 LLM 生成的 workflow_json 进行结构校验和拓扑校验，确保工作流合法。

#### Scenario: 结构校验
- **WHEN** LLM 返回 workflow_json
- **THEN** 系统 SHALL 校验：(1) 必须包含 nodes 和 edges 数组 (2) 每个 node 必须有 id、type、position、data 字段 (3) 每个 node.data 必须有 label 和 activity_kind (4) activity_kind 必须为 request/approve/process/action/end 之一 (5) 每个 edge 必须有 source 和 target 且引用有效 node id

#### Scenario: 拓扑校验
- **WHEN** workflow_json 通过结构校验
- **THEN** 系统 SHALL 校验：(1) 有且仅有一个 activity_kind=request 的起始节点 (2) 有且仅有一个 activity_kind=end 的结束节点 (3) 从起始节点到结束节点存在至少一条完整路径 (4) 不存在孤立节点（无入边也无出边，除起始节点无入边、结束节点无出边外）

#### Scenario: 校验失败触发重试
- **WHEN** 校验发现问题
- **THEN** 系统 SHALL 将校验错误附加到下一次 LLM 调用的 prompt 中作为修正提示，重新请求 LLM 生成

### Requirement: 解析引擎内置约束提示词

系统 SHALL 维护内置的工作流解析约束提示词（Generator System Prompt），定义 LLM 生成工作流时必须遵循的规则。此提示词存储在 `itsm.generator` Agent 的 system_prompt 字段中。

#### Scenario: 约束提示词内容
- **WHEN** 工作流解析引擎调用 LLM
- **THEN** system_prompt SHALL 包含以下约束：(1) 输出必须为 JSON 格式，包含 nodes 和 edges (2) 节点 activity_kind 限定为 request/approve/process/action/end (3) 参与人类型限定为 user/position/position_department (4) action 类型节点必须关联 action_code (5) 边必须有 source 和 target (6) 不可发明协作规范中未提及的角色、部门或动作 (7) 必须忠实于协作规范的流程描述

#### Scenario: 可用动作注入
- **WHEN** 请求中包含 available_actions
- **THEN** 系统 SHALL 将动作列表（code + name + description）注入到 LLM prompt 中，供 LLM 在生成 action 节点时引用匹配

### Requirement: 解析结果保存

系统 SHALL 支持将解析生成的 workflow_json 保存到服务定义。

#### Scenario: 保存工作流到服务定义
- **WHEN** 前端获取解析结果后调用 `PUT /api/v1/itsm/services/:id` 保存
- **THEN** 系统 SHALL 将 workflow_json 更新到 ServiceDefinition 的 workflow_json 字段
