## 1. Repository Layer Tests

- [x] 1.1 Create `internal/repository/message_channel_test.go` with test DB helper and seed helpers
- [x] 1.2 Add tests for `Create`, `FindByID`, `List` (pagination + keyword)
- [x] 1.3 Add tests for `Update`, `Delete` (success + not-found)
- [x] 1.4 Add tests for `ToggleEnabled` and `MaskConfig` (happy path + malformed JSON)

## 2. Service Layer Refactor & Tests

- [x] 2.1 Add injectable `driverResolver` field to `MessageChannelService` (default to `channel.GetDriver`)
- [x] 2.2 Create `internal/service/message_channel_test.go` with test DB helper and stub driver
- [x] 2.3 Add tests for `Create` (success + invalid type), `Get` (success + not-found), `List`
- [x] 2.4 Add tests for `Update` (success + password preservation + invalid JSON), `Delete`, `ToggleEnabled`
- [x] 2.5 Add tests for `TestChannel` (success + failure) and `SendTest` (success + not-found)

## 3. EmailDriver Refactor & Tests

- [x] 3.1 Extract internal `smtpClient` interface from `EmailDriver` and add a production constructor
- [x] 3.2 Update `EmailDriver` to depend on the injectable SMTP client abstraction
- [x] 3.3 Create `internal/channel/email_test.go` with a fake SMTP client implementation
- [x] 3.4 Add tests for `Send` (plain text + HTML multipart + TLS path)
- [x] 3.5 Add tests for `Test` (secure TLS + STARTTLS + plain SMTP + auth failure)

## 4. Handler Layer Tests

- [x] 4.1 Create `internal/handler/message_channel_test.go` with Gin + httptest scaffolding and stub service
- [x] 4.2 Add tests for `List`, `Get` (200 + 404)
- [x] 4.3 Add tests for `Create` (200 + 400) and `Update` (200 + 404)
- [x] 4.4 Add tests for `Delete` (200 + 404) and `Toggle` (200)
- [x] 4.5 Add tests for `Test` (success + failure) and `SendTest` (200)

## 5. Verification

- [x] 5.1 Run `go test ./internal/repository/... ./internal/service/... ./internal/channel/... ./internal/handler/...` and ensure all tests pass
- [x] 5.2 Run `go build -tags dev ./cmd/server/` to confirm no compilation regressions
