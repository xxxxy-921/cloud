# This file demonstrates the Gherkin format convention for ITSM BDD tests.
# Tag @wip marks scenarios that have no step definitions yet — they are skipped.
#
# Naming convention:
#   <module>_<capability>.feature
#   e.g. classic_engine_linear.feature, workflow_generate_branch.feature
#
# Step style (Chinese):
#   Given 一个服务定义 "xxx" 使用经典引擎
#   When  用户创建工单
#   Then  工单状态为 "进行中"

@wip
Feature: 示例 — 经典引擎线性流程

  Scenario: 线性流程完整执行
    Given 一个服务定义 "报修服务" 使用经典引擎
    And   工作流包含: 开始 → 提交表单 → IT处理 → 结束
    When  用户创建工单
    Then  工单状态为 "进行中"
