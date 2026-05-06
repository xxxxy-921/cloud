@deterministic @agentic @observability
Feature: SmartEngine 决策解释与诊断可观测性

  每轮决策都必须留下可读、可追责、可审计的解释或诊断，解释不能只有空泛话术。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已定义 VPN 开通申请协作规范
    And 已基于静态工作流发布 VPN 开通服务（智能引擎）

  Scenario: 成功创建人工活动时写入结构化决策解释
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
            {"type": "process", "participant_type": "position_department", "department_code": "it", "position_code": "network_admin", "instructions": "按 request_kind=network_access_issue 进入网络管理员处理"}
          ],
          "reasoning": "decision.ticket_context.form_data.request_kind=network_access_issue，decision.resolve_participant 返回 network-operator；下一步创建 it/network_admin 人工处理。",
          "confidence": 0.94
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 时间线包含 "ai_decision_executed" 类型事件
    And 最新决策解释包含字段:
      | basis         |
      | trigger       |
      | decision      |
      | nextStep      |
      | humanOverride |
    And 最新决策解释依据包含 "decision.ticket_context"
    And 最新决策解释引用真实事实

  Scenario: 低置信决策进入人工处置时写入解释
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    And 智能引擎置信度阈值设为 0.95
    And 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}}
        ],
        "plan": {
          "next_step_type": "process",
          "execution_mode": "single",
          "activities": [
            {"type": "process", "participant_type": "position_department", "department_code": "it", "position_code": "network_admin", "instructions": "低置信候选处理"}
          ],
          "reasoning": "decision.ticket_context 提供了 request_kind，但参与人证据不足，置信度低于阈值，等待人工处置。",
          "confidence": 0.5
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 时间线包含 "ai_decision_pending" 类型事件
    And 最新决策解释依据包含 "置信度低于阈值"
    And 最新决策解释引用真实事实

  Scenario: 决策失败时写入诊断事件与解释
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    And 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [],
        "plan": {
          "next_step_type": "teleport",
          "execution_mode": "single",
          "activities": [],
          "reasoning": "decision.ticket_context.form_data.request_kind=network_access_issue，但模型输出了不存在的 next_step_type=teleport。",
          "confidence": 0.96
        }
      }
      """
    When 智能引擎执行脚本化决策循环
    Then 决策诊断事件已记录
    And 时间线包含 "ai_decision_failed" 类型事件
