## MODIFIED Requirements

### Requirement: oklch CSS variable theme
The system SHALL define theme CSS variables using oklch color space in globals.css.

#### Scenario: Primary color token
- **WHEN** the application renders
- **THEN** `--color-primary` SHALL be `oklch(0.50 0.13 264)` (desaturated blue)

#### Scenario: Ring color token
- **WHEN** the application renders
- **THEN** `--color-ring` SHALL match the updated primary value

#### Scenario: Border color token
- **WHEN** the application renders
- **THEN** `--color-border` SHALL be `oklch(0.91 0.005 250)` (lighter than previous)

#### Scenario: Input border color token
- **WHEN** the application renders
- **THEN** `--color-input` SHALL be `oklch(0.91 0.005 250)` (matching border)

#### Scenario: Sidebar accent color token
- **WHEN** the application renders
- **THEN** `--color-sidebar-accent` SHALL be `oklch(0.95 0.02 264)` (desaturated highlight)

### Requirement: Card shadow convention
Card containers SHALL use border-only styling without box-shadow.

#### Scenario: Card rendering
- **WHEN** a Card component renders
- **THEN** it SHALL NOT apply `shadow-sm` or any visible box-shadow
- **AND** it SHALL retain its `border` for boundary definition

### Requirement: Input component sizing
The Input component SHALL use refined sizing aligned with the auth page aesthetic.

#### Scenario: Input height and radius
- **WHEN** an Input component renders in the application
- **THEN** its height SHALL be `h-[2.375rem]` (38px)
- **AND** its border-radius SHALL be `rounded-lg`

#### Scenario: Input focus ring
- **WHEN** an Input receives focus
- **THEN** the focus ring SHALL be 2px wide (`ring-[2px]`) instead of 3px

### Requirement: Badge default variant
The default Badge variant SHALL use a subtle tinted background instead of solid primary.

#### Scenario: Default badge rendering
- **WHEN** a Badge with `variant="default"` renders
- **THEN** its background SHALL be `bg-primary/10`
- **AND** its text color SHALL be the primary color (not white)

### Requirement: DataTable pagination styling
The DataTablePagination component SHALL use clean, minimal styling.

#### Scenario: Pagination rendering
- **WHEN** a DataTablePagination component renders
- **THEN** it SHALL NOT use `border-dashed` styling
- **AND** it SHALL use a simple top border (`border-t border-border/40`) or no border
