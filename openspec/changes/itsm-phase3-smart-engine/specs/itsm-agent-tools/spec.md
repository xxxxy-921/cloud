## ADDED Requirements

### Requirement: ITSM App 向 AI App 注册 Builtin Tool
ITSM App SHALL 在 `Providers()` 阶段通过 IOC 获取 AI App 的 ToolRegistry（如果可用），注册一组 ITSM 专用 Builtin Tool。AI App 不存在时 SHALL 静默跳过注册，不影响 ITSM 经典功能。

#### Scenario: AI App 存在时注册工具
- **WHEN** ITSM App 启动，IOC 容器中存在 AI App 的 ToolRegistry
- **THEN** 系统 SHALL 注册全部 6 个 ITSM Builtin Tool（itsm.search_services、itsm.create_ticket、itsm.query_ticket、itsm.list_my_tickets、itsm.cancel_ticket、itsm.add_comment）

#### Scenario: AI App 不存在时静默跳过
- **WHEN** ITSM App 启动，IOC 容器中不存在 AI App 的 ToolRegistry（edition 未包含 AI App）
- **THEN** 系统 SHALL 静默跳过工具注册，仅输出 info 级别日志 "AI App 不可用，跳过 ITSM 工具注册"，ITSM 经典功能正常运行

#### Scenario: 工具注册幂等
- **WHEN** ITSM App 重启，ToolRegistry 中已存在同名工具
- **THEN** 系统 SHALL 更新已有工具定义而非创建重复记录

### Requirement: itsm.search_services 工具
系统 SHALL 注册 `itsm.search_services` 工具，用于搜索可用的 ITSM 服务目录。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "keyword": { "type": "string", "description": "搜索关键词，匹配服务名称或描述" },
    "catalog_id": { "type": "integer", "description": "服务目录分类 ID，筛选特定分类下的服务" }
  },
  "description": "搜索可用的 IT 服务。可通过关键词模糊搜索，或按服务目录分类筛选。返回匹配的已启用服务列表。"
}
```

#### Scenario: 按关键词搜索服务
- **WHEN** Agent 调用 itsm.search_services，输入 `keyword="网络"`
- **THEN** 系统 SHALL 返回名称或描述包含 "网络" 的已启用服务列表，每项包含 id、name、description、catalog_name、form_schema 摘要（字段列表，不含完整 schema）

#### Scenario: 按服务目录筛选
- **WHEN** Agent 调用 itsm.search_services，输入 `catalog_id=5`
- **THEN** 系统 SHALL 返回该目录及其子目录下的全部已启用服务列表

#### Scenario: 同时传入 keyword 和 catalog_id
- **WHEN** Agent 调用 itsm.search_services，输入 `keyword="VPN"` 和 `catalog_id=5`
- **THEN** 系统 SHALL 返回该目录下名称或描述包含 "VPN" 的已启用服务列表（AND 条件）

#### Scenario: 无匹配结果
- **WHEN** Agent 调用 itsm.search_services，无匹配服务
- **THEN** 系统 SHALL 返回空列表和提示 "未找到匹配的服务，请尝试其他关键词"

### Requirement: itsm.create_ticket 工具
系统 SHALL 注册 `itsm.create_ticket` 工具，用于通过 Agent 对话创建 ITSM 工单。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "service_id": { "type": "integer", "description": "服务定义 ID（从 search_services 获取）" },
    "summary": { "type": "string", "description": "工单标题/摘要" },
    "description": { "type": "string", "description": "工单详细描述" },
    "priority": { "type": "string", "enum": ["low", "medium", "high", "urgent"], "description": "优先级，默认 medium" },
    "form_data": { "type": "object", "description": "服务表单数据（键值对，按服务的 form_schema 填写）" },
    "requester_id": { "type": "integer", "description": "提单人用户 ID，不传则使用当前会话用户" }
  },
  "required": ["service_id", "summary"],
  "description": "创建一个 ITSM 工单。必须指定 service_id 和 summary。创建后返回工单编号，可用于后续查询。"
}
```

#### Scenario: 成功创建工单
- **WHEN** Agent 调用 itsm.create_ticket，输入 service_id、summary、priority
- **THEN** 系统 SHALL 创建工单，`source` 设为 `"agent"`，`agent_session_id` 设为当前 Agent Session ID，返回 `{"ticket_id": 123, "ticket_code": "ITSM-20260414-0001", "message": "工单创建成功"}`

#### Scenario: 关联 Agent 会话
- **WHEN** Agent 调用 itsm.create_ticket 创建工单
- **THEN** 系统 MUST 将当前 Agent Session ID 设置为工单的 `agent_session_id` 字段，处理人可通过该 ID 回溯对话上下文

#### Scenario: 自动设置提单人
- **WHEN** Agent 调用 itsm.create_ticket 未传入 `requester_id`
- **THEN** 系统 SHALL 使用 Agent Session 关联的 user_id 作为工单的提单人

#### Scenario: 指定提单人
- **WHEN** Agent 调用 itsm.create_ticket 传入 `requester_id=42`
- **THEN** 系统 SHALL 将 user_id=42 设置为工单的提单人（需验证用户存在）

#### Scenario: 服务不存在或已禁用
- **WHEN** Agent 调用 itsm.create_ticket，输入的 service_id 不存在或服务已禁用
- **THEN** 系统 SHALL 返回错误 `{"error": "指定的服务不存在或已禁用"}`

#### Scenario: 必填字段缺失
- **WHEN** Agent 调用 itsm.create_ticket，缺少 service_id 或 summary
- **THEN** 系统 SHALL 返回 inputSchema 校验错误 `{"error": "缺少必填字段: service_id, summary"}`

#### Scenario: 智能服务自动触发引擎
- **WHEN** 通过 itsm.create_ticket 创建的工单对应的服务是智能服务（engine_type="smart"）
- **THEN** 系统 SHALL 自动触发 SmartEngine.Start()，开始 AI 决策循环

### Requirement: itsm.query_ticket 工具
系统 SHALL 注册 `itsm.query_ticket` 工具，用于查询工单详细状态。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "ticket_id": { "type": "integer", "description": "工单 ID" },
    "ticket_code": { "type": "string", "description": "工单编号（如 ITSM-20260414-0001）" }
  },
  "description": "查询工单的详细状态。支持按 ID 或编号查询。返回工单的当前状态、处理进度、处理人等信息。"
}
```

#### Scenario: 按工单 ID 查询
- **WHEN** Agent 调用 itsm.query_ticket，输入 `ticket_id=123`
- **THEN** 系统 SHALL 返回工单详情：ticket_code、summary、status、priority、current_step（当前 Activity type + status）、assignee_name、created_at、sla_status（response_deadline 剩余时间、resolution_deadline 剩余时间）

#### Scenario: 按工单编号查询
- **WHEN** Agent 调用 itsm.query_ticket，输入 `ticket_code="ITSM-20260414-0001"`
- **THEN** 系统 SHALL 按工单编号查找并返回工单详情

#### Scenario: 工单不存在
- **WHEN** Agent 调用 itsm.query_ticket，输入的 ticket_id 或 ticket_code 不存在
- **THEN** 系统 SHALL 返回错误 `{"error": "工单不存在"}`

#### Scenario: 权限校验——自己的工单
- **WHEN** Agent 调用 itsm.query_ticket，查询的工单提单人为当前会话用户
- **THEN** 系统 SHALL 返回完整工单详情（含处理人、内部评论、AI 决策信息）

#### Scenario: 权限校验——他人的工单
- **WHEN** Agent 调用 itsm.query_ticket，查询的工单提单人非当前会话用户且当前用户无 itsm_admin 角色
- **THEN** 系统 SHALL 仅返回工单基本信息（ticket_code、summary、status、priority），不返回处理人详情和内部评论

#### Scenario: 未传入任何标识
- **WHEN** Agent 调用 itsm.query_ticket，既未传入 ticket_id 也未传入 ticket_code
- **THEN** 系统 SHALL 返回错误 `{"error": "请提供 ticket_id 或 ticket_code"}`

### Requirement: itsm.list_my_tickets 工具
系统 SHALL 注册 `itsm.list_my_tickets` 工具，用于查询当前用户的工单列表。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "status": { "type": "string", "enum": ["pending", "in_progress", "waiting_approval", "completed", "cancelled"], "description": "按状态筛选" },
    "page": { "type": "integer", "description": "页码，默认 1", "default": 1 },
    "page_size": { "type": "integer", "description": "每页条数，默认 10，最大 50", "default": 10 }
  },
  "description": "查询当前用户提交的工单列表。支持按状态筛选和分页。"
}
```

#### Scenario: 查询全部工单
- **WHEN** Agent 调用 itsm.list_my_tickets，未传入 status 筛选
- **THEN** 系统 SHALL 返回当前会话用户提交的全部工单列表，按创建时间倒序，每项包含 ticket_code、summary、status、priority、created_at

#### Scenario: 按状态筛选
- **WHEN** Agent 调用 itsm.list_my_tickets，输入 `status="in_progress"`
- **THEN** 系统 SHALL 返回当前用户处于 `in_progress` 状态的工单列表

#### Scenario: 分页查询
- **WHEN** Agent 调用 itsm.list_my_tickets，输入 `page=2`、`page_size=10`
- **THEN** 系统 SHALL 返回第二页的工单记录，响应包含 `total`（总数）和 `items`（工单列表）

#### Scenario: 无工单
- **WHEN** 当前用户没有任何工单
- **THEN** 系统 SHALL 返回 `{"total": 0, "items": [], "message": "您暂无工单"}`

### Requirement: itsm.cancel_ticket 工具
系统 SHALL 注册 `itsm.cancel_ticket` 工具，用于取消工单。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "ticket_id": { "type": "integer", "description": "要取消的工单 ID" },
    "reason": { "type": "string", "description": "取消原因" }
  },
  "required": ["ticket_id", "reason"],
  "description": "取消指定工单。仅工单提单人或管理员可取消未完结的工单。"
}
```

#### Scenario: 成功取消工单
- **WHEN** Agent 调用 itsm.cancel_ticket，输入 ticket_id 和 reason，且当前会话用户为工单提单人
- **THEN** 系统 SHALL 取消工单（调用 WorkflowEngine.Cancel），状态变为 `cancelled`，在 Timeline 记录取消原因，返回 `{"message": "工单已取消"}`

#### Scenario: 无权取消
- **WHEN** Agent 调用 itsm.cancel_ticket，当前会话用户非工单提单人且非 itsm_admin 角色
- **THEN** 系统 SHALL 返回错误 `{"error": "无权取消该工单"}`

#### Scenario: 工单已完结不可取消
- **WHEN** Agent 调用 itsm.cancel_ticket，目标工单状态为 `completed`
- **THEN** 系统 SHALL 返回错误 `{"error": "已完结的工单不可取消"}`

#### Scenario: 工单已取消
- **WHEN** Agent 调用 itsm.cancel_ticket，目标工单状态已为 `cancelled`
- **THEN** 系统 SHALL 返回错误 `{"error": "工单已取消，无需重复操作"}`

### Requirement: itsm.add_comment 工具
系统 SHALL 注册 `itsm.add_comment` 工具，用于在工单 Timeline 中添加评论。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "ticket_id": { "type": "integer", "description": "工单 ID" },
    "content": { "type": "string", "description": "评论内容" }
  },
  "required": ["ticket_id", "content"],
  "description": "在指定工单中添加评论。评论会记录在工单时间线中。"
}
```

#### Scenario: 成功添加评论
- **WHEN** Agent 调用 itsm.add_comment，输入 ticket_id 和 content
- **THEN** 系统 SHALL 在工单 Timeline 中添加评论记录（type="comment"，operator=当前会话用户），返回 `{"message": "评论已添加"}`

#### Scenario: 空评论内容
- **WHEN** Agent 调用 itsm.add_comment，content 为空字符串
- **THEN** 系统 SHALL 返回 inputSchema 校验错误 `{"error": "评论内容不能为空"}`

#### Scenario: 工单不存在
- **WHEN** Agent 调用 itsm.add_comment，输入的 ticket_id 不存在
- **THEN** 系统 SHALL 返回错误 `{"error": "工单不存在"}`

### Requirement: 工具执行权限验证
每个 ITSM 工具执行时 MUST 验证调用者权限，通过 Agent Session 关联的 user_id 确定当前操作者身份。

#### Scenario: 有效会话用户
- **WHEN** Agent 工具被调用，当前 Agent Session 关联了有效的 user_id
- **THEN** 系统 SHALL 以该 user_id 的身份执行操作，遵守该用户对应的权限约束（提单人只能操作自己的工单）

#### Scenario: 无会话用户（系统级 Agent）
- **WHEN** Agent 工具被调用，当前 Agent Session 未关联 user_id（系统级 Agent / Node Token 调用）
- **THEN** 系统 SHALL 以系统身份执行操作，拥有完整的 ITSM 操作权限

### Requirement: 工具 inputSchema 定义
每个 ITSM Builtin Tool MUST 提供符合 JSON Schema 规范的 inputSchema，包含参数名称、类型、描述和必填标记。AI App 的 Agent 在调用工具前会读取 inputSchema 构造参数。

#### Scenario: Agent 按 Schema 调用
- **WHEN** Agent 根据工具的 inputSchema 构造函数调用参数
- **THEN** 系统 SHALL 按 Schema 校验输入，校验失败返回明确的错误信息（含字段名和期望类型）

#### Scenario: Schema 包含 description
- **WHEN** AI App 读取 ITSM 工具的 inputSchema
- **THEN** 每个参数 SHALL 包含 `description` 字段，描述语言为中文

### Requirement: IT 服务台 Agent 预置定义
ITSM App 的 Seed 数据 SHALL 包含一个"IT 服务台"Agent 预置定义，类型为用户侧 public Agent。

#### Scenario: Seed 创建 IT 服务台 Agent
- **WHEN** ITSM App 执行 Seed 且 AI App 可用
- **THEN** 系统 SHALL 创建名为 "IT 服务台" 的 Agent：
  - type: assistant
  - visibility: public（所有用户可见）
  - system_prompt: 定义服务台行为（引导用户描述问题 → 搜索匹配服务 → 确认后创建工单 → 返回工单编号）
  - 绑定工具: itsm.search_services、itsm.create_ticket、itsm.query_ticket、itsm.list_my_tickets、itsm.cancel_ticket、itsm.add_comment

#### Scenario: 用户通过 Agent 对话提单
- **WHEN** 用户在 AI Chat 中与 "IT 服务台" Agent 对话，描述 "我的 VPN 连不上"
- **THEN** Agent SHALL 调用 itsm.search_services 搜索 VPN 相关服务，向用户确认后调用 itsm.create_ticket 创建工单，返回工单编号给用户

#### Scenario: AI App 不可用时跳过
- **WHEN** ITSM App 执行 Seed 且 AI App 不可用
- **THEN** 系统 SHALL 跳过 Agent 创建，仅输出 info 级别日志

#### Scenario: Seed 幂等
- **WHEN** ITSM App 重复执行 Seed
- **THEN** 系统 SHALL 检查同名 Agent 是否存在，存在则跳过创建

### Requirement: 流程决策 Agent 预置定义
ITSM App 的 Seed 数据 SHALL 包含一个"流程决策"Agent 预置定义，类型为系统侧 private Agent。

#### Scenario: Seed 创建流程决策 Agent
- **WHEN** ITSM App 执行 Seed 且 AI App 可用
- **THEN** 系统 SHALL 创建名为 "流程决策" 的 Agent：
  - type: assistant
  - visibility: private（仅系统内部使用，用户不可见）
  - temperature: 0.2（低温度确保决策稳定性）
  - system_prompt: 定义决策行为（分析工单上下文 → 评估可用操作 → 选择最优下一步 → 输出结构化 DecisionPlan JSON → 评估信心分数）

#### Scenario: 服务绑定
- **WHEN** 管理员创建智能服务时选择流程决策 Agent
- **THEN** 系统 SHALL 将该 Agent 的 ID 保存到 ServiceDefinition.agent_id，SmartEngine 使用此 Agent 做工单流转决策

### Requirement: 处理协助 Agent 预置定义
ITSM App 的 Seed 数据 SHALL 包含一个"处理协助"Agent 预置定义，类型为处理人侧 team Agent。

#### Scenario: Seed 创建处理协助 Agent
- **WHEN** ITSM App 执行 Seed 且 AI App 可用
- **THEN** 系统 SHALL 创建名为 "处理协助" 的 Agent：
  - type: assistant
  - visibility: team（团队可见，ITSM 处理人可用）
  - system_prompt: 定义协助行为（查询知识库提供诊断建议、辅助编写处理记录、解答技术问题）

#### Scenario: AI Copilot 使用
- **WHEN** 处理人在工单详情页点击"AI 协助"按钮
- **THEN** 前端 SHALL 创建一个 Agent Session（agent_id=处理协助 Agent），在 initial message 中注入当前工单的摘要信息（ticket_code、summary、description、status、current_step），打开 Chat 面板与 Agent 对话
