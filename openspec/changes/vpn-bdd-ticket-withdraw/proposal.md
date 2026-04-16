## Why

验证申请人可以在工单尚未被处理前撤回 VPN 开通申请，以及各种撤回失败场景的正确处理。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_ticket_withdraw.feature`

## What Changes

- 创建 `features/vpn_ticket_withdraw.feature`，4 个 Scenario：
  - 无人认领时成功撤回
  - 已被审批人认领后撤回失败
  - 非申请人撤回失败
  - 撤回原因记录在时间线
- 创建 `steps_vpn_withdraw_test.go`：撤回步骤实现 + 时间线断言

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加工单撤回场景

## Impact

- `internal/app/itsm/features/vpn_ticket_withdraw.feature` (new)
- `internal/app/itsm/steps_vpn_withdraw_test.go` (new)

## Dependencies

- vpn-bdd-main-flow (Phase 3) — 复用工单创建和流转步骤
