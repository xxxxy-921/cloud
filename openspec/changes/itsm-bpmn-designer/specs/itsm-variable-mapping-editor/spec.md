## ADDED Requirements

### Requirement: Input Mapping 编辑
系统 SHALL 提供 input mapping 编辑器，支持配置流程变量到表单默认值的映射列表。

#### Scenario: 添加 input mapping
- **WHEN** 用户点击 "添加映射" 按钮
- **THEN** 新增一行映射，左侧为流程变量名下拉，右侧为表单字段名下拉

#### Scenario: 删除 input mapping
- **WHEN** 用户点击映射行的删除按钮
- **THEN** 移除该映射行

#### Scenario: 显示已有映射
- **WHEN** 节点 data 中已存在 inputMapping 数组
- **THEN** 编辑器展示所有映射行，每行显示 变量名 → 字段名

### Requirement: Output Mapping 编辑
系统 SHALL 提供 output mapping 编辑器，支持配置表单字段到流程变量的映射列表。

#### Scenario: 添加 output mapping
- **WHEN** 用户点击 "添加映射" 按钮
- **THEN** 新增一行映射，左侧为表单字段名下拉，右侧为流程变量名输入

#### Scenario: 变量名自动补全
- **WHEN** 用户在变量名输入框中输入
- **THEN** 系统基于已有流程变量名提供自动补全建议

### Requirement: 脚本赋值编辑
系统 SHALL 提供脚本赋值编辑器，每行为 变量名 + 表达式 的可编辑列表。

#### Scenario: 添加脚本赋值
- **WHEN** 用户在 script 节点的属性面板中点击 "添加赋值"
- **THEN** 新增一行，左侧为变量名输入，右侧为表达式输入

#### Scenario: 表达式语法提示
- **WHEN** 用户在表达式输入框中输入
- **THEN** 输入框下方显示可用的内置函数和变量引用提示
