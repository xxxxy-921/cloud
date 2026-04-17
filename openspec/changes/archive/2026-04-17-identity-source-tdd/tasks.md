## 1. Crypto Utility Tests

- [x] 1.1 Create `internal/pkg/token/crypto_test.go` with in-memory SQLite helper including `SystemConfig` AutoMigrate
- [x] 1.2 Add round-trip encrypt/decrypt test and key auto-generation test
- [x] 1.3 Add error-path tests for invalid hex and tampered ciphertext

## 2. Model Layer Tests

- [x] 2.1 Create `internal/model/identity_source_test.go`
- [x] 2.2 Add tests for `ToResponse()` OIDC secret masking, LDAP secret masking, and unknown-type passthrough
- [x] 2.3 Add test for `DefaultLDAPAttributeMapping()`

## 3. Repository Layer Tests

- [x] 3.1 Create `internal/repository/identity_source_test.go` with test DB helper and seed helpers
- [x] 3.2 Add tests for `List` ordering, `FindByID`, `FindByDomain` (match / case-insensitive / trim / no-match)
- [x] 3.3 Add tests for `CheckDomainConflict` (conflict / self-update / empty domains)
- [x] 3.4 Add tests for `Create`, `Update`, `Delete`, and `Toggle`

## 4. Service Layer Refactor & Tests

- [x] 4.1 Add injectable `testOIDC`, `testLDAP`, and `ldapAuth` fields to `IdentitySourceService` (defaults to real implementations)
- [x] 4.2 Create `internal/service/identity_source_test.go` with test DB helper and stub implementations
- [x] 4.3 Add tests for `Create` (OIDC + LDAP success, unsupported type, domain conflict)
- [x] 4.4 Add tests for `Update` (success, masked secret preservation, not-found) and `Delete`/`Toggle` (success + not-found)
- [x] 4.5 Add tests for `TestConnection` (OIDC success/failure, LDAP success/failure, not-found)
- [x] 4.6 Add tests for `AuthenticateByPassword` (success, all sources fail), `CheckDomain`, `IsForcedSSO`, and `ExtractDomain`

## 5. Handler Layer Tests

- [x] 5.1 Create `internal/handler/identity_source_test.go` with Gin + httptest scaffolding
- [x] 5.2 Add tests for `List` and `Create` (200 + 400 + 409)
- [x] 5.3 Add tests for `Update` (200 + 404) and `Delete` (200 + 404)
- [x] 5.4 Add tests for `Toggle` (200) and `TestConnection` (200 with success/message payload)

## 6. Verification

- [x] 6.1 Run `go test ./internal/pkg/token/... ./internal/model/... ./internal/repository/... ./internal/service/... ./internal/handler/...` and ensure all tests pass
- [x] 6.2 Run `go build -tags dev ./cmd/server/` to confirm no compilation regressions
