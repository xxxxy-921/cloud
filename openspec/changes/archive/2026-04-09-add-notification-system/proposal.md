## Why

Metis 缺少全局通知机制。管理员无法向用户发布系统公告（维护通知、版本更新、功能上线等），用户也没有统一的入口查看待处理信息。通知系统是后续告警、审批等模块的基础设施，需要先建立核心通知管道和公告模块作为第一个生产者。

## What Changes

- 新增 `notification` 和 `notification_read` 两张表，支持广播（all）和定向（user）两种投递模式
- 新增通知中心 API（列表、未读数、标记已读、全部已读），所有登录用户可访问
- 新增公告管理 API（CRUD），仅管理员可操作，创建公告时自动写入通知表
- 新增 `NotificationService.Send()` 统一发送接口，供各模块调用
- 前端 TopNav 右上角新增通知铃铛 + Popover 下拉列表，30 秒轮询未读数
- 前端新增公告管理页面（表格 + 创建/编辑弹窗），受权限控制
- 新增 Casbin 策略：`notification:list/read`（登录白名单）、`announcement:list/create/update/delete`
- 新增菜单项：公告管理（归入系统管理分组）

## Capabilities

### New Capabilities

- `notification-center`: 全局通知管道 — 数据模型（notification + notification_read）、统一发送接口、通知列表/未读数/标记已读 API、前端铃铛 Popover
- `announcement`: 系统公告管理 — 公告 CRUD API、公告管理页面（表格+弹窗）、权限控制，创建公告时调用通知管道发送广播通知

### Modified Capabilities

- `nav-modules`: 新增公告管理菜单项到系统管理分组

## Impact

- **后端新增文件**: `model/notification.go`, `repository/notification.go`, `service/notification.go`, `handler/notification.go`, `handler/announcement.go`
- **后端修改**: `database.go`（AutoMigrate）、`main.go`（IOC 注册）、`handler.go`（路由注册）、seed 数据（Casbin 策略 + 菜单）
- **前端新增**: `components/notification-bell.tsx`、`pages/announcements/`
- **前端修改**: `top-nav.tsx`（嵌入铃铛组件）、`App.tsx`（路由）
- **依赖**: 无新外部依赖，使用现有 shadcn/ui Popover + Badge 组件
