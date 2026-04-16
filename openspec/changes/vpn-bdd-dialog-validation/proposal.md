## Why

验证服务台 Agent 在对话层的智能识别能力：跨路由冲突识别、同路由多选合并、必填缺失追问。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_service_desk_dialog_validation.feature`

## What Changes

- 创建 `features/vpn_dialog_validation.feature`，3 个 Scenario：
  - 跨路由冲突——Agent 识别并向用户澄清
  - 同路由多选——Agent 正常推进不要求二选一
  - 必填缺失——Agent 追问缺失信息而非直接提交
- 创建 `steps_vpn_dialog_validation_test.go`

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加服务台对话校验场景

## Impact

- `internal/app/itsm/features/vpn_dialog_validation.feature` (new)
- `internal/app/itsm/steps_vpn_dialog_validation_test.go` (new)

## Dependencies

- vpn-bdd-dialog-coverage (Phase 6)
