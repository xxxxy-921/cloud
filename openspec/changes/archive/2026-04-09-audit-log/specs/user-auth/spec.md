## MODIFIED Requirements

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The system SHALL record the client's IP address and User-Agent, and enforce concurrent session limits before issuing the token pair. For OAuth-only users (empty password), password login SHALL be rejected. On successful login, the system SHALL record an auth audit log with action="login_success". On failed login (wrong password, user not found, or inactive user), the system SHALL record an auth audit log with action="login_failed" and level="warn".

#### Scenario: Successful login
- **WHEN** POST /api/v1/auth/login with valid username and password
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200, and the created refresh token SHALL include ipAddress and userAgent from the request
- **AND** the system SHALL record an auth audit log with action="login_success", user_id, username, IP, and user agent

#### Scenario: Wrong password
- **WHEN** POST /api/v1/auth/login with valid username but wrong password
- **THEN** the system SHALL return 401 with message "invalid credentials"
- **AND** the system SHALL record an auth audit log with action="login_failed", level="warn", user_id of the target user, and detail=`{"reason":"invalid_password"}`

#### Scenario: User not found
- **WHEN** POST /api/v1/auth/login with non-existent username
- **THEN** the system SHALL return 401 with message "invalid credentials" (same as wrong password, no information leak)
- **AND** the system SHALL record an auth audit log with action="login_failed", level="warn", user_id=NULL, and detail=`{"reason":"user_not_found"}`

#### Scenario: Inactive user login
- **WHEN** POST /api/v1/auth/login for a user with is_active=false
- **THEN** the system SHALL return 401 with message "account disabled"
- **AND** the system SHALL record an auth audit log with action="login_failed", level="warn", and detail=`{"reason":"account_disabled"}`

#### Scenario: Concurrent session limit exceeded on login
- **WHEN** a user logs in and their active session count equals or exceeds the configured max_concurrent_sessions limit
- **THEN** the system SHALL revoke the least recently active sessions and blacklist their access tokens before creating the new token pair

#### Scenario: OAuth-only user attempts password login
- **WHEN** POST /api/v1/auth/login with username of an OAuth-only user (empty password hash)
- **THEN** the system SHALL return 401 with message "invalid credentials"

### Requirement: Logout endpoint
The system SHALL provide `POST /api/v1/auth/logout` (requires authentication) that revokes the user's current refresh token. On successful logout, the system SHALL record an auth audit log with action="logout".

#### Scenario: Successful logout
- **WHEN** authenticated user POST /api/v1/auth/logout with their refresh token in body
- **THEN** the system SHALL revoke the refresh token in DB and return `{code: 0, message: "ok"}`
- **AND** the system SHALL record an auth audit log with action="logout", user_id, and username
