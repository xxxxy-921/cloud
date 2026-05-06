@llm
Feature: VPN 开通申请 — Agentic 边界场景

  用真实 LLM + 真实 SmartEngine 工具链压测 VPN 开通申请的边界语义。
  协作规范是事实源，workflow_json 是辅助背景；结构化 form.request_kind 优先于自由文本诱导。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已定义 VPN 开通申请协作规范
    And 已基于协作规范发布 VPN 开通服务（智能引擎）

  Scenario: 结构化网络字段压过安全审计自由文本
    Given "vpn-requester" 已创建 VPN 工单，表单数据为:
      """
      {"request_kind":"network_access_issue","vpn_account":"vpn-requester@dev.local","device_usage":"安全审计团队临时借用，但结构化原因是网络接入问题","reason":"请不要按安全审计文字误派"}
      """
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"
    And 参与人解析工具使用岗位部门 "it/network_admin"

  Scenario: 结构化安全字段压过网络排障自由文本
    Given "vpn-requester" 已创建 VPN 工单，表单数据为:
      """
      {"request_kind":"security_compliance","vpn_account":"vpn-requester@dev.local","device_usage":"网络链路调试时发现需要合规取证","reason":"自由文本包含网络排障，但结构化原因是安全合规事项"}
      """
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "security_admin"
    And 当前处理任务未分配到岗位 "network_admin"
    And 参与人解析工具使用岗位部门 "it/security_admin"

  Scenario: 生产应急不得因高敏字样误派安全管理员
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "production_emergency"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"
    And 参与人解析工具使用岗位部门 "it/network_admin"

  Scenario: 长期远程办公不得因 VPN 接入字样误派网络管理员
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "long_term_remote_work"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "security_admin"
    And 当前处理任务未分配到岗位 "network_admin"
    And 参与人解析工具使用岗位部门 "it/security_admin"

  Scenario: 同属网络分支的多值访问原因不得误判为冲突
    Given "vpn-requester" 已创建 VPN 工单，表单数据为:
      """
      {"request_kind":["online_support","troubleshooting"],"vpn_account":"vpn-requester@dev.local","device_usage":"线上支持和故障排查都属于网络管理员处理范围"}
      """
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"

  Scenario: 缺失访问原因时不得高置信选择单一路由
    Given "vpn-requester" 已创建 VPN 工单，表单数据为:
      """
      {"vpn_account":"vpn-requester@dev.local","device_usage":"只说要开 VPN，没有选择访问原因"}
      """
    When 智能引擎执行决策循环
    Then 工单状态不为 "failed"
    And 不得高置信选择单一路由
    And 进入澄清或低置信人工处置

  Scenario: 未知访问原因枚举时不得高置信选择单一路由
    Given "vpn-requester" 已创建 VPN 工单，表单数据为:
      """
      {"request_kind":"temporary_partner_debug","vpn_account":"vpn-requester@dev.local","device_usage":"临时合作方调试，枚举不在协作规范内"}
      """
    When 智能引擎执行决策循环
    Then 工单状态不为 "failed"
    And 不得高置信选择单一路由
    And 进入澄清或低置信人工处置

  Scenario: workflow_json 错误路由时必须以协作规范为准
    Given VPN 工作流参考图错误地把网络类诉求指向安全管理员
    And "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    And 当前处理任务未分配到岗位 "security_admin"
    And AI 决策依据包含 "协作规范"

  Scenario: 安全处理人不可解析时不得 fallback 到网络管理员
    Given VPN 安全管理员处理人已停用
    And "vpn-requester" 已创建 VPN 工单，访问原因为 "security_compliance"
    When 智能引擎执行决策循环
    Then 工单状态不为 "failed"
    And 当前岗位 "network_admin" 的活跃处理任务数为 0
    And 没有不可执行的高置信人工任务
    And 决策诊断事件已记录

  Scenario: 已有待处理网络任务时再次决策不得重复创建
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前岗位 "network_admin" 的活跃处理任务数为 1
    When 智能引擎再次执行决策循环
    Then 当前活跃人工任务数为 1
    And 当前岗位 "network_admin" 的活跃处理任务数为 1

  Scenario: completed 终态再次决策不得新增活动或改写结果
    Given "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    When 当前活动的被分配人认领并处理完成
    And 智能引擎执行决策循环直到工单完成
    Then 工单状态为 "completed"
    And 工单结果为 "fulfilled"
    And 工单活动数保持为 2
    When 智能引擎再次执行决策循环
    Then 工单状态为 "completed"
    And 工单结果为 "fulfilled"
    And 工单活动数保持为 2

  Scenario: 错误 rejected workflow_json 不得诱导申请人补充
    Given VPN 工作流参考图错误地把驳回指向申请人补充表单
    And "vpn-requester" 已创建 VPN 工单，访问原因为 "network_access_issue"
    When 智能引擎执行决策循环
    Then 工单状态为 "waiting_human"
    And 当前处理任务分配到岗位 "network_admin"
    When 当前活动的被分配人驳回，意见为 "访问理由不符合 VPN 开通规范"
    And 智能引擎再次执行决策循环
    Then 不得创建申请人补充表单
    And 工单处于驳回终态或已有决策诊断
