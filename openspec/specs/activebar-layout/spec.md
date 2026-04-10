# Capability: activebar-layout

## Purpose
Defines the application's sidebar navigation layout with a two-tier structure (Icon Rail + Nav Panel), TopNav bar, header with breadcrumb, and data-driven navigation configuration.

## Requirements

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
The TopNav bar SHALL be fixed at the top with height h-14. It SHALL display the sidebar toggle button and site logo/name on the left, and user actions on the right. The user dropdown menu SHALL be implemented using shadcn `DropdownMenu` component (not a hand-rolled dropdown), providing keyboard navigation, ARIA roles, theme-aware styling, and click-outside dismissal. The site logo and name SHALL be fetched from GET /api/v1/site-info.

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

#### Scenario: User opens dropdown menu
- **WHEN** user clicks the username button in TopNav
- **THEN** a `DropdownMenu` SHALL open with "修改密码" and "退出登录" items
- **AND** the menu SHALL support keyboard navigation (Arrow keys, Enter, Escape)

#### Scenario: User changes password via dropdown
- **WHEN** user selects "修改密码" from the dropdown
- **THEN** the ChangePasswordDialog SHALL open

#### Scenario: User logs out via dropdown
- **WHEN** user selects "退出登录" from the dropdown
- **THEN** the system SHALL call logout and navigate to `/login`

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

### Requirement: Sheet components a11y
All Sheet (side drawer) components SHALL include a `SheetDescription` element (visible or visually hidden) so that screen readers can announce the dialog's purpose. This eliminates the Radix "Missing Description" console warning.

#### Scenario: Role sheet has description
- **WHEN** the role edit/create sheet is opened
- **THEN** a SheetDescription SHALL be present in the DOM (may be visually hidden)

#### Scenario: User sheet has description
- **WHEN** the user edit/create sheet is opened
- **THEN** a SheetDescription SHALL be present in the DOM

#### Scenario: Config sheet has description
- **WHEN** the config edit/create sheet is opened
- **THEN** a SheetDescription SHALL be present in the DOM

#### Scenario: Menu sheet has description
- **WHEN** the menu edit/create sheet is opened
- **THEN** a SheetDescription SHALL be present in the DOM

#### Scenario: Permission dialog has description
- **WHEN** the permission assignment sheet is opened
- **THEN** a SheetDescription SHALL be present in the DOM (the "已选 X/Y 项" text can serve as description)
