## ADDED Requirements

### Requirement: Password validation function
The system SHALL provide a `ValidatePassword(plain string, policy PasswordPolicy) []string` function that checks a password against the configured policy and returns a list of violation messages (empty = valid). The PasswordPolicy struct SHALL include MinLength (int), RequireUpper (bool), RequireLower (bool), RequireNumber (bool), RequireSpecial (bool).

#### Scenario: Password meets all requirements
- **WHEN** ValidatePassword is called with "Passw0rd!" and policy {MinLength:8, RequireUpper:true, RequireLower:true, RequireNumber:true, RequireSpecial:true}
- **THEN** the function SHALL return an empty slice

#### Scenario: Password too short
- **WHEN** ValidatePassword is called with "Ab1!" and policy {MinLength:8}
- **THEN** the function SHALL return a slice containing "密码长度至少 8 位"

#### Scenario: Missing uppercase
- **WHEN** ValidatePassword is called with "password1!" and policy {RequireUpper:true}
- **THEN** the function SHALL return a slice containing "密码需要包含大写字母"

#### Scenario: Missing lowercase
- **WHEN** ValidatePassword is called with "PASSWORD1!" and policy {RequireLower:true}
- **THEN** the function SHALL return a slice containing "密码需要包含小写字母"

#### Scenario: Missing number
- **WHEN** ValidatePassword is called with "Password!" and policy {RequireNumber:true}
- **THEN** the function SHALL return a slice containing "密码需要包含数字"

#### Scenario: Missing special character
- **WHEN** ValidatePassword is called with "Password1" and policy {RequireSpecial:true}
- **THEN** the function SHALL return a slice containing "密码需要包含特殊字符"

#### Scenario: Multiple violations
- **WHEN** ValidatePassword is called with "abc" and policy {MinLength:8, RequireUpper:true, RequireNumber:true}
- **THEN** the function SHALL return a slice with three messages

### Requirement: Password policy configuration
The system SHALL read password policy from SystemConfig keys: `security.password_min_length` (default 8), `security.password_require_upper` (default "false"), `security.password_require_lower` (default "false"), `security.password_require_number` (default "false"), `security.password_require_special` (default "false"), `security.password_expiry_days` (default "0", meaning never expire).

#### Scenario: Read default policy
- **WHEN** no password policy keys exist in SystemConfig
- **THEN** the system SHALL use defaults: min length 8, no complexity requirements, no expiry

#### Scenario: Custom policy
- **WHEN** SystemConfig contains `security.password_min_length`="12" and `security.password_require_upper`="true"
- **THEN** the system SHALL enforce minimum 12 characters with uppercase requirement

### Requirement: Password policy enforcement points
The system SHALL validate passwords against the configured policy at: user creation (POST /api/v1/users), password change (PUT /api/v1/auth/password), user registration (POST /api/v1/auth/register), and admin password reset. If validation fails, the system SHALL return 400 with the list of violation messages.

#### Scenario: Create user with weak password
- **WHEN** POST /api/v1/users with password "123" and policy requires min length 8
- **THEN** the system SHALL return 400 with message "密码长度至少 8 位"

#### Scenario: Change password with weak new password
- **WHEN** PUT /api/v1/auth/password with correct old password but new password "abc" and policy requires min length 8
- **THEN** the system SHALL return 400 with violation messages before changing the password

### Requirement: Password expiry check middleware
The system SHALL provide a middleware that reads `passwordChangedAt` and `forcePasswordReset` from JWT claims, and if the password has expired (passwordChangedAt + expiry_days < now) or forcePasswordReset is true, returns HTTP 409 with `{"code": -1, "message": "password expired"}`. The middleware SHALL whitelist: PUT /api/v1/auth/password, POST /api/v1/auth/logout, POST /api/v1/auth/refresh.

#### Scenario: Password expired
- **WHEN** a request carries a JWT with passwordChangedAt=2025-01-01 and security.password_expiry_days=90 and current date is 2025-06-01
- **THEN** the middleware SHALL return 409 with message "password expired"

#### Scenario: Force password reset
- **WHEN** a request carries a JWT with forcePasswordReset=true
- **THEN** the middleware SHALL return 409 with message "password expired"

#### Scenario: Password not expired
- **WHEN** a request carries a JWT with passwordChangedAt=2025-05-01 and security.password_expiry_days=90 and current date is 2025-06-01
- **THEN** the middleware SHALL allow the request to proceed

#### Scenario: Expiry disabled
- **WHEN** security.password_expiry_days is 0
- **THEN** the middleware SHALL skip the expiry check (only check forcePasswordReset)

#### Scenario: Whitelisted routes bypass
- **WHEN** a request to PUT /api/v1/auth/password carries an expired password JWT
- **THEN** the middleware SHALL allow the request to proceed

### Requirement: User model password expiry fields
The User model SHALL include `PasswordChangedAt` (*time.Time, set to now() on user creation and password change) and `ForcePasswordReset` (bool, default false, set by admin). The ToResponse method SHALL include these fields.

#### Scenario: New user creation
- **WHEN** a new user is created with a password
- **THEN** PasswordChangedAt SHALL be set to current time

#### Scenario: Password change updates timestamp
- **WHEN** a user changes their password
- **THEN** PasswordChangedAt SHALL be updated to current time and ForcePasswordReset SHALL be set to false

#### Scenario: Admin forces password reset
- **WHEN** an admin sets ForcePasswordReset=true on a user via PUT /api/v1/users/:id
- **THEN** the user's ForcePasswordReset field SHALL be set to true

#### Scenario: OAuth user has no password timestamp
- **WHEN** an OAuth-only user is created
- **THEN** PasswordChangedAt SHALL be nil (password expiry does not apply)
