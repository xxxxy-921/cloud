## MODIFIED Requirements

### Requirement: 节点类型扩展
系统 SHALL 在 types.ts 中将 form/user_task 节点的 `formDefinitionId` 字段替换为 `formSchema` (FormSchema 对象)。WFNodeData 接口 SHALL 移除 `formDefinitionId?: string`，新增 `formSchema?: FormSchema`。

#### Scenario: 类型定义完整性
- **WHEN** 编辑器使用 NodeType 联合类型
- **THEN** 包含全部 15 种类型：start, end, form, approve, process, action, exclusive, notify, wait, timer, signal, parallel, inclusive, subprocess, script

#### Scenario: form 节点属性面板内嵌表单设计器
- **WHEN** 用户选中一个 form 或 user_task 节点
- **THEN** 右侧属性面板 SHALL 显示内嵌的 FormDesigner 组件，允许直接编辑 formSchema

#### Scenario: 保存时 formSchema 嵌入 workflowJson
- **WHEN** 用户保存工作流
- **THEN** 每个 form/user_task 节点的 data.formSchema SHALL 包含完整的表单 schema JSON
