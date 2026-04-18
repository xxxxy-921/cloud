## Context

Metis ITSM 模块已有 VPN BDD（2-way 路由）和 Server Access BDD（3-way 分支决策），均验证了智能引擎的审批路由能力。但当前所有 BDD 场景都不涉及 **Action 节点**（HTTP webhook 自动触发）。

Metis 的 Action 基础设施已完整：ServiceAction 模型、TicketActionExecution 记录、ActionExecutor（HTTP 调用+重试）、`itsm-action-execute` 调度任务、`decision.list_actions` AI 工具。但 BDD 测试中使用 `noopSubmitter`（静默丢弃所有调度任务），action 从未被真正测试过。

生产数据库备份白名单临时放行是一个典型的**预检动作→人工审批→放行动作**流程。参考 bklite-cloud 的 `db_backup_whitelist_action_flow` BDD 设计，新增 3 个场景验证 Action 节点的端到端行为。

## Goals / Non-Goals

**Goals:**
- 验证智能引擎对 Action 节点的编排能力（预检→审批→放行→完成）
- 验证 Action 执行记录（TicketActionExecution）的正确性
- 验证权限校验（错误人员无法认领/审批 DBA 审批环节）
- 验证并行工单间 Action 记录的隔离性
- 扩展 `replaceTemplateVars` 支持 `{{ticket.form_data.*}}` 模板变量
- 建立可复用的 Action 测试基础设施（LocalActionReceiver + syncActionSubmitter）

**Non-Goals:**
- 不覆盖经典引擎（经典引擎的 action 执行路径与智能引擎共用同一个 executor）
- 不覆盖服务台 Agent 全链路（draft_prepare / draft_confirm 流程）
- 不测试 Action 失败重试路径和 boundary error 处理（scope for future BDD）
- 不修改 SmartEngine 核心决策逻辑

## Decisions

### Decision 1: syncActionSubmitter 替代 noopSubmitter

**选择**: 新增 `syncActionSubmitter` 实现 `engine.TaskSubmitter`，收到 `itsm-action-execute` 任务时同步调用 `ActionExecutor.Execute()`，然后自动调用 `engine.Progress()` 标记活动完成（与生产行为一致），其他任务类型走 no-op。

**替代方案 A**: 继续用 noopSubmitter + 在 step 中手动调 `ActionExecutor.Execute()`
**替代方案 B**: 实现完整的同步调度引擎

**理由**: 方案 A 与生产行为偏离太大，step 需要手动编排 action 执行和 progress。方案 B 过度工程化。syncActionSubmitter 精确模拟生产 `HandleActionExecute` 的行为（执行+progress），让 BDD step 只需触发决策循环即可。

### Decision 2: LocalActionReceiver 使用 httptest.Server

**选择**: Go 标准库 `net/http/httptest.NewServer` 创建 in-process HTTP 服务器，记录所有收到的请求

**结构**:
```
LocalActionReceiver
├─ server: *httptest.Server
├─ records: []ActionRecord  // thread-safe, 用 sync.Mutex 保护
├─ Record(r *http.Request)  // 记录请求
├─ Records() []ActionRecord // 返回所有记录
├─ RecordsByPath(path) []   // 按 path 过滤
├─ Clear()                  // 清空记录
└─ URL(path) string         // 构建完整 URL
```

**理由**: httptest.Server 是 Go 测试的标准模式，进程内通信零延迟，端口自动分配无冲突。bklite-cloud 的 `LocalActionReceiver` 是 Python 等价物。

### Decision 3: 2 个 ServiceAction（precheck + apply）

**选择**: 在 `publishDbBackupSmartService()` 中创建 2 个 ServiceAction 记录，URL 指向 LocalActionReceiver 的不同路径

```
ServiceAction "db_backup_whitelist_precheck":
  URL: receiver.URL("/precheck")
  Body: {"ticket_code":"{{ticket.code}}","database":"{{ticket.form_data.database_name}}","source_ip":"{{ticket.form_data.source_ip}}"}

ServiceAction "db_backup_whitelist_apply":
  URL: receiver.URL("/apply")
  Body: {"ticket_code":"{{ticket.code}}","database":"{{ticket.form_data.database_name}}","whitelist_window":"{{ticket.form_data.whitelist_window}}"}
```

**理由**: precheck 在审批前验证参数合法性，apply 在审批后执行放行。两阶段模式与 bklite 对齐。body_template 使用 form_data 变量，同时验证模板引擎扩展。

### Decision 4: 扩展 replaceTemplateVars 支持 form_data

**选择**: 修改 `engine/executor_action.go` 的 `replaceTemplateVars`，解析 ticket 的 FormData JSON 字段，支持 `{{ticket.form_data.<key>}}` 和 `{{ticket.code}}` 变量

**理由**: 这是 Action 模板的核心功能缺失。body_template 需要包含工单的业务字段（数据库名、IP 等）才有实际意义。改动向后兼容，仅新增变量支持。

### Decision 5: 协作规范描述预检和放行动作

**选择**: 协作规范中明确描述两阶段动作，让 AI 知道在哪个节点触发哪个 action：

```
这是一个数据库备份白名单临时放行服务。
...
信息收集完成后，系统要先自动执行数据库备份白名单预检动作（precheck），验证参数合法性。
预检通过后，交给信息部的数据库管理员岗位审批...
审批通过后，系统要自动执行数据库备份白名单放行动作（apply）...
```

AI 通过 `decision.list_actions` 工具获取可用的 ServiceAction 列表（包含 name, code, id），然后在 DecisionPlan 中引用 `action_id`。

### Decision 6: 决策循环模式——每步一个 cycle

**选择**: 每个 BDD step 触发一次决策循环，action 通过 syncActionSubmitter 同步完成后自动 progress

**流程**:
```
When 智能引擎执行决策循环     → AI 决定: action(precheck) → sync执行 → auto-progress
When 智能引擎再次执行决策循环  → AI 决定: approve(db_admin)
When DBA 认领并审批
When 智能引擎再次执行决策循环  → AI 决定: action(apply) → sync执行 → auto-progress
When 智能引擎再次执行决策循环  → AI 决定: complete
```

**理由**: 与 smart engine 的生产行为一致——每次决策循环输出一步，action 完成后触发下一轮。syncActionSubmitter 的 auto-progress 让流程自然推进。

## Risks / Trade-offs

- **[LLM 生成不确定性]** → LLM 可能生成不符合预期的 workflow。缓解：ValidateWorkflow 验证 + 3 次重试
- **[AI Action 选择不稳定]** → AI 可能选错 action_id 或跳过 action 步骤。缓解：协作规范中明确标注动作名称和时机；`decision.list_actions` 工具返回 action 的 name+code+description 供 AI 匹配
- **[syncActionSubmitter 与生产差异]** → 生产环境中 action 是异步执行（scheduler task），测试中是同步。差异在于并发时序。缓解：BDD 测试的目标是验证逻辑正确性，不验证并发时序
- **[模板变量扩展范围]** → 仅支持一级 form_data 字段（不支持嵌套对象）。缓解：当前 BDD 用例的 form_data 全是扁平 key-value，够用
