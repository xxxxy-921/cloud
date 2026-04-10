## Context

Metis 已建立可插拔 App 架构：`App` 接口（Name/Models/Seed/Providers/Routes/Tasks），`main.go` 启动时依次调用各 App 的生命周期方法，前端通过 `registerApp()` 注册路由。目前无实际 App，身份源管理将作为第一个 App 验证整套架构。

现有认证体系：本地密码登录 + 社交 OAuth（AuthProvider + UserConnection）。企业身份源（OIDC/LDAP）与社交 OAuth 的关注点完全不同（域名绑定、强制 SSO、LDAP bind、JIT 供给等），适合作为独立 App 实现。

**核心挑战**：身份源需要介入内核的登录流程（密码失败后尝试 LDAP、域名检测触发 SSO）。App 接口当前无法 hook 内核行为。需要一个最小的内核扩展点。

## Goals / Non-Goals

**Goals:**
- 身份源管理作为自包含 App 实现（`internal/app/identity/` + `web/src/apps/identity/`）
- 内核新增最小扩展接口，允许 App 注入外部认证能力
- OIDC Authorization Code + PKCE、LDAP bind 认证、JIT 用户供给
- 域名路由、强制 SSO
- 无 identity App 时，内核功能零影响
- 验证 App 架构的实际可用性

**Non-Goals:**
- SAML 2.0（后续迭代）
- SCIM 用户同步（仅 JIT）
- 修改 App 接口本身（当前接口已足够）
- 将 AuthProvider（社交登录）迁移为 App（保持内核不变）

## Decisions

### D1: App 自包含，内核仅 1 个扩展接口

**选择**: 在内核 `internal/service/` 中定义 `ExternalAuthenticator` 接口，App 实现并注册到 IOC。AuthService 通过 `do.InvokeAs[ExternalAuthenticator]` 可选解析 — 解析失败（无 App）则跳过。

```go
// internal/service/external_auth.go （内核新增文件）
type ExternalAuthenticator interface {
    // AuthenticateByPassword 在本地密码失败后调用，尝试外部认证（如 LDAP）
    AuthenticateByPassword(username, password string) (*model.User, error)
    // CheckDomain 检查 email 域名是否匹配某个身份源
    CheckDomain(email string) (*DomainCheckResult, error)
    // IsForcedSSO 检查用户是否被强制 SSO
    IsForcedSSO(email string) bool
}

type DomainCheckResult struct {
    SourceID   uint   `json:"id"`
    Name       string `json:"name"`
    Type       string `json:"type"`
    ForceSso   bool   `json:"forceSso"`
}
```

**原因**: 最小侵入 — 内核只增加 1 个接口文件 + AuthService 增加 3 行可选调用。App 不存在时行为完全不变。这比 event bus 或 hook 注册表更简单直接。

**替代方案**: App 接口新增 `AuthHooks()` 方法。被否决，因为 AuthHooks 与多数 App 无关（AI App、License App 不需要），污染通用接口。

### D2: App 文件结构

**选择**: 所有身份源代码集中在 `internal/app/identity/` 下，按职责分文件：

```
internal/app/identity/
  app.go          # App 接口实现 + init() 注册
  model.go        # IdentitySource 模型 + Config 结构体
  repository.go   # GORM CRUD
  service.go      # 业务逻辑（CRUD、加密、测试连接）
  authenticator.go # ExternalAuthenticator 实现（LDAP fallback、域名检测）
  oidc.go         # OIDC 协议封装（Discovery、PKCE、token 交换）
  ldap.go         # LDAP 协议封装（bind、search、TLS）
  handler.go      # 管理 API handler
  sso_handler.go  # SSO 公开端点 handler
  crypto.go       # AES-256-GCM 加密工具
```

**原因**: 自包含 = 一个目录删除后项目编译通过（仅去掉 edition import）。不散落到 `internal/model/`、`internal/repository/` 等内核目录。

### D3: 配置字段 JSON + AES-256-GCM 加密

**选择**: `IdentitySource.Config` 使用 TEXT 列存 JSON（`OIDCConfig` / `LDAPConfig` 结构体序列化）。敏感字段（client_secret、bind_password）在序列化前 AES-256-GCM 加密。密钥从 `ENCRYPTION_KEY` 环境变量读取，未设置则自动生成存入 SystemConfig。

**原因**: OIDC 和 LDAP 配置结构差异大，JSON 比多列更灵活。加密通过 App 内的 `crypto.go` 自包含处理。

### D4: 域名检测 API 位于 App 但注册为公开端点

**选择**: `GET /api/v1/auth/check-domain` 由 App 的 `sso_handler.go` 提供，通过 `Routes()` 注册。但此端点需要公开访问（登录页未认证时调用），所以 App 在 `Routes()` 中直接注册到 Gin Engine（通过从 authed group 获取 parent），或者内核在 Casbin 白名单中预留 `/api/v1/auth/sso` 和 `/api/v1/auth/check-domain` 前缀。

**实际方案**: 内核 Casbin 白名单新增 2 行前缀。App 的 SSO handler 注册在 authed group 下但被白名单放行 — 与现有 OAuth 端点（`/api/v1/auth/providers`、`/api/v1/auth/oauth/*`）模式一致。

### D5: 前端登录页优雅降级

**选择**: 内核登录页增加可选的 email 域名检测。调用 `GET /api/v1/auth/check-domain` — 如果返回 404（无 identity App）则静默忽略，登录页表现不变。如果返回匹配结果，则动态显示 SSO 按钮。

**原因**: 前端不需要 import App 代码。登录页通过 API 发现能力 — 有 App 时增强，无 App 时降级。

### D6: SSO 回调页面放在 App 前端

**选择**: `/sso/callback` 路由由 `web/src/apps/identity/module.ts` 注册。如果 identity App 未加载，该路由不存在 — 但这不是问题，因为没有 App 时不会有 SSO 流程发起。

### D7: OIDC 和 LDAP 的 JIT + 身份跟踪

**选择**: 复用内核已有的 `UserConnection` 表跟踪外部身份。OIDC 用 `oidc_{sourceId}` 作 Provider，LDAP 用 `ldap_{sourceId}`。App 通过 IOC 解析内核的 `UserConnectionRepo` 和 `UserRepo`。

**原因**: App 可以引用内核 service/repo（IOC 允许跨包解析），避免重复建表。

### D8: Edition 集成

**选择**: `edition_full.go` 添加 `import _ "metis/internal/app/identity"`。`edition_lite.go` 不导入。前端 `registry.ts` 添加 `import './identity/module'`。

## Risks / Trade-offs

- **[内核扩展接口设计]** `ExternalAuthenticator` 是第一个内核扩展点，设计不当会成为模式负担 → 保持接口最小（3 个方法），后续 App 如果需要不同 hook 再按需增加
- **[Casbin 白名单预留]** 即使无 App，白名单中有 `/api/v1/auth/sso` 前缀 → 无安全风险，因为无 handler 注册时请求返回 404
- **[App 依赖内核 service]** App 通过 IOC 解析 `UserRepo`、`UserConnectionRepo` 等 → 如果内核重命名这些类型，App 编译失败。这是有意的耦合 — App 与内核同仓库，编译期发现
- **[LDAP 库体积]** `go-ldap/ldap/v3` 增加二进制体积 → edition_lite 不导入 identity App，精简版不受影响
- **[加密密钥管理]** 与 D3 相同，ENCRYPTION_KEY 丢失则加密配置不可读 → 文档说明备份
