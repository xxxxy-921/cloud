## Why

组织管理模块（`add-org-position-module`）已完成 V1 实现，但人员分配子系统存在两个问题：

1. **API 并发安全缺陷** — 当前 `PUT /users/:id/positions` 采用全量替换（DELETE ALL + INSERT），在部门维度 UI 下，两个管理员同时操作同一用户的不同部门会导致后者覆盖前者的修改，且无任何冲突提示。
2. **分配页面体验粗糙** — 左侧部门树缺少成员计数和选中反馈，右侧成员表信息密度不足（无头像、邮箱、分配时间），添加成员的交互流程繁琐，无空状态引导，整体视觉层次与系统其他页面（Users、Nodes）存在明显差距。

## What Changes

- **Assignment API 改为增量操作** — 新增 `POST`（添加分配）和 `DELETE`（移除分配）端点，替代全量 PUT。消除并发竞态风险，同时简化前端调用逻辑。
- **重新设计分配页面 UI** — 左侧部门树增加成员计数 Badge、部门搜索过滤、选中高亮；右侧成员表增加用户头像+邮箱双行展示、主岗/兼岗 Badge、分配时间列、DropdownMenu 操作菜单；完善空状态引导。
- **新增用户组织信息 Sheet** — 从成员操作菜单可打开只读 Sheet，查看该用户在所有部门的完整分配情况。
- **停用部门 scope 过滤** — `GetUserDepartmentScope` 排除 `IsActive=false` 的部门。
- **ManagerID 防御性处理** — 部门查询时对已删除/停用的 Manager 做 graceful 降级显示。

## Capabilities

### New Capabilities

- `assignment-api-incremental`: 人员分配增量 API（POST 添加、DELETE 移除），替代全量替换
- `assignment-ui-redesign`: 人员分配页面 UI 重设计（部门树增强、成员表丰富化、用户组织 Sheet、空状态）

### Modified Capabilities

- `org-assignment`: scope helper 过滤停用部门；ManagerID 防御性查询

## Impact

- **后端** — `internal/app/org/assignment_handler.go`、`assignment_service.go`、`assignment_repository.go` 新增/修改端点；`department_service.go` scope helper 修改
- **前端** — `web/src/apps/org/pages/assignments/index.tsx` 重写；新增用户组织信息 Sheet 组件
- **API 变更** — `PUT /org/users/:id/positions`（全量替换）废弃，改为 `POST` + `DELETE` 增量操作。前端是唯一调用方，无外部依赖，非 breaking change。
- **Casbin 策略** — 新增端点需要补充对应的权限策略
- **国际化** — 新增/修改 zh-CN 和 en 翻译条目
