## Context

The role management service (`internal/service/role.go`) implements business rules such as code uniqueness enforcement, system role immutability, Casbin policy migration on code rename, data scope validation, and user-assignment guards. Currently there are no service-layer tests. The user-management TDD just established a working test pattern using in-memory SQLite with real repository and Casbin instances.

## Goals / Non-Goals

**Goals:**
- Add comprehensive service-layer tests for `RoleService` following the user-management TDD pattern.
- Cover all business rules and edge cases without mocking repositories or Casbin.
- Keep tests fast and deterministic using shared-cache SQLite.

**Non-Goals:**
- Handler-layer tests (out of scope).
- Repository-layer unit tests with mocked DB.
- Changing any production behavior.

## Decisions

1. **Use in-memory SQLite with shared cache**
   - Rationale: Fast, isolated per test, and matches the existing user-management test style.
   - Alternative: Dockerized PostgreSQL. Rejected due to overhead and deviation from existing patterns.

2. **Use real Casbin enforcer via `NewEnforcerWithDB`**
   - Rationale: `RoleService` logic includes Casbin policy migration and cleanup. Testing with a real enforcer catches cross-layer issues that mocks hide.
   - Alternative: Mocked `CasbinService`. Rejected because Casbin behavior is central to the service's correctness.

3. **Seed minimal data per test**
   - Rationale: Each test creates its own roles/users/policies to stay independent. A small set of test helpers reduces boilerplate.

4. **AutoMigrate `casbin_rule` via gormadapter**
   - Rationale: `gormadapter.NewAdapterByDB(db)` automatically creates the Casbin rule table on first use, so no extra migration is needed in test setup.

## Risks / Trade-offs

- [Risk] `AutoMigrate` for `Role`, `RoleDeptScope`, and `User` may drift from production schema over time.
  → Mitigation: Only migrate tables directly used by `RoleService`.

- [Risk] Casbin enforcer creation adds ~50ms per test suite.
  → Mitigation: Create one enforcer per `newRoleServiceForTest` call; tests remain well under 1s total.
