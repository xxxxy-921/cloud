## Why

Metis 目前仅支持本地用户名+密码登录和社交 OAuth（GitHub、Google）。企业客户需要对接现有身份基础设施（Okta、Azure AD 等 OIDC Provider，或 OpenLDAP/Active Directory），而不是为每个用户创建独立账号密码。

项目已建立可插拔 App 架构（`internal/app/` + `web/src/apps/`），但尚无实际 App。身份源管理作为第一个可插拔 App，既满足企业 SSO 需求，又验证 App 架构的可行性。

## What Changes

- 新增 `internal/app/identity/` 后端 App — 实现 `app.App` 接口，包含身份源模型、OIDC/LDAP 认证逻辑、管理 API、JIT 用户供给
- 新增 `web/src/apps/identity/` 前端 App — 身份源管理页面、SSO 回调页面
- 内核新增 `ExternalAuthenticator` 接口 — 允许 App 注入认证扩展，内核 AuthService 在本地密码失败后调用
- 内核登录页增加域名检测能力 — 优雅降级：无 identity App 时接口返回空，登录页表现不变
- 在 `edition_full.go` 中导入 identity App

## Capabilities

### New Capabilities
- `identity-source`: 身份源数据模型与 CRUD 管理（App 内自包含：model/repo/service/handler）
- `oidc-auth`: OIDC 协议集成，Authorization Code + PKCE 流程、IdP 发现、token 交换
- `ldap-auth`: LDAP/AD 协议集成，bind 认证、用户搜索、属性映射
- `identity-source-ui`: 身份源管理后台页面（`web/src/apps/identity/`）

### Modified Capabilities
- `user-auth`: 内核 AuthService 新增可选的 `ExternalAuthenticator` 接口调用；新增 `check-domain` 和 SSO 公开端点的 Casbin 白名单
- `user-auth-frontend`: 登录页增加 email 域名检测和 SSO 按钮（优雅降级，无 App 时不影响现有功能）

## Impact

- **后端新增依赖**: `github.com/coreos/go-oidc/v3`、`golang.org/x/oauth2`、`github.com/go-ldap/ldap/v3`
- **数据库**: App 通过 `Models()` 注册 `IdentitySource` 表（AutoMigrate）
- **内核改动极小**: 仅新增 1 个接口定义 + AuthService 增加 1 个可选依赖 + Casbin 白名单 2 行 + 登录页前端改动
- **API 端点**: App 注册 `/api/v1/identity-sources/*`（管理）、`/api/v1/auth/sso/*`（OIDC 流程）；内核新增 `/api/v1/auth/check-domain`
- **Casbin 策略**: App 在 `Seed()` 中注册 admin 权限
- **Edition**: `edition_full.go` 导入 `_ "metis/internal/app/identity"`；`edition_lite.go` 不导入，精简版无此功能
- **前端**: `web/src/apps/identity/module.ts` 注册路由；`registry.ts` 导入模块
- **安全**: 敏感字段（client_secret、bind_password）AES-256-GCM 加密存储
