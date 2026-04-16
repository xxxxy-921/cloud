## Context

The MenuService (`internal/service/menu.go`) provides tree-structured menu retrieval, Casbin-based permission filtering, CRUD operations, and reordering. There is currently no dedicated service-layer test file. We will follow the established kernel test pattern: in-memory SQLite with a real `samber/do` injector, real repositories, and a real Casbin enforcer created via `internal/casbin.NewEnforcerWithDB`.

## Goals / Non-Goals

**Goals:**
- Add comprehensive `internal/service/menu_test.go` covering all `MenuService` public methods.
- Verify tree building, permission filtering (including parent retention logic), CRUD, and reordering.
- Keep tests fast and deterministic with shared-memory SQLite.

**Non-Goals:**
- No handler-level or frontend tests.
- No changes to production code unless tests reveal bugs.

## Decisions

1. **Test database**: Use `file:test?mode=memory&cache=shared` SQLite DSN, identical to role/user service tests.
2. **Casbin integration**: Use `casbin.NewEnforcerWithDB` so `GetUserTree` and `GetUserPermissions` tests exercise real policy storage.
3. **Seed helper**: Create helper functions `seedMenu`, `seedMenus` to quickly build directory→menu→button trees.
4. **Admin shortcut verification**: Assert that `GetUserTree("admin")` returns the full tree regardless of policies.
