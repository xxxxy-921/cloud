## ADDED Requirements

### Requirement: Task list page with statistics
The system SHALL provide a `/tasks` page showing a dashboard header with 4 statistics cards (total tasks, running, completed today, failed today) and a tab-based task list below.

#### Scenario: Page renders with stats
- **WHEN** a user navigates to `/tasks`
- **THEN** the page SHALL display 4 stat cards fetched from `GET /api/v1/tasks/stats` and a task table below

#### Scenario: Tab filtering
- **WHEN** the user clicks the "定时任务" or "异步队列" tab
- **THEN** the task list SHALL filter to show only tasks of the corresponding type

### Requirement: Task list table
The task list table SHALL display columns: Name, Description, Cron/Type, Status (badge), Last Execution (relative time + status), and Actions.

#### Scenario: Task row rendering
- **WHEN** the task list loads
- **THEN** each row SHALL show the task name, description, cron expression (or "异步" for async), a status badge (active=green, paused=yellow), last execution info, and action buttons

#### Scenario: Empty state
- **WHEN** no tasks are registered
- **THEN** the table SHALL display an empty state message

### Requirement: Task action buttons with permission guard
Action buttons (pause/resume, trigger) SHALL be conditionally rendered based on the current user's permissions using the existing `PermissionGuard` component.

#### Scenario: Pause/Resume toggle
- **WHEN** a scheduled task is active and the user has `system:task:pause` permission
- **THEN** a pause button SHALL be shown; clicking it SHALL call `POST /tasks/:name/pause` and refresh the list

#### Scenario: Resume button
- **WHEN** a scheduled task is paused and the user has `system:task:resume` permission
- **THEN** a resume button SHALL be shown; clicking it SHALL call `POST /tasks/:name/resume` and refresh the list

#### Scenario: Manual trigger button
- **WHEN** the user has `system:task:trigger` permission
- **THEN** a trigger button SHALL be shown on every task row; clicking it SHALL call `POST /tasks/:name/trigger` with a confirmation dialog

#### Scenario: No permission
- **WHEN** the user lacks `system:task:pause` permission
- **THEN** the pause button SHALL NOT be rendered

### Requirement: Task detail page with execution history
The system SHALL provide a `/tasks/:name` page showing task configuration and paginated execution history.

#### Scenario: Detail page renders
- **WHEN** a user navigates to `/tasks/log-cleanup`
- **THEN** the page SHALL display a config card (name, type, description, cron, timeout, retries, status) and an execution history table below

#### Scenario: Execution history table
- **WHEN** the detail page loads
- **THEN** the execution history table SHALL display columns: ID, Trigger (badge), Status (badge with color), Duration, Error (truncated), Time — with pagination controls

#### Scenario: Navigate back
- **WHEN** the user clicks the back button or breadcrumb
- **THEN** they SHALL return to the `/tasks` list page

### Requirement: Frontend route and menu integration
The `/tasks` route SHALL be registered in App.tsx with lazy loading and protected by permission guard (`system:task:list`). The menu entry SHALL come from the backend menu API (seeded as part of 系统管理).

#### Scenario: Route lazy loaded
- **WHEN** the user navigates to `/tasks`
- **THEN** the page component SHALL be lazy-loaded via React.lazy

#### Scenario: Menu visibility
- **WHEN** a user with `system:task:list` permission logs in
- **THEN** "任务中心" SHALL appear in the sidebar under "系统管理"

#### Scenario: No permission hides menu
- **WHEN** a user without `system:task:list` permission logs in
- **THEN** "任务中心" SHALL NOT appear in the sidebar
