## MODIFIED Requirements

### Requirement: Concurrent session limit
The system SHALL enforce a configurable maximum number of concurrent sessions per user. The limit is read from SystemConfig key `security.max_concurrent_sessions` (default: 5, 0 means unlimited). The refresh token expiry duration SHALL be read from SystemConfig key `security.session_timeout_minutes` (default: 10080, i.e. 7 days) instead of being hardcoded.

#### Scenario: Login within limit
- **WHEN** a user logs in and their active session count is below the configured limit
- **THEN** the login SHALL proceed normally

#### Scenario: Login exceeds limit
- **WHEN** a user logs in and their active session count equals or exceeds the configured limit
- **THEN** the system SHALL revoke the least recently active sessions (by LastSeenAt ascending) and blacklist their access tokens, keeping only (limit - 1) existing sessions to make room for the new one

#### Scenario: Limit set to zero
- **WHEN** the `security.max_concurrent_sessions` config is set to 0
- **THEN** no concurrent session limit SHALL be enforced

#### Scenario: Custom session timeout
- **WHEN** security.session_timeout_minutes is set to 60
- **THEN** new refresh tokens SHALL have an expiry of 60 minutes from creation

#### Scenario: Default session timeout
- **WHEN** security.session_timeout_minutes is not set
- **THEN** refresh tokens SHALL have the default 7-day (10080 minutes) expiry
