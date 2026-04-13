## Context

组织管理模块（`add-org-position-module`）已完成 V1 交付，包含部门管理、岗位管理、人员分配三个子模块。当前人员分配子系统采用全量替换 API（`PUT /users/:id/positions` → DELETE ALL + INSERT），前端在部门维度操作时需要先 GET 用户全部分配、拼接修改、再 PUT 回去，存在并发覆盖风险。

前端分配页面是最小化实现：左侧部门树无计数/搜索，右侧成员表仅显示用户名和岗位，添加成员流程需要多步 API 调用。与系统其他页面（Users、Nodes）的交互质量有明显差距。

约束：
- 前端唯一调用方，无外部 API 消费者，API 变更不构成 breaking change
- 保持 App 可插拔架构，不侵入内核
- 遵循现有 UI 模式（Table、Sheet、shadcn/ui）

## Goals / Non-Goals

**Goals:**
- 消除人员分配 API 的并发竞态风险
- 将分配页面 UI 提升到与系统其他页面一致的质量水平
- 提供用户组织全景视图（Sheet 形式）
- scope helper 正确过滤停用部门

**Non-Goals:**
- 不做拖拽式组织架构图
- 不做变更历史/审计追踪（留给 V2）
- 不修改 `(UserID, DepartmentID)` 唯一约束（保持同部门单岗位）
- 不修改部门/岗位管理页面（仅改分配页面）

## Decisions

### D1: 增量 API 替代全量替换

**选择**：新增 `POST /org/users/:id/positions`（添加）和 `DELETE /org/users/:id/positions/:assignmentId`（移除），废弃全量 PUT。

**替代方案**：
- A) 保留全量 PUT + 乐观锁（version 字段）→ 冲突时返回 409，前端需要重试/合并逻辑，复杂度高
- B) 保留全量 PUT + 无保护 → 接受数据丢失风险
- C) 增量 API（选择此方案）→ 从根本上消除竞态，前端逻辑也更简单

**理由**：增量 API 是最简方案。每个操作原子化，不需要客户端先读后写。后端改动量小（拆分现有 repository 方法），前端改动量实际减少（不需要拼接逻辑）。

### D2: 保留 Table 布局，增强列内容

**选择**：成员列表保留 DataTable，不改为 Card List。

**理由**：
- Card List 信息密度低，成员多时滚动量大
- 与 Users、Nodes 等现有页面保持一致
- Table 支持排序、分页等标准交互

**增强内容**：成员列双行显示（头像+名称 / 邮箱），新增主岗/兼岗 Badge 列、分配时间列、DropdownMenu 操作列。

### D3: 部门树成员计数 — 前端计算 vs 后端接口

**选择**：后端在 tree 接口中返回每个部门的 `memberCount` 字段。

**替代方案**：前端拿到 tree 后逐个查询 → N+1 请求，不可接受。

**实现**：`GET /org/departments/tree` 响应中每个节点增加 `memberCount` 字段。后端一次性 `GROUP BY department_id` 查出计数，O(1) 额外查询。

### D4: 用户组织信息 Sheet — 复用现有 API

**选择**：从成员操作菜单打开只读 Sheet，调用现有 `GET /org/users/:id/positions` 展示全部分配。

**理由**：无需新建 API，复用已有端点。Sheet 组件约 50 行代码，成本极低。

## Risks / Trade-offs

- **[旧 PUT 端点废弃]** → 直接删除，前端是唯一调用方，无兼容性问题。若后续有外部集成需求，可以重新评估。
- **[memberCount 一致性]** → 非事务性计数，添加/移除成员后需要 invalidate departments/tree query。可接受的短暂不一致（毫秒级）。
- **[停用部门 scope 过滤]** → 改变 `GetUserDepartmentScope` 行为，现有调用方需验证是否受影响。当前无外部调用方，风险低。
