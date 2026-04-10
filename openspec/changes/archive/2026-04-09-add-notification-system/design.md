## Context

Metis 是一个 Go + React 单体应用，后端 Gin + GORM + samber/do (IOC)，前端 Vite + React 19 + shadcn/ui。当前没有任何通知机制。

参考了 NekoAdmin 的通知系统设计（独立 notification + notificationRead 表、广播/定向投递、客户端轮询、Popover 下拉列表、公告管理页 CRUD）。

现有 Metis 层级：model → repository → service → handler，通过 `do.Provide()` 注册到 IOC 容器。路由在 `handler.go:Register()` 统一注册，分 public 和 authed 两组，authed 组经过 JWTAuth + CasbinAuth 中间件。

## Goals / Non-Goals

**Goals:**
- 建立可扩展的通知管道，各模块通过 `NotificationService.Send()` 统一发送
- 实现公告模块作为第一个通知生产者（CRUD + 权限）
- 前端右上角铃铛 + Popover 展示通知列表，30s 轮询未读数
- 广播通知（targetType=all）通过独立 notification_read 表跟踪每用户已读状态

**Non-Goals:**
- 实时推送（WebSocket / SSE）— 轮询满足当前需求
- 多渠道投递（email / SMS）— 未来扩展
- 富文本公告内容 — 先纯文本，后续可加 Markdown
- 告警、审批等通知类型 — 本次只做公告，但架构预留 type/source 字段
- 通知偏好设置（用户选择接收哪些类型）

## Decisions

### D1: 已读状态用独立表（notification_read）而非字段标记

**选择**: 独立的 `notification_read` 表，`UNIQUE(notification_id, user_id)`

**备选**: 在 notification 表里加 `is_read` + `user_id` 字段，广播通知为每个用户生成一条记录

**理由**: 广播通知（targetType=all）只存一条记录，已读状态按用户独立跟踪，避免 N 用户产生 N 条重复通知。查询需要 LEFT JOIN，但 SQLite 完全能胜任当前规模。

### D2: 两套 API 分离（通知中心 vs 公告管理）

**选择**:
- `/api/v1/notifications/*` — 面向所有登录用户，只读 + 标记已读，放入 Casbin 白名单
- `/api/v1/announcements/*` — 面向管理员，CRUD 操作，走 Casbin 权限检查

**理由**: 通知中心是通用消费端，公告管理是特定生产端。分离后权限模型清晰，未来新增告警模块只需加生产端 API，消费端不变。

### D3: 前端用 Popover 而非 Sheet

**选择**: shadcn/ui Popover，固定宽度 ~400px，锚定在铃铛图标下方

**备选**: Sheet 侧栏滑出

**理由**: 通知列表是快速预览场景，Popover 轻量、不遮挡主内容。公告详情较长时可在 Popover 内截断显示。

### D4: 客户端 30s 轮询

**选择**: React Query 的 `refetchInterval: 30_000` 轮询 `/notifications/unread-count`

**备选**: SSE / WebSocket

**理由**: 实现简单，Metis 是小规模单体应用，30s 延迟可接受。React Query 在窗口失焦时自动暂停轮询，减少无效请求。

### D5: notification_read 不使用 BaseModel

**选择**: `notification_read` 表不嵌入 BaseModel，只有 `id`（主键）、`notification_id`、`user_id`、`read_at`，无 soft delete

**理由**: 已读记录是事实型数据，不需要软删除和 updated_at。删除通知时级联清理已读记录即可。

### D6: Handler 层拆分 notification 和 announcement

**选择**: 两个 handler 文件 `notification.go` 和 `announcement.go`，但共享同一个 `NotificationService`

**理由**: 路由职责不同（消费 vs 管理），拆分文件便于维护。Service 层统一是因为公告本质上是通知的子集。

## API 设计

### 通知中心（登录即可访问，Casbin 白名单）

```
GET    /api/v1/notifications              → 通知列表（分页，含 isRead 状态）
GET    /api/v1/notifications/unread-count  → 未读数量
PUT    /api/v1/notifications/:id/read      → 标记单条已读
PUT    /api/v1/notifications/read-all      → 全部标记已读
```

### 公告管理（需要对应 Casbin 权限）

```
GET    /api/v1/announcements               → 公告列表（分页，含发布者信息）
POST   /api/v1/announcements               → 创建公告（同时写入 notification 表）
PUT    /api/v1/announcements/:id           → 编辑公告（同步更新 notification 记录）
DELETE /api/v1/announcements/:id           → 删除公告（级联删除 notification + read 记录）
```

## 数据模型

```
notification
├── id           uint    PK (BaseModel)
├── created_at   time    (BaseModel)
├── updated_at   time    (BaseModel)
├── deleted_at   time    (BaseModel, soft delete)
├── type         string  "announcement" | "alert" | "approval" ...
├── source       string  模块标识 "announcement" | "scheduler" ...
├── title        string  标题
├── content      string  内容（可为空）
├── target_type  string  "all" | "user"
├── target_id    *uint   当 target_type=user 时为 user_id
└── created_by   *uint   发布者 user_id

notification_read
├── id              uint       PK
├── notification_id uint       FK → notification.id
├── user_id         uint       FK → user.id
├── read_at         time.Time
└── UNIQUE(notification_id, user_id)

索引:
- notification: idx_type, idx_created_at
- notification_read: idx_notif_user (composite unique)
```

## Casbin 策略

```
# 通知中心 — 加入 CasbinAuth 白名单（类似 /auth/me, /menus/user-tree）
/api/v1/notifications/*  → 跳过 Casbin，JWTAuth 即可

# 公告管理 — 走 Casbin 检查
p, admin, /api/v1/announcements,   GET
p, admin, /api/v1/announcements,   POST
p, admin, /api/v1/announcements/*, PUT
p, admin, /api/v1/announcements/*, DELETE
```

## 前端组件结构

```
TopNav
└── div.ml-auto.flex
    ├── NotificationBell          ← 新增
    │   ├── Button (ghost, Bell icon)
    │   ├── Badge (unread count, 99+ 封顶)
    │   └── Popover
    │       ├── Header ("通知中心" + "全部已读" button)
    │       ├── ScrollArea (max-h-96)
    │       │   └── NotificationItem[]
    │       │       ├── 蓝点 (未读指示)
    │       │       ├── 类型图标 (📢 公告 / ⚠️ 告警 ...)
    │       │       ├── 标题 + 内容摘要
    │       │       └── 相对时间 ("2分钟前")
    │       └── Footer (空状态: "暂无通知")
    └── UserDropdown              ← 现有
```

## Risks / Trade-offs

- **[轮询延迟]** 新通知最多 30s 后才显示 → 可接受，未来可升级 SSE
- **[广播性能]** 未读数查询需要 LEFT JOIN 排除已读 → SQLite 小规模无问题，大规模需加索引优化或缓存
- **[公告与通知耦合]** 编辑/删除公告需同步操作 notification 表 → 通过 service 层封装，事务保证一致性
- **[无离线通知]** 纯客户端轮询，用户不在线时不会收到通知 → 符合当前需求，未来可加邮件渠道
