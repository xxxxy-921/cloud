## ADDED Requirements

### Requirement: List tasks
The system SHALL expose `GET /api/v1/tasks` to list all registered tasks with their runtime state. It SHALL support an optional query parameter `type` to filter by `scheduled` or `async`.

#### Scenario: List all tasks
- **WHEN** a GET request is made to `/api/v1/tasks`
- **THEN** the response SHALL be 200 with a JSON array of task objects, each containing: name, type, description, cronExpr (if scheduled), timeoutMs, maxRetries, status, and lastExecution (timestamp, status, duration of most recent run)

#### Scenario: Filter by type
- **WHEN** a GET request is made to `/api/v1/tasks?type=scheduled`
- **THEN** the response SHALL include only tasks where type is `scheduled`

### Requirement: Get task detail
The system SHALL expose `GET /api/v1/tasks/:name` to retrieve a single task's configuration and its most recent execution history (default 20 records).

#### Scenario: Task exists
- **WHEN** a GET request is made to `/api/v1/tasks/log-cleanup`
- **THEN** the response SHALL be 200 with the task's full configuration and a `recentExecutions` array

#### Scenario: Task not found
- **WHEN** a GET request is made to `/api/v1/tasks/nonexistent`
- **THEN** the response SHALL be 404 with an error message

### Requirement: List task executions with pagination
The system SHALL expose `GET /api/v1/tasks/:name/executions` to list execution history for a task. It SHALL support `page` and `pageSize` query parameters (default page=1, pageSize=20).

#### Scenario: Paginated history
- **WHEN** a GET request is made to `/api/v1/tasks/log-cleanup/executions?page=1&pageSize=10`
- **THEN** the response SHALL be 200 with a paginated result containing `list` (array of executions), `total`, `page`, `pageSize`

### Requirement: Queue statistics
The system SHALL expose `GET /api/v1/tasks/stats` to return aggregate queue statistics.

#### Scenario: Stats returned
- **WHEN** a GET request is made to `/api/v1/tasks/stats`
- **THEN** the response SHALL be 200 with JSON containing: totalTasks, pending, running, completedToday, failedToday

### Requirement: Pause scheduled task
The system SHALL expose `POST /api/v1/tasks/:name/pause` to pause a scheduled task. Pausing SHALL stop the cron scheduler from triggering the task and update the task's status to `paused`.

#### Scenario: Pause active task
- **WHEN** a POST request is made to `/api/v1/tasks/log-cleanup/pause` and the task is active
- **THEN** the task's status SHALL be updated to `paused`, the cron entry SHALL be removed, and the response SHALL be 200

#### Scenario: Pause already paused task
- **WHEN** a POST request is made to pause a task that is already paused
- **THEN** the response SHALL be 400 with a message indicating the task is already paused

#### Scenario: Pause async task
- **WHEN** a POST request is made to pause a task of type `async`
- **THEN** the response SHALL be 400 with a message indicating only scheduled tasks can be paused

### Requirement: Resume scheduled task
The system SHALL expose `POST /api/v1/tasks/:name/resume` to resume a paused scheduled task. Resuming SHALL re-add the cron entry and update the status to `active`.

#### Scenario: Resume paused task
- **WHEN** a POST request is made to `/api/v1/tasks/log-cleanup/resume` and the task is paused
- **THEN** the task's status SHALL be updated to `active`, the cron entry SHALL be re-added, and the response SHALL be 200

#### Scenario: Resume active task
- **WHEN** a POST request is made to resume a task that is already active
- **THEN** the response SHALL be 400 with a message indicating the task is already active

### Requirement: Trigger task manually
The system SHALL expose `POST /api/v1/tasks/:name/trigger` to manually trigger any registered task (both scheduled and async). It SHALL create a TaskExecution with trigger=manual and submit it immediately.

#### Scenario: Manual trigger
- **WHEN** a POST request is made to `/api/v1/tasks/log-cleanup/trigger`
- **THEN** a new TaskExecution SHALL be created with trigger=manual and status=pending, submitted to the executor, and the response SHALL be 200 with the execution ID

#### Scenario: Trigger paused task
- **WHEN** a POST request is made to trigger a paused scheduled task
- **THEN** it SHALL still execute (manual trigger ignores pause state) and the response SHALL be 200

### Requirement: Casbin permission enforcement
All task API endpoints SHALL be protected by Casbin middleware. Admin role SHALL have full access by default via seed policies.

#### Scenario: Admin access
- **WHEN** an authenticated admin user accesses any task API endpoint
- **THEN** access SHALL be granted

#### Scenario: Unauthorized access
- **WHEN** an authenticated user without task permissions accesses a task API endpoint
- **THEN** the response SHALL be 403
