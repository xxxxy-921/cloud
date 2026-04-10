## MODIFIED Requirements

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

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success. The system SHALL record the client's IP address and User-Agent, and enforce concurrent session limits before issuing the token pair. For OAuth-only users (empty password), password login SHALL be rejected.

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

#### Scenario: OAuth-only user attempts password login
- **WHEN** POST /api/v1/auth/login with username of an OAuth-only user (empty password hash)
- **THEN** the system SHALL return 401 with message "invalid credentials"

### Requirement: Get current user endpoint
The system SHALL provide `GET /api/v1/auth/me` (requires authentication) returning the current user's profile (excluding password hash), including a `hasPassword` boolean field and a `connections` list of bound OAuth providers.

#### Scenario: Get own profile
- **WHEN** authenticated user GET /api/v1/auth/me
- **THEN** the system SHALL return `{code: 0, data: {id, username, email, phone, avatar, role, isActive, createdAt, hasPassword, connections: [{provider, externalName}]}}`

#### Scenario: OAuth-only user profile
- **WHEN** authenticated OAuth-only user GET /api/v1/auth/me
- **THEN** the response SHALL include hasPassword=false and connections containing the bound provider(s)
