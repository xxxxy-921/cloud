## ADDED Requirements

### Requirement: ExternalAuthenticator interface（内核新增）
The kernel SHALL define an `ExternalAuthenticator` interface in `internal/service/external_auth.go` with three methods: `AuthenticateByPassword(username, password string) (*model.User, error)`, `CheckDomain(email string) (*DomainCheckResult, error)`, and `IsForcedSSO(email string) bool`. DomainCheckResult SHALL contain SourceID, Name, Type, ForceSso.

#### Scenario: Interface defined in kernel
- **WHEN** the kernel is compiled
- **THEN** `ExternalAuthenticator` interface SHALL exist in `internal/service/` package

#### Scenario: No implementation registered
- **WHEN** no App registers an ExternalAuthenticator and AuthService tries to resolve it
- **THEN** the resolution SHALL fail gracefully (nil) and AuthService SHALL skip external auth

### Requirement: Casbin whitelist for SSO and domain-check
The kernel SHALL add `/api/v1/auth/sso` and `/api/v1/auth/check-domain` to the Casbin whitelist prefixes in `internal/middleware/casbin.go`.

#### Scenario: SSO endpoint is public
- **WHEN** unauthenticated user accesses GET /api/v1/auth/sso/3/authorize
- **THEN** Casbin middleware SHALL skip permission checking

#### Scenario: Domain check is public
- **WHEN** unauthenticated user accesses GET /api/v1/auth/check-domain?email=test@acme.com
- **THEN** Casbin middleware SHALL skip permission checking

### Requirement: Expose GenerateTokenPair for App use
The kernel AuthService SHALL expose its `GenerateTokenPair(user *model.User, ip, ua string) (*TokenPair, error)` method (currently private `generateTokenPair`) as a public method so that Apps can issue tokens after external authentication.

#### Scenario: App calls GenerateTokenPair
- **WHEN** the identity App successfully authenticates a user via OIDC/LDAP
- **THEN** it SHALL call `authService.GenerateTokenPair(user, ip, ua)` to get a TokenPair

## MODIFIED Requirements

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The system SHALL record the client's IP address and User-Agent, and enforce concurrent session limits before issuing the token pair. For OAuth-only users (empty password), password login SHALL be rejected. **When local password verification fails and an ExternalAuthenticator is registered in IOC, the system SHALL call `ExternalAuthenticator.AuthenticateByPassword()` before returning "invalid credentials".** **When the user's email domain matches a ForceSso identity source, the system SHALL reject password login with 403.**

#### Scenario: Successful login
- **WHEN** POST /api/v1/auth/login with valid username and password
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200

#### Scenario: Wrong password
- **WHEN** POST /api/v1/auth/login with valid username but wrong password
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: User not found
- **WHEN** POST /api/v1/auth/login with non-existent username
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: Inactive user login
- **WHEN** POST /api/v1/auth/login for a user with is_active=false
- **THEN** the system SHALL return 401 with message "account disabled"

#### Scenario: OAuth-only user attempts password login
- **WHEN** POST /api/v1/auth/login with username of an OAuth-only user
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: LDAP fallback via ExternalAuthenticator
- **WHEN** local password fails and ExternalAuthenticator is registered
- **THEN** SHALL call `AuthenticateByPassword()`. On success, return TokenPair. On failure, return 401.

#### Scenario: No ExternalAuthenticator registered
- **WHEN** local password fails and no ExternalAuthenticator exists in IOC
- **THEN** SHALL return 401 "invalid credentials" immediately (current behavior, no change)

#### Scenario: Forced SSO blocks password login
- **WHEN** user's email domain is bound to a ForceSso=true identity source
- **THEN** SHALL return 403 "this domain requires SSO login"

### Requirement: Domain check endpoint（App handler, 内核路由支持）
The identity App SHALL register `GET /api/v1/auth/check-domain?email=xxx` via its Routes() method. This endpoint is public (Casbin whitelisted by kernel).

#### Scenario: Domain matches OIDC source
- **WHEN** GET /api/v1/auth/check-domain?email=john@acme.com and source matches
- **THEN** SHALL return `{code: 0, data: {id: 3, name: "Okta SSO", type: "oidc", forceSso: true}}`

#### Scenario: No match
- **WHEN** no identity source matches the email domain
- **THEN** SHALL return `{code: 0, data: null}`

#### Scenario: No identity App loaded (endpoint doesn't exist)
- **WHEN** identity App is not loaded and frontend calls check-domain
- **THEN** SHALL return 404 (no handler registered), frontend handles gracefully
