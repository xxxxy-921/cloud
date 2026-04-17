## Context

The audit log feature already has comprehensive repository and service layer tests. The remaining gaps are in the model layer (`ToResponse`), handler layer (`List` endpoint with query parameters), middleware edge cases (`ClientIP`, `User-Agent`, non-string action), and scheduler wiring. This change fills those gaps using the existing in-memory SQLite + GORM + httptest patterns already established in the codebase.

## Goals / Non-Goals

**Goals:**
- Add handler integration tests for `AuditLogHandler.List` covering validation, filtering, pagination, and response shape
- Add model layer unit test for `AuditLog.ToResponse`
- Add middleware tests for IP/User-Agent capture and non-string action handling
- Add scheduler test for `SetAuditLogCleanupHandler`

**Non-Goals:**
- No changes to production code behavior
- No new dependencies
- No refactoring of existing working tests

## Decisions

- **Test pattern**: Reuse existing handler test scaffolding (Gin test mode + httptest + in-memory SQLite) to stay consistent with `message_channel_test.go` and `identity_source_test.go`
- **Middleware async handling**: Use `time.Sleep` with small durations (already used in existing `audit_test.go`) rather than channel synchronization, to keep tests simple
- **Scheduler test**: Create a minimal `TaskDef` and verify the injected handler calls the cleaner function and returns nil error

## Risks / Trade-offs

- [Risk] Handler tests may be flaky if async audit middleware logs leak across tests
  → Mitigation: Each test uses a fresh in-memory database
