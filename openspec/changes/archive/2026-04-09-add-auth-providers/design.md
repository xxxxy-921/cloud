## Context

Metis 当前仅支持用户名+密码的本地认证方式。认证体系基于 JWT access token (30min) + opaque refresh token (7d) 的双令牌架构，已实现令牌轮换、复用检测、并发会话限制和黑名单机制。

用户模型中 Username 和 Password 均为必填字段，无任何外部身份关联机制。前端登录页是简单的用户名+密码表单，auth store 和 API 拦截器围绕本地登录流程设计。

本次变更引入 OAuth2 社交登录（GitHub/Google），需要在不破坏现有本地认证的前提下，扩展用户模型和认证流程。

## Goals / Non-Goals

**Goals:**
- 支持 GitHub 和 Google OAuth2 登录，与现有密码登录共存
- 管理员可在后台配置/启用/禁用 OAuth 认证源
- 用户可绑定/解绑外部账号，支持多种登录方式并存
- 首次 OAuth 登录自动创建本地用户（JIT provisioning）
- 架构可扩展，后续可方便地添加微信、OIDC 等 provider

**Non-Goals:**
- 不做企业 SSO（OIDC/SAML/LDAP）— 留给后续迭代
- 不做微信扫码登录 — 需要特殊适配，首期不做
- 不持久化 OAuth 平台的 access_token/refresh_token — 只用于身份识别
- 不做邮箱冲突时的自动账号关联 — 安全风险高，首期采用拒绝策略
- 不做 ClientSecret 加密存储 — Metis 定位内部管理系统，明文 DB + API 隐藏足够

## Decisions

### 1. OAuth 库选择：`golang.org/x/oauth2`

**选择**: 使用 Go 官方 `golang.org/x/oauth2` 库。

**备选方案**:
- 手写 HTTP 请求：灵活但工作量大，需处理 PKCE、token exchange 等细节
- goth 等第三方库：封装过多，引入不必要的依赖和抽象

**理由**: 官方库稳定成熟，GitHub 和 Google 都有现成的 endpoint 子包（`github.com/golang.org/x/oauth2/github`、`google.golang.org/api/oauth2`），代码量最小，后续添加新 provider 只需定义 endpoint。

### 2. 用户模型升级策略：Username/Password 可选

**选择**: 将 Username 改为可选（自动生成），Password 改为可选（OAuth 用户为空字符串）。

**备选方案**:
- 强制补全：OAuth 登录后要求填写用户名 — 用户体验差
- 独立用户表：OAuth 用户存在单独的表 — 增加复杂度

**理由**: 方案 B（自动生成用户名）兼容现有逻辑，不需要大面积改动 repository/service 层。用户名格式为 `{provider}_{externalID}`，保证唯一性。用户登录后可自行修改用户名。

### 3. OAuth State 管理：内存 Map + TTL

**选择**: 使用 `sync.Map` 存储 OAuth state → metadata 映射，10 分钟过期自动清理。

**备选方案**:
- JWT 编码 state：无状态但 state 参数较长
- 加密 Cookie：需要处理跨域和 SameSite 问题

**理由**: Metis 是单实例部署，内存方案最简单可靠。State 结构包含 `{provider, redirectURL, createdAt}`，后台 goroutine 定期清理过期条目。

### 4. 回调流程：前端中转

**选择**: OAuth provider 回调到前端路由 `/oauth/callback`，前端拿到 code+state 后调用后端 API。

**流程**:
```
用户点击 GitHub 登录
  → 前端调 GET /api/v1/auth/oauth/:provider → 返回 {authURL, state}
  → 前端 window.location = authURL
  → GitHub 授权后回调到前端 /oauth/callback?code=xxx&state=yyy
  → 前端调 POST /api/v1/auth/oauth/callback {provider, code, state}
  → 后端验证 state + 换 token + 获取用户信息 + 签发 JWT
  → 返回 TokenPair → 前端存储，跳转首页
```

**备选方案**: 后端直接 redirect — 但需要处理 token 传递（query param 不安全，fragment 不可靠），且和 SPA 架构不一致。

**理由**: 与现有密码登录流程一致，前端统一管理 token 存储和路由跳转。回调 URL 配置到前端地址，后端保持纯 API。

### 5. 账号关联策略：拒绝 + 提示

**选择**: 当 OAuth 用户邮箱与已有本地用户冲突时，返回错误提示，不自动合并。

**理由**: 自动关联的前提是邮箱已验证，但 Metis 目前没有邮箱验证机制。贸然关联可能导致账号劫持（攻击者在 GitHub 上设置别人的邮箱）。用户需先用密码登录，再手动绑定 OAuth 账号。

### 6. 数据模型：两张新表

**auth_providers 表**（认证源配置）:
- ProviderKey (unique): "github" / "google"
- DisplayName, Enabled, ClientID, ClientSecret, Scopes, CallbackURL, SortOrder
- 使用 BaseModel（ID + 时间戳 + 软删除）

**user_connections 表**（用户绑定关系）:
- UserID (FK → users), Provider, ExternalID, ExternalName, ExternalEmail, AvatarURL
- unique(Provider, ExternalID) — 同一外部身份只能绑定一个本地用户
- unique(UserID, Provider) — 一个用户每个 provider 只绑一个

### 7. Provider 接口抽象

定义 `OAuthProvider` 接口：
```go
type OAuthProvider interface {
    GetAuthURL(state string) string
    ExchangeCode(ctx context.Context, code string) (*OAuthUserInfo, error)
}
```

GitHub 和 Google 各自实现。新增 provider 只需实现此接口 + 在 auth_providers 表添加配置。

## Risks / Trade-offs

- **[内存 State 丢失]** → 服务重启会丢失所有进行中的 OAuth 流程（10 分钟窗口），用户需重新点击登录。可接受，因为 Metis 不频繁重启。
- **[ClientSecret 明文存储]** → 数据库泄露会暴露 OAuth 密钥。缓解：API 响应不返回 Secret 值；生产环境依赖数据库访问控制。
- **[无邮箱验证]** → 不能安全地自动关联账号。缓解：首期只支持手动绑定，后续可添加邮箱验证后自动关联。
- **[Username 可选后的兼容性]** → 现有代码中可能有依赖 Username 非空的逻辑。缓解：自动生成用户名确保字段始终有值，只是不再由用户手动输入。
