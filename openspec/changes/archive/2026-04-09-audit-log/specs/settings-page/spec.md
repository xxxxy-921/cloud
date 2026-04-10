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
- **AND** the card SHALL include a "日志保留策略" section with two numeric inputs: "登录活动日志保留天数" (default 90) and "操作记录日志保留天数" (default 365), with a note that 0 means never clean

#### Scenario: Security settings validation
- **WHEN** the user enters a negative number for max concurrent sessions or log retention days
- **THEN** the form SHALL display a validation error (value must be >= 0)

#### Scenario: Scheduler settings tab content
- **WHEN** the "任务设置" tab is active
- **THEN** the tab SHALL display a card with a numeric input for "历史保留天数" (history retention days), loaded from GET /api/v1/settings/scheduler, with a save button that calls PUT /api/v1/settings/scheduler
