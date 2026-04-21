## ADDED Requirements

### Requirement: Install wizard stays aligned with unified theme tokens
The install wizard SHALL continue using the authentication-page visual language and SHALL stay aligned with the same global theme token family used by the unified authenticated workspace. It SHALL NOT drift into a separate standalone style system.

#### Scenario: Install wizard evaluated after workspace theme rollout
- **WHEN** the system theme foundation has been upgraded for authenticated pages
- **THEN** the install wizard SHALL still resolve its shell styling from the same token family
- **AND** it SHALL remain visually consistent with both the login page and the authenticated workspace
