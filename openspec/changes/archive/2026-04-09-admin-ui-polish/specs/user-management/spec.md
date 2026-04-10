## MODIFIED Requirements

### Requirement: User management page table actions
The user management page table SHALL display row actions in a `DropdownMenu` triggered by a `MoreHorizontal` icon button, instead of inline buttons. The menu SHALL contain: "编辑", "停用/启用", a separator, and "删除" (in destructive styling). The delete action SHALL still trigger an AlertDialog for confirmation.

#### Scenario: User clicks action menu
- **WHEN** user clicks the MoreHorizontal button on a user row
- **THEN** a DropdownMenu SHALL open with edit, toggle-active, and delete options

#### Scenario: Delete from action menu
- **WHEN** user selects "删除" from the action menu
- **THEN** an AlertDialog SHALL appear for confirmation before deletion

### Requirement: Role management page table actions
The role management page table SHALL display row actions in a `DropdownMenu` triggered by a `MoreHorizontal` icon button. The menu SHALL contain: "权限", "编辑", a separator, and "删除" (in destructive styling, disabled for system roles).

#### Scenario: Role action menu
- **WHEN** user clicks the MoreHorizontal button on a role row
- **THEN** a DropdownMenu SHALL open with permission-assign, edit, and delete options

### Requirement: Form selects use shadcn Select
All form select inputs in Sheet forms SHALL use shadcn `Select` component instead of native `<select>` elements, ensuring visual consistency with other shadcn form components.

#### Scenario: Menu sheet parent and type selects
- **WHEN** the menu sheet form is displayed
- **THEN** the parent menu selector and type selector SHALL use shadcn `Select` components

#### Scenario: User sheet role select
- **WHEN** the user sheet form is displayed
- **THEN** the role selector SHALL use a shadcn `Select` component

### Requirement: Role sheet cancel button
The role edit/create Sheet SHALL include a "取消" (cancel) button in the footer alongside the "保存" button, consistent with the permission dialog footer.

#### Scenario: Role sheet footer buttons
- **WHEN** the role sheet is open
- **THEN** the footer SHALL display both "取消" and "保存" buttons
