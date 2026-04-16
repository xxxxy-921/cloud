## Why

验证服务台 tool chain（service_load → draft_prepare → draft_confirm → ticket_create）在不同用户表达和多轮对话下仍能稳定工作。这是从引擎层向服务台对话层的跃迁。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_service_desk_dialog_coverage.feature`

## What Changes

- 创建 `features/vpn_dialog_coverage.feature`，Scenario Outline 覆盖：
  - 3 种确认后成功建单的对话场景
  - 3 种未确认不得建单的对话场景
- 创建 `steps_vpn_dialog_test.go`：调用 service desk tool chain 的步骤实现
- 需要搭建最小 Agent/Session 上下文来驱动 tool chain

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加服务台对话覆盖场景

## Impact

- `internal/app/itsm/features/vpn_dialog_coverage.feature` (new)
- `internal/app/itsm/steps_vpn_dialog_test.go` (new)

## Dependencies

- vpn-bdd-main-flow (Phase 3)
