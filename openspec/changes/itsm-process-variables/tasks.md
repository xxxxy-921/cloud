## 1. ProcessVariable 模型 + Repository

- [x] 1.1 创建 ProcessVariable 模型：id, ticket_id (index), scope_id, key, value (text), value_type, source, created_at, updated_at。TableName `itsm_process_variables`，UNIQUE(ticket_id, scope_id, key)。ToResponse() 方法按 value_type 反序列化 value（`internal/app/itsm/model_variable.go`）
- [x] 1.2 在 ITSM App.Models() 中注册 ProcessVariable，确保 AutoMigrate
- [x] 1.3 创建 VariableRepository：SetVariable(upsert), GetVariable(ticket+scope+key), ListByTicket(ticket_id), DeleteByTicket(ticket_id)（`internal/app/itsm/variable_repository.go`）
- [x] 1.4 IOC 注册：在 app.go Providers() 中 do.Provide VariableRepository

## 2. Variable Service + Handler

- [x] 2.1 创建 VariableService：SetVariable(做类型校验), GetVariable, ListByTicket, BulkSet(batch upsert for form binding), InferValueType(fieldType→valueType 映射)（`internal/app/itsm/variable_service.go`）
- [x] 2.2 IOC 注册 VariableService
- [x] 2.3 创建 VariableHandler：GET /api/v1/itsm/tickets/:id/variables 返回变量列表（`internal/app/itsm/variable_handler.go`）
- [x] 2.4 IOC 注册 VariableHandler + 路由挂载到 app.go Routes()
- [x] 2.5 添加 Casbin 策略：GET /api/v1/itsm/tickets/:id/variables 对 admin 角色

## 3. Form Binding → 变量写入

- [x] 3.1 新增 `writeFormBindings(tx, ticketID, scopeID, formSchemaJSON, formDataJSON, source)` 工具函数：解析 form schema 中有 binding 的字段，从 form data 提取值，调用 VariableRepository.SetVariable 写入（`internal/app/itsm/engine/variable_writer.go`）
- [x] 3.2 实现 fieldTypeToValueType 映射函数：text/textarea/email/url/select/radio/rich_text→string, number→number, switch/checkbox(无options)→boolean, date/datetime→date, multi_select/checkbox(有options)/date_range→json
- [x] 3.3 修改 classic.go `Start()` 方法：在工单创建并处理 start 节点后，调用 writeFormBindings 写入初始变量（source="form:start"）
- [x] 3.4 修改 classic.go `Progress()` 方法：在 activity 完成（form_data 写入后），调用 writeFormBindings 写入变量（source="form:<activity_id>"）

## 4. Gateway 条件重构

- [x] 4.1 修改 condition.go `buildEvalContext()`：新增从 itsm_process_variables 查询变量，填充 `var.<key>` 命名空间
- [x] 4.2 保留 `form.<key>` 向前兼容：将变量同时填充到 form.* 前缀
- [x] 4.3 保留 `ticket.*` 和 `activity.outcome` 原有逻辑不变
- [x] 4.4 添加 fallback：当 ticket 无 process variables 时，回退到旧的 form_data JSON 解析逻辑

## 5. 前端变量面板

- [x] 5.1 前端 API 函数：fetchTicketVariables(ticketId) 返回变量数组（`web/src/apps/itsm/api.ts`）
- [x] 5.2 创建 VariablesPanel 组件：只读表格展示 key/value/type/source/updatedAt，支持空状态（`web/src/apps/itsm/components/variables-panel.tsx`）
- [x] 5.3 集成到工单详情页：在 ticket detail 页面中引入 VariablesPanel（修改 `web/src/apps/itsm/pages/tickets/[id]/index.tsx`）
- [x] 5.4 i18n：在 zh-CN.json 和 en.json 中添加 variables 相关翻译键（variables.title, variables.empty, variables.key, variables.value, variables.type, variables.source, variables.updatedAt）
