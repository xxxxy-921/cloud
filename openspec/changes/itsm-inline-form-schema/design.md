## Context

当前 ITSM 的表单定义（FormDefinition）作为独立实体存储在 `itsm_form_definitions` 表中，通过 `formId` 被 ServiceDefinition 和 WorkflowNode 引用。完整的管理链路包括后端 4 文件（model、repository、service、handler）、前端 2 页面（列表 + 详情）、5 个 API 端点。

实际使用中，表单几乎不会跨服务复用。独立管理引入了不必要的复杂度：
- 管理员需在表单页面和服务定义页面之间跳转
- 引擎运行时需额外查询 FormDefinition 表
- 版本同步依赖 Activity 级别的 snapshot，说明运行时本质上需要的是内嵌 schema

## Goals / Non-Goals

**Goals:**
- 将表单 schema 内嵌到 ServiceDefinition 和 WorkflowNode 中
- 移除 FormDefinition 独立管理（表、API、页面）
- 简化工单创建和流转时的表单加载逻辑

**Non-Goals:**
- 不改变 FormSchema 结构本身（field types、validation rules、binding 机制保持不变）
- 不改变 FormRenderer / FormDesigner 前端组件
- 不引入"表单模板"功能（可后续单独做）
- 不改变 TicketActivity.FormSchema 的运行时 snapshot 机制

## Decisions

### 1. ServiceDefinition 用 IntakeFormSchema 替代 FormID

```go
// Before
FormID *uint `json:"formId" gorm:"index"`

// After
IntakeFormSchema JSONField `json:"intakeFormSchema" gorm:"type:text"`
```

**理由**: 进件表单是服务定义的固有属性，不需要间接引用。内嵌后创建工单时直接从 ServiceDefinition 读取 schema，无需 JOIN FormDefinition。

### 2. WorkflowNode 用内嵌 FormSchema 替代 FormID

```go
// Before (engine/workflow.go NodeData)
FormID string `json:"formId,omitempty"`

// After
FormSchema json.RawMessage `json:"formSchema,omitempty"`
```

**理由**: 工作流 JSON 在工单创建时已经被 snapshot 到 ticket.workflow_json。如果 form schema 也内嵌在其中，则天然跟随 snapshot，无需额外处理版本一致性。

**前端 WFNodeData 对应变更**:
```typescript
// Before
formDefinitionId?: string

// After
formSchema?: FormSchema
```

### 3. 引擎加载表单的逻辑简化

```
// Before (createActivityForNode)
1. 读取 node.FormID
2. 查询 FormDefinition 表获取 schema
3. 写入 activity.FormSchema

// After
1. 读取 node.FormSchema（已在 workflow JSON 中）
2. 直接写入 activity.FormSchema
```

引擎不再需要注入 FormDefRepository。

### 4. 前端工作流编辑器变更

form/user_task 节点的属性面板：
- **Before**: 下拉选择一个 FormDefinition
- **After**: 内嵌 FormDesigner 组件，直接在节点属性面板中编辑 schema

服务定义编辑页面：
- **Before**: 下拉选择进件表单
- **After**: 内嵌 FormDesigner 组件

### 5. 数据库迁移策略

不写自动迁移脚本。原因：
- ITSM 模块尚处于开发阶段，没有生产数据需要迁移
- 直接删除 `itsm_form_definitions` 表，GORM AutoMigrate 自动处理新字段
- seed 数据更新：将 FormDefinition seed 中的 schema 内嵌到对应 ServiceDefinition seed

## Risks / Trade-offs

- **[WorkflowJSON 体积增大]** → 表单 schema 内嵌后 workflow_json 字段变大。可接受：单个表单 schema 通常 < 5KB，一个流程 5-10 个节点最多增加 50KB，对 TEXT 字段无压力。
- **[表单复用丧失]** → 如果未来需要跨服务复用表单，需引入"表单模板"机制（clone 而非 reference）。当前不做。
- **[编辑器 UX 复杂度]** → 节点属性面板内嵌 FormDesigner 增加面板复杂度。缓解：FormDesigner 已是独立组件，嵌入即可。
