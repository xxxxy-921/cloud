## ADDED Requirements

### Requirement: Announcement CRUD API
The system SHALL provide announcement management APIs under `/api/v1/announcements`:
- `GET /api/v1/announcements` — list announcements with pagination, including publisher username
- `POST /api/v1/announcements` — create announcement (title required, content optional)
- `PUT /api/v1/announcements/:id` — update announcement title and content
- `DELETE /api/v1/announcements/:id` — delete announcement

All endpoints SHALL require Casbin authorization.

#### Scenario: List announcements
- **WHEN** an authorized user requests GET /api/v1/announcements?page=1&pageSize=20
- **THEN** the system SHALL return announcements ordered by created_at DESC, each with the publisher's username

#### Scenario: Create announcement
- **WHEN** an authorized user sends POST /api/v1/announcements with title="维护通知" and content="今晚维护"
- **THEN** an announcement notification SHALL be created with type="announcement", source="announcement", target_type="all", created_by=current_user_id
- **AND** the response SHALL return the created announcement

#### Scenario: Create announcement without title
- **WHEN** an authorized user sends POST /api/v1/announcements with empty title
- **THEN** the system SHALL return a 400 validation error

#### Scenario: Update announcement
- **WHEN** an authorized user sends PUT /api/v1/announcements/5 with updated title and content
- **THEN** the notification record SHALL be updated with the new title and content

#### Scenario: Delete announcement
- **WHEN** an authorized user sends DELETE /api/v1/announcements/5
- **THEN** the notification SHALL be soft-deleted
- **AND** associated notification_read records SHALL be cleaned up

### Requirement: Announcement Casbin permissions
The following Casbin policies SHALL be seeded for the admin role:
- `(admin, /api/v1/announcements, GET)`
- `(admin, /api/v1/announcements, POST)`
- `(admin, /api/v1/announcements/*, PUT)`
- `(admin, /api/v1/announcements/*, DELETE)`

#### Scenario: Admin accesses announcement management
- **WHEN** a user with admin role requests any announcement API
- **THEN** the request SHALL be authorized by Casbin

#### Scenario: Non-admin denied
- **WHEN** a user without admin role requests POST /api/v1/announcements
- **THEN** the request SHALL be denied with 403

### Requirement: Announcement management page
The frontend SHALL provide an announcement management page at `/announcements` route, accessible via menu navigation. The page SHALL include:
- Data table with columns: Title, Publisher, Published At, Actions
- Create button (visible only with create permission)
- Edit action per row (visible only with update permission) opening a dialog/drawer
- Delete action per row (visible only with delete permission) with confirmation dialog
- Search by keyword
- Pagination

#### Scenario: Admin views announcement page
- **WHEN** admin navigates to /announcements
- **THEN** the page SHALL display a paginated table of announcements with CRUD actions

#### Scenario: User without permission
- **WHEN** a user without announcement:list permission tries to access /announcements
- **THEN** the route SHALL be protected by PermissionGuard

#### Scenario: Create new announcement
- **WHEN** admin clicks "新建公告" and fills in the form
- **THEN** a dialog/drawer SHALL appear with title (required) and content (optional) fields
- **AND** on submit, the announcement SHALL be created and the table SHALL refresh

#### Scenario: Delete with confirmation
- **WHEN** admin clicks the delete action on an announcement
- **THEN** a confirmation dialog SHALL appear before the deletion proceeds

### Requirement: Announcement menu item
A menu item for "公告管理" SHALL be seeded in the system management menu group with appropriate icon and permission key `system:announcement:list`.

#### Scenario: Menu item visible to admin
- **WHEN** an admin user loads the sidebar navigation
- **THEN** "公告管理" SHALL appear under the system management group

#### Scenario: Menu item hidden from unauthorized user
- **WHEN** a user without announcement permissions loads the sidebar
- **THEN** "公告管理" SHALL NOT appear in the navigation
