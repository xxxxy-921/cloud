## ADDED Requirements

### Requirement: Config list page
The application SHALL display all system configs in a table at /config.

#### Scenario: Table displays configs
- **WHEN** the user navigates to /config
- **THEN** a table SHALL display all system config entries with columns: Key, Value, Remark, UpdatedAt

#### Scenario: Empty state
- **WHEN** no system configs exist
- **THEN** the table SHALL display an empty state message

### Requirement: Create config via Sheet drawer
The application SHALL use a Sheet (right-side drawer) for creating new configs.

#### Scenario: Open create drawer
- **WHEN** the user clicks the "新建" button on the config page
- **THEN** a Sheet SHALL slide in from the right with a form containing Key, Value, and Remark fields

#### Scenario: Submit create form
- **WHEN** the user fills in the form and submits
- **THEN** the system SHALL call PUT /api/v1/config and refresh the table on success

### Requirement: Edit config via Sheet drawer
The application SHALL use a Sheet for editing existing configs.

#### Scenario: Open edit drawer
- **WHEN** the user clicks the edit action on a config row
- **THEN** a Sheet SHALL slide in from the right with the form pre-filled with existing values

#### Scenario: Key field read-only on edit
- **WHEN** the edit drawer opens
- **THEN** the Key field SHALL be read-only (primary key cannot change)

### Requirement: Delete config with confirmation
The application SHALL use an AlertDialog for delete confirmation.

#### Scenario: Delete confirmation
- **WHEN** the user clicks the delete action on a config row
- **THEN** an AlertDialog SHALL appear asking for confirmation

#### Scenario: Confirmed delete
- **WHEN** the user confirms deletion
- **THEN** the system SHALL call DELETE /api/v1/config/:key and refresh the table on success
