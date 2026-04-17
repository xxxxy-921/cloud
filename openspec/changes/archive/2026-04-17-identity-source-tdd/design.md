## Context

The identity source feature supports OIDC and LDAP authentication backends, including encrypted configuration storage (`token.Encrypt/Decrypt`), domain-based routing, and real-time connectivity tests against external servers. The implementation spans `internal/model`, `internal/repository`, `internal/service`, `internal/handler`, `internal/pkg/identity`, and `internal/pkg/token`. Currently there are **zero unit tests** across all of these layers, making refactors risky and regressions invisible.

## Goals / Non-Goals

**Goals:**
- Achieve high unit-test coverage for identity source backend logic
- Make `IdentitySourceService` testable without requiring real OIDC/LDAP servers
- Verify `token.Encrypt/Decrypt` round-trip behavior using an in-memory database
- Cover domain conflict detection, secret masking, and password-preservation logic
- Verify HTTP handler boundaries (status codes, JSON shapes, audit fields)

**Non-Goals:**
- No changes to frontend React code
- No changes to public API contracts
- No integration tests against real LDAP/OIDC servers
- No changes to the `sync.Once`-based `token.GetEncryptionKey` design

## Decisions

### 1. Use real `token.Encrypt/Decrypt` with in-memory SQLite + `SystemConfig`
**Rationale:** `token.GetEncryptionKey` relies on a package-level `sync.Once` that initializes from `SystemConfig`. Rather than refactor the crypto layer (which would touch many files), tests will `AutoMigrate(&model.SystemConfig{})` and let `Encrypt` auto-generate the key on first call. This exercises the real code path with zero network dependencies.

**Alternative considered:** Abstract `Encrypt/Decrypt` behind an interface injected into `IdentitySourceService`. Rejected because it adds boilerhead for marginal gain and strays from the existing codebase style.

### 2. Inject external network/auth functions into `IdentitySourceService`
**Rationale:** `TestConnection` calls `identity.TestOIDCDiscovery` and `identity.TestLDAPConnection`. `AuthenticateByPassword` calls `identity.LDAPAuthenticate`. These perform real network I/O. We will add three function fields to the service:
- `testOIDC func(ctx context.Context, issuerURL string) error`
- `testLDAP func(cfg *model.LDAPConfig) error`
- `ldapAuth func(cfg *model.LDAPConfig, username, password string) (*identity.LDAPAuthResult, error)`

The constructor defaults them to the real implementations; tests override with stubs.

### 3. Use in-memory SQLite for repository and service tests
**Rationale:** This is the established Metis pattern (`notification_test.go`, `message_channel_test.go`). It gives us real GORM behavior (sorting, soft deletes, counts, transactions) with fast isolation.

### 4. Handler tests use `gin.New()` + `httptest`
**Rationale:** Handler tests verify routing, binding, status codes, and response JSON. We construct a minimal Gin engine and mount the handler, using the production service wired to a memory DB but with network stubs injected.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `sync.Once` in `token.GetEncryptionKey` makes parallel tests fragile | Tests that touch `Encrypt/Decrypt` will run sequentially within their package. We do not add parallel test execution for the identity source package. |
| Masked secret preservation logic (`••••••`) is string-comparison based | Add explicit tests for OIDC `ClientSecret` and LDAP `BindPassword` preservation paths, plus the "new secret provided" path. |
| Domain conflict logic involves string parsing (split/trim/lower) | Repository tests will cover exact match, case insensitivity, whitespace trimming, and partial-match rejection. |
| Stubbing `ldapAuth` might miss integration issues | Real LDAP integration is explicitly out of scope for unit tests; manual smoke tests remain the responsibility of ops/deployment. |
