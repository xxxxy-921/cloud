## Context

Metis 当前具备 JWT 认证、token 黑名单、并发会话限制、RBAC、OAuth 社交登录等基础安全能力。但缺少密码策略、登录锁定、2FA、自助注册、验证码等精细化安全管控。这些功能在 NekoAdmin 参考实现中已有完整设计，本次将其适配到 Metis 的 Go + Gin + GORM + samber/do 架构中。

**当前约束：**
- SQLite only，无 Redis
- 单进程部署，内存缓存可靠
- SystemConfig K/V 表已有成熟的配置读写模式
- 前端安全设置页面已存在，需扩展

## Goals / Non-Goals

**Goals:**
- 补齐 6 大安全管控能力，全部通过 SystemConfig 配置化
- 所有新功能默认关闭/宽松，不影响现有部署
- Login 流程有序编排：锁定检查 → 验证码 → 密码验证 → 2FA → 签发 token
- 密码过期通过 JWT Claims + 轻量 middleware 实现，零 DB 开销
- 验证码用内存存储（sync.Map + TTL），适配无 Redis 环境

**Non-Goals:**
- 密码历史（禁止复用最近 N 个密码）—— 需额外表，后续迭代
- TOTP "信任此设备" —— 增加复杂度，后续迭代
- Turnstile / reCAPTCHA 集成 —— 预留配置位但不实现
- 邮件验证注册 —— 注册即激活，无邮件确认流程
- WebAuthn / FIDO2 —— 超出本期范围

## Decisions

### D1: 密码过期检查方式 — JWT Claims + Middleware

**选择**：在 JWT claims 中嵌入 `passwordChangedAt` 和 `forcePasswordReset`，由轻量 middleware 每次请求检查。

**替代方案**：
- A) 仅登录时检查，token claims 带标记 → 无法捕捉会话期间密码到期
- B) Middleware 每次查 DB → 性能开销不可接受

**理由**：从 claims 读取零 DB 开销，且能实时捕捉过期。返回 HTTP 409 + `PASSWORD_EXPIRED` code，前端拦截跳转改密页。白名单路由：change-password、logout、refresh。

### D2: 验证码存储 — 内存 sync.Map

**选择**：`sync.Map` + 后台 goroutine 定期清理过期条目（5 分钟 TTL）。

**替代方案**：
- A) SQLite 表 → 频繁写入+清理对 SQLite 不友好
- B) Redis → Metis 无 Redis 依赖

**理由**：验证码本质短命（5 分钟），重启丢失无影响。内存方案零依赖，性能最优。

### D3: 2FA 临时 Token — 短命 JWT

**选择**：密码验证通过但需 2FA 时，签发一个 5 分钟有效的 `twoFactorToken`（JWT，claims 含 userId + purpose="2fa"），仅用于 POST /api/v1/auth/2fa/login。

**替代方案**：
- A) Session-based（内存存临时 state）→ 需要额外的 session 管理
- B) 在原 login 响应中直接返回 needsTwoFactor 标记 + 让用户带用户名密码重新请求 → 安全隐患

**理由**：JWT 无状态，5 分钟自动过期，middleware 可通过 purpose claim 限制使用范围。

### D4: 登录锁定 — User 字段 + 时间自愈

**选择**：在 User model 上加 `FailedLoginAttempts` 和 `LockedUntil` 字段，锁定到期自动解除。

**理由**：最简设计，不需要额外表。原子更新 `failed_login_attempts = failed_login_attempts + 1` 防竞态。管理员可通过 user edit API 重置。

### D5: Go TOTP 库 — github.com/pquerna/otp

**选择**：Go 生态最成熟的 TOTP 库，支持 RFC 6238，API 简洁。

### D6: Go 图形验证码库 — github.com/mojocn/base64Captcha

**选择**：输出 base64 PNG，支持数字/字母/算术，无外部依赖。

### D7: 登录流程编排顺序

```
POST /api/v1/auth/login
  │
  ├─ 1. 查用户（不存在 → 401）
  ├─ 2. 检查锁定（LockedUntil > now → 423 LOCKED）
  ├─ 3. 验证验证码（captcha_provider != "none" → 校验 X-Captcha-Id/Answer）
  ├─ 4. 验证密码（失败 → FailedAttempts++ → 可能触发锁定 → 401）
  ├─ 5. 检查 is_active（disabled → 401）
  ├─ 6. 检查 2FA（TwoFactorEnabled → 返回 twoFactorToken → 202）
  ├─ 7. 密码过期检查 + 并发会话清理
  └─ 8. 签发 token pair（200）
```

### D8: 注册 API — 复用现有 User 创建逻辑

注册端点 `POST /api/v1/auth/register` 加入 Casbin 白名单，内部复用 UserService.Create()，额外步骤：检查 registration_open、密码策略验证、分配 default_role_code、自动登录签发 token。

## Risks / Trade-offs

- **[内存验证码重启丢失]** → 验证码 5 分钟 TTL，重启概率低，用户重新获取即可
- **[密码过期 claims 与实际不同步]** → 改密时签发新 token 更新 claims；access token 最长 30 分钟，误差可控
- **[2FA Secret 明文存储]** → SQLite 单机部署，DB 文件已是安全边界。如需加密可后续加 AES 包装，当前优先简洁
- **[锁定状态绕过]** → 锁定检查在密码验证之前，且 FailedAttempts 原子更新，竞态风险极低
- **[注册滥用]** → 默认关闭注册；开启时依赖验证码保护。后续可加 IP 限速
