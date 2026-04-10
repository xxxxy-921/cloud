## Context

Metis 当前使用 JWT (30min access token) + Refresh Token (7day, DB-backed rotation) 的认证模式。RefreshToken 模型只有 Token、UserID、ExpiresAt、Revoked 四个业务字段，不记录登录来源。管理员无法查看在线用户，也无法强制踢出。

同时，`/config` 页面暴露了 SystemConfig 表的原始 CRUD，任何有权限的用户可以直接读写任意 key。系统设置应该通过类型安全的专用 API 管理。

## Goals / Non-Goals

**Goals:**
- 管理员可查看所有活跃会话（用户、IP、设备、登录时间、最后活跃）
- 管理员可立即踢出指定会话（不需要等 access token 30分钟过期）
- 每用户并发会话数可配置，超限自动踢出最旧会话
- 系统设置页面改为分类 Tab 布局，提供类型化的设置管理
- 移除通用 KV 配置接口，消除安全隐患

**Non-Goals:**
- 审计日志（auth_log）——后续单独做
- WebSocket 实时推送会话变更——页面手动刷新即可
- 用户自助管理自己的会话——仅管理员可操作
- 分布式黑名单（Redis）——单进程内存 map 足够，Metis 是单二进制部署

## Decisions

### D1: 会话 = Refresh Token（不新建表）

**选择**: 在 `refresh_tokens` 表上扩展字段（IPAddress, UserAgent, LastSeenAt, AccessTokenJTI），不新建 `user_sessions` 表。

**理由**: Metis 中一次登录产生一个 refresh token，天然对应一个会话。新建表会引入两个实体的同步问题。扩展现有表字段 AutoMigrate 自动加列，零迁移成本。

**备选**: 新建 `user_sessions` 表，refresh token 关联 session_id。概念更清晰，但对 Metis 的规模来说过度设计。

### D2: 内存黑名单实现即时踢出

**选择**: 进程内 `sync.RWMutex + map[string]time.Time`（key=jti, value=过期时间），JWT 中间件每次请求检查。

**理由**: JWT 的无状态特性意味着 revoke refresh token 不能立即使 access token 失效。黑名单将 access token 的 jti 标记为无效，剩余有效期最多 30 分钟后自动清除。单进程 map 查询 O(1)，无外部依赖。

**备选 A**: 只撤销 refresh token，等 access token 自然过期（最多 30 分钟延迟）——对"踢出"场景不可接受。
**备选 B**: 用 userId 粒度黑名单——会误伤同用户的其他正常会话。

### D3: AccessTokenJTI 字段关联 access token 和 refresh token

**选择**: refresh_tokens 表新增 `AccessTokenJTI` 字段，每次 login/refresh 时写入当前 access token 的 jti。踢出时通过此字段拿到 jti 加入黑名单。

**理由**: 踢出操作的输入是 refresh token 的 ID，必须能映射到对应的 access token jti 才能精确拦截。

### D4: 黑名单清理注册为 Scheduler 任务

**选择**: 在 `internal/scheduler/builtin.go` 注册 `blacklist_cleanup` 任务（每5分钟），和 `expired_token_cleanup` 任务（每天凌晨3点）。

**理由**: 复用现有 scheduler 引擎，任务可在管理页面查看状态、暂停/恢复，与 HistoryCleanupTask 一致。黑名单 cleanup 也做惰性清理（IsBlocked 时删过期条目），定时任务作为兜底。

### D5: 删除通用 Config API，新增类型化 Settings API

**选择**: 移除 `/api/v1/config` 全部路由，新增 `GET/PUT /api/v1/settings/security` 和 `GET/PUT /api/v1/settings/scheduler`。SettingsService 内部读写 SystemConfig 表。

**理由**: 通用 KV API 允许任意读写配置，是安全隐患。类型化 API 有明确的请求/响应结构、验证逻辑和权限控制。

**备选**: 保留 KV 路由但隐藏菜单——仍然可通过 API 直接访问，不彻底。

### D6: Settings 页面 Tab 布局

**选择**: settings 页面从两张卡片改为 Tabs 组件（站点信息 / 安全设置 / 任务设置），每个 Tab 内用 Card 组织表单。

**理由**: 随着可配置项增多，Tab 比平铺卡片更好扩展。shadcn/ui 的 Tabs 组件已在项目中可用。

## Risks / Trade-offs

- **[进程重启丢失黑名单]** → 黑名单最大条目生命周期 30 分钟。重启后最坏情况：已踢出用户的 access token 在剩余有效期内仍可用（最多 30 分钟），但 refresh token 已撤销，不会续期。可接受。

- **[并发登录竞态]** → 同一用户在并发限制边界同时从两个设备登录，可能短暂超限。影响极小，下次登录会矫正。不做分布式锁。

- **[删除 /config 路由是 breaking change]** → 如果有外部脚本依赖该 API 会中断。当前 Metis 无外部集成，风险可控。种子数据和前端同步移除。

- **[UserAgent 解析精度]** → 简单正则提取浏览器名和操作系统，不引入重型 UA 解析库。极少数 UA 可能显示为"未知"。

## Open Questions

无——所有决策已在 explore 阶段与用户确认。
