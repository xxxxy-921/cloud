## MODIFIED Requirements

### Requirement: Floating input card pattern
The system SHALL provide a floating input card pattern for full-page input areas. The card SHALL use `shadow-lg rounded-2xl bg-background/95 backdrop-blur` styling with a two-zone layout: content area (textarea/input) on top and a toolbar row on bottom separated by a subtle border.

#### Scenario: Floating input card visual
- **WHEN** the floating input card is rendered
- **THEN** it SHALL have rounded corners (2xl), elevated shadow (lg), and semi-transparent background with backdrop blur

## ADDED Requirements

### Requirement: Workspace container quiet visual style
The `workspace-surface` CSS class SHALL use a semi-transparent background (`oklch(... / 0.4)`) and a weak border (`oklch(... / 0.5)`) with NO backdrop-filter, NO linear-gradient background, and NO box-shadow. All components using `workspace-surface` (including Card) SHALL inherit this quiet visual treatment automatically.

#### Scenario: Card inherits quiet surface style
- **WHEN** a Card component is rendered on any page
- **THEN** it SHALL display with a barely-visible container (semi-transparent background, weak border, no blur, no shadow)

#### Scenario: Card content remains readable
- **WHEN** a Card with text content is rendered
- **THEN** the text foreground color SHALL maintain sufficient contrast against the semi-transparent background

### Requirement: Workspace page header quiet style
The `workspace-page-header` CSS class SHALL provide ONLY structural layout (flex, padding, gap) and a bottom border separator (`border-b border-border/50`). It SHALL NOT apply backdrop-filter, linear-gradient background, box-shadow, or rounded corners.

#### Scenario: Page header renders as flat separator
- **WHEN** a page using `workspace-page-header` is rendered
- **THEN** the header area SHALL have no visible container (no background, no shadow, no rounded corners), only a subtle bottom border separating it from content

#### Scenario: Page header layout preserved
- **WHEN** a page header has title on left and actions on right
- **THEN** the flex layout, padding, and responsive breakpoints SHALL work identically to before

### Requirement: Workspace table card quiet style
The `workspace-table-card` CSS class SHALL use a semi-transparent background and weak border, matching `workspace-surface` visual weight. It SHALL NOT apply backdrop-filter, linear-gradient, or box-shadow.

#### Scenario: DataTable container is visually light
- **WHEN** a DataTableCard is rendered
- **THEN** its container SHALL be barely visible with semi-transparent background and weak border

### Requirement: Workspace table toolbar quiet style
The `workspace-table-toolbar` CSS class SHALL provide ONLY structural layout and a bottom border separator. It SHALL NOT apply backdrop-filter, linear-gradient, background, or box-shadow.

#### Scenario: Toolbar renders without decoration
- **WHEN** a DataTableToolbar is rendered inside a DataTableCard
- **THEN** it SHALL have no visible background or shadow, only a subtle bottom border separating it from the table content

### Requirement: Workspace toolbar input quiet style
The `workspace-toolbar-input` CSS class SHALL use a transparent background and weak border color. It SHALL NOT apply box-shadow.

#### Scenario: Search input is visually minimal
- **WHEN** a search input with `workspace-toolbar-input` class is rendered
- **THEN** it SHALL have a subtle border and transparent background with no shadow

### Requirement: Workspace panel quiet style
The `workspace-panel` CSS class SHALL use a light opaque background and a right border for structural separation. It SHALL NOT apply backdrop-filter or linear-gradient. A minimal box-shadow MAY be retained for structural anchoring.

#### Scenario: Sidebar panel has subtle boundary
- **WHEN** the sidebar panel is rendered
- **THEN** it SHALL have a right border and slightly elevated background compared to the content area, without blur or gradient effects
