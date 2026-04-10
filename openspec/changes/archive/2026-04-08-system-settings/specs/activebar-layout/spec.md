## MODIFIED Requirements

### Requirement: TopNav bar
The application SHALL have a fixed TopNav bar at the top (h-14, 56px).

#### Scenario: TopNav content
- **WHEN** the dashboard layout renders
- **THEN** the TopNav SHALL display the site logo (if set) and site name on the left, fetched from GET /api/v1/site-info
- **AND** user actions on the right

#### Scenario: TopNav with no custom settings
- **WHEN** no site info has been configured
- **THEN** the TopNav SHALL display the default name "Metis" with no logo

#### Scenario: TopNav reflects setting changes
- **WHEN** the site name or logo is updated via the settings page
- **THEN** the TopNav SHALL reflect the new values after the query cache is invalidated
