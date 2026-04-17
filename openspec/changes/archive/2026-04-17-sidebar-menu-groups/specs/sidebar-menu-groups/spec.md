## ADDED Requirements

### Requirement: App module supports optional menu group declaration
AppModule interface SHALL accept an optional `menuGroups` field. Each menu group SHALL have a `label` (i18n key) and an `items` array of permission strings identifying which menu items belong to the group.

#### Scenario: App registers with menu groups
- **WHEN** an App calls `registerApp()` with a `menuGroups` array
- **THEN** the groups are stored and retrievable via `getMenuGroups(appName)`

#### Scenario: App registers without menu groups
- **WHEN** an App calls `registerApp()` without `menuGroups`
- **THEN** `getMenuGroups(appName)` SHALL return `undefined`

### Requirement: Sidebar renders grouped menu items when groups are declared
When the active App has `menuGroups` configured, the Tier 2 nav panel SHALL render menu items organized by group, with a visual group label preceding each group's items.

#### Scenario: Active app has menu groups
- **WHEN** the active App has `menuGroups` defined
- **THEN** Tier 2 renders items bucketed into groups, each group preceded by a label header
- **THEN** items within each group maintain their original sort order

#### Scenario: Active app has no menu groups
- **WHEN** the active App has no `menuGroups` (undefined or empty)
- **THEN** Tier 2 renders all items as a flat list (current behavior, unchanged)

### Requirement: Ungrouped items fall back to end of list
Menu items whose `permission` does not match any group's `items` array SHALL be rendered after all groups, without a group label.

#### Scenario: Menu item not in any group
- **WHEN** a visible menu item's `permission` is not listed in any group's `items`
- **THEN** the item appears after all grouped sections, without a label header

### Requirement: Group labels support i18n
Group labels SHALL be resolved through i18n using the key pattern `menu.group.{appName}.{label}`, falling back to the raw `label` string if no translation exists.

#### Scenario: Translation exists for group label
- **WHEN** the i18n key `menu.group.itsm.service` has a translation
- **THEN** the group header displays the translated text

#### Scenario: No translation for group label
- **WHEN** the i18n key has no translation
- **THEN** the group header displays the raw label string

### Requirement: ITSM declares three menu groups
The ITSM App SHALL declare three menu groups in its `registerApp()` call:
- **service**: `itsm:catalog:list`, `itsm:service:list`, `itsm:form:list`
- **ticket**: `itsm:ticket:list`, `itsm:ticket:mine`, `itsm:ticket:todo`, `itsm:ticket:history`, `itsm:ticket:approvals`
- **config**: `itsm:priority:list`, `itsm:sla:list`, `itsm:engine:config`

#### Scenario: ITSM sidebar displays grouped menus
- **WHEN** user navigates to any ITSM page
- **THEN** the Tier 2 panel shows three labeled sections: 服务 (service), 工单 (ticket), 配置 (config)
- **THEN** each section contains the corresponding menu items in their original sort order

### Requirement: Group label visual style
Group labels SHALL be rendered as non-interactive text with muted styling: small font size (11px), semibold weight, wider letter spacing, muted foreground color. Groups after the first SHALL have top spacing for visual separation.

#### Scenario: Visual rendering of group labels
- **WHEN** Tier 2 renders grouped items
- **THEN** group labels appear as small, muted, uppercase (for Latin text) headers
- **THEN** the first group has no extra top spacing
- **THEN** subsequent groups have additional top spacing for separation
