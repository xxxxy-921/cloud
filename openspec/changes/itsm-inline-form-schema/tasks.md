## 1. 后端模型变更

- [ ] 1.1 ServiceDefinition 模型：移除 `FormID *uint`，新增 `IntakeFormSchema JSONField`；同步更新 Response 结构和 ToResponse()
- [ ] 1.2 engine/workflow.go NodeData：移除 `FormID string`，新增 `FormSchema json.RawMessage`
- [ ] 1.3 移除 model_form.go（FormDefinition 模型）
- [ ] 1.4 从 app.go Models() 中移除 FormDefinition 注册

## 2. 后端服务层变更

- [ ] 2.1 移除 form_def_repository.go、form_def_service.go、form_def_handler.go
- [ ] 2.2 从 app.go Providers() 中移除 FormDef 相关 IOC 注册
- [ ] 2.3 从 app.go Routes() 中移除 `/forms` 路由组
- [ ] 2.4 ServiceDefService：创建/更新时对 IntakeFormSchema 调用 form.ValidateSchema()（如非 nil），移除 formId 存在性校验
- [ ] 2.5 更新 seed.go：移除 FormDefinition seed，将 schema 内嵌到 ServiceDefinition seed 的 IntakeFormSchema 字段

## 3. 引擎逻辑变更

- [ ] 3.1 ticket_service.go：创建工单时从 ServiceDefinition.IntakeFormSchema 读取表单 schema，不再通过 formId 查询 FormDefinition
- [ ] 3.2 engine executor（createActivityForNode）：从 node.FormSchema 直接读取 schema 写入 activity.FormSchema，移除 FormDefinition 查询逻辑
- [ ] 3.3 variable_writer.go writeFormBindings：创建工单时从 IntakeFormSchema 解析 binding，确认不依赖 FormDefinition

## 4. 前端变更

- [ ] 4.1 移除 pages/forms/ 目录（列表页 + 详情页）
- [ ] 4.2 移除 module.ts 中的 forms 路由注册和导航菜单项
- [ ] 4.3 服务定义编辑 UI：将进件表单从"选择表单"下拉改为内嵌 FormDesigner 组件，编辑 intakeFormSchema
- [ ] 4.4 工作流编辑器 form/user_task 节点属性面板：将"选择表单"下拉改为内嵌 FormDesigner 组件，编辑 node.data.formSchema
- [ ] 4.5 更新 API 调用：service definition 的创建/更新 payload 中 formId → intakeFormSchema
- [ ] 4.6 工单创建页面：从 service.intakeFormSchema 读取表单 schema 而非通过 formId 获取

## 5. 验证

- [ ] 5.1 go build 编译通过
- [ ] 5.2 确认 seed 数据正确加载（服务定义带有内嵌表单 schema）
- [ ] 5.3 前端 lint 通过，无 FormDefinition 残留引用
