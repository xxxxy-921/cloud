## Context

当前 Classic Engine 中数据以 JSON blob 形式分散存储在 `ticket.form_data` 和各 `activity.form_data` 中。Gateway 的 `buildEvalContext()` 每次求值时临时从最新 activity 的 form_data 解析 JSON 构建 map，这意味着：

- 只能看到**最后一个**完成节点的表单数据，前序节点数据不可引用
- Action 节点的 webhook 响应无法进入条件求值
- 通知模板无法引用结构化数据
- 无法审计"某变量何时被谁改为什么值"

FormField 类型中已有 `binding` 字段（form-engine change 中定义），但引擎未使用它。本 change 激活 binding，建立统一变量存储。

**依赖**: itsm-form-engine（FormField.Binding 定义 + FormDefinition 模型）

## Goals / Non-Goals

**Goals:**
- 建立 `itsm_process_variables` 独立存储表，支持按 ticket + scope + key 管理流程变量
- 工单创建时，根据 start form 的 field binding 自动初始化变量
- Activity 完成时，根据 form field binding 自动写入/更新变量
- 重构 `buildEvalContext()` 从变量表读取，替代临时 JSON 解析
- 提供变量查看 API 供前端和调试使用
- 前端工单详情页新增变量面板（只读展示）

**Non-Goals:**
- 子流程变量 scope 隔离（等 subprocess change，当前只支持 scope="root"）
- 通用表达式引擎（当前 `evaluateCondition()` 的 7 种运算符够用）
- 变量变更历史追踪（后续需求驱动）
- NodeData input/output mapping 配置（binding 语法糖够用）
- Action 节点 webhook 结果自动写入变量（后续 advanced-nodes change）
- 前端变量编辑功能（仅只读查看）

## Decisions

### D1: 独立表 vs JSON 列

**选择: 独立 `itsm_process_variables` 表**

```
id | ticket_id | scope_id | key | value (TEXT/JSON) | value_type | source | updated_at
```

UNIQUE(ticket_id, scope_id, key)

替代方案: 在 ticket 上加 `variables` JSON 列 — 单次读取更快，但无法对单个变量索引查询，SQLite 下并发 JSON merge 也有问题。独立表对后续变量历史审计更友好，也是 Activiti/Camunda 的标准做法。

### D2: value_type 类型系统

支持 5 种类型: `string | number | boolean | json | date`

- `value` 列统一存为 TEXT（JSON 序列化）
- `value_type` 列记录原始类型
- 读取时按 value_type 做类型还原
- 写入时由引擎根据表单字段类型自动推断

不做复杂类型（array、object），`json` 类型作为逃生舱口覆盖任意结构。

### D3: binding 触发时机

**选择: Activity 完成时引擎自动写入**

在 `classic.go` 的 `Progress()` 方法中，activity 完成后：
1. 解析 activity 的 form_schema，提取所有有 `binding` 的字段
2. 从 form_data 中取出对应值
3. 写入 `itsm_process_variables` 表（UPSERT by ticket_id + scope_id + key）

source 字段记录 `"form:<activity_id>"` 以标明数据来源。

工单创建时同理：解析 service 的 start form schema，binding 字段写入变量。

### D4: buildEvalContext 重构

将 `condition.go` 中的 `buildEvalContext()` 改为：
1. 从 `itsm_process_variables` 查询 ticket 的所有 root scope 变量
2. 构建 `var.<key>` 命名空间
3. 保留 `ticket.*` 和 `activity.outcome` 原有键
4. 废弃 `form.*` 前缀 → 迁移为 `var.*`（向前兼容：同时填充 `form.*` 和 `var.*`，后续版本移除 form.*）

### D5: 前端变量面板

在工单详情页（`tickets/[id]`）的右侧或 tab 区域新增 `VariablesPanel` 组件：
- 调用 `GET /api/v1/itsm/tickets/:id/variables`
- 只读表格展示: key | value | type | source | updated_at
- 不做编辑功能

## Risks / Trade-offs

**[性能] 每次 Gateway 求值多一次 DB 查询** → 变量数量通常 < 50，单次 WHERE ticket_id=? 查询在 SQLite 下 < 1ms。可接受。

**[迁移] 现有工单没有 process variables** → 不做回填。存量工单继续用旧的 form_data 解析逻辑（`form.*` 向前兼容）。新工单自动写入变量。

**[binding 遗漏] 表单设计者忘记配置 binding** → 没有 binding 的字段数据只存在 activity.form_data 中，不会写入变量。这是设计意图：只有显式 bind 的字段才升级为流程变量。

**[并发写入] parallel 审批模式下多人同时完成** → 不同 activity 写入不同变量键（因为不同表单），UNIQUE 约束 + UPSERT 保证幂等。如果意外写同一个键，后完成者覆盖（last-write-wins），可接受。
