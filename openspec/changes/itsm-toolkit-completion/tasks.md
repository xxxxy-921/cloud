## 1. 数据模型变更

- [x] 1.1 AgentSession 模型增加 `State` 字段（JSON text），用于存储服务台会话状态机 (`internal/app/ai/agent_model.go`)

## 2. 通用工具注册（AI App）

- [x] 2.1 在 AI App seed 中新增 3 个通用工具定义：`general.current_time`、`system.current_user_profile`、`organization.org_context`（toolkit: "general"），写入 `ai_tools` 表 (`internal/app/ai/seed.go`)
- [x] 2.2 实现 `general.current_time` handler：返回服务器时间、UTC、中国时间、目标时区时间
- [x] 2.3 实现 `system.current_user_profile` handler：查询 User + Org App 的 AssignmentService 获取部门/岗位/角色，Org App 未安装时优雅降级
- [x] 2.4 实现 `organization.org_context` handler：支持按 username/department_code/position_code 筛选，返回用户/部门/岗位结构，Org App 未安装时返回空结果

## 3. ITSM 工具定义重写

- [x] 3.1 重写 `tools/provider.go` 的 `AllTools()` 函数：删除旧的 6 个工具定义，替换为 10 个新工具定义（service_match、service_confirm、service_load、new_request、draft_prepare、draft_confirm、validate_participants、ticket_create、my_tickets、ticket_withdraw），每个包含完整的 JSON Schema
- [x] 3.2 更新 `SeedTools()` 函数：增加旧工具清理逻辑（删除 itsm.search_services、itsm.query_ticket、itsm.cancel_ticket、itsm.add_comment 及其 agent_tools 绑定），已有工具更新而非重复创建

## 4. ITSM 工具 Handler 实现

- [x] 4.1 重写 `tools/handlers.go`：拆分 `TicketQuerier` 为 `ServiceDeskOperator` 接口（MatchServices、ConfirmService、LoadService、NewRequest、PrepareDraft、ConfirmDraft、ValidateParticipants、CreateTicket、MyTickets、WithdrawTicket）
- [x] 4.2 实现 `itsm.service_match` handler：查询已启用服务，关键词权重匹配，返回 0-3 候选 + 置信度 + confirmation_required，写入 session state
- [x] 4.3 实现 `itsm.service_confirm` handler：校验 service_id 在候选列表中，更新 session state 的 confirmed_service_id
- [x] 4.4 实现 `itsm.service_load` handler：加载服务的协作规范、表单定义、动作配置、路由字段提示，校验确认前置条件，更新 session state
- [x] 4.5 实现 `itsm.new_request` handler：清空 session state 中的所有服务台状态
- [x] 4.6 实现 `itsm.draft_prepare` handler：校验必填字段、选项值合法性、单选多值检测，自增 draft_version，更新 session state
- [x] 4.7 实现 `itsm.draft_confirm` handler：校验 stage 为 awaiting_confirmation，设置 confirmed_draft_version，检测 fields_hash 变更
- [x] 4.8 实现 `itsm.validate_participants` handler：解析 workflow_json 的审批节点，检查岗位+部门是否能解析到有效用户
- [x] 4.9 实现 `itsm.ticket_create` handler（增强版）：校验前置条件（loaded_service_id + confirmed_draft_version），创建工单，重置 session state
- [x] 4.10 实现 `itsm.my_tickets` handler：查询当前用户非终态工单，计算 can_withdraw 标志
- [x] 4.11 实现 `itsm.ticket_withdraw` handler：校验提交人身份和工单状态，调用取消逻辑

## 5. 智能体 Seed 重写

- [x] 5.1 重写 `SeedAgents()` 函数：先删除旧的 3 个智能体（"IT 服务台"、"ITSM 流程决策"、"ITSM 处理协助"）及其 ai_agent_tools 绑定
- [x] 5.2 创建"IT 服务台智能体"：type=assistant, strategy=react, visibility=public, temp=0.3, max_tokens=4096, max_turns=20，复刻 bklite 的 19 条约束版 system prompt
- [x] 5.3 创建"流程决策智能体"：type=assistant, strategy=react, visibility=private, temp=0.2, max_tokens=2048, max_turns=1，复刻 bklite 的决策原则版 system prompt
- [x] 5.4 绑定工具到服务台智能体：10 个 ITSM 工具 + 3 个通用工具
- [x] 5.5 确保 seed 幂等性：同名智能体存在时跳过创建，旧智能体只删除一次

## 6. 集成与验证

- [x] 6.1 确保 `go build -tags dev ./cmd/server/` 编译通过
- [x] 6.2 确保 `go test ./internal/app/ai/... ./internal/app/itsm/...` 通过
- [x] 6.3 验证 seed 顺序：AI App seed（通用工具）先于 ITSM App seed（智能体绑定通用工具），检查 main.go 中的 App 注册顺序
