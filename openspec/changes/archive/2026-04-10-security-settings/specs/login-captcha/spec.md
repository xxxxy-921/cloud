## ADDED Requirements

### Requirement: Captcha store
The system SHALL provide an in-memory captcha store using sync.Map with entries keyed by UUID and values containing the answer string and expiration time (5 minutes from creation). A background goroutine SHALL clean up expired entries every minute.

#### Scenario: Store captcha
- **WHEN** a new captcha is generated
- **THEN** the system SHALL store {answer, expiresAt: now+5min} keyed by a generated UUID

#### Scenario: Verify captcha
- **WHEN** Verify(id, answer) is called with a valid ID and correct answer (case-insensitive)
- **THEN** the system SHALL return true and delete the entry (one-time use)

#### Scenario: Verify wrong answer
- **WHEN** Verify(id, answer) is called with a wrong answer
- **THEN** the system SHALL return false and delete the entry (prevent brute-force)

#### Scenario: Verify expired captcha
- **WHEN** Verify(id, answer) is called with an expired captcha ID
- **THEN** the system SHALL return false

#### Scenario: Cleanup expired entries
- **WHEN** the cleanup goroutine runs
- **THEN** all entries where expiresAt < now SHALL be removed

### Requirement: Captcha generation endpoint
The system SHALL provide `GET /api/v1/captcha` (no authentication required) that returns captcha data based on the configured provider. When `security.captcha_provider` is "none", it SHALL return `{enabled: false}`. When "image", it SHALL generate a captcha using github.com/mojocn/base64Captcha and return `{enabled: true, id: "<uuid>", image: "<base64-png>"}`.

#### Scenario: Captcha disabled
- **WHEN** GET /api/v1/captcha and captcha_provider="none"
- **THEN** the system SHALL return `{code: 0, data: {enabled: false}}`

#### Scenario: Image captcha
- **WHEN** GET /api/v1/captcha and captcha_provider="image"
- **THEN** the system SHALL return `{code: 0, data: {enabled: true, id: "uuid", image: "data:image/png;base64,xxx"}}`

### Requirement: Captcha verification in login
When `security.captcha_provider` is not "none", the login endpoint SHALL require `X-Captcha-Id` and `X-Captcha-Answer` headers. Verification SHALL happen AFTER lockout check but BEFORE password verification. Missing or invalid captcha SHALL return 400.

#### Scenario: Missing captcha headers
- **WHEN** POST /api/v1/auth/login without X-Captcha-Id header and captcha is enabled
- **THEN** the system SHALL return 400 with message "请输入验证码"

#### Scenario: Invalid captcha
- **WHEN** POST /api/v1/auth/login with wrong captcha answer
- **THEN** the system SHALL return 400 with message "验证码错误"

#### Scenario: Valid captcha
- **WHEN** POST /api/v1/auth/login with correct captcha ID and answer
- **THEN** the captcha SHALL be consumed and login flow SHALL proceed to password verification

#### Scenario: Captcha disabled skips check
- **WHEN** POST /api/v1/auth/login and captcha_provider="none"
- **THEN** the system SHALL skip captcha verification entirely

### Requirement: Captcha configuration
The system SHALL read captcha settings from SystemConfig: `security.captcha_provider` (default "none", valid values: "none", "image").

#### Scenario: Default configuration
- **WHEN** no captcha config exists in SystemConfig
- **THEN** captcha SHALL be disabled (provider="none")

### Requirement: Captcha route whitelist
The captcha endpoint `GET /api/v1/captcha` SHALL be added to the JWT middleware whitelist and Casbin whitelist (no auth required).

#### Scenario: Public access
- **WHEN** an unauthenticated user calls GET /api/v1/captcha
- **THEN** the request SHALL succeed without JWT
