## ADDED Requirements

### Requirement: Navigation config split by App
Navigation configuration SHALL be organized as one file per App in lib/nav/ directory.

#### Scenario: Adding a new app
- **WHEN** a developer creates a new app file in lib/nav/ and imports it in index.ts
- **THEN** the new app SHALL appear in the Icon Rail and Nav Panel

#### Scenario: Adding a nav item to existing app
- **WHEN** a developer adds a NavItemDef to an app file
- **THEN** the new item SHALL appear in that app's Nav Panel

### Requirement: Central navigation export
The lib/nav/index.ts SHALL export the assembled apps array, findActiveApp helper, and breadcrumbLabels.

#### Scenario: All apps assembled
- **WHEN** the application imports from lib/nav
- **THEN** it SHALL receive the complete apps[] array with all registered apps

#### Scenario: Breadcrumb labels aggregated
- **WHEN** the header component renders breadcrumbs
- **THEN** breadcrumbLabels SHALL include labels from all app modules
