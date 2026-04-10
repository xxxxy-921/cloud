## MODIFIED Requirements

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

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The system SHALL record the client's IP address and User-Agent, and enforce concurrent session limits before issuing the token pair.

#### Scenario: Successful login
- **WHEN** POST /api/v1/auth/login with valid username and password
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200, and the created refresh token SHALL include ipAddress and userAgent from the request

#### Scenario: Wrong password
- **WHEN** POST /api/v1/auth/login with valid username but wrong password
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: User not found
- **WHEN** POST /api/v1/auth/login with non-existent username
- **THEN** the system SHALL return 401 with message "invalid credentials" (same as wrong password, no information leak)

#### Scenario: Inactive user login
- **WHEN** POST /api/v1/auth/login for a user with is_active=false
- **THEN** the system SHALL return 401 with message "account disabled"

#### Scenario: Concurrent session limit exceeded on login
- **WHEN** a user logs in and their active session count equals or exceeds the configured max_concurrent_sessions limit
- **THEN** the system SHALL revoke the least recently active sessions and blacklist their access tokens before creating the new token pair

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

### Requirement: Change password endpoint
The system SHALL provide `PUT /api/v1/auth/password` (requires authentication) accepting old password and new password. On success, all refresh tokens for the user SHALL be revoked and their access tokens blacklisted.

#### Scenario: Successful password change
- **WHEN** authenticated user PUT /api/v1/auth/password with correct old password and valid new password
- **THEN** the system SHALL update the password hash, revoke all refresh tokens for this user, and blacklist all their active access token JTIs

#### Scenario: Wrong old password
- **WHEN** PUT /api/v1/auth/password with incorrect old password
- **THEN** the system SHALL return 400 with message "old password incorrect"
