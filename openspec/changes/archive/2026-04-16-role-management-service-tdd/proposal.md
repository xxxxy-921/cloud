## Why

The role management service (`internal/service/role.go`) contains critical business rules—code uniqueness guards, system role immutability, Casbin policy migration on code rename, data scope validation, and user-assignment checks—but currently has zero test coverage. Adding service-layer tests will prevent regressions and make the RBAC core safer to refactor.

## What Changes

- Add `internal/service/role_test.go` with TDD-style tests for all `RoleService` methods.
- Use in-memory SQLite (shared cache) following the existing user-management test pattern.
- Test real dependencies (`RoleRepo`, `CasbinService`) without mocking.
- Cover business rules: duplicate code, system role guards, Casbin policy migration, data scope validation, and user-assignment blocks.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `role-management`: Add service-layer test coverage requirements. The behavior being tested already exists; this delta captures the test requirements as spec-level acceptance criteria.

## Impact

- `internal/service/role_test.go` (new)
- `internal/service/*` (minor test helper additions if needed)
- No API or behavior changes; purely additive test coverage.
