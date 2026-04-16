## Context

当前工作流编辑器基于 ReactFlow，所有节点共用矩形/菱形模板，属性面板仅支持 label、executionMode、waitMode 等基础字段。缺少参与人选择器、表单绑定、条件构建器等关键配置组件，也没有自动布局、撤销重做等编辑器基础能力。

现有代码结构：
- `workflow-editor.tsx` — 主编辑器（~175 行），NodePalette + ReactFlow + PropertyPanel 三栏布局
- `custom-nodes.tsx` — 单一 CustomNode 组件（~80 行），start/end 药丸形、gateway 菱形、其他矩形
- `property-panel.tsx` — NodePropertyPanel + EdgePropertyPanel（~163 行），字段有限
- `types.ts` — 9 种 NodeType，WFNodeData/WFEdgeData 接口
- `workflow-viewer.tsx` — 只读查看器，需复用新节点渲染

后端已就绪的 API：
- `GET /api/v1/users?keyword=` — 用户搜索
- `GET /api/v1/org/departments/tree` — 部门树
- `GET /api/v1/org/positions` — 岗位列表
- `GET /api/v1/itsm/forms?scope=global&scope=service` — 表单定义列表
- `GET /api/v1/itsm/services/:id/actions` — 服务动作列表

约束：React Compiler 启用（无早期 return 在 hooks 前、无 IIFE、无 render 中读写 ref）；使用 shadcn/ui + Tailwind CSS 4；bun 构建。

## Goals / Non-Goals

**Goals:**
- 14 种节点类型差异化 BPMN 风格渲染（Event 圆形、Task 信息卡片、Gateway 菱形+符号、Subprocess 粗框）
- 属性面板全面重做：参与人选择器、表单绑定、条件构建器、变量映射、脚本赋值、审批模式等
- 编辑器 UX 增强：自动布局（dagre）、Undo/Redo、Copy/Paste、右键菜单、键盘快捷键、校验高亮
- WorkflowViewer 复用新节点渲染组件

**Non-Goals:**
- 后端 API 改动（所有数据接口已就绪）
- Subprocess 子画布递归编辑（仅视觉展示，递归编辑留后续迭代）
- 边界事件（Boundary Event）完整实现（仅预留类型定义，不实现 attach 交互）
- 流程模拟/调试能力
- 导入/导出 BPMN XML 标准格式

## Decisions

### D1: 节点渲染拆分为独立组件文件

**选择**：将 `custom-nodes.tsx` 拆分为 `nodes/` 目录，每种形状类别一个文件（event-node、task-node、gateway-node、subprocess-node），通过 `nodes/index.ts` 统一导出 nodeTypes map。

**理由**：当前单文件 80 行将膨胀到 400+ 行；按类别拆分后每个文件 80-120 行，独立开发和测试。Viewer 直接复用同一套 nodeTypes。

**备选**：保持单文件 → 维护困难；每种节点一个文件 → 文件过多（14 个），粒度过细。

### D2: 属性面板拆分为 Panel + 子面板组件

**选择**：`property-panel.tsx` 保留为路由壳（根据 nodeType 渲染对应子面板），实际配置逻辑拆入 `panels/` 目录：`participant-picker.tsx`、`form-binding-picker.tsx`、`condition-builder.tsx`、`variable-mapping-editor.tsx`、`script-assignment-editor.tsx`。

**理由**：当前 163 行将膨胀到 1000+ 行；各子面板组件可独立开发、独立复用（如条件构建器可用于规则引擎）。

**备选**：全部写在 property-panel.tsx → 文件过大；每个 nodeType 一个面板 → 大量重复代码。

### D3: Undo/Redo 基于 zustand 操作栈

**选择**：创建 `use-undo-redo.ts` hook，维护 `{ past: State[], present: State, future: State[] }` 结构。每次 nodes/edges 变更 push 到 past，Ctrl+Z pop past → present，Ctrl+Shift+Z pop future → present。使用 debounce（300ms）合并高频拖拽操作。

**理由**：ReactFlow 不内置 undo/redo；zustand 轻量且项目已在使用；操作栈模式简单可靠。

**备选**：Command Pattern（每种操作一个 Command 类）→ 过度设计；immer patches → 引入新依赖且 patch 格式与 ReactFlow 不兼容。

### D4: 自动布局使用 @dagrejs/dagre

**选择**：引入 `@dagrejs/dagre`，创建 `auto-layout.ts` 工具函数，将 ReactFlow nodes/edges 转为 dagre graph，计算布局后写回 position。

**理由**：dagre 是 ReactFlow 生态最成熟的布局库，体积小（~40KB），API 简单，支持 TB/LR 方向。

**备选**：elkjs（更强大但 ~200KB，WASM 加载慢）；d3-dag（API 复杂）；手写布局算法（工作量大）。

### D5: 条件构建器支持多条件 AND/OR 组合

**选择**：扩展 `GatewayCondition` 类型为 `ConditionGroup`：

```typescript
interface ConditionGroup {
  logic: "and" | "or"
  conditions: Array<SimpleCondition | ConditionGroup>
}
interface SimpleCondition {
  field: string
  operator: string
  value: unknown
}
```

向后兼容：旧的单条件 `GatewayCondition` 在加载时自动包装为 `{ logic: "and", conditions: [old] }`。

**理由**：实际业务场景中单条件远不够用（如 "优先级=高 AND 金额>10000"）；嵌套结构支持未来扩展。

**备选**：保持单条件 → 功能不足；表达式字符串 → 用户需手写、易出错。

### D6: 新增节点类型扩展 NodeType 联合类型

**选择**：在 types.ts 中扩展 NODE_TYPES 数组，新增 `timer`、`signal`、`parallel`、`inclusive`、`subprocess`、`script` 六种类型。保留原有 9 种不变，`exclusive` 继续作为排他网关。

**理由**：proposal 要求 14 种差异化视觉。新增类型与后端引擎无关（后端 node routing 只看 nodeType 字段值），纯前端渲染差异。

### D7: 右键菜单使用 shadcn ContextMenu

**选择**：使用 `@radix-ui/react-context-menu`（shadcn/ui 已包含），在 ReactFlow 的 onNodeContextMenu/onPaneContextMenu 事件中触发。

**理由**：项目已有 shadcn/ui 依赖，ContextMenu 开箱即用；无需引入额外库。

### D8: Edge label 使用 ReactFlow 自定义 EdgeLabel

**选择**：创建自定义 Edge 组件 `custom-edges.tsx`，在 smoothstep 路径上渲染条件摘要或 outcome 标签。Gateway 出边显示条件表达式简写，approve 出边显示 approved/rejected badge。

**理由**：ReactFlow 原生 label 仅支持字符串；自定义 Edge 可渲染 JSX（badge、图标）。

## Risks / Trade-offs

**[类型扩展兼容性]** → 新增 6 种 NodeType 后，已有的 workflowJson 中不会包含这些类型，无需迁移。新类型仅在编辑器中可用。后端引擎需要同步支持新类型的路由逻辑，但这是 Non-Goal（后续迭代）。

**[dagre 布局精度]** → dagre 不支持不同尺寸的节点（默认等宽等高），需要在布局前为每种节点类型设置正确的 width/height。如果节点内容动态变化（如长标签），布局可能不完美。→ 缓解：为每种 nodeType 预设固定尺寸，布局后用户可手动微调。

**[ConditionGroup 复杂度]** → 嵌套条件组 UI 在小屏幕上可能拥挤。→ 缓解：限制最大嵌套深度为 2 层；默认展示平铺 AND 模式，点击切换高级模式。

**[React Compiler 兼容性]** → 新增的 useUndoRedo hook 和 useAutoLayout 需严格遵守 compiler 规则（hooks 在顶层调用，无条件 return 在 hooks 前）。→ 缓解：CI 中 eslint-plugin-react-hooks 会捕获违规。

**[属性面板数据获取]** → 参与人选择器需要调用 users/departments/positions API，表单绑定需要调用 forms API。这些请求可能增加编辑器初始化时间。→ 缓解：使用 React Query 的 staleTime + 懒加载（展开面板时才请求）。

**[WorkflowViewer 兼容]** → Viewer 复用新 nodeTypes 后，旧的 workflowJson（使用旧 nodeType）仍需正常渲染。→ 缓解：新 nodeTypes map 覆盖所有类型，旧类型的渲染走原有逻辑。
