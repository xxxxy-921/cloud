## Why

当前工作流编辑器虽然基于 ReactFlow 可以拖拽连线，但视觉和交互远未达到专业 BPMN 设计器水准：所有节点共享一个矩形模板（仅 gateway 为菱形），属性面板只能配置 label 和少数基础属性，缺少参与人选择器、表单绑定、条件构建器、动作选择器等关键配置能力。没有自动布局、撤销重做、右键菜单、快捷键等编辑器基础能力。

这是用户直接接触的界面——工作流设计器的质量直接决定了管理员对 ITSM 系统专业度的感知。需要达到 ServiceNow Flow Designer / Camunda Modeler 级别的交互体验。

## What Changes

### 节点渲染全面重做
- **Event 节点**：圆形设计——Start(细线圆/绿), End(粗线圆/红), Timer(双线圆+时钟图标), Signal(双线圆+闪电图标)
- **Task 节点**：差异化圆角矩形——顶部图标+标题行，下方嵌入摘要信息（表单字段数、参与人名称、SLA 标记等）；Form(蓝), Approve(琥珀), Process(紫), Action(青), Script(灰蓝), Notify(粉)
- **Gateway 节点**：菱形 + 内部符号——Exclusive(✕), Parallel(✛), Inclusive(○)，下方显示标签
- **Subprocess 节点**：粗边框矩形，内部可展开显示子流程缩略图，底部 [+] 折叠/展开按钮
- **Boundary Event**：半圆图标附着在 Task 节点底部边缘，虚线连接到目标节点
- **Edge**：smoothstep 连线，gateway 出边显示条件摘要标签，approve 出边显示 approved/rejected 标签

### Property Panel 全面重做
- **参与人选择器**：统一组件，支持 user(搜索用户)、position(选择岗位)、department(选择部门)、requester_manager(固定选项) 四种类型，可添加多个参与人
- **表单绑定选择器**：下拉选择 FormDefinition，带表单字段预览
- **服务动作选择器**：下拉选择 ServiceAction，显示 URL/Method 预览
- **条件可视化构建器**：变量下拉 + 运算符下拉 + 值输入，支持多条件 AND/OR 组合，替代原始 input
- **变量映射编辑器**：input mapping (流程变量→表单默认值) 和 output mapping (表单字段→流程变量) 的可编辑列表
- **脚本赋值编辑器**：变量名 + 表达式的可编辑列表，带语法提示
- **边界事件配置**：类型选择、时长输入、interrupting 开关、目标路径选择
- **子流程编辑器**：点击打开子画布（复用 WorkflowEditor 组件递归）
- **审批模式**：single/parallel/sequential + 自动通过/拒绝条件

### 编辑器增强
- **Auto-layout**：集成 dagre 库，一键自动排版（保持手动调整过的位置可 override）
- **Undo/Redo**：基于 zustand 的操作栈，Ctrl+Z / Ctrl+Shift+Z
- **Copy/Paste**：Ctrl+C / Ctrl+V 复制选中的节点和边
- **右键上下文菜单**：删除、复制、查看属性、添加边界事件
- **键盘快捷键**：Delete/Backspace 删除选中、方向键微调位置、Ctrl+A 全选
- **校验高亮**：校验错误的节点显示红色边框 + 悬浮提示气泡，替代底部列表
- **Edge labels**：gateway 出边上显示条件表达式摘要或 outcome 值
- **并行网关配对**：fork 和 join 之间用虚线框视觉关联

## Capabilities

### New Capabilities
- `itsm-bpmn-nodes`: BPMN-style 节点渲染（14 种节点类型差异化视觉）
- `itsm-participant-picker`: 参与人选择器组件（用户搜索 + 岗位 + 部门 + 上级）
- `itsm-condition-builder`: 可视化条件构建器组件（变量+运算符+值 组合）
- `itsm-variable-mapping-editor`: 变量映射编辑器组件
- `itsm-editor-ux`: 自动布局 + Undo/Redo + Copy/Paste + 右键菜单 + 快捷键
- `itsm-form-binding-picker`: 表单绑定选择器（FormDefinition 下拉 + 预览）

### Modified Capabilities
- `itsm-workflow-editor`: WorkflowEditor 组件全面重构（节点渲染 + 属性面板 + 编辑器能力）
- `itsm-workflow-viewer`: WorkflowViewer 复用新的节点渲染组件

## Impact

- **前端**：`web/src/apps/itsm/components/workflow/` 几乎完全重写——custom-nodes.tsx (~600 行), property-panel.tsx (~1200 行, 拆分为多个子面板), workflow-editor.tsx (~400 行增强), 新增 condition-builder.tsx (~300 行), participant-picker.tsx (~250 行), variable-mapping.tsx (~200 行), auto-layout.ts (~100 行), use-undo-redo.ts (~80 行)；types.ts 扩展节点类型定义
- **后端**：无改动（所有后端能力由 ①②③④⑤ 提供）
- **新依赖**：dagre (auto-layout), @dagrejs/dagre 或 elkjs
- **依赖**：①②③④⑤（所有后端能力就绪后开始）
