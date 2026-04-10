## REMOVED Requirements

### Requirement: Config list page
**Reason**: The raw KV config editor exposes direct database manipulation, replaced by typed settings APIs and settings page tabs.
**Migration**: System configuration is now managed through the settings page at /settings with dedicated tabs for each config category (security, scheduler). Backend reads/writes go through SettingsService.

### Requirement: Create config via Sheet drawer
**Reason**: No longer needed — config entries are managed through typed settings APIs.
**Migration**: Use PUT /api/v1/settings/security or PUT /api/v1/settings/scheduler to update specific settings.

### Requirement: Edit config via Sheet drawer
**Reason**: No longer needed — config entries are managed through typed settings APIs.
**Migration**: Use the settings page tabs to edit security or scheduler settings.

### Requirement: Delete config with confirmation
**Reason**: No longer needed — config keys are managed internally by the system, not user-deletable.
**Migration**: No migration needed. Config keys are lifecycle-managed by the application.
