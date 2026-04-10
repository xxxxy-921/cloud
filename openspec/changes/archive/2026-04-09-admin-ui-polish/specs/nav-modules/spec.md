## MODIFIED Requirements

### Requirement: TopNav bar
The TopNav bar SHALL be fixed at the top with height h-14. It SHALL display the sidebar toggle button and site logo/name on the left, and user actions on the right. The user dropdown menu SHALL be implemented using shadcn `DropdownMenu` component (not a hand-rolled dropdown), providing keyboard navigation, ARIA roles, theme-aware styling, and click-outside dismissal.

#### Scenario: User opens dropdown menu
- **WHEN** user clicks the username button in TopNav
- **THEN** a `DropdownMenu` SHALL open with "修改密码" and "退出登录" items
- **AND** the menu SHALL support keyboard navigation (Arrow keys, Enter, Escape)

#### Scenario: User changes password via dropdown
- **WHEN** user selects "修改密码" from the dropdown
- **THEN** the ChangePasswordDialog SHALL open

#### Scenario: User logs out via dropdown
- **WHEN** user selects "退出登录" from the dropdown
- **THEN** the system SHALL call logout and navigate to `/login`
