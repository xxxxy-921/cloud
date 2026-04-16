## Why

验证 VPN 开通申请从创建工单到审批完成的主链路。这是引擎层的核心场景：申请人提交 → 经典引擎路由到正确审批人（网络管理员 vs 安全管理员）→ 审批通过 → 工单完成。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_request_main_flow.feature`

## What Changes

- 创建 `features/vpn_main_flow.feature`：一个 Scenario 覆盖网络管理员审批路径
- 创建 `steps_vpn_main_flow_test.go`：步骤实现（创建工单、提交表单、引擎 Progress、认领、审批、断言状态）
- 验证排他网关根据 `request_kind` 字段正确路由
- 验证不分派给 security_admin

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加 VPN 主链路场景

## Impact

- `internal/app/itsm/features/vpn_main_flow.feature` (new)
- `internal/app/itsm/steps_vpn_main_flow_test.go` (new)
- `internal/app/itsm/bdd_test.go` (modified — register steps)

## Dependencies

- vpn-bdd-workflow-fixture (Phase 2)
