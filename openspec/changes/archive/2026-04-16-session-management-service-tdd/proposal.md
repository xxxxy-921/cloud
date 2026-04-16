## Why

The session management module currently lacks service-layer test coverage. Following the established TDD pattern for kernel services, we need dedicated tests for `SessionService` to ensure active session listing, force kick logic, and token blacklist interactions are correct and regression-safe.

## What Changes

- Add `internal/service/session_test.go` with TDD-style service-layer tests for `SessionService`.
- Cover `ListSessions` and `KickSession` with real `RefreshTokenRepo` and `TokenBlacklist` (no mocks).
- Use in-memory SQLite consistent with existing kernel service test patterns.
- Add test requirements to the `session-management` capability spec.

## Capabilities

### New Capabilities
- `session-management-service-test`: Service-layer test requirements and scenarios for session management.

### Modified Capabilities
- `session-management`: Add requirements covering service-layer test scenarios for listing and kicking sessions.

## Impact

- `internal/service/session_test.go` (new)
- `openspec/specs/session-management/spec.md` (modified)
- No breaking changes to APIs or frontend.
