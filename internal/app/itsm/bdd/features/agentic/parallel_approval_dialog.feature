@llm
Feature: 多角色并签申请 — 服务台智能体服务匹配

  验证服务台智能体能够识别并匹配多角色并签申请服务，正确调用 itsm.service_match
  和 itsm.service_load 工具，并收集申请所需的全部信息。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份         | 用户名                   | 部门 | 岗位            |
      | 申请人       | pa-dialog-requester      | -    | -               |
      | 网络管理员   | pa-dialog-netadmin       | it   | network_admin   |
      | 安全管理员   | pa-dialog-secadmin       | it   | security_admin  |
      | 运维管理员   | pa-dialog-opsadmin       | it   | ops_admin       |
    And 已发布多角色并签申请对话测试服务

  Scenario: 服务台智能体识别并签申请场景并加载服务
    Given 用户消息为 "我需要提一个防火墙策略变更申请，需要网络和安全团队同时审批"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And 工具调用序列包含 "itsm.service_load"

  Scenario: 信息不完整时服务台智能体追问必填字段
    Given 用户消息为 "帮我提一个需要多人审批的变更申请"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And Agent 未调用 draft_prepare 或未继续到 draft_confirm
