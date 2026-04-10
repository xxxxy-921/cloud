## ADDED Requirements

### Requirement: Two-tier sidebar with Icon Rail and Nav Panel
The sidebar SHALL consist of a Tier 1 Icon Rail (w-12, 48px) and a Tier 2 Nav Panel (w-40, 160px).

#### Scenario: Icon Rail displays app icons
- **WHEN** the dashboard layout renders
- **THEN** the Icon Rail SHALL display one icon button per AppDef in the navigation config

#### Scenario: Clicking Icon Rail switches Nav Panel
- **WHEN** a user clicks an app icon in the Icon Rail
- **THEN** the Nav Panel SHALL display the navigation items belonging to that app
- **AND** the router SHALL navigate to the first item of that app

#### Scenario: Active app detection by pathname
- **WHEN** the current URL pathname matches an app's basePath
- **THEN** that app's icon SHALL be highlighted in the Icon Rail
- **AND** that app's items SHALL be displayed in the Nav Panel

### Requirement: Data-driven navigation config
Navigation SHALL be defined by an AppDef[] array in lib/nav.ts.

#### Scenario: Navigation configuration structure
- **WHEN** the sidebar loads navigation config
- **THEN** each AppDef SHALL have id, label, icon (lucide-react name), basePath, and items array
- **AND** each NavItemDef SHALL have id, href, icon, and title

#### Scenario: Adding a new module
- **WHEN** a developer adds a new AppDef to the config array
- **THEN** the Icon Rail SHALL automatically display the new app icon
- **AND** its items SHALL appear in the Nav Panel when selected

### Requirement: TopNav bar
The application SHALL have a fixed TopNav bar at the top (h-14, 56px).

#### Scenario: TopNav content
- **WHEN** the dashboard layout renders
- **THEN** the TopNav SHALL display the application logo/name on the left and user actions on the right

### Requirement: Header with breadcrumb
Below the TopNav, within the content area, a Header bar SHALL display breadcrumb navigation.

#### Scenario: Breadcrumb display
- **WHEN** the user navigates to /config
- **THEN** the header SHALL display a breadcrumb showing the current location

### Requirement: Active state styling
Active navigation items SHALL use accent background tint.

#### Scenario: Active Icon Rail item
- **WHEN** an app is active in the Icon Rail
- **THEN** its icon button SHALL display `bg-sidebar-accent text-sidebar-accent-foreground`

#### Scenario: Active Nav Panel item
- **WHEN** a nav item matches the current pathname
- **THEN** it SHALL display `bg-sidebar-accent text-sidebar-accent-foreground`
