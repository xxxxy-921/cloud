# Capability: totp-two-factor

## Purpose
TOTP-based two-factor authentication system with backup codes, supporting setup, verification during login, enforcement policies, and management endpoints.

## Requirements

### Requirement: TwoFactorSecret model
The system SHALL provide a `TwoFactorSecret` model with fields: ID (uint, PK), UserID (uint, FK unique), Secret (string, TOTP base32 secret), BackupCodes (string, JSON array of 10 hashed backup codes), CreatedAt, UpdatedAt. The model SHALL be registered in AutoMigrate.

#### Scenario: One secret per user
- **WHEN** a user enables 2FA
- **THEN** exactly one TwoFactorSecret record SHALL exist for that user (UserID unique constraint)

#### Scenario: Delete on disable
- **WHEN** a user disables 2FA
- **THEN** the TwoFactorSecret record SHALL be hard-deleted (not soft-deleted)

### Requirement: User model TwoFactorEnabled field
The User model SHALL include `TwoFactorEnabled` (bool, default false). This field SHALL be included in ToResponse output and user list API responses.

#### Scenario: Default state
- **WHEN** a new user is created
- **THEN** TwoFactorEnabled SHALL be false

#### Scenario: Enabled after setup
- **WHEN** a user completes 2FA setup and confirms with a valid TOTP code
- **THEN** TwoFactorEnabled SHALL be set to true

### Requirement: 2FA setup endpoint
The system SHALL provide `POST /api/v1/auth/2fa/setup` (requires authentication) that generates a new TOTP secret using github.com/pquerna/otp, returns the secret and QR code URI (otpauth://totp/Metis:{username}?secret=xxx&issuer=Metis), but does NOT save it yet. The secret SHALL be returned to the client for QR code display.

#### Scenario: Generate setup data
- **WHEN** an authenticated user calls POST /api/v1/auth/2fa/setup
- **THEN** the system SHALL return `{code: 0, data: {secret, qrUri}}` where secret is a base32 TOTP secret and qrUri is an otpauth:// URI

#### Scenario: Already enabled
- **WHEN** a user with TwoFactorEnabled=true calls POST /api/v1/auth/2fa/setup
- **THEN** the system SHALL return 400 with message "two-factor authentication already enabled"

### Requirement: 2FA confirm endpoint
The system SHALL provide `POST /api/v1/auth/2fa/confirm` (requires authentication) accepting `{secret, code}` that validates the TOTP code against the provided secret. On success, the system SHALL save the TwoFactorSecret record, generate 10 random backup codes (8-char alphanumeric each), store their bcrypt hashes, set TwoFactorEnabled=true, and return the plaintext backup codes.

#### Scenario: Successful confirmation
- **WHEN** POST /api/v1/auth/2fa/confirm with valid secret and matching 6-digit TOTP code
- **THEN** the system SHALL save the secret, generate 10 backup codes, set TwoFactorEnabled=true, and return `{code: 0, data: {backupCodes: ["xxxx-xxxx", ...]}}`

#### Scenario: Invalid TOTP code
- **WHEN** POST /api/v1/auth/2fa/confirm with valid secret but wrong TOTP code
- **THEN** the system SHALL return 400 with message "invalid verification code"

### Requirement: 2FA disable endpoint
The system SHALL provide `DELETE /api/v1/auth/2fa` (requires authentication) accepting `{code}` (TOTP code or backup code) for verification. On success, the system SHALL delete the TwoFactorSecret record and set TwoFactorEnabled=false.

#### Scenario: Disable with TOTP code
- **WHEN** DELETE /api/v1/auth/2fa with a valid TOTP code
- **THEN** the system SHALL set TwoFactorEnabled=false and delete the TwoFactorSecret record

#### Scenario: Disable with backup code
- **WHEN** DELETE /api/v1/auth/2fa with a valid backup code
- **THEN** the system SHALL set TwoFactorEnabled=false and delete the TwoFactorSecret record

#### Scenario: Invalid code
- **WHEN** DELETE /api/v1/auth/2fa with an invalid code
- **THEN** the system SHALL return 400 with message "invalid verification code"

### Requirement: 2FA login verification
The system SHALL provide `POST /api/v1/auth/2fa/login` accepting `{twoFactorToken, code}`. The twoFactorToken is a short-lived JWT (5 minutes, purpose="2fa") issued during login when the user has 2FA enabled. The code can be either a 6-digit TOTP code or an 8-char backup code. On success, the system SHALL issue a full token pair (accessToken + refreshToken).

#### Scenario: Valid TOTP code
- **WHEN** POST /api/v1/auth/2fa/login with valid twoFactorToken and correct 6-digit TOTP code
- **THEN** the system SHALL return `{code: 0, data: {accessToken, refreshToken, expiresIn}}` with HTTP 200

#### Scenario: Valid backup code
- **WHEN** POST /api/v1/auth/2fa/login with valid twoFactorToken and correct backup code
- **THEN** the system SHALL issue token pair AND invalidate the used backup code (set its hash to empty string)

#### Scenario: Invalid code
- **WHEN** POST /api/v1/auth/2fa/login with valid twoFactorToken but wrong code
- **THEN** the system SHALL return 401 with message "invalid verification code"

#### Scenario: Expired twoFactorToken
- **WHEN** POST /api/v1/auth/2fa/login with an expired twoFactorToken (>5 minutes)
- **THEN** the system SHALL return 401 with message "verification expired, please login again"

#### Scenario: All backup codes used
- **WHEN** a user has used all 10 backup codes
- **THEN** the user SHALL only be able to authenticate with TOTP codes from their authenticator app

### Requirement: 2FA enforcement
When SystemConfig key `security.require_two_factor` is "true", users without TwoFactorEnabled SHALL be required to set up 2FA. The login response for such users SHALL include `{requireTwoFactorSetup: true}` alongside the normal token pair. The frontend SHALL redirect these users to the 2FA setup page.

#### Scenario: Forced 2FA for user without it
- **WHEN** require_two_factor=true and a user without 2FA logs in successfully
- **THEN** the login response SHALL include `requireTwoFactorSetup: true` alongside the token pair

#### Scenario: Forced 2FA not applicable
- **WHEN** require_two_factor=true and a user with 2FA enabled logs in
- **THEN** the normal 2FA login flow SHALL proceed (twoFactorToken issued)

#### Scenario: Enforcement disabled
- **WHEN** require_two_factor=false (default)
- **THEN** 2FA SHALL be optional for all users

### Requirement: 2FA route whitelist
The 2FA setup and confirm endpoints SHALL be accessible when the user has a valid access token even if 2FA enforcement is active. The Casbin whitelist SHALL include: POST /api/v1/auth/2fa/setup, POST /api/v1/auth/2fa/confirm, POST /api/v1/auth/2fa/login.

#### Scenario: Setup endpoint accessible
- **WHEN** a user with requireTwoFactorSetup flag calls POST /api/v1/auth/2fa/setup
- **THEN** the endpoint SHALL be accessible (whitelisted from Casbin)
