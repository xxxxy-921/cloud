## ADDED Requirements

### Requirement: 单条件可视化编辑
系统 SHALL 提供可视化条件编辑行，每行包含：变量下拉 + 运算符下拉 + 值输入。

#### Scenario: 选择变量字段
- **WHEN** 用户点击条件行的变量下拉
- **THEN** 系统展示当前流程中可用的变量列表（来自上游节点的 output mapping 和表单字段），用户选择后填入字段名

#### Scenario: 选择运算符
- **WHEN** 用户点击条件行的运算符下拉
- **THEN** 系统展示可用运算符列表：equals、not_equals、contains_any、gt、lt、gte、lte、is_empty、is_not_empty

#### Scenario: 输入比较值
- **WHEN** 用户在值输入框中输入内容
- **THEN** 值保存到条件的 value 字段，支持字符串和数字类型

### Requirement: 多条件 AND/OR 组合
系统 SHALL 支持多个条件通过 AND/OR 逻辑组合，最大嵌套深度为 2 层。

#### Scenario: 添加 AND 条件
- **WHEN** 用户点击 "添加条件" 按钮
- **THEN** 在当前组内新增一行条件，组内条件默认以 AND 连接

#### Scenario: 切换 AND/OR 逻辑
- **WHEN** 用户点击条件组左侧的逻辑标签（AND/OR）
- **THEN** 该组的逻辑在 AND 和 OR 之间切换

#### Scenario: 添加条件组
- **WHEN** 用户点击 "添加条件组" 按钮
- **THEN** 在顶层新增一个子条件组，子组内可添加多个条件

#### Scenario: 删除条件
- **WHEN** 用户点击条件行的删除按钮
- **THEN** 移除该条件行；如果组内无条件则移除整个组

### Requirement: 条件向后兼容
系统 SHALL 自动将旧格式的单条件（GatewayCondition）升级为 ConditionGroup 格式。

#### Scenario: 加载旧格式条件
- **WHEN** 编辑器加载包含旧格式 `{ field, operator, value }` 条件的 edge
- **THEN** 自动包装为 `{ logic: "and", conditions: [{ field, operator, value }] }` 格式展示

#### Scenario: 保存新格式条件
- **WHEN** 用户编辑条件后保存
- **THEN** 条件以 ConditionGroup 格式保存到 edge data 中

### Requirement: 条件摘要显示
系统 SHALL 在条件构建器上方显示当前条件的自然语言摘要。

#### Scenario: 单条件摘要
- **WHEN** 条件组仅包含一个条件 `{ field: "priority", operator: "equals", value: "high" }`
- **THEN** 摘要显示 "priority = high"

#### Scenario: 多条件摘要
- **WHEN** 条件组包含多个条件以 AND 连接
- **THEN** 摘要显示 "priority = high AND amount > 10000"
