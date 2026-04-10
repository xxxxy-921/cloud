## ADDED Requirements

### Requirement: User model
The system SHALL store users with username (unique), password hash (bcrypt), email, phone, avatar, role (enum: admin/user), and is_active flag. The User model SHALL embed BaseModel for ID, timestamps, and soft delete.

#### Scenario: Create user record
- **WHEN** a new user is created with username "alice" and role "user"
- **THEN** the system SHALL store a User record with bcrypt-hashed password, is_active=true, and auto-generated ID/timestamps

#### Scenario: Username uniqueness
- **WHEN** a user with username "alice" already exists and another user is created with the same username
- **THEN** the system SHALL return a unique constraint violation error

### Requirement: RefreshToken model
The system SHALL store refresh tokens with token (unique, opaque random string), user_id (FK to User), expires_at, and revoked flag. The RefreshToken model SHALL embed BaseModel.

#### Scenario: Create refresh token
- **WHEN** a user logs in successfully
- **THEN** the system SHALL create a RefreshToken record with a cryptographically random 32-byte base64url token, 7-day expiry, and revoked=false

#### Scenario: Revoke refresh token
- **WHEN** a user logs out or a token is rotated
- **THEN** the system SHALL set revoked=true on the corresponding RefreshToken record

### Requirement: JWT access token generation and validation
The system SHALL generate signed JWT access tokens using HS256 with configurable secret (JWT_SECRET env var). Claims SHALL include userId, role, iss ("metis"), sub, jti (UUID), iat, and exp (30 minutes).

#### Scenario: Generate access token
- **WHEN** a user logs in or refreshes their token
- **THEN** the system SHALL return a signed JWT with userId, role in custom claims and 30-minute expiry

#### Scenario: Validate access token
- **WHEN** a request includes a valid Bearer token in the Authorization header
- **THEN** the JWT middleware SHALL parse the token, validate signature and expiry, and set userId and userRole in the Gin context

#### Scenario: Expired access token
- **WHEN** a request includes an expired JWT
- **THEN** the middleware SHALL return 401 with message "token expired"

#### Scenario: Invalid access token
- **WHEN** a request includes a malformed or tampered JWT
- **THEN** the middleware SHALL return 401 with message "invalid token"

### Requirement: Login endpoint
The system SHALL provide `POST /api/v1/auth/login` accepting username and password in JSON body, returning a token pair on success.

#### Scenario: Successful login
- **WHEN** POST /api/v1/auth/login with valid username and password
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200

#### Scenario: Wrong password
- **WHEN** POST /api/v1/auth/login with valid username but wrong password
- **THEN** the system SHALL return 401 with message "invalid credentials"

#### Scenario: User not found
- **WHEN** POST /api/v1/auth/login with non-existent username
- **THEN** the system SHALL return 401 with message "invalid credentials" (same as wrong password, no information leak)

#### Scenario: Inactive user login
- **WHEN** POST /api/v1/auth/login for a user with is_active=false
- **THEN** the system SHALL return 401 with message "account disabled"

### Requirement: Logout endpoint
The system SHALL provide `POST /api/v1/auth/logout` (requires authentication) that revokes the user's current refresh token.

#### Scenario: Successful logout
- **WHEN** authenticated user POST /api/v1/auth/logout with their refresh token in body
- **THEN** the system SHALL revoke the refresh token in DB and return `{code: 0, message: "ok"}`

### Requirement: Token refresh endpoint
The system SHALL provide `POST /api/v1/auth/refresh` accepting a refresh token and returning a new token pair with refresh token rotation.

#### Scenario: Successful refresh
- **WHEN** POST /api/v1/auth/refresh with a valid, non-revoked, non-expired refresh token
- **THEN** the system SHALL revoke the old refresh token, create a new one, generate a new access token, and return the new token pair

#### Scenario: Expired refresh token
- **WHEN** POST /api/v1/auth/refresh with an expired refresh token
- **THEN** the system SHALL return 401 with message "refresh token expired"

#### Scenario: Revoked refresh token reuse
- **WHEN** POST /api/v1/auth/refresh with an already-revoked refresh token
- **THEN** the system SHALL revoke ALL refresh tokens for that user (theft detection) and return 401

### Requirement: Get current user endpoint
The system SHALL provide `GET /api/v1/auth/me` (requires authentication) returning the current user's profile (excluding password hash).

#### Scenario: Get own profile
- **WHEN** authenticated user GET /api/v1/auth/me
- **THEN** the system SHALL return `{code: 0, data: {id, username, email, phone, avatar, role, isActive, createdAt}}`

### Requirement: Change password endpoint
The system SHALL provide `PUT /api/v1/auth/password` (requires authentication) accepting old password and new password.

#### Scenario: Successful password change
- **WHEN** authenticated user PUT /api/v1/auth/password with correct old password and valid new password
- **THEN** the system SHALL update the password hash and revoke all refresh tokens for this user

#### Scenario: Wrong old password
- **WHEN** PUT /api/v1/auth/password with incorrect old password
- **THEN** the system SHALL return 400 with message "old password incorrect"

### Requirement: JWTAuth middleware
The system SHALL provide a Gin middleware that extracts Bearer token from Authorization header, validates the JWT, and sets userId and userRole in the Gin context. Unauthenticated requests SHALL receive 401.

#### Scenario: Missing Authorization header
- **WHEN** a protected route receives a request without Authorization header
- **THEN** the middleware SHALL return 401 with message "missing authorization header"

#### Scenario: Invalid Bearer format
- **WHEN** Authorization header is present but not in "Bearer <token>" format
- **THEN** the middleware SHALL return 401 with message "invalid authorization format"

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
