## 1. Engine 接口与 ClassicEngine 核心

- [x] 1.1 定义 WorkflowEngine 接口 — `internal/app/itsm/engine/engine.go`，包含 Start/Progress/Cancel 三个方法签名
- [x] 1.2 实现 ClassicEngine 结构体 — `internal/app/itsm/engine/classic.go`，包含 workflow_json 解析（构建 nodeMap/edgeMap）、Start() 逻辑（找 start 节点→沿出边→创建首个 Activity）
- [x] 1.3 实现 Progress() 核心流转 — 当前 Activity → 匹配 outcome 出边 → 创建目标 Activity → 自动节点递归处理（最大深度 50）
- [x] 1.4 实现 Cancel() — 批量取消所有活跃 Activity，设工单状态为 cancelled，记录 Timeline
- [x] 1.5 IOC 注册 — 在 ITSMApp.Providers() 中注册 ClassicEngine 到容器

## 2. 节点类型执行

- [x] 2.1 start/end 节点 — start 跳过直接处理出边目标；end 设工单 completed + 记录 Timeline
- [x] 2.2 form 节点 — 创建 pending Activity + 分配参与人，处理人提交后 outcome="submitted"，表单数据保存到 result
- [x] 2.3 approve 节点 — 支持三种模式：single（单人）、parallel（并行会签，任一驳回即驳回）、sequential（串行依次），outcome 为 approved/rejected
- [x] 2.4 process 节点 — 创建 pending Activity + 分配参与人，处理人提交结果后 outcome="completed"
- [x] 2.5 action 节点 — 创建 in_progress Activity，提交 itsm-action-execute 异步任务，HTTP 调用成功 outcome="success"，失败 outcome="failed"
- [x] 2.6 gateway 节点 — 评估条件列表（equals/not_equals/contains_any/gt/lt/gte/lte），选择匹配出边，无匹配走默认边，无默认边报错
- [x] 2.7 notify 节点 — 通过 Kernel Channel 发送通知（模板变量替换 `{{ticket.code}}` 等），非阻塞立即继续
- [x] 2.8 wait 节点 — signal 模式创建 pending Activity 等待 API 触发；timer 模式提交 itsm-wait-timer 异步任务（execute_after）

## 3. 参与人解析

- [x] 3.1 ParticipantResolver 结构体 — `internal/app/itsm/engine/resolver.go`，通过 IOC 可选注入 Org App 服务
- [x] 3.2 user 类型 + requester_manager 类型解析 — user 直接返回指定 ID；requester_manager 查询提交人上级
- [x] 3.3 position 类型 + department 类型解析 — 调用 Org App 查询，Org App 不存在时返回明确错误
- [x] 3.4 解析结果为空处理 — 记录 Timeline 警告，Activity 保持 pending 等待管理员手动指派

## 4. Workflow JSON 校验

- [x] 4.1 Validator 结构体 — `internal/app/itsm/engine/validator.go`，Validate(workflowJSON) 返回错误列表
- [x] 4.2 结构校验 — 有且仅有一个 start（仅一条出边）、至少一个 end（无出边）、节点类型合法（9 种之一）
- [x] 4.3 边合法性 + 孤立节点检测 — source/target 引用存在的节点 ID，非 start 节点至少一条入边
- [x] 4.4 gateway 条件完整性 — 至少两条出边，非默认出边必须有 condition
- [x] 4.5 ServiceDefinition Handler 集成 — 保存 workflow_json 时调用 Validator，校验失败返回详细错误

## 5. 动作执行与等待任务

- [x] 5.1 ActionExecutor — `internal/app/itsm/engine/executor_action.go`，HTTP 调用逻辑 + 指数退避重试（默认 3 次）+ TicketActionExecution 记录
- [x] 5.2 itsm-action-execute 任务注册 — ITSMApp.Tasks() 注册 async 任务，handler 读取 payload 调用 ActionExecutor
- [x] 5.3 itsm-wait-timer 任务注册 — ITSMApp.Tasks() 注册 async 任务，检查 execute_after 时间到达后调用 Progress（未到时间跳过不计失败）

## 6. TicketService 集成

- [x] 6.1 修改 TicketService.Create() — engine_type="classic" 时创建工单后调用 ClassicEngine.Start()，保存 workflow_json 快照到工单
- [x] 6.2 新增 TicketService.Progress() — 验证权限（处理人是 assignee 或管理员），委托 ClassicEngine.Progress()
- [x] 6.3 新增 TicketService.Signal() — 验证 Activity 为 wait 节点 + pending 状态，调用 ClassicEngine.Progress()
- [x] 6.4 新增 Handler API + Casbin 策略 — `POST /api/v1/itsm/tickets/:id/progress`、`POST /api/v1/itsm/tickets/:id/signal`，更新 Casbin 策略

## 7. 前端：ReactFlow 编辑器

- [ ] 7.1 安装 @xyflow/react 依赖 — `bun add @xyflow/react`
- [ ] 7.2 编辑器画布组件 — ReactFlow 实例 + 缩放/平移/拖拽 + minimap + 左侧节点面板（9 种节点可拖拽）
- [ ] 7.3 自定义节点组件 — 为每种节点类型创建 React 组件（不同颜色/图标/形状），显示名称和简要配置
- [ ] 7.4 节点属性面板 — 右侧面板按节点类型显示配置项（form: Schema+参与人 / approve: 模式+参与人 / action: ServiceAction 选择 / gateway: 条件列表 / notify: 渠道+模板 / wait: 模式+时长）
- [ ] 7.5 边属性配置 — 点击边显示：outcome 输入、默认边开关、网关条件配置（source 为 gateway 时）
- [ ] 7.6 保存/加载 + 校验提示 — 序列化 JSON 调用后端 API 保存，校验失败时显示错误并高亮有问题的节点/边；打开时从 workflow_json 恢复画布
- [ ] 7.7 集成到服务定义编辑 — engine_type="classic" 时在服务定义编辑 Sheet 中嵌入编辑器（全屏或大面板模式）
- [ ] 7.8 i18n — 编辑器所有文本（节点类型名、属性标签、按钮文案）加入 itsm 的 zh-CN.json 和 en.json

## 8. 前端：流程可视化

- [ ] 8.1 只读流程图组件 — ReactFlow 只读模式，从工单快照 workflow_json 渲染
- [ ] 8.2 状态高亮渲染 — 当前活跃节点高亮、已完成节点绿色、已走过的边加粗、未到达灰色
- [ ] 8.3 节点点击查看 Activity 详情 — 点击已完成节点弹出 Popover 显示处理人、时间、结果
- [ ] 8.4 集成到工单详情页 + 流转操作面板 — 嵌入只读流程图（仅 engine_type="classic"），当前处理人显示操作按钮（审批通过/驳回、表单提交、处理完成）

## 9. 集成验证

- [ ] 9.1 端到端：简单审批流 — 画流程（start→form→approve→end）→ 提单 → 填表 → 审批 → 完结
- [ ] 9.2 端到端：审批驳回返工 — 含返工边 → 驳回 → 回表单 → 重新填写 → 再次审批通过
- [ ] 9.3 端到端：网关条件路由 + 动作节点 — 含 gateway 分支 + action 节点的完整流程
- [ ] 9.4 端到端：等待节点 + 通知节点 — signal API 触发 + timer 定时触发 + 通知发送
- [ ] 9.5 流程可视化 + 权限验证 — 进行中/已完成工单的流程图高亮正确，非处理人不能操作 Activity
