# Capability: oidc-auth

## Purpose
OIDC Authorization Code + PKCE flow implementation within the identity App, providing SSO login initiation, callback processing, JIT user provisioning, provider caching, and external identity tracking.

## Requirements

### Requirement: OIDC authorization flow（App 内实现）
The identity App SHALL implement OIDC Authorization Code + PKCE flow in `internal/app/identity/oidc.go`. The SSO endpoints SHALL be registered via the App's `Routes()` method.

#### Scenario: Initiate OIDC SSO
- **WHEN** GET /api/v1/auth/sso/:id/authorize for an enabled OIDC identity source
- **THEN** the App SHALL generate PKCE code_verifier/challenge, create an OAuth state token (storing source ID + code_verifier), build the authorization URL, and return `{authURL, state}`

#### Scenario: OIDC SSO for disabled source
- **WHEN** GET /api/v1/auth/sso/:id/authorize for a disabled identity source
- **THEN** SHALL return 400 "identity source not available"

#### Scenario: OIDC SSO for non-existent source
- **WHEN** GET /api/v1/auth/sso/:id/authorize for a non-existent ID
- **THEN** SHALL return 404 "identity source not found"

### Requirement: OIDC callback processing
The App SHALL provide `POST /api/v1/auth/sso/callback` accepting `{code, state}` that completes the OIDC flow and performs JIT user provisioning. The callback handler SHALL delegate user provisioning to the kernel `AuthService.ProvisionExternalUser` method instead of implementing provisioning logic directly. The handler SHALL NOT hold references to UserRepo, UserConnectionRepo, or RoleRepo.

#### Scenario: Successful OIDC callback (new user)
- **WHEN** POST /api/v1/auth/sso/callback with valid code/state, no existing local account
- **THEN** SHALL exchange code for tokens, validate ID token via JWKS, extract claims, call `AuthService.ProvisionExternalUser` with provider="oidc_{sourceId}", then call `AuthService.GenerateTokenPair` and return the TokenPair

#### Scenario: Successful OIDC callback (existing user, link strategy)
- **WHEN** OIDC user's email matches existing local user and conflict strategy is "link"
- **THEN** SHALL call `AuthService.ProvisionExternalUser` which links the identity to the existing user, then return a TokenPair

#### Scenario: Email conflict with "fail" strategy
- **WHEN** OIDC user's email matches existing local user and conflict strategy is "fail"
- **THEN** SHALL return 409 "email already registered"

#### Scenario: Returning OIDC user
- **WHEN** OIDC user has previously logged in via this source
- **THEN** SHALL find existing user via `AuthService.ProvisionExternalUser`, update attributes if changed, return TokenPair

#### Scenario: Invalid state token
- **WHEN** POST /api/v1/auth/sso/callback with invalid/expired state
- **THEN** SHALL return 400 "invalid or expired state"

#### Scenario: Identity source not found error handling
- **WHEN** the identity source ID does not exist in the database
- **THEN** the Service layer SHALL translate the database error to a domain sentinel error, and the Handler SHALL use `errors.Is()` to match it (SHALL NOT check `gorm.ErrRecordNotFound` directly)

### Requirement: OIDC provider caching
The App SHALL cache OIDC provider metadata (discovery document, JWKS) per identity source with 1-hour TTL.

#### Scenario: Cached provider reuse
- **WHEN** a second SSO request arrives within 1 hour
- **THEN** SHALL use cached metadata without HTTP calls

### Requirement: SSO state management
The App SHALL extend the existing kernel `StateManager` (resolved via IOC) to support SSO states storing source ID + PKCE code_verifier. SSO states SHALL expire after 10 minutes.

#### Scenario: Generate SSO state
- **WHEN** user initiates OIDC SSO for identity source ID 3
- **THEN** SHALL generate a state token storing `{sourceID: 3, codeVerifier: "..."}`

### Requirement: External identity tracking via UserConnection
The App SHALL track external identities using the kernel's `UserConnection` table (resolved via IOC). OIDC uses Provider `"oidc_{sourceId}"` with ExternalID as the OIDC `sub` claim.

#### Scenario: Store OIDC identity on first login
- **WHEN** user first logs in via OIDC source ID 3 with sub "user123"
- **THEN** SHALL create UserConnection with Provider="oidc_3", ExternalID="user123"

#### Scenario: Find existing user by OIDC identity
- **WHEN** returning user logs in with sub "user123" via source ID 3
- **THEN** SHALL query UserConnection by Provider="oidc_3", ExternalID="user123"

### Requirement: Token pair generation via kernel AuthService
The App SHALL resolve the kernel's `AuthService` via IOC and call its `GenerateTokenPair` method (or equivalent) to issue JWT access + refresh tokens after successful SSO/LDAP authentication. The kernel AuthService SHALL expose this method for App use.

#### Scenario: App generates tokens after SSO
- **WHEN** OIDC callback successfully identifies/creates a user
- **THEN** the App SHALL call kernel AuthService to generate and return a TokenPair
