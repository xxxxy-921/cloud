## Why

验证草稿版本校验机制（服务字段变更后旧草稿不能继续提交）和多值输入检测（radio/select 字段传入多值时返回结构化 warning）。

参考来源：bklite-cloud `tests/bdd/itsm/features/vpn_draft_version_and_multivalue.feature`

## What Changes

- 创建 `features/vpn_draft_version_multivalue.feature`，2 个 Scenario：
  - 草稿版本校验——字段变更后 draft_confirm 返回 service_fields_changed 错误
  - 多值输入检测——radio 字段传入多值时返回 multivalue_on_single_field warning
- 创建 `steps_vpn_draft_version_test.go`

## Capabilities

### Modified Capabilities
- `itsm-bdd-infrastructure`: 增加草稿版本与多值检测场景

## Impact

- `internal/app/itsm/features/vpn_draft_version_multivalue.feature` (new)
- `internal/app/itsm/steps_vpn_draft_version_test.go` (new)

## Dependencies

- vpn-bdd-dialog-coverage (Phase 6)
