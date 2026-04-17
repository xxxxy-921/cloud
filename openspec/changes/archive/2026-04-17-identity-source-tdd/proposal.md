## Why

The identity source subsystem (`internal/service/identity_source.go`, `internal/repository/identity_source.go`, `internal/pkg/identity/`) manages OIDC/LDAP configuration, domain binding, secret encryption, and external authentication. Despite being a security-critical feature, it currently has **zero automated test coverage**. Adding comprehensive tests now is essential before any future work on SSO enhancements or new identity protocols.

## What Changes

- Add unit tests for `internal/pkg/token/crypto.go` (AES-256-GCM round-trip, key auto-generation, error paths)
- Add model tests for `IdentitySource.ToResponse()` (secret masking) and `DefaultLDAPAttributeMapping()`
- Add repository tests for `IdentitySourceRepo` (CRUD, domain lookup, domain conflict detection)
- Refactor `IdentitySourceService` to inject external network dependencies (`TestOIDCDiscovery`, `TestLDAPConnection`, `LDAPAuthenticate`) for testability
- Add service tests covering CRUD, config encryption, masked-secret preservation, `TestConnection`, `AuthenticateByPassword`, and domain utilities
- Add handler integration tests for `/api/v1/identity-sources/*` endpoints

## Capabilities

### New Capabilities
- `identity-source-test-coverage`: Comprehensive unit-test coverage for the identity source model, repository, service, crypto utilities, and HTTP handler layers.

### Modified Capabilities
- *(none — this change does not alter spec-level behavior, only adds tests and minimal internal refactor for testability)*

## Impact

- `internal/pkg/token/crypto_test.go` (new)
- `internal/model/identity_source_test.go` (new)
- `internal/repository/identity_source_test.go` (new)
- `internal/service/identity_source.go` (minor refactor: inject network/auth functions)
- `internal/service/identity_source_test.go` (new)
- `internal/handler/identity_source_test.go` (new)
- No API or frontend changes
