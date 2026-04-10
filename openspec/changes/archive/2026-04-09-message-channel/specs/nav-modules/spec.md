## MODIFIED Requirements

### Requirement: Navigation config split by App
Navigation configuration SHALL be organized as one file per App in lib/nav/ directory.

#### Scenario: Adding a new app
- **WHEN** a developer creates a new app file in lib/nav/ and imports it in index.ts
- **THEN** the new app SHALL appear in the Icon Rail and Nav Panel

#### Scenario: Adding a nav item to existing app
- **WHEN** a developer adds a NavItemDef to an app file
- **THEN** the new item SHALL appear in that app's Nav Panel

#### Scenario: Announcement nav item added to system management
- **WHEN** the system management app navigation is loaded
- **THEN** it SHALL include a "公告管理" item pointing to /announcements with the Megaphone icon

#### Scenario: Message channel nav item added to system management
- **WHEN** the system management app navigation is loaded
- **THEN** it SHALL include a "消息通道" item pointing to /channels with the Mail icon, positioned after "公告管理"
