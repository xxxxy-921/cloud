@deterministic @agentic @parallel
Feature: SmartEngine 并行活动收敛

  并行计划必须创建共享并行组的活动，只有全部分支完成后才允许推进下一轮。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已定义 VPN 开通申请协作规范
    And 已基于静态工作流发布 VPN 开通服务（智能引擎）

  Scenario: 并行计划创建共享 activity_group_id 的多个活跃活动
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 执行确定性并行处理决策，岗位为:
      | 部门 | 岗位           |
      | it   | network_admin  |
      | it   | security_admin |
    Then 工单状态为 "waiting_human"
    And 当前活跃人工任务数为 2
    And 所有活跃活动共享同一并行组

  Scenario: 仅完成一个并行活动时不得触发后续决策
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 执行确定性并行处理决策，岗位为:
      | 部门 | 岗位           |
      | it   | network_admin  |
      | it   | security_admin |
    And 完成一个并行活动
    Then 当前仍有未完成并行活动
    And 当前活跃人工任务数为 1
    And 时间线不包含 "workflow_completed" 类型事件

  Scenario: 全部并行活动完成后并行组收敛且当前活动清空
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 执行确定性并行处理决策，岗位为:
      | 部门 | 岗位           |
      | it   | network_admin  |
      | it   | security_admin |
    And 完成剩余并行活动
    Then 并行组已收敛且当前活动已清空
