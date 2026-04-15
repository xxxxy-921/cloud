## Why

当前 ITSM 工作流引擎没有流程变量的概念。表单数据以 raw JSON 存在 ticket.FormData 和 activity.FormData 中，网关条件评估通过 `buildEvalContext` 临时从多个 JSON 字段拼装 map。这导致：
- 表单提交的数据无法在后续节点中被条件引用（除非手动解析 JSON）
- 服务任务的输入/输出无法映射到统一数据源
- 通知模板无法引用结构化的流程数据
- 脚本任务（后续 change）没有可操作的变量存储
- 子流程（后续 change）没有变量作用域隔离

流程变量是 BPMN 引擎的血液系统——所有节点通过变量交换数据，所有条件通过变量求值。

## What Changes

- **新增 ProcessVariable 模型**：`itsm_process_variables` 表，存储 ticket_id + scope_id + key + value(JSON) + value_type + source，支持 UNIQUE(ticket_id, scope_id, key)
- **新增 VariableScope 机制**：scope_id 默认为 "root"（流程级），子流程场景下为 subprocess node id（局部变量），变量解析按 local → parent → root 逐层查找
- **新增表达式引擎基础**：简单表达式求值器（支持变量引用、比较运算、字符串操作、三元运算），供 gateway 条件和后续 script task 使用
- **重构 condition.go**：`buildEvalContext` 改为从 `itsm_process_variables` 表读取，替代从 form_data JSON 临时解析
- **重构 ClassicEngine 表单提交**：Activity 完成时，根据 form field binding 写入 process variables
- **Ticket 创建集成**：工单创建时，将 start form 数据根据 binding 初始化为流程变量
- **变量查看 API**：`GET /api/v1/itsm/tickets/:id/variables` 查询流程变量（管理/调试用）
- **前端变量面板**：工单详情页新增流程变量查看面板

## Capabilities

### New Capabilities
- `itsm-process-variable`: ProcessVariable 模型 + 作用域机制 + CRUD + 查询 API
- `itsm-expression-engine`: 简单表达式求值器（变量引用、比较、字符串、三元）
- `itsm-variable-binding`: 表单字段 → 流程变量的自动写入机制

### Modified Capabilities
- `itsm-classic-engine`: Gateway 条件从变量表求值；表单提交写入变量
- `itsm-ticket-create`: 创建时初始化流程变量
- `itsm-ticket-detail-ui`: 新增变量面板

## Impact

- **后端**：`internal/app/itsm/` 新增 process_variable model/repository/service (~200 行)；新增 `internal/app/itsm/expr/` 表达式引擎 (~300 行)；重构 `engine/condition.go` (~50 行改动)；重构 `engine/classic.go` 表单提交部分 (~30 行改动)
- **前端**：工单详情页新增变量面板组件 (~150 行)
- **依赖**：① itsm-form-engine（form field binding 定义）
