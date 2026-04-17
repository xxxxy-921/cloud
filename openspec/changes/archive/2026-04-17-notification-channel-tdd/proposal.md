## Why

The message channel feature (`message-channel`) is fully implemented in both backend and frontend, but currently has **zero automated test coverage**. Without tests, refactors to the driver layer (e.g. adding new channel types like webhook or DingTalk) or changes to the SMTP implementation risk regressions. Adding comprehensive unit tests now establishes a safety net and enables true TDD for future channel extensions.

## What Changes

- Add repository-layer tests for `MessageChannelRepo` (CRUD, toggle, masking)
- Add service-layer tests for `MessageChannelService` (business logic, password preservation, error paths)
- Refactor `EmailDriver` to accept an injectable SMTP client interface so it can be unit-tested without a real mail server
- Add driver-layer tests for `EmailDriver` (plain text, HTML, TLS, STARTTLS, error branches)
- Add handler-layer integration tests for `/api/v1/channels/*` endpoints

## Capabilities

### New Capabilities
- `message-channel-test-coverage`: Comprehensive unit-test coverage for the existing message channel repository, service, email driver, and HTTP handler layers.

### Modified Capabilities
- *(none — this change does not alter spec-level behavior, only adds tests and a minimal internal refactor for testability)*

## Impact

- `internal/repository/message_channel_test.go` (new)
- `internal/service/message_channel_test.go` (new)
- `internal/channel/email.go` (minor refactor: extract SMTP client interface)
- `internal/channel/email_test.go` (new)
- `internal/handler/message_channel_test.go` (new)
- No API or frontend changes
