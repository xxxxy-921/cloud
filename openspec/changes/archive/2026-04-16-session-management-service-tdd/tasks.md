## 1. Test Infrastructure

- [x] 1.1 Create `internal/service/session_test.go` with `newTestDBForSession`, `newSessionServiceForTest`, `seedUserForSessionTest`, and `seedRefreshToken` helpers using in-memory SQLite and real `TokenBlacklist`.

## 2. List Sessions Tests

- [x] 2.1 Implement `TestSessionServiceListSessions_Pagination` to verify page size and total count.
- [x] 2.2 Implement `TestSessionServiceListSessions_MarksCurrentJTI` to verify `IsCurrent` flag.
- [x] 2.3 Implement `TestSessionServiceListSessions_ExcludesRevokedAndExpired` to verify only active tokens are returned.
- [x] 2.4 Implement `TestSessionServiceListSessions_Empty` to verify empty result with total=0.

## 3. Kick Session Tests

- [x] 3.1 Implement `TestSessionServiceKickSession_Success` to verify revocation and blacklist addition.
- [x] 3.2 Implement `TestSessionServiceKickSession_PreventsSelfKick` to verify `ErrCannotKickSelf`.
- [x] 3.3 Implement `TestSessionServiceKickSession_NotFound` to verify `ErrSessionNotFound` for missing IDs.
- [x] 3.4 Implement `TestSessionServiceKickSession_AlreadyRevoked` to verify `ErrSessionNotFound` for revoked tokens.
- [x] 3.5 Implement `TestSessionServiceKickSession_EmptyJTI` to verify no panic when `AccessTokenJTI` is empty.

## 4. Verification

- [x] 4.1 Run `go test ./internal/service/ -run TestSessionService -v` and ensure all tests pass.
- [x] 4.2 Run `go test ./...` to confirm no regressions.
