# Capability: user-registration

## Purpose
User self-registration system allowing new users to create accounts when registration is open, with configurable default role assignment.

## Requirements

### Requirement: Registration endpoint
The system SHALL provide `POST /api/v1/auth/register` accepting `{username, password, email (optional)}`. The endpoint SHALL: check that `security.registration_open` is "true" (else return 403), validate password against the current password policy, create the user with is_active=true, assign the role from `security.default_role_code` if non-empty, and auto-login by returning a token pair.

#### Scenario: Successful registration
- **WHEN** POST /api/v1/auth/register with valid data and registration_open=true
- **THEN** the system SHALL create the user, assign default role (if configured), and return `{code: 0, data: {accessToken, refreshToken, expiresIn}}`

#### Scenario: Registration closed
- **WHEN** POST /api/v1/auth/register and registration_open=false
- **THEN** the system SHALL return 403 with message "registration not open"

#### Scenario: Username already exists
- **WHEN** POST /api/v1/auth/register with an existing username
- **THEN** the system SHALL return 400 with message "username already exists"

#### Scenario: Password policy violation
- **WHEN** POST /api/v1/auth/register with password "123" and policy requires min length 8
- **THEN** the system SHALL return 400 with password violation messages

#### Scenario: Default role assignment
- **WHEN** registration succeeds and security.default_role_code="user"
- **THEN** the created user SHALL have role code "user"

#### Scenario: No default role
- **WHEN** registration succeeds and security.default_role_code is empty
- **THEN** the created user SHALL have no role assigned

### Requirement: Registration configuration
The system SHALL read registration settings from SystemConfig: `security.registration_open` (default "false") and `security.default_role_code` (default "").

#### Scenario: Default configuration
- **WHEN** no registration config keys exist in SystemConfig
- **THEN** registration SHALL be closed (open=false) and no default role assigned

### Requirement: Registration route whitelist
The registration endpoint `POST /api/v1/auth/register` SHALL be added to both the JWT middleware whitelist (no auth required) and the Casbin whitelist.

#### Scenario: Unauthenticated access
- **WHEN** an unauthenticated user calls POST /api/v1/auth/register
- **THEN** the request SHALL not require a JWT token

### Requirement: Registration public config endpoint
The existing public configuration endpoint (or a new `GET /api/v1/auth/registration-status`) SHALL return `{registrationOpen: bool}` without authentication, so the login page can conditionally show the registration link.

#### Scenario: Check registration status
- **WHEN** GET /api/v1/auth/registration-status is called without authentication
- **THEN** the system SHALL return `{code: 0, data: {registrationOpen: true/false}}`
