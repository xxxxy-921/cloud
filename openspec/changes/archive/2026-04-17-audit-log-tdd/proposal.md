## Why

The audit log feature currently has good repository and service layer test coverage, but gaps remain in the model layer, handler layer, middleware edge cases, and scheduler integration. Adding comprehensive TDD coverage to these layers will prevent regressions and document expected behavior for the audit log API and middleware.

## What Changes

- Add model layer unit tests for `AuditLog.ToResponse()` field mapping
- Add handler layer integration tests for `AuditLogHandler.List()` including category validation, missing category, pagination, date range parsing, keyword/action/resource filtering, and response serialization
- Add middleware tests for `ClientIP`/`User-Agent` capture and non-string `audit_action` handling
- Add scheduler test for `SetAuditLogCleanupHandler` wiring

## Capabilities

### New Capabilities
- `audit-log-test-coverage`: Test coverage for the audit log handler, model, middleware edge cases, and scheduler cleanup task

### Modified Capabilities
<!-- No existing capability requirements are changing; this change only adds tests -->

## Impact

- `internal/model/audit_log_test.go` (new)
- `internal/handler/audit_log_test.go` (new)
- `internal/middleware/audit_test.go` (additional tests)
- `internal/scheduler/builtin_test.go` (new)
- No API or behavior changes
