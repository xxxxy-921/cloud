## ADDED Requirements

### Requirement: Login failure tracking
The User model SHALL include `FailedLoginAttempts` (int, default 0) and `LockedUntil` (*time.Time, nullable). On each failed password verification, FailedLoginAttempts SHALL be atomically incremented via `UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = ?`. On successful login, both fields SHALL be reset to 0 and nil respectively.

#### Scenario: Increment on failure
- **WHEN** a user enters an incorrect password
- **THEN** the system SHALL atomically increment FailedLoginAttempts by 1

#### Scenario: Reset on success
- **WHEN** a user enters a correct password and logs in successfully
- **THEN** the system SHALL set FailedLoginAttempts=0 and LockedUntil=nil

#### Scenario: Concurrent login attempts
- **WHEN** two failed login attempts occur simultaneously for the same user
- **THEN** the atomic SQL update SHALL correctly increment the counter to 2

### Requirement: Account lockout
The system SHALL lock a user's account when FailedLoginAttempts reaches the configured `security.login_max_attempts` (default 5). Locking SHALL set `LockedUntil` to now + `security.login_lockout_minutes` (default 30) minutes. The lockout is time-based and self-healing: once LockedUntil passes, the user can attempt login again.

#### Scenario: Lockout triggered
- **WHEN** a user's FailedLoginAttempts reaches 5 (with login_max_attempts=5)
- **THEN** the system SHALL set LockedUntil to now + 30 minutes and record an audit log with action="lockout"

#### Scenario: Login while locked
- **WHEN** a locked user (LockedUntil > now) attempts to login
- **THEN** the system SHALL return 423 with message "账户已锁定，请 N 分钟后重试" where N is the remaining lockout minutes, WITHOUT verifying the password

#### Scenario: Lockout expired
- **WHEN** a locked user's LockedUntil has passed and they attempt to login with correct password
- **THEN** the system SHALL allow login and reset FailedLoginAttempts=0 and LockedUntil=nil

#### Scenario: Lockout disabled
- **WHEN** security.login_max_attempts is set to 0
- **THEN** the system SHALL not track failed attempts or lock accounts

### Requirement: Lockout configuration
The system SHALL read lockout settings from SystemConfig: `security.login_max_attempts` (default "5", 0=disabled) and `security.login_lockout_minutes` (default "30").

#### Scenario: Custom lockout settings
- **WHEN** SystemConfig has login_max_attempts="10" and login_lockout_minutes="60"
- **THEN** accounts SHALL lock after 10 failed attempts for 60 minutes

### Requirement: Admin unlock
The system SHALL allow administrators to unlock a locked user account by setting FailedLoginAttempts=0 and LockedUntil=nil via the existing PUT /api/v1/users/:id endpoint. An audit log with action="unlock" SHALL be recorded.

#### Scenario: Admin unlocks user
- **WHEN** an admin calls PUT /api/v1/users/:id with a request to unlock the account
- **THEN** the system SHALL set FailedLoginAttempts=0 and LockedUntil=nil and record an audit log

#### Scenario: User lockout status in list
- **WHEN** GET /api/v1/users returns user data
- **THEN** the response SHALL include lockedUntil and failedLoginAttempts fields so the admin UI can display lockout status

### Requirement: Lockout check order
The lockout check SHALL execute BEFORE password verification in the login flow. This prevents timing-based attacks where an attacker can determine if a password is correct based on response time differences.

#### Scenario: Locked user with correct password
- **WHEN** a locked user provides the correct password
- **THEN** the system SHALL reject the login with 423 without checking the password
