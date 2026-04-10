# Capability: session-management

## Purpose
Session management system providing active session listing, force kick, in-memory token blacklist, concurrent session limits, and cleanup tasks.

## Requirements

### Requirement: Active sessions list API
The system SHALL provide `GET /api/v1/sessions` to list all active (non-revoked, non-expired) sessions with user info, protected by `system:session:list` permission.

#### Scenario: List active sessions
- **WHEN** an authorized admin calls GET /api/v1/sessions
- **THEN** the system SHALL return a paginated list of active refresh tokens joined with user data, including fields: id, userId, username, ipAddress, userAgent, loginAt (createdAt), lastSeenAt, isCurrent (boolean)

#### Scenario: isCurrent flag
- **WHEN** the sessions list is returned
- **THEN** each session whose AccessTokenJTI matches the requesting user's current JWT jti SHALL have isCurrent=true

#### Scenario: No active sessions
- **WHEN** no active refresh tokens exist
- **THEN** the system SHALL return an empty list with total=0

### Requirement: Force kick session API
The system SHALL provide `DELETE /api/v1/sessions/:id` to forcefully terminate a session, protected by `system:session:delete` permission.

#### Scenario: Successful kick
- **WHEN** an admin calls DELETE /api/v1/sessions/:id with a valid session ID
- **THEN** the system SHALL revoke the refresh token, add its AccessTokenJTI to the in-memory blacklist with TTL equal to the access token's remaining lifetime, and return 200

#### Scenario: Prevent self-kick
- **WHEN** an admin calls DELETE /api/v1/sessions/:id where the session belongs to their own current JWT jti
- **THEN** the system SHALL return 400 with message "不能踢出当前会话"

#### Scenario: Session not found
- **WHEN** DELETE /api/v1/sessions/:id is called with a non-existent or already-revoked session ID
- **THEN** the system SHALL return 404

### Requirement: In-memory token blacklist
The system SHALL maintain a process-level in-memory blacklist mapping JWT jti strings to their expiration times. The JWTAuth middleware SHALL check the blacklist on every request.

#### Scenario: Blacklisted token rejected
- **WHEN** a request carries a JWT whose jti is in the blacklist and has not expired
- **THEN** the middleware SHALL return 401 with message "session terminated"

#### Scenario: Expired blacklist entry ignored
- **WHEN** a blacklist entry's expiration time has passed
- **THEN** the entry SHALL be treated as non-existent (lazy cleanup) and the JWT SHALL be validated normally

#### Scenario: Blacklist Add
- **WHEN** a session is kicked or a user's tokens are bulk-revoked
- **THEN** the system SHALL call blacklist.Add(jti, accessTokenExpiresAt) to immediately block the access token

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

### Requirement: Blacklist cleanup scheduled task
The system SHALL register a scheduled task `blacklist_cleanup` (cron: `*/5 * * * *`) in the scheduler engine that calls blacklist.Cleanup() to remove all expired entries.

#### Scenario: Cleanup runs on schedule
- **WHEN** the blacklist_cleanup task fires
- **THEN** all blacklist entries whose expiration time is in the past SHALL be removed

### Requirement: Expired token cleanup scheduled task
The system SHALL register a scheduled task `expired_token_cleanup` (cron: `0 3 * * *`) that hard-deletes refresh token records that are either expired for more than 7 days or revoked for more than 7 days.

#### Scenario: Old expired tokens cleaned
- **WHEN** the expired_token_cleanup task fires
- **THEN** all refresh_tokens where (expires_at < now - 7 days) OR (revoked=true AND updated_at < now - 7 days) SHALL be permanently deleted from the database

### Requirement: Session management menu and permissions
The system SHALL include a "会话管理" menu item under the system management group at path `/sessions` with permission code `system:session:list`. A child button permission `system:session:delete` SHALL control the kick action.

#### Scenario: Menu seed data
- **WHEN** the seed command runs
- **THEN** the menu tree SHALL include a "会话管理" entry with icon "Monitor", path "/sessions", and associated Casbin policies for GET /api/v1/sessions and DELETE /api/v1/sessions/:id

### Requirement: Sessions management frontend page
The system SHALL provide a sessions management page at `/sessions` displaying active sessions in a table with columns: username, IP address, device (parsed from UserAgent), login time, last active time, and action.

#### Scenario: Sessions table display
- **WHEN** the admin navigates to /sessions
- **THEN** the page SHALL display a table of active sessions fetched from GET /api/v1/sessions

#### Scenario: Kick button for non-current sessions
- **WHEN** a session row has isCurrent=false
- **THEN** a "踢出" button SHALL be enabled

#### Scenario: Current session indicator
- **WHEN** a session row has isCurrent=true
- **THEN** the action column SHALL show "(当前会话)" text instead of a kick button

#### Scenario: Kick confirmation dialog
- **WHEN** the admin clicks "踢出" on a session
- **THEN** an AlertDialog SHALL appear asking for confirmation before calling DELETE /api/v1/sessions/:id

#### Scenario: Table refresh after kick
- **WHEN** a session is successfully kicked
- **THEN** the sessions table SHALL refresh to reflect the change
