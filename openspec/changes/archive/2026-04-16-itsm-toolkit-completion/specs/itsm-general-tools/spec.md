## ADDED Requirements

### Requirement: general.current_time 工具
系统 SHALL 在 AI App seed 中注册 `general.current_time` 工具（toolkit: "general"），用于获取当前时间（多时区）。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "timezone": { "type": "string", "description": "IANA 时区名（如 Asia/Shanghai），可选，默认返回多个时区" }
  }
}
```

**返回结构**:
```json
{
  "server_time": "2026-04-16T14:30:00+08:00",
  "utc_time": "2026-04-16T06:30:00Z",
  "china_formatted_time": "2026-04-16 14:30:00",
  "target_time": "2026-04-16T14:30:00+08:00",
  "target_timezone": "Asia/Shanghai"
}
```

#### Scenario: 不传时区
- **WHEN** Agent 调用 general.current_time 不传 timezone 参数
- **THEN** 系统 SHALL 返回服务器时间、UTC 时间、中国时间（UTC+8），target_time 和 target_timezone 为空

#### Scenario: 传入 IANA 时区
- **WHEN** Agent 调用 general.current_time，传入 timezone="America/New_York"
- **THEN** 系统 SHALL 额外返回该时区的 target_time

#### Scenario: 无效时区
- **WHEN** Agent 调用 general.current_time，传入无效的 timezone 值
- **THEN** 系统 SHALL 返回错误 `{"error": "无效的时区名称"}`

### Requirement: system.current_user_profile 工具
系统 SHALL 在 AI App seed 中注册 `system.current_user_profile` 工具（toolkit: "general"），用于获取当前会话用户的完整档案。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {}
}
```

**返回结构**:
```json
{
  "user": { "id": 1, "username": "zhangsan", "email": "z@example.com", "phone": "13800138000" },
  "department": { "id": 3, "code": "it", "name": "信息部" },
  "manager": { "id": 2, "username": "admin" },
  "positions": [ { "id": 1, "code": "it_admin", "name": "IT管理员", "is_primary": true } ],
  "role": { "id": 1, "code": "admin", "name": "管理员" },
  "missing_fields": []
}
```

#### Scenario: 用户有完整组织信息
- **WHEN** Agent 调用 system.current_user_profile，当前用户有部门、岗位、角色分配
- **THEN** 系统 SHALL 返回完整用户档案，missing_fields 为空列表

#### Scenario: 用户缺少组织信息
- **WHEN** Agent 调用 system.current_user_profile，当前用户未分配部门或岗位
- **THEN** 系统 SHALL 返回基础用户信息，department 和 positions 为 null/空，missing_fields 包含 `["department", "positions"]`

#### Scenario: Org App 未安装
- **WHEN** Agent 调用 system.current_user_profile，但 Org App 未安装（edition 不包含）
- **THEN** 系统 SHALL 返回仅包含 user 和 role 的基础信息，department/manager/positions 为 null，missing_fields 包含 `["department", "positions"]`

### Requirement: organization.org_context 工具
系统 SHALL 在 AI App seed 中注册 `organization.org_context` 工具（toolkit: "general"），用于查询组织架构信息。

**inputSchema**:
```json
{
  "type": "object",
  "properties": {
    "username": { "type": "string", "description": "按用户名查询" },
    "department_code": { "type": "string", "description": "按部门代码筛选" },
    "position_code": { "type": "string", "description": "按岗位代码筛选" },
    "include_inactive": { "type": "boolean", "description": "是否包含停用记录，默认 false" }
  }
}
```

**返回结构**:
```json
{
  "filters": { "department_code": "it" },
  "users": [ { "id": 1, "username": "admin", "email": "...", "department": {...}, "positions": [...] } ],
  "departments": [ { "id": 3, "code": "it", "name": "信息部", "parent_code": "headquarters" } ],
  "positions": [ { "id": 1, "code": "it_admin", "name": "IT管理员" } ],
  "summary": "用户 2 个；部门 1 个；岗位 3 个"
}
```

#### Scenario: 按部门查询
- **WHEN** Agent 调用 organization.org_context，传入 department_code="it"
- **THEN** 系统 SHALL 返回 IT 部门下的所有用户、相关岗位信息

#### Scenario: 按岗位查询
- **WHEN** Agent 调用 organization.org_context，传入 position_code="network_admin"
- **THEN** 系统 SHALL 返回所有担任网络管理员岗位的用户

#### Scenario: 按用户名查询
- **WHEN** Agent 调用 organization.org_context，传入 username="admin"
- **THEN** 系统 SHALL 返回该用户的完整组织信息（部门、岗位、角色）

#### Scenario: 组合筛选
- **WHEN** Agent 调用 organization.org_context，同时传入 department_code 和 position_code
- **THEN** 系统 SHALL 返回满足 AND 条件的结果

#### Scenario: Org App 未安装
- **WHEN** Agent 调用 organization.org_context，但 Org App 未安装
- **THEN** 系统 SHALL 返回 `{"users": [], "departments": [], "positions": [], "summary": "组织管理模块未安装"}`

### Requirement: 通用工具自动绑定 ITSM 智能体
AI App seed 中注册的通用工具 SHALL 在 ITSM App seed 中自动绑定到服务台智能体。

#### Scenario: 服务台智能体绑定通用工具
- **WHEN** ITSM App 执行 seed 且 AI App 可用
- **THEN** 系统 SHALL 将 general.current_time、system.current_user_profile、organization.org_context 三个工具绑定到 IT 服务台智能体

#### Scenario: 通用工具不存在时跳过绑定
- **WHEN** ITSM App 执行 seed 但通用工具尚未注册（AI App seed 顺序问题）
- **THEN** 系统 SHALL 跳过绑定，输出 warn 级别日志
