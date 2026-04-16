## ADDED Requirements

### Requirement: Test session list retrieval
The service-layer test suite SHALL verify that `ListSessions` returns active sessions paginated and correctly marks the current session via JTI matching.

#### Scenario: List active sessions with pagination
- **WHEN** multiple active refresh tokens exist and `ListSessions` is called with page=1 and pageSize=2
- **THEN** it returns exactly 2 session items and the correct total count

#### Scenario: Mark current session via JTI
- **WHEN** `ListSessions` is called with a `currentJTI` that matches one token's `AccessTokenJTI`
- **THEN** the matching session SHALL have `IsCurrent=true` and all others `IsCurrent=false`

#### Scenario: Exclude revoked and expired sessions
- **WHEN** some refresh tokens are revoked or expired and `ListSessions` is called
- **THEN** only non-revoked, non-expired tokens appear in the result

#### Scenario: Empty sessions list
- **WHEN** no active refresh tokens exist and `ListSessions` is called
- **THEN** it returns an empty items slice and total=0

### Requirement: Test session kick
The service-layer test suite SHALL verify that `KickSession` revokes the refresh token, blacklists the access token JTI, prevents self-kick, and handles missing sessions.

#### Scenario: Kick session successfully
- **WHEN** `KickSession` is called for a valid active session with a different JTI
- **THEN** the refresh token is revoked, its `AccessTokenJTI` is added to the blacklist, and no error is returned

#### Scenario: Prevent self-kick
- **WHEN** `KickSession` is called for a session whose `AccessTokenJTI` matches the current JTI
- **THEN** it returns `ErrCannotKickSelf`

#### Scenario: Kick non-existent session
- **WHEN** `KickSession` is called for a session ID that does not exist
- **THEN** it returns `ErrSessionNotFound`

#### Scenario: Kick already revoked session
- **WHEN** `KickSession` is called for a session that has already been revoked
- **THEN** it returns `ErrSessionNotFound`

#### Scenario: Kick session with empty JTI
- **WHEN** `KickSession` is called for a valid session that has no `AccessTokenJTI`
- **THEN** the refresh token is revoked and no panic occurs
