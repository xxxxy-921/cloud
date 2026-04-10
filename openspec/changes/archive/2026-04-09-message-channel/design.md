## Context

Metis 当前有站内通知（Notification + Announcement）但缺少外部消息推送能力。需要新增 MessageChannel 模块，支持配置和管理外部消息通道，本次仅实现邮件（SMTP）。

现有架构模式：BaseModel 嵌入 → Repository（GORM）→ Service（业务逻辑）→ Handler（Gin）→ samber/do IOC 注册。种子数据通过 `internal/seed/` 维护菜单和 Casbin 策略。

## Goals / Non-Goals

**Goals:**
- 实现 MessageChannel 的 CRUD（创建、查询、更新、删除、切换启用状态）
- 实现 ChannelDriver 接口抽象，当前仅 EmailDriver（SMTP）
- 实现"测试连接"功能（验证 SMTP 配置可用性）
- 实现"发送测试邮件"功能（实际发送一封测试邮件到指定地址）
- 前端管理页面：列表 + 新建/编辑对话框 + 测试交互
- 种子数据：菜单项 + Casbin 策略

**Non-Goals:**
- Config 字段不加密，明文 JSON 存储
- 不与现有 Notification 系统联动
- 不设计企业微信/钉钉等其他通道（但 Driver 接口保留扩展性）
- 不做消息发送记录/日志表

## Decisions

### 1. 数据模型

新增 `MessageChannel` 表，嵌入 BaseModel：

```go
type MessageChannel struct {
    BaseModel
    Name    string `gorm:"size:100;not null"`
    Type    string `gorm:"size:32;not null"`          // "email"
    Config  string `gorm:"type:text;not null"`         // JSON 明文
    Enabled bool   `gorm:"not null;default:true"`
}
```

Config 字段存储 JSON 字符串，不同 type 有不同结构。邮件示例：
```json
{
  "host": "smtp.example.com",
  "port": 465,
  "secure": true,
  "username": "user@example.com",
  "password": "xxx",
  "from": "系统通知 <noreply@example.com>"
}
```

**为何不用 JSONB/单独字段**：SQLite 不支持 JSONB，明文 JSON text 兼容两种数据库。不同 type 的 config 结构差异大，固定字段不灵活。

### 2. Driver 抽象

```go
// internal/channel/driver.go
type Payload struct {
    To      []string
    Subject string
    Body    string  // 纯文本
    HTML    string  // 可选 HTML
}

type Driver interface {
    Send(config map[string]any, payload Payload) error
    Test(config map[string]any) error
}
```

注册表模式：
```go
var drivers = map[string]Driver{
    "email": &EmailDriver{},
}

func GetDriver(channelType string) (Driver, error) { ... }
```

**为何用全局注册表**：简单，无需 IOC。Driver 无状态，每次 Send/Test 从 config 创建连接。

### 3. EmailDriver 实现

使用 Go 标准库 `net/smtp` + `crypto/tls`，不引入额外依赖。

- `Test()`: 建立 SMTP 连接 → TLS 握手 → AUTH 验证 → 断开。仅验证配置可用性。
- `Send()`: 构建 MIME 消息 → 发送邮件。

### 4. API 设计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/channels` | 列表（分页，密码字段脱敏返回） |
| POST | `/api/v1/channels` | 创建 |
| GET | `/api/v1/channels/:id` | 详情（密码脱敏） |
| PUT | `/api/v1/channels/:id` | 更新 |
| DELETE | `/api/v1/channels/:id` | 删除 |
| PUT | `/api/v1/channels/:id/toggle` | 切换启用/禁用 |
| POST | `/api/v1/channels/:id/test` | 测试连接 |
| POST | `/api/v1/channels/:id/send-test` | 发送测试邮件（body 传 `{to, subject, body}`） |

**API 响应脱敏**：查询返回时，config 中的 `password` 字段替换为 `"******"`。更新时若收到 `"******"` 则保留原值不变。

### 5. 前端页面

参考现有 announcements 页面结构，使用 Dialog 模式（非独立页面）：

- **列表页** `/channels`：DataTable 展示名称、类型（Badge）、启用状态（Switch）、操作
- **新建/编辑**：Dialog 弹窗，左侧基础信息（名称+类型选择），右侧根据类型动态渲染 config 表单
- **测试连接**：表单内按钮，调用 test API，Toast 反馈结果
- **发送测试**：编辑/详情中提供按钮，弹出输入收件人地址，发送测试邮件

通道类型元数据在前端维护（`CHANNEL_TYPES`常量），定义每种类型的 config 字段 schema，用于动态渲染表单和字段验证。

### 6. 种子数据

menus.go 中在"系统管理"下追加：
```
消息通道 /channels Mail system:channel:list Sort:7
├── 新增通道 Button system:channel:create
├── 编辑通道 Button system:channel:update
└── 删除通道 Button system:channel:delete
```

policies.go 中追加 admin 的 API 策略：
```
/api/v1/channels GET/POST
/api/v1/channels/:id GET/PUT/DELETE
/api/v1/channels/:id/toggle PUT
/api/v1/channels/:id/test POST
/api/v1/channels/:id/send-test POST
```

### 7. 前端路由

在 router 中新增 `/channels` lazy 路由，PermissionGuard 检查 `system:channel:list`。

## Risks / Trade-offs

- **[密码明文存储]** → 本次明确不加密。生产环境建议通过数据库访问控制和文件权限保护 SQLite 文件。后续可按需追加加密层。
- **[net/smtp 局限]** → 标准库 SMTP 不支持某些高级特性（如 OAuth2 认证的 Gmail）。对于企业 SMTP 服务器足够。如遇不兼容可替换为 gomail 库，Driver 接口不变。
- **[类型不可变]** → 创建后不可更改通道类型。避免 config schema 不匹配的问题，参考 NekoAdmin 同样做法。
