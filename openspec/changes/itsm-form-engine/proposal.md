## Why

ITSM 工作流中的表单目前没有统一规范：ServiceDefinition.FormSchema 和 NodeData.FormSchema 都是未定义结构的 json.RawMessage，前端硬编码表单字段而非动态渲染，后端无法做 schema-based 校验。没有表单设计器，管理员无法可视化地创建和管理工单表单。这意味着每个新的服务定义都需要开发介入来编写表单 UI。

一个完整的 BPMN 引擎由流程引擎和表单引擎两大核心组成。表单引擎是后续所有工作流能力（流程变量绑定、节点表单配置、运行时动态渲染）的前置基础设施。

## What Changes

- **定义 Form Schema v1 规范**：JSON 格式的表单描述规范，覆盖 16 种字段类型（text, textarea, number, select, multi_select, radio, checkbox, switch, date, datetime, date_range, user_picker, dept_picker, position_picker, file_upload, rich_text），支持校验规则、条件显隐、布局分区、流程变量绑定、节点级字段权限
- **新增 FormDefinition 模型**：`itsm_form_definitions` 表，独立管理表单定义，支持版本号、代码引用、服务关联
- **新增 Form CRUD API**：`/api/v1/itsm/forms` 完整增删改查，含 schema 结构校验
- **新增 Form Designer 页面**：左侧字段类型面板（拖拽）+ 中间画布预览（排序）+ 右侧字段属性编辑器，ITSM 菜单新增「表单管理」入口
- **新增 FormRenderer 组件**：根据 schema 动态渲染 shadcn/ui 表单，支持 create/edit/view 三种模式和节点级字段权限
- **新增 Form Validator**：前端从 schema 动态生成 Zod schema 做客户端校验；后端 Go 端实现相同校验逻辑做服务端校验
- **迁移 ServiceDefinition**：`form_schema` JSON 字段改为 `form_id` FK 引用 FormDefinition
- **Seed 内置表单**：预创建「通用事件」「变更申请」等默认表单模板

## Capabilities

### New Capabilities
- `itsm-form-schema`: Form Schema v1 规范定义，覆盖字段类型、校验规则、条件显隐、布局、变量绑定、权限控制
- `itsm-form-definition`: FormDefinition 模型 + CRUD API + schema 结构校验
- `itsm-form-designer`: 可视化表单设计器页面（字段面板 + 画布 + 属性编辑器）
- `itsm-form-renderer`: 动态表单渲染组件，支持多模式 + 字段权限
- `itsm-form-validator`: 前后端统一的 schema-based 表单校验

### Modified Capabilities
- `itsm-service-definition`: FormSchema JSON 字段迁移为 form_id FK；工作流节点的 formSchema 改为 formId 引用
- `itsm-seed`: 新增内置表单模板

## Impact

- **后端**：`internal/app/itsm/` 新增 form_def model/repository/service/handler (~400 行)；新增 `internal/app/itsm/form/` schema 规范 + validator (~300 行)；修改 ServiceDefinition 模型
- **前端**：`web/src/apps/itsm/` 新增 `components/form-engine/` (FormRenderer ~500 行, FormDesigner ~800 行)；新增 `pages/forms/` 管理页面；修改 api.ts
- **菜单**：ITSM 侧边栏新增「表单管理」
- **Seed**：新增 3-5 个内置表单模板
- **依赖**：无前置依赖
