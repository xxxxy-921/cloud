## MODIFIED Requirements

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
