## ADDED Requirements

### Requirement: 表单定义下拉选择
系统 SHALL 在 form 节点的属性面板中提供 FormDefinition 下拉选择器。

#### Scenario: 展示可用表单列表
- **WHEN** 用户选中 form 节点并打开属性面板
- **THEN** 表单绑定选择器调用 `GET /api/v1/itsm/forms` 展示可用的 FormDefinition 列表，按名称显示

#### Scenario: 选择表单后预览字段
- **WHEN** 用户从下拉列表中选择一个 FormDefinition
- **THEN** 选择器下方显示该表单的字段预览列表（字段名 + 类型），formSchema 保存到节点 data

#### Scenario: 已绑定表单的回显
- **WHEN** 节点 data 中已存在 formDefinitionId
- **THEN** 选择器回显当前绑定的表单名称，并展示字段预览

#### Scenario: 清除表单绑定
- **WHEN** 用户点击选择器的清除按钮
- **THEN** formDefinitionId 和 formSchema 从节点 data 中移除

### Requirement: 服务动作选择器
系统 SHALL 在 action 节点的属性面板中提供 ServiceAction 下拉选择器。

#### Scenario: 展示可用动作列表
- **WHEN** 用户选中 action 节点并打开属性面板
- **THEN** 动作选择器调用 `GET /api/v1/itsm/services/:serviceId/actions` 展示当前服务的 ServiceAction 列表

#### Scenario: 选择动作后显示预览
- **WHEN** 用户从下拉列表中选择一个 ServiceAction
- **THEN** 选择器下方显示该动作的 URL 和 HTTP Method 预览信息，actionId 保存到节点 data

#### Scenario: 已绑定动作的回显
- **WHEN** 节点 data 中已存在 actionId
- **THEN** 选择器回显当前绑定的动作名称，并展示 URL/Method 预览
