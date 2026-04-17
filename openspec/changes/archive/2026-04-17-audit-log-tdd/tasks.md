## 1. Model Layer Tests

- [x] 1.1 Create `internal/model/audit_log_test.go`
- [x] 1.2 Add test for `ToResponse()` field mapping

## 2. Handler Layer Tests

- [x] 2.1 Create `internal/handler/audit_log_test.go` with test DB helper and seed helper
- [x] 2.2 Add test for `List` success (200 with items/total/page/pageSize)
- [x] 2.3 Add tests for `List` category validation (missing category = 400, invalid category = 400)
- [x] 2.4 Add tests for `List` filters (keyword, action, resource)
- [x] 2.5 Add test for `List` date range parsing
- [x] 2.6 Add test verifying response items are `AuditLogResponse` shape

## 3. Middleware Tests

- [x] 3.1 Add test to `internal/middleware/audit_test.go` for `ClientIP` and `User-Agent` recording
- [x] 3.2 Add test for non-string `audit_action` handling

## 4. Scheduler Tests

- [x] 4.1 Create `internal/scheduler/builtin_test.go`
- [x] 4.2 Add test for `SetAuditLogCleanupHandler` invoking cleaner and returning nil

## 5. Verification

- [x] 5.1 Run `go test ./internal/model/... ./internal/handler/... ./internal/middleware/... ./internal/scheduler/...` and ensure all tests pass
- [x] 5.2 Run `go build -tags dev ./cmd/server/` to confirm no compilation regressions
