## ADDED Requirements

### Requirement: Themed page surface pattern
The system SHALL provide a themed page surface pattern for authenticated pages, covering page headers, primary content sections, secondary sections, and empty states. These surfaces SHALL share a consistent hierarchy derived from the unified workspace theme instead of page-by-page ad hoc card styling.

#### Scenario: List page uses themed surfaces
- **WHEN** an authenticated list page renders a header area, filters, table container, and empty state
- **THEN** each area SHALL follow the shared themed surface hierarchy
- **AND** the page SHALL not assemble unrelated card styles that break visual consistency

### Requirement: Accent usage stays sparse in shared patterns
Shared UI patterns SHALL use accent color sparingly. Accent color SHALL identify focus, active state, key actions, and status anchors, and SHALL NOT become the default large-area background treatment for common management screens.

#### Scenario: Section container in a management workflow
- **WHEN** a section container or card is rendered in an authenticated management page
- **THEN** its primary visual identity SHALL come from surface hierarchy and typography
- **AND** accent color SHALL remain limited to focused controls, highlights, or lightweight emphasis
