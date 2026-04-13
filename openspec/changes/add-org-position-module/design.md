## Context

Metis 内核当前仅有扁平的 `User`/`Role` 模型，缺少组织架构层。现有 App 架构（`internal/app/<name>`）已经成熟，支持模型注册、IOC 依赖注入、路由挂载、菜单与 Casbin 策略种子。本模块作为新的可选 App `org` 接入，不侵入内核，保证 lite 版仍可裁剪。

## Goals / Non-Goals

**Goals:**
- 提供可插拔的 `org` App，支持部门（树形）、岗位（字典）、人员分配的完整 CRUD
- 支持一人多岗，并明确标记主岗
- 前端提供部门管理、岗位管理、人员分配三个管理页面
- 为未来的数据范围权限（如"仅看本部门"）预留 scope helper

**Non-Goals:**
- 不修改内核 `User`/`Role` 模型或 `user-management` 页面
- 不支持汇报线、职级晋升路径、编制人数限制
- 不与现有 Casbin 中间件耦合做动态数据范围过滤（仅留扩展接口）

## Decisions

### 1. 采用纯 App 实现，不改造内核
**理由**：组织架构属于可选能力，部分轻量部署不需要。保持内核精简符合 Metis "可裁剪 edition" 的设计原则。App 可以通过独立 API（如 `/api/v1/org/users/:id/positions`）为内核用户补充组织信息。

### 2. 使用 `UserPosition` 中间表实现一人多岗
**理由**：直接用 `User` 加 `DepartmentID`/`PositionID` 字段只能支持一岗一部门，无法满足一人兼任多部门职位的需求。中间表增加 `IsPrimary` 字段即可解决主岗标识问题，且对现有 `users` 表零侵入。

### 3. 数据权限 scope 由 App service 提供，业务层调用
**理由**：Casbin 当前是按 "Role + Path + Method" 做粗粒度鉴权，不适合做按部门 ID 列表的动态数据过滤。通过在 App 内暴露 `GetUserDepartmentIDs(userID)` 等 helper，其他模块的 repository 层在需要时注入条件即可，改动最小且不影响现有权限链路性能。

### 4. 部门树前端采用扁平列表 + `parentId` 缩进，暂不实现拖拽排序
**理由**：Metis 现有组件库没有 TreeTable 组件，从零引入会增加复杂度。先用缩进表格 + 独立 TreeSelect 选择上级部门即可满足 MVP 需求，后续可视需求升级。

### 5. 人员分配采用"部门视角"为主交互
**理由**：管理员通常按部门查看「这个部门有哪些人」。页面左侧部门树、右侧成员列表，符合传统组织架构管理心智模型。也支持从成员处快速调整其主岗/移除。

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| 用户管理页无法直接显示部门/岗位 | 在用户详情页可额外调 `GET /api/v1/org/users/:id/positions` 拼接；当前方案下不做侵入式改动 |
| 部门层级过深导致递归查询性能下降 | 部门数量在常规企业场景下可控（<1000），GORM 预加载已足够；如未来出现瓶颈可引入 materialized path 或闭包表 |
| 删除部门/岗位时存在未解除关联的人员 | API 层先做关联检查，有成员时返回 400 并提示"请先移除该部门下的人员" |
| 主岗冲突（同一用户多个主岗） | Service 层在保存 `UserPosition` 时强制重置其他记录 `IsPrimary=false` |
