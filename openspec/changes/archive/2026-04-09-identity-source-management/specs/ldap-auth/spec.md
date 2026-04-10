## ADDED Requirements

### Requirement: LDAP authentication flow（App 内实现）
The identity App SHALL implement LDAP bind authentication in `internal/app/identity/ldap.go`. The App's `ExternalAuthenticator` implementation SHALL attempt LDAP authentication when called by the kernel's AuthService.

#### Scenario: LDAP auth triggered by kernel
- **WHEN** kernel AuthService calls `ExternalAuthenticator.AuthenticateByPassword("john", "secret")` after local password failure
- **THEN** the App SHALL find matching LDAP sources and attempt LDAP bind authentication

#### Scenario: Successful LDAP authentication (new user)
- **WHEN** LDAP bind succeeds for a user with no existing local account
- **THEN** SHALL JIT-create a local User with attributes mapped from LDAP, return the User

#### Scenario: Successful LDAP authentication (existing user)
- **WHEN** LDAP bind succeeds for a returning user
- **THEN** SHALL find existing local user via UserConnection, update mapped attributes, return the User

#### Scenario: LDAP auth failure
- **WHEN** LDAP bind fails with invalid credentials
- **THEN** SHALL return error, kernel proceeds to return "invalid credentials"

#### Scenario: No matching LDAP source
- **WHEN** no enabled LDAP source matches the user's context
- **THEN** SHALL return error, kernel proceeds normally

### Requirement: LDAP bind and search
The App SHALL perform LDAP authentication in two steps: (1) bind with admin BindDN/BindPassword to search for user DN, (2) re-bind with user DN and provided password.

#### Scenario: Admin bind and user search
- **WHEN** LDAP auth triggered for username "john" with UserFilter "(uid={{username}})"
- **THEN** SHALL bind as BindDN, search under SearchBase with filter "(uid=john)", obtain user DN

#### Scenario: User not found in LDAP
- **WHEN** admin bind search returns zero results
- **THEN** SHALL treat as authentication failure

#### Scenario: User re-bind verification
- **WHEN** user DN is found
- **THEN** SHALL attempt bind with user DN + provided password

### Requirement: LDAP TLS support
The App SHALL support LDAP TLS: `ldaps://` for implicit TLS, `ldap://` with UseTLS=true for StartTLS upgrade.

#### Scenario: LDAPS connection
- **WHEN** ServerURL is "ldaps://ldap.corp.com:636"
- **THEN** SHALL establish TLS connection directly

#### Scenario: StartTLS upgrade
- **WHEN** ServerURL is "ldap://ldap.corp.com:389" and UseTLS is true
- **THEN** SHALL connect plain then upgrade via StartTLS

### Requirement: LDAP attribute mapping
The App SHALL map LDAP user attributes to local User fields per the source's AttributeMapping. Default: username←uid, email←mail, display_name←cn.

#### Scenario: Custom attribute mapping
- **WHEN** source has AttributeMapping `{"username": "sAMAccountName", "email": "userPrincipalName"}`
- **THEN** SHALL read sAMAccountName as username, userPrincipalName as email

#### Scenario: Missing attribute
- **WHEN** LDAP entry lacks the mapped email attribute
- **THEN** SHALL leave local user's email empty (not fail)

### Requirement: LDAP identity tracking via UserConnection
The App SHALL track LDAP identities using Provider `"ldap_{sourceId}"` and ExternalID as the user's DN.

#### Scenario: Store LDAP identity on first login
- **WHEN** user first authenticates via LDAP source ID 5 with DN "uid=john,ou=People,dc=corp,dc=com"
- **THEN** SHALL create UserConnection with Provider="ldap_5", ExternalID="uid=john,ou=People,dc=corp,dc=com"

### Requirement: LDAP connection lifecycle
The App SHALL NOT maintain persistent LDAP connections. Each auth attempt SHALL dial, authenticate, and close.

#### Scenario: Connection lifecycle
- **WHEN** LDAP auth is attempted
- **THEN** SHALL dial new connection, perform auth, close regardless of outcome
