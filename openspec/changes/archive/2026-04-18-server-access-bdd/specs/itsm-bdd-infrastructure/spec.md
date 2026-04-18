## ADDED Requirements

### Requirement: 审批岗位分配断言 step

系统 SHALL 提供通用 BDD step `当前审批分配到岗位 "<position_code>"`，断言当前活动的 TicketAssignment 中 position 的 code 匹配期望值。该 step 不绑定具体服务类型，可被所有 BDD 场景复用。

#### Scenario: 岗位分配断言匹配
- **WHEN** 当前活动有 TicketAssignment 且关联的 Position.Code 为 "ops_admin"
- **AND** 执行断言 `当前审批分配到岗位 "ops_admin"`
- **THEN** 断言通过

#### Scenario: 岗位分配断言不匹配
- **WHEN** 当前活动有 TicketAssignment 且关联的 Position.Code 为 "network_admin"
- **AND** 执行断言 `当前审批分配到岗位 "ops_admin"`
- **THEN** 断言失败并报告实际岗位

### Requirement: 审批可见性断言 step

系统 SHALL 提供通用 BDD step `当前审批仅对 "<username>" 可见`，断言当前活动的 TicketAssignment 通过 position_department 解析后，仅指定用户在可处理人列表中。

#### Scenario: 可见性断言通过
- **WHEN** 当前审批分配到 it/ops_admin 岗位，且 ops-operator 是该岗位唯一成员
- **AND** 执行断言 `当前审批仅对 "ops-operator" 可见`
- **THEN** 断言通过（ops-operator 可见，network-operator 和 security-operator 不可见）

### Requirement: 越权认领失败断言 step

系统 SHALL 提供通用 BDD step `"<username>" 认领当前工单应失败`，尝试让指定用户认领当前活动的 assignment，断言操作失败（用户不在该 assignment 的 position_department 可处理人中）。

#### Scenario: 非目标岗位用户认领失败
- **WHEN** 当前审批分配到 it/ops_admin 岗位
- **AND** network-operator（属于 it/network_admin）尝试认领
- **THEN** 认领操作失败

### Requirement: 越权审批失败断言 step

系统 SHALL 提供通用 BDD step `"<username>" 审批当前工单应失败`，尝试让指定用户直接审批当前活动，断言操作失败。

#### Scenario: 非目标岗位用户审批失败
- **WHEN** 当前审批分配到 it/ops_admin 岗位
- **AND** security-operator（属于 it/security_admin）尝试审批
- **THEN** 审批操作失败
