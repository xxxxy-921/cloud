Feature: SLA 保障岗 Agentic 处理
  系统需要确认 SLA 保障岗在 Agentic 模式下能读取风险工单、SLA 上下文和升级规则，并触发正确保障动作。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份       | 用户名        | 部门 | 岗位      |
      | 申请人     | sla-requester | it   | staff     |
      | 当前处理人 | ops-current   | it   | ops_admin |
      | 升级处理人 | ops-lead      | it   | ops_lead  |
    And 已发布带 SLA 的智能服务和 SLA 保障岗

  Scenario: SLA 保障岗触发通知升级
    Given 存在响应 SLA 已超时且命中 "notify" 升级规则的工单
    When 执行 SLA 保障扫描
    Then SLA 保障岗已调用工具 "sla.risk_queue"
    And SLA 保障岗已调用工具 "sla.ticket_context"
    And SLA 保障岗已调用工具 "sla.escalation_rules"
    And SLA 保障岗已调用工具 "sla.trigger_escalation"
    And 时间线包含 "sla_escalation" 类型事件

  Scenario: SLA 保障岗触发转派升级
    Given 存在响应 SLA 已超时且命中 "reassign" 升级规则的工单
    When 执行 SLA 保障扫描
    Then SLA 保障岗已调用工具 "sla.risk_queue"
    And SLA 保障岗已调用工具 "sla.ticket_context"
    And SLA 保障岗已调用工具 "sla.escalation_rules"
    And SLA 保障岗已调用工具 "sla.trigger_escalation"
    And 工单已转派给 "ops-lead"
    And 时间线包含 "sla_escalation" 类型事件

  Scenario: SLA 保障岗触发优先级升级
    Given 存在响应 SLA 已超时且命中 "escalate_priority" 升级规则的工单
    When 执行 SLA 保障扫描
    Then SLA 保障岗已调用工具 "sla.risk_queue"
    And SLA 保障岗已调用工具 "sla.ticket_context"
    And SLA 保障岗已调用工具 "sla.escalation_rules"
    And SLA 保障岗已调用工具 "sla.trigger_escalation"
    And 工单优先级为 "urgent"
    And 时间线包含 "sla_escalation" 类型事件

  Scenario: SLA 保障岗不得处理尚未超时的工单
    Given 存在响应 SLA 尚未超时但配置 "notify" 升级规则的工单
    When 执行 SLA 保障扫描，使用 "正常触发" 模拟智能体
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 0

  Scenario: SLA 保障岗不得提前触发未到等待时间的升级规则
    Given 存在响应 SLA 已超时但 "notify" 升级规则需等待 10 分钟的工单
    When 执行 SLA 保障扫描，使用 "正常触发" 模拟智能体
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 工单 SLA 状态为 "breached_response"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 0

  Scenario: SLA 保障岗不得重复触发已处理过的升级规则
    Given 存在响应 SLA 已超时且命中 "notify" 升级规则的工单
    And 已记录当前 SLA 升级规则的 "sla_escalation" 时间线
    When 执行 SLA 保障扫描，使用 "正常触发" 模拟智能体
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 时间线中 "sla_escalation" 类型事件数量为 1
    And 时间线中 "sla_assurance_pending" 类型事件数量为 0

  Scenario: SLA 保障岗不得越权触发非候选工单
    Given 存在响应 SLA 已超时且命中 "reassign" 升级规则的工单
    When 执行 SLA 保障扫描，使用 "错误工单" 模拟智能体
    Then SLA 保障岗已调用工具 "sla.trigger_escalation"
    And SLA 保障工具 "sla.trigger_escalation" 返回错误包含 "只允许触发当前候选工单和已命中规则"
    And 工单处理人为 "ops-current"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 1

  Scenario: SLA 保障岗不得越权触发非命中规则
    Given 存在响应 SLA 已超时且命中 "escalate_priority" 升级规则的工单
    When 执行 SLA 保障扫描，使用 "错误规则" 模拟智能体
    Then SLA 保障岗已调用工具 "sla.trigger_escalation"
    And SLA 保障工具 "sla.trigger_escalation" 返回错误包含 "只允许触发当前候选工单和已命中规则"
    And 工单优先级为 "normal"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 1

  Scenario: SLA 保障岗只输出文本不得被视为已处理
    Given 存在响应 SLA 已超时且命中 "notify" 升级规则的工单
    When 执行 SLA 保障扫描，使用 "只说不做" 模拟智能体
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 1
    And 最新 "sla_assurance_pending" 时间线原因包含 "声称已经处理"

  Scenario: 未绑定 SLA 保障岗时升级动作进入人工待处理
    Given 存在响应 SLA 已超时且命中 "notify" 升级规则的工单
    And SLA 保障岗未绑定
    When 执行 SLA 保障扫描，使用 "正常触发" 模拟智能体
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 1
    And 最新 "sla_assurance_pending" 时间线详情包含当前 SLA 规则

  Scenario: AI 执行器不可用时升级动作进入人工待处理
    Given 存在响应 SLA 已超时且命中 "notify" 升级规则的工单
    When 执行 SLA 保障扫描，AI 执行器不可用
    Then SLA 保障岗未调用工具 "sla.trigger_escalation"
    And 时间线中 "sla_escalation" 类型事件数量为 0
    And 时间线中 "sla_assurance_pending" 类型事件数量为 1
    And 最新 "sla_assurance_pending" 时间线详情包含当前 SLA 规则
