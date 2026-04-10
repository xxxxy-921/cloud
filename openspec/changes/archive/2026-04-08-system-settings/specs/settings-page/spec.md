## ADDED Requirements

### Requirement: Settings page with name and logo sections
The application SHALL display a settings page at /settings with two card sections.

#### Scenario: Page layout
- **WHEN** the user navigates to /settings
- **THEN** the page SHALL display a "基本信息" card with a name input and a "系统 Logo" card with upload controls

### Requirement: Edit system name
The settings page SHALL allow editing the system name with inline save.

#### Scenario: Load current name
- **WHEN** the settings page loads
- **THEN** the name input SHALL be pre-filled with the current system name from GET /api/v1/site-info

#### Scenario: Save name
- **WHEN** the user edits the name and clicks the save button
- **THEN** the system SHALL call PUT /api/v1/site-info with the new name
- **AND** the TopNav SHALL reflect the updated name

#### Scenario: Empty name validation
- **WHEN** the user clears the name input and clicks save
- **THEN** the form SHALL display a validation error and NOT submit

### Requirement: Upload logo
The settings page SHALL allow uploading a logo image.

#### Scenario: Upload via file picker
- **WHEN** the user selects an image file (PNG, JPEG, or WebP, max 2MB)
- **THEN** the system SHALL convert it to base64 and call PUT /api/v1/site-info/logo
- **AND** the preview SHALL update to show the new logo

#### Scenario: File too large
- **WHEN** the user selects a file larger than 2MB
- **THEN** the system SHALL display an error message without submitting

### Requirement: Preview logo
The settings page SHALL display a preview of the current logo.

#### Scenario: Logo exists
- **WHEN** the settings page loads and a logo is set
- **THEN** the logo card SHALL display the logo image from /api/v1/site-info/logo

#### Scenario: No logo
- **WHEN** the settings page loads and no logo is set
- **THEN** the logo card SHALL display a placeholder with upload instructions

### Requirement: Remove logo
The settings page SHALL allow removing the current logo.

#### Scenario: Remove logo
- **WHEN** the user clicks the remove button
- **THEN** the system SHALL call DELETE /api/v1/site-info/logo
- **AND** the preview SHALL revert to the placeholder state
