## ADDED Requirements

### Requirement: Notification data model
The system SHALL maintain a `notification` table with fields: id, type (string), source (string), title (string), content (text, nullable), target_type ("all" | "user"), target_id (nullable uint), created_by (nullable uint), and BaseModel timestamps. The system SHALL maintain a `notification_read` table with fields: id (uint PK), notification_id (uint FK), user_id (uint FK), read_at (timestamp), with a unique constraint on (notification_id, user_id).

#### Scenario: Broadcast notification created
- **WHEN** a notification is created with target_type="all"
- **THEN** a single notification record SHALL be stored, visible to all users

#### Scenario: Targeted notification created
- **WHEN** a notification is created with target_type="user" and target_id=42
- **THEN** the notification SHALL only be visible to user 42

#### Scenario: Read status tracked per user
- **WHEN** user A marks a broadcast notification as read
- **THEN** a notification_read record SHALL be created for (notification_id, user_A_id)
- **AND** the notification SHALL still show as unread for user B

### Requirement: Unified send interface
The NotificationService SHALL expose a `Send(type, source, title, content, targetType, targetID, createdBy)` method that any module can call to create notifications.

#### Scenario: Module sends notification
- **WHEN** the announcement module calls `NotificationService.Send("announcement", "announcement", "维护通知", "今晚22点维护", "all", nil, adminUserID)`
- **THEN** a notification record SHALL be created with the provided fields

### Requirement: Notification list API
The system SHALL provide `GET /api/v1/notifications` that returns paginated notifications for the authenticated user. Each item SHALL include an `isRead` boolean derived from LEFT JOIN with notification_read. Results SHALL be ordered by created_at DESC. The API SHALL accept `page` and `pageSize` query parameters.

#### Scenario: User views notification list
- **WHEN** an authenticated user requests GET /api/v1/notifications?page=1&pageSize=20
- **THEN** the system SHALL return notifications where target_type="all" OR (target_type="user" AND target_id=current_user_id), with isRead status for each

#### Scenario: Soft-deleted notifications excluded
- **WHEN** a notification has been soft-deleted
- **THEN** it SHALL NOT appear in the notification list

### Requirement: Unread count API
The system SHALL provide `GET /api/v1/notifications/unread-count` that returns the count of unread notifications for the authenticated user.

#### Scenario: User has unread notifications
- **WHEN** there are 5 broadcast notifications and user has read 2
- **THEN** the unread count SHALL return 3

#### Scenario: No unread notifications
- **WHEN** user has read all notifications
- **THEN** the unread count SHALL return 0

### Requirement: Mark as read API
The system SHALL provide `PUT /api/v1/notifications/:id/read` to mark a single notification as read, and `PUT /api/v1/notifications/read-all` to mark all unread notifications as read for the authenticated user. Both operations SHALL be idempotent.

#### Scenario: Mark single notification as read
- **WHEN** user sends PUT /api/v1/notifications/5/read
- **THEN** a notification_read record SHALL be created (or no-op if already read)
- **AND** the unread count SHALL decrease by 1 (if was unread)

#### Scenario: Mark all as read
- **WHEN** user sends PUT /api/v1/notifications/read-all
- **THEN** notification_read records SHALL be created for all unread notifications of the user

#### Scenario: Idempotent read marking
- **WHEN** user marks an already-read notification as read again
- **THEN** no error SHALL occur and no duplicate record SHALL be created

### Requirement: Notification APIs skip Casbin
The notification center APIs (`/api/v1/notifications/*`) SHALL be whitelisted in CasbinAuth middleware. Only JWT authentication SHALL be required.

#### Scenario: Regular user accesses notifications
- **WHEN** an authenticated user with no special permissions requests GET /api/v1/notifications
- **THEN** the request SHALL succeed without Casbin permission check

### Requirement: Notification bell UI
The frontend SHALL display a bell icon in the TopNav header, positioned to the left of the user dropdown. The bell SHALL show a badge with the unread notification count. Counts above 99 SHALL display as "99+". The badge SHALL be hidden when count is 0.

#### Scenario: Unread notifications exist
- **WHEN** user has 5 unread notifications
- **THEN** the bell icon SHALL display a badge with "5"

#### Scenario: Over 99 unread
- **WHEN** user has 150 unread notifications
- **THEN** the badge SHALL display "99+"

#### Scenario: No unread notifications
- **WHEN** user has 0 unread notifications
- **THEN** no badge SHALL be displayed on the bell icon

### Requirement: Notification popover
Clicking the bell icon SHALL open a Popover (~400px wide) displaying the notification list. Each item SHALL show: unread indicator (blue dot), type icon, title, content preview (truncated), and relative time. The popover SHALL include a "全部已读" button in the header. The popover SHALL show "暂无通知" when empty.

#### Scenario: User opens notification popover
- **WHEN** user clicks the bell icon
- **THEN** a Popover SHALL appear showing the most recent notifications with isRead visual indicators

#### Scenario: User clicks a notification item
- **WHEN** user clicks an unread notification in the popover
- **THEN** the notification SHALL be marked as read and the blue dot SHALL disappear

#### Scenario: User clicks "全部已读"
- **WHEN** user clicks the "全部已读" button
- **THEN** all notifications SHALL be marked as read, all blue dots SHALL disappear, and the badge count SHALL reset to 0

#### Scenario: Empty state
- **WHEN** the user has no notifications
- **THEN** the popover SHALL display "暂无通知"

### Requirement: Polling for unread count
The frontend SHALL poll `GET /api/v1/notifications/unread-count` every 30 seconds using React Query's `refetchInterval`. Polling SHALL pause when the browser tab is not focused.

#### Scenario: Periodic polling
- **WHEN** 30 seconds have elapsed since the last poll
- **THEN** the frontend SHALL fetch the updated unread count

#### Scenario: Tab loses focus
- **WHEN** the browser tab is backgrounded
- **THEN** polling SHALL pause until the tab regains focus
