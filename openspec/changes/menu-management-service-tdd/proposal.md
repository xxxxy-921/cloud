## Why

The menu management module currently lacks comprehensive service-layer test coverage. Following the TDD patterns established in user-management and role-management, we need to add dedicated tests for the MenuService to ensure tree retrieval, permission filtering, CRUD operations, and sorting behavior are correct and regression-safe.

## What Changes

- Add `internal/service/menu_test.go` with TDD-style service-layer tests for `MenuService`.
- Cover `GetTree`, `GetUserTree`, `GetUserPermissions`, `Create`, `Update`, `ReorderMenus`, and `Delete`.
- Use in-memory SQLite and real Casbin enforcer (no mocks), consistent with existing kernel service test patterns.
- Add test requirements to the `menu-system` capability spec.

## Capabilities

### New Capabilities
- `menu-management-service-test`: Service-layer test requirements and scenarios for menu management.

### Modified Capabilities
- `menu-system`: Add requirements covering service-layer test scenarios for menu tree, permission filtering, CRUD, and reordering.

## Impact

- `internal/service/menu_test.go` (new)
- `openspec/specs/menu-system/spec.md` (modified)
- No breaking changes to APIs or frontend.
