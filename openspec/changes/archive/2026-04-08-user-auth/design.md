## Context

Metis 是一个 Go (Gin + GORM + samber/do) + React (Vite + TypeScript) 单二进制 Web 应用。当前所有 API 完全开放，无任何认证机制。需要引入用户体系，同时保持单二进制部署的简洁性。

参考了 bklite-cloud (Django) 的业务设计，包含完整的 RBAC、多 Provider SSO、审计日志等能力。本次设计提取其核心——用户认证和基础管理，并适配 Go 技术栈和单二进制架构。

**约束**:
- 单二进制部署，不依赖外部服务 (Redis 等)
- SQLite 为默认数据库，需兼容 PostgreSQL
- 前端为 SPA，通过 API 通信
- 需预留 Casbin RBAC、SSO、审计日志的扩展口

## Goals / Non-Goals

**Goals:**
- 实现 JWT 双 token 认证 (access + refresh)，支持登录、登出、token 刷新
- 实现用户 CRUD 管理 (admin)，包括创建、编辑、激活/停用、重置密码
- 实现基础角色控制 (admin/user 两级)
- 提供 CLI 子命令创建初始管理员
- 前端实现登录页、token 管理、受保护路由
- 为未来 Casbin RBAC、SSO、审计日志预留扩展点

**Non-Goals:**
- 不实现 Casbin RBAC (仅预留扩展口)
- 不实现 SSO / 多 Provider 登录
- 不实现审计日志
- 不实现注册功能 (用户由 admin 创建)
- 不实现强密码策略

## Decisions

### D1: JWT access token + opaque refresh token (非纯 JWT 或纯 Session)

**选择**: Access token 为签名 JWT (30min)，refresh token 为随机字符串存 DB (7天)。

**替代方案**:
- 纯 Session (bklite-cloud 方案): 需要服务端 session 存储，增加单二进制复杂度
- 纯 JWT (无 refresh): 无法实现登出，token 泄露风险大
- 双 JWT: refresh token 无法即时吊销

**理由**: Access token 无状态验证 (不查 DB)，性能好。Refresh token 存 DB 可精确吊销，支持登出和会话管理。refresh token rotation 防止 token 盗用。

### D2: bcrypt 密码哈希

**选择**: golang.org/x/crypto/bcrypt，DefaultCost (10)。

**替代方案**: argon2id (更现代但 Go 标准库支持不如 bcrypt 成熟)、scrypt。

**理由**: bcrypt 是 Go 生态最成熟的选择，DefaultCost 在安全性和性能间平衡好。

### D3: 角色硬编码枚举 + 中间件检查 (非 Casbin)

**选择**: User.Role 为字符串枚举 (admin/user)，RequireRole() 中间件硬编码检查。

**替代方案**: 直接引入 Casbin RBAC。

**理由**: 当前只需两级角色，Casbin 过重。硬编码中间件简单直接，未来迁移到 Casbin 时只需替换中间件，不影响业务层。

### D4: CLI 子命令创建初始 admin (非自动种子)

**选择**: `metis create-admin --username=xxx --password=xxx` 子命令。

**替代方案**: 首次启动自动创建默认 admin、环境变量配置。

**理由**: CLI 子命令显式、安全，不会意外创建默认账号。使用 cobra 或简单的 os.Args 解析。

### D5: 前端 localStorage 存储 token

**选择**: access token 和 refresh token 存 localStorage。

**替代方案**: httpOnly cookie (更安全但需处理 CSRF)。

**理由**: 单二进制内部应用场景，XSS 风险可控。localStorage 实现简单，配合 access token 短寿命 (30min) 降低风险。

### D6: 路由分三组——公开 / 认证 / admin

**选择**:
```
/api/v1/auth/login, /auth/refresh     → 公开
/api/v1/auth/logout, /auth/me, ...    → JWTAuth 中间件
/api/v1/users/*                       → JWTAuth + RequireRole("admin")
```

**理由**: 清晰的权限边界，中间件链式组合，符合 Gin 分组路由的惯用模式。

## Risks / Trade-offs

- **[Access token 无法即时吊销]** → 短寿命 (30min) + 可选内存黑名单 (改密码等场景)。内存黑名单重启后清空，但 access token 最多 30 分钟内自然过期，可接受。
- **[Refresh token 单点存储]** → 存在用户 DB 中，SQLite 单写者限制下并发刷新可能冲突 → 通过数据库事务保证一致性，SQLite WAL 模式支持并发读。
- **[localStorage XSS 风险]** → 适用于内部工具场景。若未来面向公网，需迁移到 httpOnly cookie。
- **[CLI 子命令增加二进制复杂度]** → 使用简单的 subcommand 分发 (if/switch)，不引入 cobra 框架，保持轻量。
- **[角色硬编码不够灵活]** → 第一版足够，未来通过 Casbin 替换 RequireRole 中间件即可，业务层无需改动。
