## ADDED Requirements

### Requirement: Task center menu entry
The seed data SHALL include a "任务中心" menu entry under "系统管理" directory with path `/tasks`, icon `Clock`, permission `system:task:list`, and sort order 5.

#### Scenario: Menu seeded
- **WHEN** the database is seeded
- **THEN** a menu item "任务中心" SHALL exist under "系统管理" with type=menu, path=/tasks, permission=system:task:list

### Requirement: Task center button permissions
The seed data SHALL include button-level permissions under "任务中心": "暂停任务" (system:task:pause), "恢复任务" (system:task:resume), "触发任务" (system:task:trigger).

#### Scenario: Button permissions seeded
- **WHEN** the database is seeded
- **THEN** three button menu items SHALL exist under "任务中心" with the specified permissions

### Requirement: Admin Casbin policies for task APIs
The admin role seed SHALL include Casbin policies for all task API endpoints: GET /api/v1/tasks, GET /api/v1/tasks/stats, GET /api/v1/tasks/:name, GET /api/v1/tasks/:name/executions, POST /api/v1/tasks/:name/pause, POST /api/v1/tasks/:name/resume, POST /api/v1/tasks/:name/trigger.

#### Scenario: Admin policies seeded
- **WHEN** the database is seeded
- **THEN** the admin role SHALL have Casbin policies for all 7 task API endpoints
