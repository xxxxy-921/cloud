## MODIFIED Requirements

### Requirement: Settings page with tabbed layout
The application SHALL display a settings page at /settings with a Tabs component containing three tabs: "站点信息" (site info), "安全设置" (security), and "任务设置" (scheduler).

#### Scenario: Page layout
- **WHEN** the user navigates to /settings
- **THEN** the page SHALL display a Tabs component with three tabs, defaulting to the "站点信息" tab

#### Scenario: Site info tab content
- **WHEN** the "站点信息" tab is active
- **THEN** the tab SHALL display the existing "基本信息" card with name input and "系统 Logo" card with upload controls

#### Scenario: Security settings tab content
- **WHEN** the "安全设置" tab is active
- **THEN** the tab SHALL display a card with a numeric input for "最大并发会话数" (max concurrent sessions), loaded from GET /api/v1/settings/security, with a save button that calls PUT /api/v1/settings/security

#### Scenario: Security settings validation
- **WHEN** the user enters a negative number for max concurrent sessions
- **THEN** the form SHALL display a validation error (value must be >= 0)

#### Scenario: Scheduler settings tab content
- **WHEN** the "任务设置" tab is active
- **THEN** the tab SHALL display a card with a numeric input for "历史保留天数" (history retention days), loaded from GET /api/v1/settings/scheduler, with a save button that calls PUT /api/v1/settings/scheduler

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
