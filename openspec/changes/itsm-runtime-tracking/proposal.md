## Why

流程引擎和设计器就绪后，最后一环是运行时体验。当前工单详情页无法可视化展示流程执行状态——管理员不知道工单"走到了哪一步"、"哪些分支在等待"、"流程变量当前值是什么"。表单也是硬编码的静态 UI，没有利用 Form Engine 的动态渲染能力。

运行时追踪将流程设计器从"设计工具"升级为"可视化运维面板"，让 ITSM 管理员能够：看到实时流程状态、理解工单卡在哪里、查看决策数据、以正确的表单处理工单。

## What Changes

### Runtime Workflow Viewer
- **流程图只读视图**：复用 ⑥ 的 BPMN 节点渲染组件，在工单详情页嵌入
- **Token 位置高亮**：查询活跃 token 列表，当前活跃节点显示绿色脉冲动画
- **已完成路径标记**：已完成的节点和边显示灰色 + 勾选标记
- **并行分支可视化**：多个活跃 token 同时高亮各自所在节点
- **失败/取消标记**：失败节点红色标记，取消节点灰色删除线

### 流程变量实时面板
- **侧边栏面板**：显示当前流程所有变量（key, value, type, source, updated_at）
- **变量修改历史**：点击变量查看修改时间线（哪个节点/表单修改了此变量）
- **管理员编辑**：管理员可手动修改流程变量（调试/紧急干预用）

### 工单表单动态渲染
- **创建工单**：根据 ServiceDefinition.form_id 加载 FormDefinition，使用 FormRenderer(mode=create) 渲染
- **处理工单**：根据当前活动节点的 formId 加载表单，使用 FormRenderer(mode=edit, permissions=节点权限) 渲染，提交时写入 activity.form_data + process variables
- **查看历史**：查看已完成活动的表单数据，使用 FormRenderer(mode=view, data=activity.form_data) 渲染
- **审批任务**：FormRenderer 配合审批操作按钮（通过/拒绝 + 可选意见输入）

### 后端 API
- **Token 状态 API**：`GET /api/v1/itsm/tickets/:id/tokens` 返回 token 树（含 node_id, status, type）
- **变量 API 增强**：`PUT /api/v1/itsm/tickets/:id/variables/:key` 管理员手动修改
- **活动历史聚合 API**：`GET /api/v1/itsm/tickets/:id/activities` 返回按节点分组的活动历史（含 form_data、操作人、时间）

### 节点点击交互
- **点击节点查看历史**：弹出 Popover 显示该节点的活动记录（谁操作、什么时间、提交了什么数据、outcome）
- **查看表单快照**：历史活动的表单以 FormRenderer(mode=view) 渲染

## Capabilities

### New Capabilities
- `itsm-runtime-viewer`: 运行时流程图组件（token 高亮 + 路径标记 + 节点交互）
- `itsm-variable-panel`: 流程变量实时面板（查看 + 历史 + 管理员编辑）
- `itsm-dynamic-form-render`: 工单表单动态渲染（创建/处理/查看模式切换）
- `itsm-activity-history-overlay`: 节点点击查看活动历史弹出层

### Modified Capabilities
- `itsm-ticket-detail-ui`: 工单详情页集成 Runtime Viewer + Variable Panel + Dynamic Form
- `itsm-ticket-create-ui`: 创建工单页使用 FormRenderer 替代硬编码表单
- `itsm-ticket-process-ui`: 处理工单使用 FormRenderer + 节点权限

## Impact

- **后端**：新增 token 查询 API (~40 行)；变量 PUT API (~30 行)；活动聚合 API (~60 行)
- **前端**：新增 `components/runtime/` 目录——RuntimeViewer (~400 行), VariablePanel (~250 行), ActivityHistory (~200 行)；重构工单创建页和处理页集成 FormRenderer (~300 行改动)
- **依赖**：⑥ itsm-bpmn-designer（复用节点渲染组件）
