## Why

当前 ITSM 的表单定义（FormDefinition）作为独立实体管理，服务定义和工作流节点通过 `formId` 引用。但在真实 ITSM 场景中，表单几乎不会跨服务复用——每个服务的进件字段、审批字段、中间环节字段都高度定制化。独立管理增加了跳转成本、版本同步风险，且与引擎分离（经典引擎在流程中定义表单、智能引擎只需进件表单）的自然模型不匹配。

## What Changes

- **BREAKING** 移除 `itsm_form_definitions` 表及其全部后端代码（model、repository、service、handler）
- **BREAKING** 移除 `ServiceDefinition.FormID` 外键字段，替换为 `IntakeFormSchema JSONField`（内嵌 JSON）
- 工作流节点 `NodeData.FormID` 改为 `NodeData.FormSchema json.RawMessage`（内嵌 JSON）
- 移除前端表单管理页面（`/itsm/forms`），表单设计直接在服务定义 UI 和工作流编辑器中完成
- 移除相关 API 端点（`/api/v1/itsm/forms/*`）
- 更新工单创建和流转逻辑，从内嵌 schema 读取而非通过 formId 查询
- 更新 seed 数据，移除 FormDefinition seed，将 schema 内嵌到 ServiceDefinition seed

## Capabilities

### New Capabilities

_无新能力，本次为架构简化重构。_

### Modified Capabilities

- `itsm-form-definition`: 移除独立表单管理，schema 内嵌到 ServiceDefinition
- `itsm-service-definition`: ServiceDefinition 模型变更，FormID → IntakeFormSchema
- `itsm-workflow-editor`: 工作流编辑器中节点表单从引用改为内嵌
- `itsm-ticket-create`: 工单创建时从 ServiceDefinition.IntakeFormSchema 读取表单
- `itsm-classic-engine`: 引擎从节点内嵌 formSchema 加载表单，不再查询 FormDefinition

## Impact

- **后端**: 删除 4 个文件（model_form.go, form_def_handler/service/repository.go），修改 model_catalog.go、engine/workflow.go、ticket_service.go、engine executor 相关文件、seed.go、app.go（路由注册）
- **前端**: 删除 forms 页面（pages/forms/），修改服务定义 UI、工作流编辑器的表单节点面板
- **数据库**: 需要迁移脚本将现有 FormDefinition 的 schema 内嵌到引用它的 ServiceDefinition 和 WorkflowJSON 中
- **API**: 移除 `/api/v1/itsm/forms/*` 端点，修改 service definition API 的请求/响应结构
