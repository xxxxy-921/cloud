## 1. Repository Layer Tests

- [x] 1.1 Create `internal/repository/audit_log_test.go` with helper `newTestDBForAuditLog`
- [x] 1.2 Test `Create` — verify ID generation and field persistence
- [x] 1.3 Test `List` category filtering, pagination, and default page/pageSize
- [x] 1.4 Test `List` keyword filter for `auth` (matches username) and `operation` (matches summary)
- [x] 1.5 Test `List` action, resource, and date-range filters
- [x] 1.6 Test `List` ordering by `created_at DESC`
- [x] 1.7 Test `DeleteBefore` — removes old logs, returns rows affected, scoped by category, preserves newer logs
- [x] 1.8 Test `Migrate` — runs without error and is idempotent

## 2. Service Layer Tests

- [x] 2.1 Create `internal/service/audit_log_test.go` with helper to build `AuditLogService` using `do.Injector`
- [x] 2.2 Test `Log` — async write persists to DB, defaults level to `info`, errors are swallowed (not propagated)
- [x] 2.3 Test `List` — correctly delegates to repository
- [x] 2.4 Test `Cleanup` — reads retention settings from `SystemConfig`, deletes only expired logs per category, returns localized summary

## 3. Middleware & Handler Tests

- [x] 3.1 Create `internal/middleware/audit_test.go` — verify `Audit` middleware records only 2xx responses with `audit_action` set, and extracts metadata correctly
- [x] 3.2 Create `internal/handler/message_channel_test.go` 风格的 auth audit tests（或独立的 auth handler 登录失败审计路径测试）— 验证登录失败写入 `AuditCategoryAuth` 日志

## 4. Verification

- [x] 4.1 Run `go test ./internal/repository/ ./internal/service/ ./internal/middleware/ ./internal/handler/` and ensure all new tests pass
- [x] 4.2 Run `go build -tags dev ./cmd/server/` to confirm no compilation regressions
