@deterministic @agentic @smart_engine @rejected
Feature: SmartEngine 驳回恢复路径 — 不允许默认退回补充

  人工节点 rejected 后，下一轮决策必须读取 completed_activity 与 operator opinion，
  并以协作规范为恢复事实源，不能把驳回自动解释成“退回申请人补充”。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已定义 VPN 开通申请协作规范
    And 已基于静态工作流发布 VPN 开通服务（智能引擎）

  Scenario: rejected 后 ticket_context 必须暴露恢复事实且不得创建申请人补充表单
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 执行确定性决策 type="process" 参与者为 "network-operator"
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    When 当前活动的被分配人驳回，意见为 "访问理由不符合 VPN 开通规范"
    Given 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}}
        ],
        "plan": {
          "next_step_type": "complete",
          "execution_mode": "single",
          "activities": [],
          "reasoning": "decision.ticket_context.completed_activity.outcome=rejected 且 operator_opinion=访问理由不符合 VPN 开通规范；协作规范未显式定义补充或返工路径，因此不得创建申请人补充表单，按驳回恢复策略结束。",
          "confidence": 0.9
        }
      }
      """
    When 智能引擎再次执行决策循环
    Then 决策工具调用顺序为:
      | decision.ticket_context |
    And 决策工具 "decision.ticket_context" 返回结果包含 "completed_activity"
    And 决策工具 "decision.ticket_context" 返回结果包含 "requires_recovery_decision"
    And 决策工具 "decision.ticket_context" 返回结果包含 "访问理由不符合 VPN 开通规范"
    And 不得创建申请人补充表单
    And AI 决策依据包含 "协作规范未显式定义补充或返工路径"

  Scenario: 错误参考图把 rejected 指向申请人补充时仍必须被协作规范拦住
    Given VPN 工作流参考图错误地把驳回指向申请人补充表单
    And "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 执行确定性决策 type="process" 参与者为 "network-operator"
    And 当前活动的被分配人驳回，意见为 "不允许开通"
    Given 已启用脚本化智能决策器:
      """
      {
        "tool_calls": [
          {"name": "decision.ticket_context", "args": {}}
        ],
        "plan": {
          "next_step_type": "complete",
          "execution_mode": "single",
          "activities": [],
          "reasoning": "workflow_json 的 rejected 参考路径不具备高于协作规范的权威性；协作规范未写退回申请人补充，所以不能创建 requester form。",
          "confidence": 0.9
        }
      }
      """
    When 智能引擎再次执行决策循环
    Then 不得创建申请人补充表单
    And AI 决策依据包含 "协作规范"
