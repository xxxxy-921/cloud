## Context

ITSM 当前的表单能力散落在多处且缺乏规范：

- `ServiceDefinition.FormSchema`：`JSONField`（json.RawMessage），存储工单创建表单，结构未定义
- `engine.NodeData.FormSchema`：`json.RawMessage`，存储工作流节点表单，同样无结构
- `TicketActivity.FormSchema` / `FormData`：运行时表单快照和用户提交数据
- 前端工单创建页 (`tickets/create/index.tsx`)：使用 react-hook-form + zod 硬编码 title/description/serviceId/priorityId 四个字段，无法根据服务定义动态渲染

项目使用三层架构（model → repository → service → handler）+ samber/do IOC，前端使用 React Query + shadcn/ui + React Hook Form + Zod。ITSM 表名前缀 `itsm_`，API 前缀 `/api/v1/itsm/`。用户确认可删库重建，不需要数据迁移兼容。

本 change 是 BPMN 引擎 7 个 change 中的第一个，无前置依赖。后续 ② itsm-process-variables 将基于 form field binding 写入流程变量，⑥ itsm-bpmn-designer 将通过 formId 在工作流节点中引用表单，⑦ itsm-runtime-tracking 将使用 FormRenderer 动态渲染工单表单。

## Goals / Non-Goals

**Goals:**
- 定义 Form Schema v1 JSON 规范，覆盖 16 种字段类型，支持校验规则、条件显隐、布局分区、流程变量绑定、节点级字段权限
- FormDefinition 模型独立存储，支持版本号、code 引用、服务关联，遵循现有 ITSM 三层 CRUD 模式
- 可视化 Form Designer 页面——左侧字段面板 + 中间画布 + 右侧属性编辑
- FormRenderer 组件——根据 schema 动态渲染 shadcn/ui 表单，支持 create/edit/view 三种模式
- 前后端统一校验逻辑——前端动态生成 Zod schema，后端 Go 实现相同规则
- ServiceDefinition.FormSchema 迁移为 form_id FK

**Non-Goals:**
- 不实现表单版本 diff/回滚 UI（只记录 version 数字，不存历史快照）
- 不实现复杂计算字段（如公式引擎），留给 ② itsm-process-variables 的表达式引擎
- 不实现 table 字段类型（可编辑重复行），复杂度高，MVP 后再加
- 不实现文件上传的实际存储——file_upload 字段类型定义 schema 规范，但实际上传功能依赖内核文件服务（当前未实现），本次只预留字段类型
- 不实现前端表单拖拽排序的动画效果——使用简单的上下移动按钮

## Decisions

### Decision 1: Form Schema 规范设计——自定义 JSON 而非 JSON Schema

**选择**：自定义 `FormSchema v1` JSON 格式，专门面向 ITSM 表单场景。

**替代方案**：
- A) JSON Schema (Draft 2020-12) + RJSF：标准化但 UI 定制性差，schema 结构冗长，ITSM 特有概念（变量绑定、节点权限）需要大量 extension
- B) Formily Schema：功能强大但依赖 Formily 运行时，与 shadcn/ui 体系不兼容

**理由**：自定义 schema 可以精确匹配 shadcn/ui 组件 props（Select options、DatePicker format），直接表达 ITSM 领域概念（binding、permissions），且前后端解析逻辑简单一致。15 种字段类型（去掉 table，file_upload 保留但暂不实现上传）足以覆盖 ITSM 工单场景。

**Schema 顶层结构：**
```json
{
  "version": 1,
  "fields": [FormField],
  "layout": FormLayout
}
```

**FormField 结构：**
```json
{
  "key": "urgency",
  "type": "select",
  "label": "紧急程度",
  "placeholder": "请选择紧急程度",
  "description": "影响 SLA 计算",
  "defaultValue": "medium",
  "required": true,
  "disabled": false,
  "validation": [
    {"rule": "required", "message": "请选择紧急程度"}
  ],
  "options": [
    {"label": "低", "value": "low"},
    {"label": "中", "value": "medium"},
    {"label": "高", "value": "high"}
  ],
  "visibility": {
    "conditions": [
      {"field": "category", "operator": "equals", "value": "incident"}
    ],
    "logic": "and"
  },
  "binding": "urgency",
  "permissions": {
    "node_approve_1": "readonly",
    "node_form_2": "hidden"
  },
  "width": "half",
  "props": {}
}
```

**字段类型 → 渲染组件映射（15 种）：**

| type | shadcn 组件 | 特有 props |
|------|------------|-----------|
| text | Input | maxLength |
| textarea | Textarea | maxLength, rows |
| number | Input type=number | min, max, step |
| email | Input type=email | — |
| url | Input type=url | — |
| select | Select | options, searchable |
| multi_select | Combobox 多选 | options, maxItems |
| radio | RadioGroup | options |
| checkbox | Checkbox/CheckboxGroup | options (多个时为组) |
| switch | Switch | — |
| date | DatePicker (shadcn) | format, minDate, maxDate |
| datetime | DateTimePicker | format |
| date_range | DateRangePicker | format |
| user_picker | Combobox + User API | multiple |
| dept_picker | TreeSelect + Org API | multiple |
| rich_text | Textarea + Markdown 预览 | — |

**FormLayout 结构：**
```json
{
  "columns": 2,
  "sections": [
    {
      "title": "基本信息",
      "description": "请填写工单基本信息",
      "collapsible": false,
      "fields": ["title", "description", "category"]
    }
  ]
}
```

如果 layout 为 null，则所有字段按 fields 数组顺序单列渲染。

**Validation rules：**

| rule | 适用类型 | value 类型 | 说明 |
|------|---------|-----------|------|
| required | all | — | 必填 |
| minLength | text/textarea/email/url | number | 最小长度 |
| maxLength | text/textarea/email/url | number | 最大长度 |
| min | number | number | 最小值 |
| max | number | number | 最大值 |
| pattern | text/email/url | string (regex) | 正则匹配 |
| email | text | — | 邮箱格式 |
| url | text | — | URL 格式 |

**VisibilityRule**：字段条件显隐，当 conditions 满足时字段可见。operator 支持 equals/not_equals/in/not_in/is_empty/is_not_empty。logic 默认 and。

**Permissions**：key 为 workflow node ID，value 为 `editable`（默认）/ `readonly` / `hidden`。仅在工作流节点上下文中使用——创建工单时无 node context，所有字段均可编辑。

**Binding**：映射到流程变量名。默认 binding = field key。submit 时由 ② itsm-process-variables 负责写入变量表，本 change 只定义 binding 字段。

### Decision 2: FormDefinition 独立模型 + code 引用

**选择**：独立 `itsm_form_definitions` 表，通过 code 字符串在工作流节点中引用。

**替代方案**：
- A) 继续内联在 ServiceDefinition/NodeData 中：无法复用，无版本管理
- B) 通过 ID 引用：ID 在不同环境不稳定，code 更适合 seed 和工作流 JSON

**理由**：独立表让表单可以跨服务复用（如"通用事件表单"被多个服务引用），code 引用在 seed 和工作流 JSON 中稳定。version 字段每次修改 +1，Activity 创建时快照当时的 schema 到 activity.form_schema（保留现有机制）。

**模型字段：**
```
itsm_form_definitions
├── id          uint PK (BaseModel)
├── name        string NOT NULL, max 128
├── code        string UNIQUE NOT NULL, max 64
├── description string, max 512
├── schema      text NOT NULL (FormSchema JSON)
├── version     int NOT NULL, default 1
├── scope       string NOT NULL, default "global" ("global"|"service")
├── service_id  *uint INDEX (scope=service 时关联具体服务)
├── is_active   bool NOT NULL, default true
├── created_at, updated_at, deleted_at (BaseModel)
```

### Decision 3: 后端 Schema 校验——结构校验而非语义校验

**选择**：后端在 Create/Update 时做 schema 结构校验（JSON 可解析、字段类型合法、key 唯一、layout 引用的字段存在），不做语义校验（如 select 类型是否有 options）。

**理由**：语义校验规则复杂且容易过时（新增字段类型需同步更新），结构校验足以防止损坏的 schema 入库。前端 FormDesigner 在设计时提供完整的语义引导（select 必须配 options 等），是更好的防错层。

**校验规则：**
1. schema JSON 可正确解析为 FormSchema struct
2. version >= 1
3. 所有 field.key 非空且唯一
4. 所有 field.type 在允许列表中
5. layout.sections 中引用的 field key 全部存在于 fields 数组中
6. validation rule 的 rule 值在允许列表中

### Decision 4: FormRenderer 组件——三模式 + 权限控制

**选择**：单个 `<FormRenderer>` 组件，通过 props 控制行为：

```tsx
<FormRenderer
  schema={formSchema}        // FormSchema JSON
  data={existingData}        // 已有数据 (edit/view 模式)
  mode="create|edit|view"    // 渲染模式
  nodeId="node_approve_1"    // 当前工作流节点 ID (用于权限查找)
  onSubmit={handleSubmit}    // 提交回调 (view 模式无)
  onChange={handleChange}    // 实时变更回调 (可选)
  disabled={false}           // 全局禁用
/>
```

**模式行为：**
- `create`：所有字段可编辑（忽略 permissions），有 defaultValue
- `edit`：根据 `permissions[nodeId]` 决定每个字段是 editable/readonly/hidden，data 填充已有值
- `view`：所有字段只读，纯展示

**内部实现**：使用 React Hook Form 管理表单状态，从 schema 动态生成 Zod validation schema，每个 field type 对应一个渲染函数（策略模式）。

### Decision 5: FormDesigner 页面——三栏布局 + section 管理

**选择**：三栏布局，不使用拖拽库，使用上/下移动按钮 + 添加/删除操作。

**替代方案**：
- A) dnd-kit 拖拽排序：体验好但引入新依赖，与 React Compiler 兼容性需验证
- B) @hello-pangea/dnd：较重

**理由**：表单设计器不像工作流编辑器那样需要自由画布，字段排序是线性的。按钮式排序足够且无兼容风险。后续如需升级为拖拽，可局部引入。

**三栏：**
- 左栏（200px）：字段类型面板，分组展示（基础输入、选择器、日期、高级），点击添加到当前 section
- 中栏（flex-1）：表单预览画布，按 section 分组展示字段卡片，每个卡片显示字段名+类型+必填标记，支持选中/上下移动/删除
- 右栏（320px）：选中字段的属性编辑器——通用属性（key/label/placeholder/required/width/binding）+ 类型专有属性（options for select, min/max for number 等）+ 校验规则编辑 + 条件显隐编辑

上方 toolbar：表单名称/code 编辑 + section 管理（添加/重命名/删除）+ 预览切换 + 保存按钮

### Decision 6: ServiceDefinition 迁移策略

**选择**：`ServiceDefinition.FormSchema` (JSONField) → `ServiceDefinition.FormID` (*uint FK)。由于可删库重建，直接修改模型，不做数据迁移。

**改动点：**
- model: `FormSchema JSONField` → `FormID *uint` + `gorm:"index"`
- seed: 先创建 FormDefinition，再在 ServiceDefinition 中引用 form_id
- handler: Create/Update 请求中 `formSchema` 字段改为 `formId`
- response: 新增 `formId` 字段，去掉 `formSchema`（需要 schema 内容时前端单独查 form API）
- 工作流 NodeData: `formSchema json.RawMessage` → `formId string`（引用 FormDefinition.code）

### Decision 7: Seed 内置表单

**选择**：创建 3 个内置表单模板，在 seed.go 中以 Go struct 方式定义（与现有 priority/SLA seed 模式一致）。

内置表单：
1. `form_general_incident`（通用事件表单）：标题、描述、紧急程度(select)、影响范围(select)、联系方式(text)
2. `form_change_request`（变更申请表单）：标题、描述、变更原因(textarea)、计划时间(datetime)、影响评估(textarea)、回滚方案(textarea)
3. `form_service_request`（服务请求表单）：标题、描述、期望完成日期(date)、备注(textarea)

## Risks / Trade-offs

**[自定义 Schema 锁定]** → 自定义 schema 意味着没有现成的生态工具可复用。Mitigation: schema 结构简单，转换为 JSON Schema 或 Formily Schema 在未来是可行的。

**[15 种字段类型的前端工作量]** → 每种类型需要独立的渲染函数和属性编辑面板。Mitigation: 大部分类型是 Input 的变体（text/email/url/number），实际需要独立实现的约 8 种。

**[file_upload 和 dept_picker/user_picker 依赖外部服务]** → file_upload 需要内核文件服务，user_picker/dept_picker 需要 User API 和 Org App API。Mitigation: 渲染组件定义接口（onSearch/onUpload），具体实现通过 props 注入或 context 提供，本 change 实现 mock/基础版本。

**[FormRenderer 性能]** → 大表单（30+ 字段）动态生成 Zod schema + 条件显隐计算可能有性能问题。Mitigation: Zod schema 在 schema 不变时缓存（useMemo），条件显隐只在依赖字段变化时重算。

**[permissions 字段依赖 workflow node ID]** → node ID 在工作流编辑器中生成（类似 `node_1713184200000_1`），如果工作流重新生成，ID 会变化导致 permissions 失效。Mitigation: 这是设计时配置，管理员在工作流编辑器中配置节点时选择表单权限，保存时 nodeId 已确定。
