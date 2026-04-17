## Context

The message channel subsystem (`internal/channel`, `internal/service/message_channel.go`, `internal/handler/message_channel.go`) provides SMTP-based notification channels and a management UI. The implementation is complete and functional, but there are currently **no unit tests** for any layer. This makes future extensions (e.g. webhook, DingTalk, WeCom drivers) risky and prevents TDD for new channel types.

## Goals / Non-Goals

**Goals:**
- Achieve near-100% unit-test coverage for the message channel backend
- Refactor `EmailDriver` to be testable without a real SMTP server
- Ensure `MessageChannelService` business logic (password masking preservation, error mapping) is fully tested
- Verify HTTP handler boundaries (status codes, JSON shapes, audit field setting)

**Non-Goals:**
- No changes to the frontend React code
- No changes to public API contracts (request/response shapes remain identical)
- No integration tests against real SMTP servers (unit tests only)
- No new channel types (webhook, SMS, etc.) — this change is purely testing infrastructure

## Decisions

### 1. Extract an internal `smtpClient` interface for `EmailDriver`
**Rationale:** `EmailDriver` currently calls `smtp.NewClient`, `tls.Dial`, and `smtp.SendMail` directly. These are concrete implementations that require network access. By extracting a small interface (`Dial(addr string) (smtpClient, error)`, `Auth(...)`, `Mail(...)`, `Rcpt(...)`, `Data() (io.WriteCloser, error)`, `Quit()`), we can inject a fake SMTP client in tests and assert on the exact protocol interactions without sending real mail.

**Alternative considered:** Spin up a local fake SMTP server in tests. Rejected because it adds port-management flakiness and slower test execution.

### 2. Make `MessageChannelService` accept an injectable `driverResolver`
**Rationale:** `MessageChannelService` currently calls the global `channel.GetDriver()` function. This couples service tests to the real driver registry. We will add a `driverResolver func(string) (channel.Driver, error)` field on the service, initialized to `channel.GetDriver` in the constructor. Tests can override this with a stub driver.

### 3. Use in-memory SQLite for repository and service tests
**Rationale:** This is the established pattern in the codebase (`notification_test.go`, `user_test.go`). It provides fast, isolated tests with real GORM behavior (AutoMigrate, soft deletes, counts).

### 4. Handler tests use `gin.New()` + `httptest`
**Rationale:** Handler tests in Metis are lightweight integration tests that verify routing, binding, status codes, and JSON responses. We will construct a minimal Gin engine, mount the handler, and assert on the response recorder. Service dependencies will be replaced with thin stub implementations.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Mocking SMTP too aggressively may miss TLS/STARTTLS edge cases | The fake client exercises all code paths (TLS dial, STARTTLS, plain). We keep one manual smoke test checklist for real SMTP validation. |
| Password masking logic (`MaskConfig`) is string-based and fragile | Add dedicated tests for malformed JSON, missing password key, and nested objects. |
| Service tests accidentally depend on real `channel.GetDriver` | Always override `driverResolver` in service tests; never use the production constructor directly in test helpers. |
| Handler tests become large maintenance burden | Only test happy path + one error path per endpoint; avoid exhaustive table-driven tests for handlers. |
