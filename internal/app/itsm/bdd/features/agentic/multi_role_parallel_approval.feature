Feature: 多角色并签申请 — 智能引擎生成参考路径与全流程审批
  验证协作规范可生成健康参考路径（可发布标准），以及 Smart Engine
  支持多角色并签审批：AI Agent 输出 execution_mode: "parallel" 后创建并行
  审批组，全部通过后汇聚触发下一轮决策，最终由运维管理员单签审批完成工单。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份           | 用户名                     | 部门 | 岗位            |
      | 申请人         | pa-requester               | -    | -               |
      | 并签审批人A    | pa-netadmin                | it   | network_admin   |
      | 并签审批人B    | pa-secadmin                | it   | security_admin  |
      | 最终审批人     | pa-opsadmin                | it   | ops_admin       |
    And 已定义多角色并签申请协作规范
    And 已基于协作规范发布多角色并签申请服务（智能引擎）

  @bdd @itsm @parallel_approval
  Scenario: 全部并签审批通过后汇聚推进到运维管理员终审并完成
    Given "pa-requester" 已创建并签申请工单，场景为 "standard"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 应存在一个并签审批活动组，包含 2 个并行活动
    When 并签审批组中岗位 "network_admin" 的审批人认领并审批通过
    Then 并签审批组仍有未完成活动，不应触发下一步
    When 并签审批组中岗位 "security_admin" 的审批人认领并审批通过
    Then 并签审批组全部完成，应触发下一轮决策
    When 智能引擎再次执行决策循环
    Then 当前活动类型为 "approve"
    When 当前活动的被分配人认领并审批通过
    And 智能引擎执行决策循环直到工单完成
    Then 工单状态为 "completed"

  @bdd @itsm @parallel_approval
  Scenario: 部分并签审批完成不得提前创建后续审批活动
    Given "pa-requester" 已创建并签申请工单，场景为 "standard"
    When 智能引擎执行决策循环
    Then 应存在一个并签审批活动组，包含 2 个并行活动
    When 并签审批组中岗位 "network_admin" 的审批人认领并审批通过
    Then 并签审批组仍有未完成活动，不应触发下一步
    And 不应存在分配给岗位 "ops_admin" 的待处理审批活动
