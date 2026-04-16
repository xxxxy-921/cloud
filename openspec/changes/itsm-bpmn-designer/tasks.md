## 1. 类型定义与基础设施

- [ ] 1.1 扩展 `types.ts`：NODE_TYPES 新增 timer/signal/parallel/inclusive/subprocess/script，NODE_COLORS 新增对应颜色，WFNodeData 新增 formDefinitionId/inputMapping/outputMapping/scriptAssignments/subprocessJson 字段
- [ ] 1.2 扩展 `types.ts`：WFEdgeData.condition 类型从 GatewayCondition 改为 `GatewayCondition | ConditionGroup`，新增 ConditionGroup/SimpleCondition 接口
- [ ] 1.3 新增 `auto-layout.ts`：引入 @dagrejs/dagre 依赖，实现 applyDagreLayout(nodes, edges) 函数，为每种 nodeType 预设宽高
- [ ] 1.4 新增 `use-undo-redo.ts`：实现 useUndoRedo hook，维护 past/present/future 栈，支持 push/undo/redo/canUndo/canRedo，300ms debounce 合并拖拽

## 2. BPMN 节点渲染

- [ ] 2.1 创建 `nodes/` 目录结构：event-node.tsx、task-node.tsx、gateway-node.tsx、subprocess-node.tsx、index.ts
- [ ] 2.2 实现 `event-node.tsx`：start（细线圆/绿）、end（粗线圆/红）、timer（双线圆+时钟）、signal（双线圆+闪电），处理 handle 位置
- [ ] 2.3 实现 `task-node.tsx`：form/approve/process/action/script/notify 六种卡片渲染，顶部图标+标题，下方摘要信息（表单名+字段数、参与人摘要、动作 URL/Method、通道类型）
- [ ] 2.4 实现 `gateway-node.tsx`：exclusive(✕)/parallel(✛)/inclusive(○) 菱形渲染，下方标签
- [ ] 2.5 实现 `subprocess-node.tsx`：粗边框矩形，折叠/展开按钮，展开时显示子流程缩略图
- [ ] 2.6 `nodes/index.ts` 导出 nodeTypes map，将所有节点类型映射到对应组件
- [ ] 2.7 新增 `custom-edges.tsx`：自定义 Edge 组件，smoothstep 路径 + 中点 label（条件摘要 badge、outcome badge、default 标记）

## 3. 属性面板子组件

- [ ] 3.1 新增 `panels/participant-picker.tsx`：参与人类型选择（user/position/department/requester_manager）+ 对应选择器（用户搜索 API、岗位列表 API、部门树 API）+ 参与人列表管理（添加/删除/排序）
- [ ] 3.2 新增 `panels/form-binding-picker.tsx`：FormDefinition 下拉选择（调用 forms API）+ 字段预览列表 + 清除绑定
- [ ] 3.3 新增 `panels/condition-builder.tsx`：单条件行（变量下拉+运算符下拉+值输入）+ 多条件 AND/OR 组合 + 添加条件/条件组 + 旧格式自动升级 + 条件摘要显示
- [ ] 3.4 新增 `panels/variable-mapping-editor.tsx`：input mapping 和 output mapping 编辑器（变量名下拉+字段名下拉，添加/删除映射行）
- [ ] 3.5 新增 `panels/script-assignment-editor.tsx`：变量名+表达式可编辑列表，添加/删除赋值行
- [ ] 3.6 新增 `panels/action-picker.tsx`：ServiceAction 下拉选择（调用 service actions API）+ URL/Method 预览

## 4. 属性面板重构

- [ ] 4.1 重构 `property-panel.tsx` NodePropertyPanel：根据 nodeType 路由到对应子面板组合——form（参与人+表单绑定+变量映射）、approve（参与人+审批模式+变量映射）、process（参与人+变量映射）、action（动作选择器）、script（脚本赋值）、notify（通道类型+模板）、wait（等待模式+时长）、subprocess（子流程配置）、timer/signal（事件配置）
- [ ] 4.2 重构 `property-panel.tsx` EdgePropertyPanel：集成条件构建器替代原始 field/operator/value 输入

## 5. 编辑器 UX 增强

- [ ] 5.1 重构 `workflow-editor.tsx`：集成 useUndoRedo hook，nodes/edges 变更时 push 到 undo 栈
- [ ] 5.2 集成自动布局：工具栏添加自动布局按钮，调用 applyDagreLayout
- [ ] 5.3 实现 Copy/Paste：Ctrl+C 记录选中节点/边到剪贴板，Ctrl+V 粘贴（新 ID、偏移位置、排除 start/end）
- [ ] 5.4 实现右键上下文菜单：使用 shadcn ContextMenu，节点菜单（复制/删除/查看属性）、边菜单（编辑条件/删除）、画布菜单（粘贴/自动布局/全选）
- [ ] 5.5 实现键盘快捷键：Delete/Backspace 删除、Ctrl+A 全选、方向键微调（10px，Shift+方向键 1px）
- [ ] 5.6 实现校验错误高亮：节点红色边框 + tooltip，边红色 + tooltip，移除原有底部错误列表面板
- [ ] 5.7 工具栏重构：顶部右侧显示 Undo/Redo/自动布局/保存按钮，Undo/Redo 按钮根据 canUndo/canRedo 灰显

## 6. 节点面板更新

- [ ] 6.1 重构 `node-palette.tsx`：分组显示所有 15 种节点类型——事件（start/end/timer/signal）、任务（form/approve/process/action/script/notify）、网关（exclusive/parallel/inclusive）、其他（subprocess/wait）

## 7. Viewer 适配

- [ ] 7.1 更新 `workflow-viewer.tsx`：使用新 nodeTypes map 和 edgeTypes map，替换原有的单一 CustomNode
- [ ] 7.2 验证旧 workflowJson（9 种 nodeType）在新 Viewer 中正常渲染

## 8. 依赖安装与 i18n

- [ ] 8.1 安装 @dagrejs/dagre 依赖（`bun add @dagrejs/dagre @types/dagre`）
- [ ] 8.2 补充 i18n 翻译：新节点类型名称（timer/signal/parallel/inclusive/subprocess/script）、新属性面板标签、条件构建器文案、编辑器操作按钮文案（zh-CN + en）
