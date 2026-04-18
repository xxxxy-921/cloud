## ADDED Requirements

### Requirement: syncActionSubmitter 同步执行 Action 任务

系统 SHALL 提供 `syncActionSubmitter` 实现 `engine.TaskSubmitter`，在 BDD 测试中同步执行 `itsm-action-execute` 任务（调用 ActionExecutor + auto-progress），其他任务类型 no-op。

#### Scenario: itsm-action-execute 任务被同步执行
- **WHEN** smart engine 创建 action 类型活动并提交 `itsm-action-execute` 任务
- **THEN** syncActionSubmitter SHALL 同步调用 `ActionExecutor.Execute()`
- **AND** 执行完成后 SHALL 自动调用 `engine.Progress()` 标记活动完成
- **AND** TicketActionExecution 表中 SHALL 存在对应记录

#### Scenario: 非 action 任务被忽略
- **WHEN** engine 提交 `itsm-smart-progress` 或其他任务
- **THEN** syncActionSubmitter SHALL 静默忽略（no-op）

### Requirement: LocalActionReceiver HTTP 测试接收器

系统 SHALL 提供 `LocalActionReceiver`，基于 `httptest.Server` 在测试进程内启动 HTTP 服务，记录所有收到的请求。

#### Scenario: 记录 HTTP 请求
- **WHEN** ActionExecutor 向 LocalActionReceiver 的 /precheck 路径发送 POST 请求
- **THEN** receiver.Records() SHALL 包含该请求
- **AND** 记录中包含 Path、Method、Body 字段

#### Scenario: 按路径过滤记录
- **WHEN** receiver 收到 /precheck 和 /apply 各 1 个请求
- **THEN** receiver.RecordsByPath("/precheck") SHALL 返回 1 条记录
- **AND** receiver.RecordsByPath("/apply") SHALL 返回 1 条记录

#### Scenario: 清空记录
- **WHEN** 调用 receiver.Clear()
- **THEN** receiver.Records() SHALL 返回空列表

### Requirement: replaceTemplateVars 支持 form_data 和 code 变量

`replaceTemplateVars` SHALL 支持 `{{ticket.form_data.<key>}}` 格式的模板变量（从 ticket 的 FormData JSON 字段中提取一级键值），以及 `{{ticket.code}}` 变量。

#### Scenario: form_data 变量替换
- **WHEN** body 模板为 `{"db":"{{ticket.form_data.database_name}}"}`
- **AND** ticket.FormData 为 `{"database_name":"prod-db-01"}`
- **THEN** 替换结果 SHALL 为 `{"db":"prod-db-01"}`

#### Scenario: code 变量替换
- **WHEN** body 模板为 `{"code":"{{ticket.code}}"}`
- **AND** ticket.Code 为 "DB-001"
- **THEN** 替换结果 SHALL 为 `{"code":"DB-001"}`

#### Scenario: 向后兼容已有变量
- **WHEN** body 模板包含 `{{ticket.id}}` 和 `{{ticket.status}}`
- **THEN** 替换行为 SHALL 与扩展前一致
