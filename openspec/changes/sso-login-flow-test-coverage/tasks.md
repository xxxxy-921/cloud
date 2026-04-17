## 1. Refactor SSOHandler for Injection

- [ ] 1.1 Define internal `oidcProvider` interface in `internal/handler/sso.go`
- [ ] 1.2 Add `getOIDCProvider`, `provisionExternalUser`, `generateTokenPair` fields to `SSOHandler`
- [ ] 1.3 Add `resolveOIDCProvider`, `resolveProvisionExternalUser`, `resolveGenerateTokenPair` helper methods with nil-fallback to real implementations
- [ ] 1.4 Update `InitiateSSO` to use `resolveOIDCProvider`
- [ ] 1.5 Update `SSOCallback` to use `resolveOIDCProvider`, `resolveProvisionExternalUser`, and `resolveGenerateTokenPair`
- [ ] 1.6 Run `go build -tags dev ./cmd/server/` and fix any compile errors

## 2. SSO State Manager Tests

- [ ] 2.1 Create `internal/pkg/identity/sso_state_test.go`
- [ ] 2.2 Add test: `Generate` returns non-empty state string
- [ ] 2.3 Add test: `Validate` returns correct `SourceID` and `CodeVerifier`
- [ ] 2.4 Add test: validating the same state twice returns error
- [ ] 2.5 Add test: expired state returns error (inject `nowFn` or use short TTL if refactored)
- [ ] 2.6 Run `go test ./internal/pkg/identity/...` and ensure all pass

## 3. CheckDomain Endpoint Tests

- [ ] 3.1 Create `internal/handler/sso_test.go` with test DB helper and `newSSOHandlerForTest` constructor
- [ ] 3.2 Add test: `CheckDomain` returns 200 with source info for a bound domain
- [ ] 3.3 Add test: `CheckDomain` returns 400 when `email` query param is missing
- [ ] 3.4 Add test: `CheckDomain` returns 400 for invalid email format
- [ ] 3.5 Add test: `CheckDomain` returns 404 when domain has no identity source

## 4. InitiateSSO Endpoint Tests

- [ ] 4.1 Add test: `InitiateSSO` returns 200 with `authUrl` and `state` for enabled OIDC source
- [ ] 4.2 Add test: `InitiateSSO` returns 400 for invalid `:id` parameter
- [ ] 4.3 Add test: `InitiateSSO` returns 404 when source does not exist
- [ ] 4.4 Add test: `InitiateSSO` returns 400 when source is disabled
- [ ] 4.5 Add test: `InitiateSSO` returns 400 when source type is not `oidc`

## 5. SSOCallback Endpoint Tests

- [ ] 5.1 Add test: `SSOCallback` returns 200 with token pair for successful new-user JIT provision
- [ ] 5.2 Add test: `SSOCallback` returns 200 for existing connection user
- [ ] 5.3 Add test: `SSOCallback` returns 400 when request body lacks `code` or `state`
- [ ] 5.4 Add test: `SSOCallback` returns 400 for invalid/expired state
- [ ] 5.5 Add test: `SSOCallback` returns 404 when identity source not found
- [ ] 5.6 Add test: `SSOCallback` returns 400 when source type is not `oidc`
- [ ] 5.7 Add test: `SSOCallback` returns 502 when `getOIDCProvider` fails (discovery error)
- [ ] 5.8 Add test: `SSOCallback` returns 502 when `ExchangeCode` fails
- [ ] 5.9 Add test: `SSOCallback` returns 502 when `VerifyIDToken` fails
- [ ] 5.10 Add test: `SSOCallback` returns 409 when `ProvisionExternalUser` returns `ErrEmailConflict`
- [ ] 5.11 Add test: `SSOCallback` returns 500 for other `ProvisionExternalUser` errors

## 6. Verification & Regression Check

- [ ] 6.1 Run `go test ./internal/handler/... ./internal/pkg/identity/...` and ensure all tests pass
- [ ] 6.2 Run `go build -tags dev ./cmd/server/` to confirm no compilation regressions
- [ ] 6.3 Run `go test ./...` to ensure no other tests are broken by the handler refactor
