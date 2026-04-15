## 1. Schema 规范 + 后端模型

- [ ] 1.1 定义 Go 端 FormSchema 类型体系：FormSchema, FormField, FormLayout, LayoutSection, ValidationRule, VisibilityRule, FieldOption structs（`internal/app/itsm/form/schema.go`）
- [ ] 1.2 实现 schema 结构校验函数 ValidateSchema(schema FormSchema) []ValidationError：检查 version、field key 唯一性、field type 合法性、layout 引用存在性（`internal/app/itsm/form/schema.go`）
- [ ] 1.3 创建 FormDefinition 模型：itsm_form_definitions 表，字段 id/name/code/description/schema/version/scope/service_id/is_active，TableName(), ToResponse()（`internal/app/itsm/model_form.go`）
- [ ] 1.4 在 ITSM App.Models() 中注册 FormDefinition，确保 AutoMigrate

## 2. 后端 Repository + Service + Handler

- [ ] 2.1 FormDefRepository：Create, FindByID, FindByCode, Update, Delete, List(keyword, page, pageSize)（`internal/app/itsm/form_def_repository.go`）
- [ ] 2.2 FormDefService：Create（code 去重 + schema 校验）、Update（version +1 + schema 校验）、Delete（检查引用）、Get、List。定义 sentinel errors: ErrFormDefCodeExists, ErrFormDefNotFound, ErrFormDefInUse（`internal/app/itsm/form_def_service.go`）
- [ ] 2.3 FormDefHandler：POST/GET/PUT/DELETE /api/v1/itsm/forms，含请求结构体、审计字段、错误映射（`internal/app/itsm/form_def_handler.go`）
- [ ] 2.4 IOC 注册：在 app.go Providers() 中 do.Provide FormDefRepository → FormDefService → FormDefHandler
- [ ] 2.5 路由注册：在 app.go Routes() 中挂载 forms CRUD 路由

## 3. 后端表单数据校验器

- [ ] 3.1 实现 ValidateFormData(schema FormSchema, data map[string]any) []FieldValidationError：遍历 schema.fields，按 validation rules 校验 data 中对应字段值，支持 required/minLength/maxLength/min/max/pattern/email/url（`internal/app/itsm/form/validator.go`）

## 4. ServiceDefinition 迁移

- [ ] 4.1 修改 ServiceDefinition 模型：FormSchema JSONField → FormID *uint gorm:"index"
- [ ] 4.2 修改 ServiceDefinitionResponse：formSchema → formId
- [ ] 4.3 修改 ServiceDef handler Create/Update 请求结构体：formSchema → formId，Create 时校验 formId 引用存在
- [ ] 4.4 修改 engine/workflow.go NodeData：formSchema json.RawMessage → FormID string json:"formId"
- [ ] 4.5 修改 engine/classic.go handleForm/handleProcess：通过 formId 查找 FormDefinition，快照 schema 到 activity.form_schema

## 5. Seed

- [ ] 5.1 在 seed.go 中创建 3 个内置 FormDefinition：form_general_incident、form_change_request、form_service_request，含完整 FormSchema JSON
- [ ] 5.2 修改现有 ServiceDefinition seed：formSchema 内联改为 formId 引用
- [ ] 5.3 添加"表单管理"菜单项 + Casbin 策略（itsm:form:list, itsm:form:create, itsm:form:update, itsm:form:delete）

## 6. 前端 FormRenderer 组件

- [ ] 6.1 定义 TypeScript 类型：FormSchema, FormField, FormLayout, LayoutSection, ValidationRule, VisibilityRule, FieldOption, FormRendererProps（`web/src/apps/itsm/components/form-engine/types.ts`）
- [ ] 6.2 实现字段渲染策略：每种 field type 一个渲染函数，映射到 shadcn/ui 组件（`web/src/apps/itsm/components/form-engine/field-renderers.tsx`）
- [ ] 6.3 实现 Zod schema 动态生成：buildZodSchema(schema: FormSchema, visibleFields: Set<string>) → ZodObject，支持 required/minLength/maxLength/min/max/pattern/email/url 规则映射（`web/src/apps/itsm/components/form-engine/build-zod-schema.ts`）
- [ ] 6.4 实现条件显隐引擎：useFieldVisibility(schema, watchValues) → Set<string> of visible field keys，实时评估 visibility conditions（`web/src/apps/itsm/components/form-engine/use-visibility.ts`）
- [ ] 6.5 实现 FormRenderer 主组件：集成 React Hook Form + Zod + 字段渲染 + 布局 sections + 权限控制 + create/edit/view 三种模式（`web/src/apps/itsm/components/form-engine/form-renderer.tsx`）
- [ ] 6.6 导出 barrel：`web/src/apps/itsm/components/form-engine/index.ts`

## 7. 前端 FormDesigner 页面

- [ ] 7.1 前端 API 函数：fetchFormDefs, fetchFormDef, createFormDef, updateFormDef, deleteFormDef（`web/src/apps/itsm/api.ts`）
- [ ] 7.2 表单列表页：使用 useListPage 展示 FormDefinition 列表，支持 keyword 搜索、新建按钮、点击进入编辑（`web/src/apps/itsm/pages/forms/index.tsx`）
- [ ] 7.3 FieldTypePalette 组件：左侧字段类型面板，分 4 组展示 15 种类型，点击添加字段（`web/src/apps/itsm/components/form-engine/designer/field-palette.tsx`）
- [ ] 7.4 DesignerCanvas 组件：中间画布，按 section 分组显示字段卡片，支持选中、上下移动、删除（`web/src/apps/itsm/components/form-engine/designer/designer-canvas.tsx`）
- [ ] 7.5 FieldPropertyEditor 组件：右侧属性编辑器，包含通用属性 + 类型专有属性 + 校验规则编辑器 + 条件显隐编辑器（`web/src/apps/itsm/components/form-engine/designer/field-property-editor.tsx`）
- [ ] 7.6 FormDesigner 主页面：三栏布局 + toolbar（名称/code/section 管理/预览/保存）+ 状态管理，加载/保存 FormDefinition（`web/src/apps/itsm/pages/forms/[id].tsx`）
- [ ] 7.7 预览模式：toolbar 预览按钮切换画布为 FormRenderer(mode=create) 渲染

## 8. 路由 + 菜单集成

- [ ] 8.1 前端路由注册：在 ITSM module.ts 中添加 /itsm/forms 和 /itsm/forms/:id 路由（lazy import）
- [ ] 8.2 i18n：在 zh-CN.json 和 en.json 中添加表单管理相关翻译键
