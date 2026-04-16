## Why

验证流程决策在审批节点参与者缺失时能安全兜底（不导致工单失败），参与者完整时能正确路由。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_workflow_progression_participant.feature`

## What Changes

- 创建 `features/vpn_participant_validation.feature`，2 个 Scenario：
  - 审批节点缺失参与者时引擎安全兜底
  - 审批节点参与者完整时正确路由并走完全流程
- 创建 `steps_vpn_participant_test.go`：构造缺失/完整参与者的工作流 fixture + 断言

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加参与者校验场景

## Impact

- `internal/app/itsm/features/vpn_participant_validation.feature` (new)
- `internal/app/itsm/steps_vpn_participant_test.go` (new)

## Dependencies

- vpn-bdd-main-flow (Phase 3)
