## Why

一个功能完备的 BPMN 引擎需要超越简单的"人工任务 → 网关"线性模式。三种高级节点类型填补关键能力缺口：

- **Script Task（脚本任务）**：自动化变量计算和数据转换。如"根据优先级和影响范围自动计算 SLA 等级"、"格式化通知内容"。没有 Script Task，这些逻辑只能硬编码在服务端或依赖外部 webhook。
- **Boundary Timer Event（边界定时器）**：为人工任务设置超时。ITSM 中几乎所有审批和处理都需要 SLA 超时升级机制。当前 wait 节点是独立的"等待"步骤，无法附着在审批任务上实现"审批超时自动升级"。
- **Embedded Subprocess（嵌入子流程）**：将一组节点封装为可复用的子单元。如"多级审批"子流程可被不同工作流引用。子流程有独立的变量作用域和边界事件附着点。

## What Changes

### Script Task
- **新增 script 节点类型**：自动节点（IsAutoNode=true），执行后立即继续
- **NodeData 扩展**：`assignments: [{variable: string, expression: string}]` 定义变量赋值列表
- **handleScript 实现**：遍历 assignments，通过表达式引擎求值，写入 process variables，然后自动推进到下一节点
- **安全约束**：表达式引擎仅支持变量引用 + 算术 + 比较 + 字符串操作 + 三元运算，不支持函数调用或 IO 操作

### Boundary Timer Event
- **新增 b_timer 节点类型**：附着在 UserTask（form/approve/process）上
- **NodeData 扩展**：`boundaryEvents: [{id, type, interrupting, duration, targetEdgeId}]`
- **执行逻辑**：
  - 创建 UserTask activity 时，同时为每个 boundary event 创建 boundary token（suspended 状态）
  - 注册调度器定时任务（复用 itsm-wait-timer 机制）
  - 超时触发：interrupting=true 时取消主 token 及其 activity，激活 boundary token 走 targetEdge；non-interrupting 时主 token 保持，boundary token fork 出通知分支
  - 主 task 正常完成时：取消所有 boundary token 和关联的定时器

### Boundary Error Event
- **新增 b_error 节点类型**：附着在 ServiceTask（action）上
- **执行逻辑**：action webhook 执行失败（超过重试次数）时，激活 error boundary token，走错误处理分支而非标记工单失败

### Embedded Subprocess
- **新增 subprocess 节点类型**
- **NodeData 扩展**：`subProcessDef: {nodes: [], edges: []}` 内嵌子流程定义
- **执行逻辑**：
  - 进入 subprocess 时创建子 token（token_type=subprocess, scope_id=node.id）
  - 子流程内有独立的 start/end 节点
  - 子流程内的变量操作使用 scope_id 隔离
  - 子流程 end → 子 token completed → 恢复父 token 继续
- **边界事件支持**：subprocess 节点可附着 b_timer / b_error
- **ValidateWorkflow 增强**：递归校验 subProcessDef 内部结构

## Capabilities

### New Capabilities
- `itsm-script-task`: 脚本任务节点 + 变量赋值执行
- `itsm-boundary-timer`: 边界定时器事件（interrupting + non-interrupting）
- `itsm-boundary-error`: 边界错误事件（ServiceTask 失败处理）
- `itsm-subprocess`: 嵌入子流程执行 + 变量作用域隔离

### Modified Capabilities
- `itsm-classic-engine`: processNode 新增 script/subprocess case；UserTask 创建集成 boundary token 注册
- `itsm-action-execute`: action 失败时检查并触发 boundary error
- `itsm-workflow-validator`: subprocess 递归校验；boundary event 配置校验

## Impact

- **后端**：`engine/classic.go` 新增 handleScript/handleSubprocess/boundary 相关逻辑 (~400 行)；`engine/tasks.go` 修改 boundary timer 集成 (~50 行)；`engine/executor_action.go` 增加 boundary error 触发 (~30 行)；`engine/validator.go` 递归校验 (~80 行)
- **前端**：本 change 无前端改动（节点 UI 在 ⑥ 中实现）
- **依赖**：③ itsm-execution-tokens + ② itsm-process-variables
