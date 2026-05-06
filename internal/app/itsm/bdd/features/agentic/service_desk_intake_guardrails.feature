@llm
Feature: IT 服务台智能体 — 服务加载与表单收集护栏

  服务台智能体在接受用户诉求后，必须以 service_match / service_load 返回的权威服务定义为准。
  它不能跳过服务加载、不能凭空补字段，也不能把用户名当成邮箱类账号写入可确认草稿。

  Background:
    Given 已完成系统初始化
    And 已准备好以下参与人、岗位与职责
      | 身份               | 用户名             | 部门 | 岗位           |
      | 申请人             | vpn-requester      | -    | -              |
      | 网络管理员处理人   | network-operator   | it   | network_admin  |
      | 安全管理员处理人   | security-operator  | it   | security_admin |
    And 已发布 VPN 对话测试服务

  Scenario: 服务加载 — 返回权威表单字段与可用预填
    Given 用户消息为 "我要申请VPN，账号是 wenhaowu@dev.com，线上支持用的，申请原因是远程处理线上问题"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And 工具调用序列包含 "itsm.service_load"
    And service_load 的字段收集包含必填字段 "vpn_account,device_usage,request_kind,reason"
    And service_load 已从用户请求预填 "vpn_account" 为 "wenhaowu@dev.com"
    And service_load 已从用户请求预填 "request_kind" 为 "online_support"
    And 工具调用序列包含 "itsm.draft_prepare"
    And draft_prepare 表单字段 "vpn_account" 等于 "wenhaowu@dev.com"
    And draft_prepare 表单字段 "request_kind" 等于 "online_support"
    And 回复内容不匹配 "请补充.*VPN账号|请补充.*访问原因|设备型号"

  Scenario: 表单收集 — 用户名不能冒充邮箱账号进入可确认草稿
    Given 用户消息为 "我要申请VPN，账号是 vpn-requester，线上支持用的，申请原因是远程处理线上问题"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And 工具调用序列包含 "itsm.service_load"
    And Agent 未进入可确认草稿
    And 回复内容匹配 "邮箱|邮件|完整.*地址|账号.*格式"

  Scenario: 表单收集 — 只说想开 VPN 时必须基于服务字段追问
    Given 用户消息为 "帮我开个VPN"
    When 服务台 Agent 处理用户消息
    Then 工具调用序列包含 "itsm.service_match"
    And 工具调用序列包含 "itsm.service_load"
    And service_load 的字段收集缺失必填字段 "vpn_account,device_usage,request_kind,reason"
    And Agent 未调用 draft_prepare 或未继续到 draft_confirm
    And 回复内容匹配 "VPN账号|访问原因|申请原因|用途"
