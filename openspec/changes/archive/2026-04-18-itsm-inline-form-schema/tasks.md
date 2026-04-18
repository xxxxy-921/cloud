## 1. 后端模型变更

- [x] 1.1 ServiceDefinition 模型：移除 `FormID *uint`，新增 `IntakeFormSchema JSONField`；同步更新 Response 结构和 ToResponse()
- [x] 1.2 engine/workflow.go NodeData：移除 `FormID string`，新增 `FormSchema json.RawMessage`
- [x] 1.3 移除 model_form.go（FormDefinition 模型）
- [x] 1.4 从 app.go Models() 中移除 FormDefinition 注册

## 2. 后端服务层变更

- [x] 2.1 移除 form_def_repository.go、form_def_service.go、form_def_handler.go
- [x] 2.2 从 app.go Providers() 中移除 FormDef 相关 IOC 注册
- [x] 2.3 从 app.go Routes() 中移除 `/forms` 路由组
- [x] 2.4 ServiceDefService：创建/更新时对 IntakeFormSchema 调用 form.ValidateSchema()（如非 nil），移除 formId 存在性校验
- [x] 2.5 更新 seed.go：移除 FormDefinition seed，将 schema 内嵌到 ServiceDefinition seed 的 IntakeFormSchema 字段

## 3. 引擎逻辑变更

- [x] 3.1 ticket_service.go：创建工单时从 ServiceDefinition.IntakeFormSchema 读取表单 schema，不再通过 formId 查询 FormDefinition
- [x] 3.2 engine executor（createActivityForNode）：从 node.FormSchema 直接读取 schema 写入 activity.FormSchema，移除 FormDefinition 查询逻辑
- [x] 3.3 variable_writer.go writeFormBindings：创建工单时从 IntakeFormSchema 解析 binding，确认不依赖 FormDefinition

## 4. 前端变更

- [x] 4.1 移除 pages/forms/ 目录（列表页 + 详情页）
- [x] 4.2 移除 module.ts 中的 forms 路由注册和导航菜单项
- [x] 4.3 工作流编辑器 FormBindingPicker：从"选择表单定义"下拉改为读取内嵌 formSchema 的只读字段展示
- [x] 4.4 工作流编辑器 task-node 摘要：从检查 formDefinitionId 改为检查 formSchema
- [x] 4.5 移除 api.ts 中 FormDef 相关类型和函数
- [x] 4.6 WFNodeData 类型：移除 formDefinitionId 字段

## 5. 验证

- [x] 5.1 go build 编译通过
- [x] 5.2 前端无 FormDefinition/formDefinitionId 残留引用
- [x] 5.3 前端 lint 通过（预有错误均为既有问题，非本次变更引入）
