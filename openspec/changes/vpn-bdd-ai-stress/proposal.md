## Why

验证 VPN 服务台在复杂服务集合（噪音服务干扰）和复杂组织图下仍能正确匹配服务并分派审批。这是 AI 能力的压力测试。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_service_desk_ai_stress.feature`

## What Changes

- 创建 `features/vpn_ai_stress.feature`，3 个 Scenario：
  - 网络故障排查对话在噪音服务集合下匹配 VPN 并路由给网络管理员
  - 口语化外部协作语境在复杂组织图下路由给安全管理员
  - 模糊表达在噪音服务集合下保持等待不误建单
- 创建 `steps_vpn_ai_stress_test.go`
- 需要创建多个干扰服务 + 增强版组织结构（备选审批人等）

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加 AI 压力测试场景

## Impact

- `internal/app/itsm/features/vpn_ai_stress.feature` (new)
- `internal/app/itsm/steps_vpn_ai_stress_test.go` (new)

## Dependencies

- vpn-bdd-dialog-coverage (Phase 6)
