## MODIFIED Requirements

### Requirement: WorkflowViewer 复用 BPMN 节点渲染
系统 SHALL 使 WorkflowViewer 使用与 WorkflowEditor 相同的 BPMN 风格节点渲染组件。

#### Scenario: Viewer 使用新 nodeTypes
- **WHEN** WorkflowViewer 渲染工作流
- **THEN** 使用与 WorkflowEditor 相同的 nodeTypes map（event-node、task-node、gateway-node 等），节点视觉风格一致

#### Scenario: Viewer 保持运行时状态叠加
- **WHEN** WorkflowViewer 显示工单的工作流状态
- **THEN** 在 BPMN 风格节点基础上叠加运行时状态：completed 节点低透明度、active 节点蓝色 ring、执行过的边绿色高亮

#### Scenario: 旧 workflowJson 兼容
- **WHEN** Viewer 加载仅包含旧 9 种 nodeType 的 workflowJson
- **THEN** 节点正常渲染，使用新的 BPMN 风格（旧类型在新渲染组件中有对应处理）
