@deterministic @agentic @smart_engine
Feature: SmartEngine 决策完整性 — 防幻觉主门禁

  使用脚本化决策器驱动真实 SmartEngine 工具与执行层，锁定“不凭空决策”的核心合同。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已定义 VPN 开通申请协作规范
    And 已基于静态工作流发布 VPN 开通服务（智能引擎）

  Scenario: 首轮人工决策必须先读取工单上下文再解析参与人
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    And 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}},
          {"name": "decision.resolve_participant", "args": {"type": "position_department", "department_code": "it", "position_code": "network_admin"}}
        ],
        "plan": {
          "next_step_type": "process",
          "execution_mode": "single",
          "activities": [
            {"type": "process", "participant_type": "position_department", "department_code": "it", "position_code": "network_admin", "instructions": "根据 decision.ticket_context.form_data.request_kind=network_access_issue 派给网络管理员"}
          ],
          "reasoning": "decision.ticket_context 显示 form.request_kind=network_access_issue，协作规范要求 it/network_admin 处理；decision.resolve_participant 返回候选人后才创建人工任务。",
          "confidence": 0.93
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 决策工具调用顺序为:
      | decision.ticket_context       |
      | decision.resolve_participant  |
    And 所有决策工具均返回成功
    And 决策工具 "decision.ticket_context" 返回结果包含 "workflow_context"
    And 决策工具 "decision.resolve_participant" 返回结果包含 "network-operator"
    And 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"

  Scenario: workflow_json 与协作规范冲突时不得被错误参考图带偏
    Given VPN 工作流参考图错误地把网络类诉求指向安全管理员
    And "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    And 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}},
          {"name": "decision.resolve_participant", "args": {"type": "position_department", "department_code": "it", "position_code": "network_admin"}}
        ],
        "plan": {
          "next_step_type": "process",
          "execution_mode": "single",
          "activities": [
            {"type": "process", "participant_type": "position_department", "department_code": "it", "position_code": "network_admin", "instructions": "协作规范优先于错误 workflow_json，网络诉求仍由网络管理员处理"}
          ],
          "reasoning": "协作规范是事实源，request_kind=network_access_issue 属于网络分支；workflow_json 仅为辅助背景，冲突时不能派给 security_admin。",
          "confidence": 0.91
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"
    And AI 决策依据包含 "协作规范是事实源"

  Scenario: terminal 工单再次触发决策时不得创建新活动
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    And 当前工单状态强制设为 "completed"
    And 记录当前工单活动数
    And 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}}
        ],
        "plan": {
          "next_step_type": "process",
          "activities": [
            {"type": "process", "participant_type": "position_department", "department_code": "it", "position_code": "network_admin", "instructions": "这条计划不应被执行"}
          ],
          "reasoning": "terminal guard should stop before agentic decision",
          "confidence": 0.99
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 工单活动数未变化
    And 时间线不包含 "ai_decision_executed" 类型事件
