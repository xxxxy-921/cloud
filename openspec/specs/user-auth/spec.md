# Capability: user-auth

## Purpose
JWT-based authentication system providing user login/logout, token generation/refresh, password management, and middleware for route protection with role-based access control.

## Requirements

### Requirement: User model
The system SHALL store users with username (unique, auto-generated for OAuth users in format `{provider}_{externalID}`), password hash (bcrypt, empty string for OAuth-only users), email, phone, avatar, role (FK to roles table), and is_active flag. The User model SHALL embed BaseModel for ID, timestamps, and soft delete. Username and Password SHALL be optional at the application level (Password may be empty for OAuth users; Username is auto-generated if not provided).

#### Scenario: Create user record
- **WHEN** a new user is created with username "alice" and role "user"
- **THEN** the system SHALL store a User record with bcrypt-hashed password, is_active=true, and auto-generated ID/timestamps

#### Scenario: Username uniqueness
- **WHEN** a user with username "alice" already exists and another user is created with the same username
- **THEN** the system SHALL return a unique constraint violation error

#### Scenario: Create OAuth user without password
- **WHEN** a new user is created via OAuth login with provider "github" and external ID "12345"
- **THEN** the system SHALL store a User record with username "github_12345", empty password hash, is_active=true, and avatar populated from the OAuth provider

#### Scenario: Check if user has password set
- **WHEN** the system needs to determine if a user can login with password
- **THEN** the system SHALL check if the password hash field is non-empty

### Requirement: RefreshToken model
The system SHALL store refresh tokens with token (unique, opaque random string), user_id (FK to User), expires_at, revoked flag, ip_address (string, login/refresh client IP), user_agent (string, login/refresh client User-Agent), last_seen_at (timestamp, updated on each token refresh), and access_token_jti (string, the jti claim of the most recently issued access token for this session). The RefreshToken model SHALL embed BaseModel.

#### Scenario: Create refresh token
- **WHEN** a user logs in successfully
- **THEN** the system SHALL create a RefreshToken record with a cryptographically random 32-byte base64url token, 7-day expiry, revoked=false, ip_address from request, user_agent from request header, last_seen_at=now, and access_token_jti set to the generated access token's jti claim

#### Scenario: Revoke refresh token
- **WHEN** a user logs out or a token is rotated
- **THEN** the system SHALL set revoked=true on the corresponding RefreshToken record

#### Scenario: Token refresh updates metadata
- **WHEN** a refresh token is successfully rotated via POST /api/v1/auth/refresh
- **THEN** the new RefreshToken record SHALL have ip_address and user_agent from the current request, last_seen_at=now, and access_token_jti set to the new access token's jti

### Requirement: JWT access token generation and validation
The system SHALL generate signed JWT access tokens using HS256 with configurable secret (JWT_SECRET env var). Claims SHALL include userId, role, iss ("metis"), sub, jti (UUID), iat, exp (30 minutes), passwordChangedAt (unix timestamp or 0 if nil), and forcePasswordReset (bool).

#### Scenario: Generate access token
- **WHEN** a user logs in or refreshes their token
- **THEN** the system SHALL return a signed JWT with userId, role, passwordChangedAt, forcePasswordReset in custom claims and 30-minute expiry

#### Scenario: Validate access token
- **WHEN** a request includes a valid Bearer token in the Authorization header
- **THEN** the JWT middleware SHALL parse the token, validate signature and expiry, and set userId, userRole, tokenJTI, passwordChangedAt, forcePasswordReset in the Gin context

#### Scenario: Expired access token
- **WHEN** a request includes an expired JWT
- **THEN** the middleware SHALL return 401 with message "token expired"

#### Scenario: Invalid access token
- **WHEN** a request includes a malformed or tampered JWT
- **THEN** the middleware SHALL return 401 with message "invalid token"

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The login flow SHALL execute in this order: (1) find user, (2) check lockout, (3) verify captcha (if enabled), (4) verify password (with lockout tracking), (5) check is_active, (6) check 2FA, (7) check password expiry + enforce concurrent sessions, (8) issue token pair. The system SHALL record the client's IP address and User-Agent. On successful login, the system SHALL record an auth audit log with action="login_success". On failed login, the system SHALL record an auth audit log with action="login_failed" and level="warn". **When local password verification fails and an ExternalAuthenticator is registered in IOC, the system SHALL call `ExternalAuthenticator.AuthenticateByPassword()` before returning "invalid credentials".** **When the user's email domain matches a ForceSso identity source, the system SHALL reject password login with 403.**

#### Scenario: Successful login
- **WHEN** POST /api/v1/auth/login with valid username and password, no lockout, valid captcha (if enabled), no 2FA
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200
- **AND** the system SHALL record an auth audit log with action="login_success"

#### Scenario: Wrong password
- **WHEN** POST /api/v1/auth/login with valid username but wrong password
- **THEN** the system SHALL increment FailedLoginAttempts, potentially trigger lockout, and return 401 with message "invalid credentials"
- **AND** the system SHALL record an auth audit log with action="login_failed", level="warn", detail=`{"reason":"invalid_password"}`

#### Scenario: User not found
- **WHEN** POST /api/v1/auth/login with non-existent username
- **THEN** the system SHALL return 401 with message "invalid credentials" (same as wrong password)

#### Scenario: Account locked
- **WHEN** POST /api/v1/auth/login for a locked user (LockedUntil > now)
- **THEN** the system SHALL return 423 with message "account locked, please try again in N minutes" WITHOUT verifying password

#### Scenario: Captcha required but missing
- **WHEN** POST /api/v1/auth/login without captcha headers and captcha is enabled
- **THEN** the system SHALL return 400 with message "please enter captcha"

#### Scenario: 2FA required
- **WHEN** POST /api/v1/auth/login with valid credentials and user has TwoFactorEnabled=true
- **THEN** the system SHALL return HTTP 202 with `{code: 0, data: {needsTwoFactor: true, twoFactorToken: "<jwt>"}}`

#### Scenario: 2FA enforcement for user without 2FA
- **WHEN** POST /api/v1/auth/login with valid credentials, user has TwoFactorEnabled=false, and require_two_factor=true
- **THEN** the system SHALL return normal token pair with additional field `requireTwoFactorSetup: true`

#### Scenario: Inactive user login
- **WHEN** POST /api/v1/auth/login for a user with is_active=false
- **THEN** the system SHALL return 401 with message "account disabled"

#### Scenario: OAuth-only user attempts password login
- **WHEN** POST /api/v1/auth/login with username of an OAuth-only user (empty password hash)
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: Concurrent session limit exceeded on login
- **WHEN** a user logs in and their active session count equals or exceeds the configured max_concurrent_sessions limit
- **THEN** the system SHALL revoke the least recently active sessions and blacklist their access tokens before creating the new token pair

#### Scenario: LDAP fallback via ExternalAuthenticator
- **WHEN** local password fails and ExternalAuthenticator is registered
- **THEN** SHALL call `AuthenticateByPassword()`. On success, return TokenPair. On failure, return 401.

#### Scenario: No ExternalAuthenticator registered
- **WHEN** local password fails and no ExternalAuthenticator exists in IOC
- **THEN** SHALL return 401 "invalid credentials" immediately (current behavior, no change)

#### Scenario: Forced SSO blocks password login
- **WHEN** user's email domain is bound to a ForceSso=true identity source
- **THEN** SHALL return 403 "this domain requires SSO login"

### Requirement: Logout endpoint
The system SHALL provide `POST /api/v1/auth/logout` (requires authentication) that revokes the user's current refresh token. On successful logout, the system SHALL record an auth audit log with action="logout".

#### Scenario: Successful logout
- **WHEN** authenticated user POST /api/v1/auth/logout with their refresh token in body
- **THEN** the system SHALL revoke the refresh token in DB and return `{code: 0, message: "ok"}`
- **AND** the system SHALL record an auth audit log with action="logout", user_id, and username

### Requirement: Token refresh endpoint
The system SHALL provide `POST /api/v1/auth/refresh` accepting a refresh token and returning a new token pair with refresh token rotation. The new access token SHALL include updated passwordChangedAt and forcePasswordReset claims read from the user's current DB state.

#### Scenario: Successful refresh
- **WHEN** POST /api/v1/auth/refresh with a valid, non-revoked, non-expired refresh token
- **THEN** the system SHALL revoke the old refresh token, create a new one, generate a new access token with current user state in claims, and return the new token pair

#### Scenario: Expired refresh token
- **WHEN** POST /api/v1/auth/refresh with an expired refresh token
- **THEN** the system SHALL return 401 with message "refresh token expired"

#### Scenario: Revoked refresh token reuse
- **WHEN** POST /api/v1/auth/refresh with an already-revoked refresh token
- **THEN** the system SHALL revoke ALL refresh tokens for that user (theft detection) and return 401

### Requirement: Get current user endpoint
The system SHALL provide `GET /api/v1/auth/me` (requires authentication) returning the current user's profile (excluding password hash), including a `hasPassword` boolean field and a `connections` list of bound OAuth providers.

#### Scenario: Get own profile
- **WHEN** authenticated user GET /api/v1/auth/me
- **THEN** the system SHALL return `{code: 0, data: {id, username, email, phone, avatar, role, isActive, createdAt, hasPassword, connections: [{provider, externalName}]}}`

#### Scenario: OAuth-only user profile
- **WHEN** authenticated OAuth-only user GET /api/v1/auth/me
- **THEN** the response SHALL include hasPassword=false and connections containing the bound provider(s)

### Requirement: Change password endpoint
The system SHALL provide `PUT /api/v1/auth/password` (requires authentication) accepting old password and new password. The new password SHALL be validated against the current password policy. On success, all refresh tokens for the user SHALL be revoked, their access tokens blacklisted, PasswordChangedAt SHALL be set to now(), and ForcePasswordReset SHALL be set to false.

#### Scenario: Successful password change
- **WHEN** authenticated user PUT /api/v1/auth/password with correct old password and policy-compliant new password
- **THEN** the system SHALL update the password hash, set PasswordChangedAt=now(), set ForcePasswordReset=false, revoke all refresh tokens, and blacklist all active access token JTIs

#### Scenario: Wrong old password
- **WHEN** PUT /api/v1/auth/password with incorrect old password
- **THEN** the system SHALL return 400 with message "old password incorrect"

#### Scenario: New password violates policy
- **WHEN** PUT /api/v1/auth/password with correct old password but new password fails policy validation
- **THEN** the system SHALL return 400 with policy violation messages

### Requirement: JWTAuth middleware
The system SHALL provide a Gin middleware that extracts Bearer token from Authorization header, validates the JWT, checks the token's jti against the in-memory blacklist, and sets userId, userRole, and tokenJTI in the Gin context. Unauthenticated or blacklisted requests SHALL receive 401.

#### Scenario: Missing Authorization header
- **WHEN** a protected route receives a request without Authorization header
- **THEN** the middleware SHALL return 401 with message "missing authorization header"

#### Scenario: Invalid Bearer format
- **WHEN** Authorization header is present but not in "Bearer <token>" format
- **THEN** the middleware SHALL return 401 with message "invalid authorization format"

#### Scenario: Blacklisted token
- **WHEN** a valid JWT is presented but its jti is in the blacklist
- **THEN** the middleware SHALL return 401 with message "session terminated"

#### Scenario: Valid token sets context
- **WHEN** a valid, non-blacklisted JWT is presented
- **THEN** the middleware SHALL set userId, userRole, and tokenJTI in the Gin context

### Requirement: RequireRole middleware
The system SHALL provide a Gin middleware that checks if the authenticated user's role matches one of the required roles.

#### Scenario: Authorized role
- **WHEN** an admin user accesses a route protected by RequireRole("admin")
- **THEN** the request SHALL proceed to the handler

#### Scenario: Unauthorized role
- **WHEN** a user with role "user" accesses a route protected by RequireRole("admin")
- **THEN** the middleware SHALL return 403 with message "forbidden"

### Requirement: JWT secret configuration
The system SHALL read JWT signing secret from `JWT_SECRET` environment variable. If not set, the system SHALL generate a random secret on startup and log a warning.

#### Scenario: Custom JWT secret
- **WHEN** `JWT_SECRET=my-secret-key` is set
- **THEN** the system SHALL use "my-secret-key" for JWT signing

#### Scenario: No JWT secret configured
- **WHEN** `JWT_SECRET` is not set
- **THEN** the system SHALL generate a random 32-byte secret, use it for the session, and log a warning that tokens will be invalidated on restart

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
