## 1. 生产代码扩展

- [x] 1.1 扩展 `engine/executor_action.go` 的 `replaceTemplateVars`：新增 `{{ticket.code}}` 变量支持
- [x] 1.2 扩展 `engine/executor_action.go` 的 `replaceTemplateVars`：解析 ticket.FormData JSON，支持 `{{ticket.form_data.<key>}}` 一级键值替换
- [x] 1.3 运行 `go build -tags dev ./cmd/server/` 确认编译通过

## 2. 测试基础设施

- [x] 2.1 在 `steps_common_test.go` 新增 `LocalActionReceiver`：基于 httptest.Server，提供 Records()、RecordsByPath()、Clear()、URL() 方法，handler 记录 Path/Method/Body
- [x] 2.2 在 `steps_common_test.go` 新增 `syncActionSubmitter`：实现 engine.TaskSubmitter，收到 `itsm-action-execute` 时同步调用 ActionExecutor.Execute() + classicEngine.Progress()，其他任务 no-op
- [x] 2.3 修改 `steps_common_test.go` 的 `bddContext.reset()`：当 `bc.actionReceiver != nil` 时，使用 syncActionSubmitter 替代 noopSubmitter 初始化引擎

## 3. DB Backup Support 文件

- [x] 3.1 创建 `db_backup_support_test.go`：定义 `dbBackupCollaborationSpec` 常量（描述预检→DBA审批→放行流程）
- [x] 3.2 定义 `dbBackupCasePayload` 结构体和 `dbBackupCasePayloads`（2 组 case payload：requester-1 和 requester-2，包含 database_name、source_ip、whitelist_window、access_reason）
- [x] 3.3 实现 `generateDbBackupWorkflow()`：复用 generateVPNWorkflow 模式，替换协作规范为 db backup 版本
- [x] 3.4 实现 `publishDbBackupSmartService()`：创建 ServiceCatalog + Priority + Agent + ServiceDefinition + 2 个 ServiceAction（precheck + apply，URL 指向 LocalActionReceiver）

## 4. DB Backup Step Definitions

- [x] 4.1 创建 `steps_db_backup_test.go`，实现 `registerDbBackupSteps()`
- [x] 4.2 实现 Given step `已定义数据库备份白名单临时放行协作规范`
- [x] 4.3 实现 Given step `已基于协作规范发布数据库备份白名单放行服务（智能引擎）`：调用 publishDbBackupSmartService，初始化 LocalActionReceiver 和 syncActionSubmitter
- [x] 4.4 实现 Given step `"<username>" 已创建数据库备份白名单放行工单，场景为 "<case_key>"`：根据 case_key 选择 payload，创建 Ticket
- [x] 4.5 实现 Then step `预检动作已为当前工单触发`：断言 TicketActionExecution 中存在 precheck action 的 success 记录 + receiver /precheck 有请求
- [x] 4.6 实现 Then step `放行动作已为当前工单触发`：断言 TicketActionExecution 中存在 apply action 的 success 记录 + receiver /apply 有请求
- [x] 4.7 实现 Then step `放行动作未为当前工单触发`：断言 apply action 的 TicketActionExecution 不存在 + receiver /apply 无请求
- [x] 4.8 实现 Then step `工单 "<ticket_ref>" 的动作记录与工单 "<other_ref>" 完全隔离`：断言两张工单的 TicketActionExecution 记录互不包含对方的 ticket_id

## 5. Feature 文件

- [x] 5.1 创建 `features/db_backup_whitelist_action_flow.feature`：Background（系统初始化、参与人、协作规范、服务发布）+ 3 个 Scenario（完整流程、权限校验、并行隔离）

## 6. 注册与验证

- [x] 6.1 在 `bdd_test.go` 的 `initializeScenario` 中注册 `registerDbBackupSteps(sc, bc)`
- [x] 6.2 运行 `go build -tags dev ./cmd/server/` 确认编译通过
- [x] 6.3 运行 `make test-bdd` 验证全部 3 个 scenario 通过
