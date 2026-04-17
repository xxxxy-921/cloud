## Why

The audit log subsystem currently has zero test coverage. This creates regression risk for a critical compliance feature and makes future refactors unsafe. We need comprehensive unit tests to ensure the repository, service, and middleware layers behave correctly.

## What Changes

- Add unit tests for `internal/repository/audit_log.go` covering `Create`, `List` (pagination, filters, sorting), `DeleteBefore`, and `Migrate`.
- Add unit tests for `internal/service/audit_log.go` covering `Log`, `List`, and `Cleanup` retention logic.
- Add unit tests for `internal/middleware/audit.go` covering success-only recording and metadata extraction.
- Add unit tests for auth-handler audit paths (login failure logging).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- None. This change only adds tests; no behavioral requirements are modified.

## Impact

- Test files added under `internal/repository/`, `internal/service/`, `internal/handler/`, and `internal/middleware/`.
- No production code changes except minor testability adjustments if strictly necessary (none expected).
