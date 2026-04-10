## MODIFIED Requirements

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The login flow SHALL execute in this order: (1) find user, (2) check lockout, (3) verify captcha (if enabled), (4) verify password (with lockout tracking), (5) check is_active, (6) check 2FA, (7) check password expiry + enforce concurrent sessions, (8) issue token pair. The system SHALL record the client's IP address and User-Agent. On successful login, the system SHALL record an auth audit log with action="login_success". On failed login, the system SHALL record an auth audit log with action="login_failed" and level="warn".

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
- **THEN** the system SHALL return 423 with message "账户已锁定，请 N 分钟后重试" WITHOUT verifying password

#### Scenario: Captcha required but missing
- **WHEN** POST /api/v1/auth/login without captcha headers and captcha is enabled
- **THEN** the system SHALL return 400 with message "请输入验证码"

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
