## MODIFIED Requirements

### Requirement: My Approvals page
The system SHALL provide a "我的审批" page at route `/itsm/tickets/approvals` displaying a table of pending approval items. The page SHALL render two kinds of items with visual distinction:

1. **Workflow approvals** (`approvalKind: "workflow"`): 标准审批行，展示 Ticket Code, Title, Service, Priority, SLA Badge, Activity Name, Created At, 和内联 [通过] [驳回] 按钮
2. **AI decision confirmations** (`approvalKind: "ai_confirm"`): AI 确认行，展示 Ticket Code, Title, Service, Priority, AI 置信度（百分比 + 色彩编码：绿 ≥80% / 黄 50-80% / 红 <50%）, Activity Name, Created At, 和内联 [确认] [拒绝] 按钮

Table SHALL support pagination. AI 确认行 SHALL 有视觉区分（如 🤖 图标或不同背景色）。

#### Scenario: View mixed approval list
- **WHEN** user navigates to "我的审批" and has both workflow approvals and AI confirmations
- **THEN** page displays all items with visual distinction between the two types

#### Scenario: AI confirmation inline actions
- **WHEN** user clicks [确认] on an AI confirmation item
- **THEN** system calls confirmActivity API, removes the row, shows success toast

#### Scenario: AI confirmation rejection
- **WHEN** user clicks [拒绝] on an AI confirmation item and enters reason
- **THEN** system calls rejectActivity API with reason, removes the row, shows success toast

#### Scenario: Empty approval list
- **WHEN** user has no pending approvals nor AI confirmations
- **THEN** page shows empty state message "暂无待审批工单"

### Requirement: Inline approve/deny actions
The "我的审批" table SHALL provide inline action buttons for each row:
- Workflow approvals: "通过" (approve) and "驳回" (deny). "驳回" SHALL open a popover for entering denial reason.
- AI confirmations: "确认" (confirm) and "拒绝" (reject). "拒绝" SHALL open a popover for entering rejection reason.

After action completion, the row SHALL be removed from the list and a success toast shown.

#### Scenario: Approve workflow item from list
- **WHEN** user clicks "通过" button on a workflow approval item
- **THEN** system calls approve API, removes the row from list, shows success toast

#### Scenario: Deny workflow item with reason from list
- **WHEN** user clicks "驳回" button, enters reason "不符合规范", and confirms
- **THEN** system calls deny API with reason, removes the row, shows success toast

#### Scenario: Confirm AI decision from list
- **WHEN** user clicks "确认" button on an AI confirmation item
- **THEN** system calls confirmActivity API, removes the row, shows success toast

#### Scenario: Reject AI decision with reason from list
- **WHEN** user clicks "拒绝" button on an AI confirmation item, enters reason, and confirms
- **THEN** system calls rejectActivity API with reason, removes the row, shows success toast
