## REMOVED Requirements

### Requirement: FormDefinition model
**Reason**: 表单 schema 内嵌到 ServiceDefinition 和 WorkflowNode 中，不再需要独立表单实体
**Migration**: 将 FormDefinition.Schema 内容内嵌到引用它的 ServiceDefinition.IntakeFormSchema 和 WorkflowNode.FormSchema 中

### Requirement: Form CRUD API
**Reason**: FormDefinition 实体移除，对应 API 端点不再需要
**Migration**: 表单管理在服务定义 UI 和工作流编辑器中内联完成

### Requirement: Schema validation on save
**Reason**: 校验逻辑移入 ServiceDefinition 和 WorkflowNode 的保存路径
**Migration**: form.ValidateSchema() 函数保留，调用方从 FormDefService 改为 ServiceDefService 和引擎

### Requirement: Form definition audit trail
**Reason**: FormDefinition 实体移除，审计跟踪随之移除
**Migration**: 表单变更的审计通过 ServiceDefinition 的 update 审计覆盖

### Requirement: IOC registration
**Reason**: FormDefRepository、FormDefService、FormDefHandler 全部移除
**Migration**: 无需替代
