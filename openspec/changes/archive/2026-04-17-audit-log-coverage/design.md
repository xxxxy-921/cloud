## Context

The audit log subsystem (`internal/model/audit_log.go`, `internal/repository/audit_log.go`, `internal/service/audit_log.go`, `internal/middleware/audit.go`) currently has zero unit tests. This is inconsistent with the rest of the kernel, where services like `message_channel`, `user`, and `settings` already have comprehensive repository and service layer tests using sqlite in-memory databases and the `samber/do` injector.

## Goals / Non-Goals

**Goals:**
- Achieve high unit-test coverage for the audit log repository, service, middleware, and handler audit paths.
- Follow existing project conventions (sqlite in-memory, table-driven tests where appropriate, `do` injector for service tests).
- Keep tests deterministic and fast (no goroutine sleeps in assertions).

**Non-Goals:**
- Refactoring production code for better testability unless absolutely unavoidable.
- Adding integration tests or frontend tests.
- Changing audit log behavior or data model.

## Decisions

- **Repository tests**: Use `github.com/glebarez/sqlite` in-memory DB with `AutoMigrate(&model.AuditLog{})`. Seed helpers accept an explicit `CreatedAt` so `DateRange` and `DeleteBefore` tests are deterministic.
- **Service tests**: `Log()` spawns a goroutine. To avoid sleeps, provide a test seam by invoking `Log()` and then reading the DB directly with a small timeout/poll, or by accepting that the goroutine finishes quickly in sqlite in-memory. For determinism, tests will call `Log()` and then assert via the repository after a channel-based sync or short `sync.WaitGroup` injection if needed. However, because we explicitly decided *not* to change production code, we will use a small retry loop (max 100ms) against the in-memory DB.
- **Middleware tests**: Construct a `gin.Engine` with the `Audit` middleware, invoke handlers that set/unset audit metadata, and assert on DB state.
- **Handler tests (auth)**: Test the login-failure audit path by mocking `AuthService` and asserting `AuditLogRepo` state.

## Risks / Trade-offs

- [Risk] `Log()` uses a naked goroutine, making synchronous assertions slightly flaky → Mitigation: use sqlite in-memory (very fast), poll with a short timeout, and document the limitation.
- [Risk] `Cleanup()` depends on `SysConfigRepo` → Mitigation: seed `SystemConfig` rows in service tests.
