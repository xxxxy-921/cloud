## 1. SmartEngine 核心

- [ ] 1.1 新建 `internal/app/itsm/engine/smart.go`：定义 SmartEngine struct，持有 IOC 注入的 AI App 服务引用（AgentService、LLM Client、KnowledgeService）
- [ ] 1.2 SmartEngine 实现 `WorkflowEngine.Start()`：构建初始 TicketCase 快照，调用 Agent 获取第一步决策，根据信心分数创建 Activity
- [ ] 1.3 SmartEngine 实现 `WorkflowEngine.Progress()`：Activity 完成后更新快照，重新调用 Agent 决策下一步
- [ ] 1.4 SmartEngine 实现 `WorkflowEngine.Cancel()`：取消当前 Activity，更新工单状态为 cancelled
- [ ] 1.5 `app.go` 的 `Providers()` 中通过 IOC 可选注入 AI App 服务（AgentService、LLM Client、KnowledgeService、ToolRegistry），不可用时记录 info 日志；EngineFactory 注册 SmartEngine

## 2. 决策循环

- [ ] 2.1 新建 `engine/snapshot.go`：实现 `BuildTicketCase()`——构建工单快照（基本信息、service、collaboration_spec、sla_status 剩余时间计算、activity_history 摘要、form_data）
- [ ] 2.2 新建 `engine/policy.go`：实现 `CompilePolicy()`——编译策略快照（allowed_step_types、participant_candidates 通过 Org App 可选获取、available_actions、allowed_status_transitions）
- [ ] 2.3 新建 `engine/planner.go`：实现 `CallAgent()`——构建 system prompt（Collaboration Spec + Agent system_prompt + 输出格式要求）+ user message（TicketCase + Policy JSON），调用 LLM Client 获取 JSON 输出
- [ ] 2.4 Agent 调用中注入知识库上下文：service 的 knowledge_base_ids 不为空时通过 KnowledgeService 检索相关知识节点并注入 system prompt
- [ ] 2.5 新建 `engine/validator_smart.go`：实现 `ValidateDecisionPlan()`——校验 next_step_type 合法性、participant_id 在候选列表中、action_id 存在；JSON 解析失败时附带格式纠正提示重试一次
- [ ] 2.6 决策执行器：DecisionPlan 校验通过后，根据 plan.activities 创建 TicketActivity 和 TicketAssignment 记录

## 3. 信心机制与人工覆盖

- [ ] 3.1 信心阈值对比逻辑：confidence >= threshold 时自动执行（Activity 设为 in_progress），< threshold 时等待确认（Activity 设为 pending_approval，ai_decision 存储 DecisionPlan）
- [ ] 3.2 人工确认/拒绝 API：`POST .../activities/:aid/confirm`（按原计划执行）和 `POST .../activities/:aid/reject`（丢弃计划，记录 overridden_by）
- [ ] 3.3 强制跳转 API：`POST .../override/jump`——取消当前 Activity，创建指定类型新 Activity，记录覆盖原因到 Timeline
- [ ] 3.4 改派 API：`POST .../override/reassign`——更新 Assignment 的 assignee_id，Timeline 记录改派（原→新）
- [ ] 3.5 驳回 API：`POST .../override/reject`——取消当前 Activity，触发新一轮决策循环
- [ ] 3.6 覆盖操作权限检查（itsm_admin 角色）+ Casbin 策略 Seed 为 confirm/reject/jump/reassign/retry-ai 端点添加策略

## 4. 超时与 Fallback

- [ ] 4.1 决策超时：Agent 调用使用 `context.WithTimeout`，超时时间从 agent_config.decision_timeout_seconds 读取（默认 30s），超时后 ai_failure_count +1 并转人工队列
- [ ] 4.2 ai_failure_count 逻辑：失败（超时/解析错误/校验不通过/模型不可用）时 +1，成功时归零；>= 3 时停用 AI 决策
- [ ] 4.3 重新尝试 AI 决策 API：`POST .../override/retry-ai`——重置 ai_failure_count 为 0，重新执行决策循环

## 5. Agent 工具注册

- [ ] 5.1 新建 `internal/app/itsm/tools/provider.go`：定义 ITSMToolProvider，持有 TicketService/ServiceDefinitionService 等引用
- [ ] 5.2 实现 RegisterTools：从 IOC 获取 ToolRegistry，注册 6 个工具的 name、description、inputSchema、handler；ToolRegistry 不可用时静默跳过
- [ ] 5.3 工具执行权限获取：从 Agent Session context 提取 user_id，无 user_id 时按系统身份执行

## 6. Agent 工具实现

- [ ] 6.1 实现 `itsm.search_services` handler：调用 ServiceDefinitionService 搜索已启用服务，支持 keyword 和 catalog_id 筛选
- [ ] 6.2 实现 `itsm.create_ticket` handler：调用 TicketService.Create()，设置 source="agent" + agent_session_id，触发引擎 Start
- [ ] 6.3 实现 `itsm.query_ticket` handler：支持 ticket_id/ticket_code 查询，权限校验（自己的完整信息，他人基本信息）
- [ ] 6.4 实现 `itsm.list_my_tickets` handler：查询当前用户工单列表，支持 status 筛选和分页
- [ ] 6.5 实现 `itsm.cancel_ticket` handler：权限校验（提单人或 admin），调用 WorkflowEngine.Cancel()
- [ ] 6.6 实现 `itsm.add_comment` handler：在工单 Timeline 添加 comment 类型记录

## 7. 预置 Agent Seed

- [ ] 7.1 Seed 中检测 AI App 可用性，不可用时跳过全部 Agent 创建（仅 info 日志）
- [ ] 7.2 Seed 创建 "IT 服务台" Agent：visibility=public，system_prompt 定义服务台引导行为，绑定 6 个 ITSM 工具
- [ ] 7.3 Seed 创建 "流程决策" Agent：visibility=private，temperature=0.2，system_prompt 定义决策输出 DecisionPlan JSON 行为
- [ ] 7.4 Seed 创建 "处理协助" Agent：visibility=team，system_prompt 定义知识库检索+诊断建议行为；所有 Agent 按 name 幂等检查

## 8. TicketService 集成

- [ ] 8.1 修改 `TicketService.Create()`：engine_type="smart" 时调用 SmartEngine.Start()；支持 source="agent" + agent_session_id 字段
- [ ] 8.2 修改 Activity 完成回调：engine_type="smart" 时提交 itsm-smart-progress 异步任务

## 9. Scheduler 任务

- [ ] 9.1 在 `ITSMApp.Tasks()` 注册 `itsm-smart-progress` 异步任务：payload 含 ticket_id + completed_activity_id，handler 执行完整决策循环
- [ ] 9.2 任务错误处理：决策失败记录日志，由 Fallback 逻辑处理，不抛出 panic

## 10. 前端：智能服务配置

- [ ] 10.1 新建 `components/smart-service-config.tsx`：智能服务配置面板（engine_type="smart" 时显示），包含以下子组件
- [ ] 10.2 Collaboration Spec Markdown 编辑器（Textarea，自适应行数）+ Agent 下拉选择器（调用 AI Agent 列表 API）
- [ ] 10.3 知识库多选绑定（MultiSelect，调用知识库列表 API）+ 信心阈值滑块（0.0-1.0，步长 0.05）+ 决策超时输入框（10-120s）
- [ ] 10.4 AI App 不可用检测：禁用 engine_type="smart" 选项并显示提示

## 11. 前端：人工覆盖面板

- [ ] 11.1 新建 `components/ai-decision-panel.tsx`：展示 ai_reasoning、confidence 进度条、DecisionPlan 摘要；pending_approval 时显示确认/拒绝按钮
- [ ] 11.2 新建 `components/override-actions.tsx`：覆盖操作下拉菜单（强制跳转、改派、驳回、重新尝试 AI），itsm_admin 权限控制显示
- [ ] 11.3 强制跳转 Sheet（选择 activity_type + 参与人 + 覆盖原因）和改派 Sheet（选择新处理人 + 改派原因）

## 12. 前端：动态流程图

- [ ] 12.1 新建 `components/smart-flow-visualization.tsx`：基于 Activity 历史的流程图，每个 Activity 作为节点按 sequence_order 排列连线
- [ ] 12.2 节点标记：AI 决策节点显示 confidence badge（绿色高信心/黄色低信心），人工覆盖节点显示人形图标 + 覆盖人
- [ ] 12.3 当前活跃节点高亮（脉冲动画）+ 节点点击展开详情 Popover（ai_reasoning、confidence、overridden_by、完成时间）

## 13. 前端：AI Copilot 入口

- [ ] 13.1 工单详情页添加 "AI 协助" 按钮（AI App 可用时显示），点击创建 Agent Session（agent_id=处理协助 Agent），initial message 注入工单上下文
- [ ] 13.2 嵌入式 Chat 面板：复用 AI Chat 的消息列表和输入框组件；已有关联 Session 时直接打开继续对话

## 14. 集成与验证

- [ ] 14.1 端到端：Agent 对话提单 → SmartEngine 启动 → 高信心自动执行 → Activity 完成 → 下一轮决策 → 完结
- [ ] 14.2 端到端：低信心决策 → 人工确认/拒绝 → 手动跳转 → 覆盖记录
- [ ] 14.3 端到端：AI 连续失败 3 次 → 自动转人工 → retry-ai → 恢复 AI 决策
- [ ] 14.4 Go build 验证 `go build -tags dev ./cmd/server/` + 前端 lint `cd web && bun run lint`
- [ ] 14.5 更新 i18n：`locales/zh-CN.json` 和 `locales/en.json` 新增智能引擎、Agent 工具、人工覆盖、AI Copilot 相关翻译 key
