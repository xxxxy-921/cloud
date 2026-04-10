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
- **THEN** the tab SHALL display 6 card sections loaded from GET /api/v1/settings/security with a single save button:
- **AND** "密码策略" card: numeric input for min length (min 1), 4 checkboxes for upper/lower/number/special requirements, numeric input for expiry days (0=never)
- **AND** "登录安全" card: numeric input for max failed attempts (0=disabled), numeric input for lockout minutes, select for captcha provider ("关闭"/"图形验证码")
- **AND** "会话管理" card: numeric input for session timeout minutes, numeric input for max concurrent sessions (0=unlimited)
- **AND** "两步验证" card: radio group with "可选（用户自行启用）" and "强制（所有用户必须启用）"
- **AND** "注册设置" card: checkbox for open registration, role select dropdown for default role (fetched from GET /api/v1/roles, empty option for "不分配角色")
- **AND** "日志保留策略" card: numeric inputs for auth and operation log retention days (0=never clean)

#### Scenario: Security settings validation
- **WHEN** the user enters invalid values (negative numbers, password min length < 1)
- **THEN** the form SHALL display validation errors

#### Scenario: Security settings save
- **WHEN** the user modifies security settings and clicks save
- **THEN** the system SHALL call PUT /api/v1/settings/security with all fields and display a success toast

#### Scenario: Scheduler settings tab content
- **WHEN** the "任务设置" tab is active
- **THEN** the tab SHALL display a card with a numeric input for "历史保留天数" (history retention days)
