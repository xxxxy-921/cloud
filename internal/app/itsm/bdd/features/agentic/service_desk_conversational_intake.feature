@llm @agentic @service_desk
Feature: IT 服务台对话式提单 — 最小可决策集与自愈

  服务台 Agent 应以对话方式收集关键字段，满足最小可决策集后再准备草稿；
  信息不完整、跨路由冲突、字段版本变化和时间语义都不能被模型幻觉绕过。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已发布 VPN 对话测试服务

  Scenario: 缺失关键字段时只追问而不准备草稿
    Given 用户消息为 "帮我开个VPN"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And Agent 未调用 draft_prepare 或未继续到 draft_confirm
    And 回复内容匹配 "补充|提供|需要|VPN|账号|原因|访问"

  Scenario: 完整输入满足最小可决策集时不得重复追问已给信息
    Given 用户消息为 "我要申请VPN，线上支持用的，VPN账号wenhaowu@dev.com，访问时段2026-05-01 09:00:00~18:00:00"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And 工具调用序列包含 "itsm.service_load"
    And 工具调用序列包含 "itsm.draft_prepare"
    And 回复内容不匹配 "请补充.*VPN账号|请补充.*访问原因|是否还有其他具体原因|设备型号"

  Scenario: 跨路由诉求必须澄清而不是替用户选择
    Given 用户消息为 "我要申请VPN，既要网络调试，也要安全审计"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_load"
    And Agent 未调用 draft_prepare 或未继续到 draft_confirm
    And 回复内容匹配 "不同.*路|处理.*路|选择|冲突|分属|哪一个|分别"

  Scenario: 字段版本变化后重新加载服务并重建草稿
    Given 服务字段将在草稿准备后变更
    And 用户消息为 "我要申请VPN开通，VPN账号wenhaowu@dev.com，类型L2TP，原因网络调试，访问时段2026-05-01 09:00:00~18:00:00"
    When 服务台 Agent 处理用户消息（含字段变更）
    Then 工具调用序列包含 "itsm.draft_confirm"
    And "itsm.service_load" 被调用至少 2 次
    And "itsm.draft_prepare" 被调用至少 2 次
