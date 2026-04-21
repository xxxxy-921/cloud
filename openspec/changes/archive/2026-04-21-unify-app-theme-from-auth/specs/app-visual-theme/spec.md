# Capability: app-visual-theme

## Purpose

Defines the unified visual theme for the authenticated application workspace, using the current login page as the source of visual language while preserving the restrained, system-oriented baseline documented in `DESIGN.md`.

## Requirements

### Requirement: Unified authenticated workspace theme
The system SHALL provide a unified visual theme for authenticated pages that is visually consistent with the login page's design language while remaining appropriate for a productivity-oriented back-office workspace. The authenticated workspace SHALL NOT remain a plain black-white-gray shell disconnected from the authentication experience.

#### Scenario: User enters authenticated workspace after login
- **WHEN** a user signs in and navigates from `/login` into the main application
- **THEN** the authenticated workspace SHALL present a visual language that feels like the same product family as the login page
- **AND** the workspace SHALL preserve readability, scanability, and operational efficiency expected of a management console

### Requirement: Theme derived from login page principles, not login page effects
The system theme SHALL inherit the login page's principles of restrained brand emphasis, layered surfaces, and refined spacing. The system SHALL NOT directly reuse highly decorative auth-stage effects such as large atmospheric orbs, full-screen hero staging, or heavy glassmorphism as the default pattern for business pages.

#### Scenario: Business page uses unified theme
- **WHEN** a user views an authenticated list page, detail page, or form page
- **THEN** the page SHALL use restrained layered surfaces and subtle brand emphasis
- **AND** it SHALL NOT rely on full-page decorative auth effects to create identity

### Requirement: Global theme token support for workspace surfaces
The frontend SHALL expose a global theme token model that covers workspace background, navigation surface, content surface, elevated surface, weak separators, focus emphasis, and brand-accent usage. Auth pages and authenticated pages SHALL consume the same token family instead of maintaining disconnected visual systems.

#### Scenario: Shared tokens power auth and workspace shells
- **WHEN** the frontend renders the login page and the authenticated dashboard layout
- **THEN** both shells SHALL resolve color, radius, and emphasis values from the same global token family

### Requirement: Dashboard shell expresses unified hierarchy
The authenticated application shell, including the top navigation, sidebar, and main content container, SHALL express a coherent hierarchy through background, borders, and surface contrast. The shell SHALL provide more visual structure than a flat monochrome frame, while keeping navigation and content legible.

#### Scenario: Authenticated shell renders navigation hierarchy
- **WHEN** a user views the dashboard layout
- **THEN** the top navigation, sidebar, and content area SHALL have intentional surface relationships and separation cues
- **AND** the layout SHALL avoid looking like an unstyled monochrome admin scaffold

### Requirement: Shared page surfaces follow restrained system baseline
Common authenticated page patterns such as page headers, section containers, cards, sheet forms, empty states, and inline feedback blocks SHALL follow a shared restrained system baseline: light surfaces, subtle boundaries, moderate contrast, and limited accent usage. These patterns SHALL prioritize content readability over decoration.

#### Scenario: Shared container on management page
- **WHEN** a management page renders a section container or information card
- **THEN** the container SHALL read as a light structural surface rather than a heavy decorative card
- **AND** accent color SHALL be reserved for focus, action, or status anchors instead of large area fills

### Requirement: Unified theme rollout supports phased migration
The system SHALL support phased adoption of the unified theme. Theme foundation and application shell updates SHALL be able to land before all module pages are visually migrated, without requiring a single all-or-nothing redesign release.

#### Scenario: Shell updated before every module page is migrated
- **WHEN** the theme foundation and dashboard shell have been upgraded but some business modules still use older page internals
- **THEN** the application SHALL remain visually coherent at the shell level
- **AND** module pages SHALL be allowed to migrate in later batches without violating the capability

### Requirement: Unified theme preserves existing design constraints
The unified theme SHALL remain compatible with the design constraints documented in `DESIGN.md`: the product is not a marketing site, strong brand blocks and exaggerated gradients are discouraged, and containers are not the visual protagonist. The theme SHALL treat the login page as a source of tone and structure, not a license for visual excess.

#### Scenario: Theme review against design baseline
- **WHEN** a designer or engineer evaluates an authenticated page against the unified theme capability
- **THEN** the page SHALL satisfy the restrained, systemized, readability-first baseline from `DESIGN.md`
- **AND** it SHALL fail review if it depends on oversized brand washes, excessive glow, or decorative glass effects to appear modern
