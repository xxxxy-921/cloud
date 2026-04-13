## Why

当前 Metis 的内核仅有扁平的 User/Role 体系，缺乏组织架构支撑。随着业务扩展，管理员需要在系统中管理部门层级、岗位字典，并支持一人多岗的灵活分配。这为后续的数据权限隔离（如"只能查看本部门数据"）奠定基础。

## What Changes

- 新增可插拔 App `org`，包含部门管理、岗位管理、人员分配三大子模块
- 引入自关联树形的 `Department` 模型和岗位字典 `Position` 模型
- 引入 `UserPosition` 关联表支持一人多岗，并标记主岗
- 新增前端页面：部门管理、岗位管理、人员分配
- App 提供 scope helper 供未来其他模块做数据范围隔离
- 不修改内核 `User`/`Role` 模型，保持 App 可裁剪

## Capabilities

### New Capabilities
- `org-department`: 部门管理（自关联树形结构，支持增删改查、部门负责人）
- `org-position`: 岗位管理（岗位字典，职级、编码、状态）
- `org-assignment`: 人员分配（用户与部门/岗位的多对多关联，支持主岗标记）
- `org-department-ui`: 部门管理前端页面（树形表格、Sheet 表单）
- `org-position-ui`: 岗位管理前端页面（DataTable、Sheet 表单）
- `org-assignment-ui`: 人员分配前端页面（按部门筛选成员、分配岗位）

### Modified Capabilities
- （无现有 spec 需求变更，此模块纯增量）

## Impact

- 后端：新增 `internal/app/org/` 目录，包含 model、repo、service、handler、seed
- 前端：新增 `web/src/apps/org/` 目录及路由注册
- 数据库：新增 `departments`、`positions`、`user_positions` 表
- 菜单与权限：新增 `org` 目录菜单及 Casbin 策略种子
