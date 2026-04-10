## 1. 数据模型与数据库

- [x] 1.1 创建 `internal/model/notification.go` — Notification 结构体（嵌入 BaseModel）+ NotificationRead 结构体（独立 PK，无 BaseModel）+ ToResponse 方法
- [x] 1.2 在 `internal/database/database.go` 的 AutoMigrate 中注册 Notification 和 NotificationRead

## 2. Repository 层

- [x] 2.1 创建 `internal/repository/notification.go` — NotificationRepo，包含：
  - `Create(notification)` — 创建通知
  - `FindByID(id)` — 查找单条通知
  - `Update(notification)` — 更新通知
  - `Delete(id)` — 软删除通知
  - `ListForUser(userID, params)` — 查询用户可见通知（target_type=all OR target_id=userID），LEFT JOIN notification_read 得到 isRead，分页返回
  - `CountUnreadForUser(userID)` — 用户未读数
  - `MarkAsRead(notificationID, userID)` — 插入 notification_read（幂等，忽略重复）
  - `MarkAllAsRead(userID)` — 批量标记已读
  - `ListAnnouncements(params)` — 公告列表（type=announcement），JOIN user 表获取发布者用户名
  - `DeleteReadByNotificationID(notificationID)` — 删除通知的已读记录（级联清理）

## 3. Service 层

- [x] 3.1 创建 `internal/service/notification.go` — NotificationService，包含：
  - `Send(type, source, title, content, targetType, targetID, createdBy)` — 统一发送接口
  - `ListForUser(userID, params)` — 通知列表
  - `GetUnreadCount(userID)` — 未读数
  - `MarkAsRead(notificationID, userID)` — 标记已读
  - `MarkAllAsRead(userID)` — 全部已读
  - `ListAnnouncements(params)` — 公告列表
  - `CreateAnnouncement(title, content, createdBy)` — 创建公告（调用 Send）
  - `UpdateAnnouncement(id, title, content)` — 更新公告
  - `DeleteAnnouncement(id)` — 删除公告（含级联清理 notification_read）

## 4. Handler 层 — 通知中心 API

- [x] 4.1 创建 `internal/handler/notification.go` — NotificationHandler，包含：
  - `List` — GET /api/v1/notifications（从 JWT 提取 userID，分页查询）
  - `GetUnreadCount` — GET /api/v1/notifications/unread-count
  - `MarkAsRead` — PUT /api/v1/notifications/:id/read
  - `MarkAllAsRead` — PUT /api/v1/notifications/read-all

## 5. Handler 层 — 公告管理 API

- [x] 5.1 创建 `internal/handler/announcement.go` — AnnouncementHandler，包含：
  - `List` — GET /api/v1/announcements（分页，含发布者信息）
  - `Create` — POST /api/v1/announcements（标题必填校验）
  - `Update` — PUT /api/v1/announcements/:id
  - `Delete` — DELETE /api/v1/announcements/:id

## 6. 路由注册与 IOC

- [x] 6.1 修改 `cmd/server/main.go` — 注册 NotificationRepo、NotificationService 到 IOC 容器
- [x] 6.2 修改 `internal/handler/handler.go` — Handler 结构体添加 notification 和 announcement 字段，New() 中 MustInvoke，Register() 中注册路由
- [x] 6.3 通知中心路由放入 authed 组但添加到 CasbinAuth 白名单（`/api/v1/notifications` 前缀）
- [x] 6.4 公告管理路由放入 authed 组，走 Casbin 权限检查

## 7. Seed 数据

- [x] 7.1 添加公告管理相关的 Casbin 策略到 seed 数据（admin 角色对 /api/v1/announcements 的 GET/POST/PUT/DELETE）
- [x] 7.2 添加公告管理菜单项到系统管理分组（标题"公告管理"、路径 /announcements、图标 Megaphone、权限键 system:announcement:list）

## 8. 前端 — 通知铃铛组件

- [x] 8.1 创建 `web/src/components/notification-bell.tsx` — NotificationBell 组件：
  - React Query 轮询 unread-count（refetchInterval: 30000）
  - Bell 图标 + Badge（0 隐藏，>99 显示 99+）
  - Popover 容器（~400px 宽）
  - 通知列表（带未读蓝点、类型图标、标题、内容摘要截断、相对时间）
  - 点击通知项标记已读
  - "全部已读"按钮
  - 空状态 "暂无通知"
- [x] 8.2 修改 `web/src/components/layout/top-nav.tsx` — 在用户下拉菜单左侧嵌入 NotificationBell 组件

## 9. 前端 — 公告管理页面

- [x] 9.1 创建 `web/src/pages/announcements/index.tsx` — 公告管理页面：
  - 使用 useListPage hook 实现分页 + 搜索
  - 数据表格（标题、发布者、发布时间、操作列）
  - 新建按钮（usePermission 控制显示）
  - 编辑/删除操作（usePermission 控制显示）
  - 删除确认 AlertDialog
- [x] 9.2 创建公告表单弹窗（Dialog 或 Sheet）— 标题（必填）+ 内容（可选）+ 提交/取消
- [x] 9.3 在 `web/src/App.tsx` 添加 /announcements 路由，包裹 PermissionGuard

## 10. 前端 — 导航菜单

- [x] 10.1 在前端导航配置中添加公告管理菜单项（如果使用后端动态菜单，则通过 seed 数据完成，无需前端修改）
