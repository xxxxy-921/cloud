## ADDED Requirements

### Requirement: 我的工单引擎类型标识
"我的工单"列表 SHALL 在每行展示引擎类型标识，Smart 引擎工单显示 🤖 图标或"智能"标签，Classic 引擎工单显示"经典"标签或不标识。

#### Scenario: Smart 工单显示标识
- **WHEN** 用户查看"我的工单"列表，其中包含 engineType=smart 的工单
- **THEN** 该行展示 Smart 引擎标识

### Requirement: 我的工单关键词搜索
"我的工单"列表 SHALL 支持关键词搜索，搜索范围包括 code、title、description 字段。

#### Scenario: 搜索匹配
- **WHEN** 用户在"我的工单"输入关键词"VPN"
- **THEN** 列表仅显示 code/title/description 包含"VPN"的工单

### Requirement: 我的待办多维参与者查询
"我的待办"后端查询 SHALL 使用多维参与者解析，通过 JOIN TicketAssignment 匹配 `user_id = currentUser OR position_id IN userPositions OR department_id IN userDepts`，替代当前仅按 `assignee_id` 过滤的逻辑。查询 SHALL 限定为活跃状态工单（`status IN {pending, in_progress, waiting_approval}`）。

#### Scenario: 按用户直接匹配
- **WHEN** 用户查看"我的待办"，有一个工单的 assignment.userId 等于当前用户
- **THEN** 该工单出现在待办列表中

#### Scenario: 按岗位匹配
- **WHEN** 用户持有"运维管理员"岗位，有一个工单的 assignment.positionId 匹配该岗位
- **THEN** 该工单出现在待办列表中

#### Scenario: 无关工单不显示
- **WHEN** 工单的 assignment 不匹配当前用户的任何维度
- **THEN** 该工单不出现在待办列表中

### Requirement: 我的待办列表增强
"我的待办"列表 SHALL 增加以下列和筛选能力：
- 新增"当前活动"列，展示当前活动名称
- 支持关键词搜索（code、title）
- 支持状态过滤（Tab 或下拉）

#### Scenario: 展示当前活动列
- **WHEN** 用户查看待办列表
- **THEN** 每行展示当前活动名称（如"运维管理员审批"）

#### Scenario: 关键词搜索
- **WHEN** 用户在待办列表输入关键词
- **THEN** 列表按 code/title 过滤结果

### Requirement: 历史工单用户范围限定
"历史工单"后端查询 SHALL 增加用户范围过滤，仅返回 `requester_id = currentUser OR assignee_id = currentUser` 的终态工单。管理员查看全局历史 SHALL 使用"全部工单"页面。

#### Scenario: 仅展示我参与的历史工单
- **WHEN** 用户查看"历史工单"，数据库中有 100 个已完成工单，其中 5 个由该用户提交，3 个由该用户处理
- **THEN** 列表展示 8 条记录（去重后）

#### Scenario: 其他人的工单不显示
- **WHEN** 历史工单中有一个工单，requester 和 assignee 都不是当前用户
- **THEN** 该工单不出现在历史列表中
