## Context

The `SessionService` (`internal/service/session.go`) provides active session listing and force kick functionality. It depends on `RefreshTokenRepo` for database queries and `TokenBlacklist` for in-memory access token revocation. There is currently no dedicated service-layer test file. We will follow the established kernel test pattern: in-memory SQLite with a real `samber/do` injector and real dependencies.

## Goals / Non-Goals

**Goals:**
- Add comprehensive `internal/service/session_test.go` covering `ListSessions` and `KickSession`.
- Verify pagination, `IsCurrent` flag logic, self-kick prevention, not-found handling, and blacklist integration.
- Use real `TokenBlacklist` (in-memory map) rather than mocks.

**Non-Goals:**
- No handler-level or scheduler task tests.
- No changes to production code unless tests reveal bugs.

## Decisions

1. **Test database**: Use `file:test?mode=memory&cache=shared` SQLite DSN, migrating `User` and `RefreshToken` tables.
2. **TokenBlacklist**: Use the real `token.NewBlacklist()` implementation since it is a pure in-memory structure with no external dependencies.
3. **Seed helpers**: Create `seedUserForSessionTest` and `seedRefreshToken` to quickly set up active/revoked/expired tokens.
