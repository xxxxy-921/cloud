## 1. 内核最小扩展

- [x] 1.1 新增 `internal/service/external_auth.go` — 定义 `ExternalAuthenticator` 接口（AuthenticateByPassword、CheckDomain、IsForcedSSO）和 `DomainCheckResult` 结构体
- [x] 1.2 修改 `internal/service/auth.go` — AuthService 可选解析 `ExternalAuthenticator`（`do.InvokeAs`，失败则 nil）；Login 方法本地密码失败后调用 `AuthenticateByPassword`；ForceSso 检查
- [x] 1.3 修改 `internal/service/auth.go` — 将 `generateTokenPair` 改为公开方法 `GenerateTokenPair`，供 App 调用
- [x] 1.4 修改 `internal/middleware/casbin.go` — 在 `casbinWhitelistPrefixes` 中添加 `/api/v1/auth/sso` 和 `/api/v1/auth/check-domain`

## 2. App 后端骨架

- [x] 2.1 新增 `internal/app/identity/app.go` — 实现 `app.App` 接口（Name/Models/Seed/Providers/Routes/Tasks），`init()` 中调用 `app.Register()`
- [x] 2.2 新增 `internal/app/identity/model.go` — IdentitySource 模型、OIDCConfig/LDAPConfig 结构体、ToResponse()（敏感字段脱敏）
- [x] 2.3 新增 `internal/app/identity/crypto.go` — AES-256-GCM 加密/解密，密钥从 ENCRYPTION_KEY 或 SystemConfig 读取
- [x] 2.4 新增 `internal/app/identity/repository.go` — GORM CRUD、按域名查找、列表

## 3. App 管理 Service & Handler

- [x] 3.1 新增 `internal/app/identity/service.go` — 业务逻辑：创建（域名唯一性校验）、更新（保留密文）、删除、切换、测试连接、列表
- [x] 3.2 新增 `internal/app/identity/handler.go` — 管理 API handler：List/Create/Update/Delete/Toggle/TestConnection
- [x] 3.3 在 `app.go` 的 `Providers()` 中注册 repository → service → handler 到 IOC
- [x] 3.4 在 `app.go` 的 `Routes()` 中注册 `/identity-sources` 路由组
- [x] 3.5 在 `app.go` 的 `Seed()` 中注册 Casbin 策略 + "身份源管理" 菜单项

## 4. OIDC 协议集成（App 内）

- [x] 4.1 添加依赖 `github.com/coreos/go-oidc/v3` 和 `golang.org/x/oauth2`
- [x] 4.2 新增 `internal/app/identity/oidc.go` — OIDC Provider 封装：Discovery 缓存（1h TTL）、PKCE 生成、AuthURL 构建、Code 交换、ID Token 验证
- [x] 4.3 扩展内核 `StateManager` 或在 App 内实现 SSO 状态管理（存储 sourceID + codeVerifier）

## 5. LDAP 协议集成（App 内）

- [x] 5.1 添加依赖 `github.com/go-ldap/ldap/v3`
- [x] 5.2 新增 `internal/app/identity/ldap.go` — LDAP 认证：连接建立（TLS/StartTLS）、Admin Bind + 搜索 + Re-bind 验证、属性映射

## 6. ExternalAuthenticator 实现 + SSO Handler（App 内）

- [x] 6.1 新增 `internal/app/identity/authenticator.go` — 实现 `ExternalAuthenticator` 接口：LDAP fallback 认证、域名检测、ForceSso 判断
- [x] 6.2 在 `app.go` 的 `Providers()` 中注册 `ExternalAuthenticator` 到 IOC（`do.Provide`）
- [x] 6.3 新增 `internal/app/identity/sso_handler.go` — 公开端点：CheckDomain / InitiateSSO / SSOCallback
- [x] 6.4 在 `Routes()` 中注册 SSO 端点：`/auth/check-domain`、`/auth/sso/:id/authorize`、`/auth/sso/callback`
- [x] 6.5 实现 JIT 用户供给：复用内核 UserRepo + UserConnectionRepo（IOC 解析），用 `oidc_{id}` / `ldap_{id}` 作 Provider
- [x] 6.6 调用内核 `AuthService.GenerateTokenPair()` 返回 TokenPair

## 7. Edition 集成

- [x] 7.1 修改 `cmd/server/edition_full.go` — 添加 `import _ "metis/internal/app/identity"`
- [x] 7.2 验证 `edition_lite.go` 不导入 identity App，精简版编译通过

## 8. 前端 App：身份源管理页面

- [x] 8.1 新增 `web/src/apps/identity/module.ts` — 调用 `registerApp()` 注册路由（管理页 + SSO 回调页）
- [x] 8.2 新增 `web/src/apps/identity/pages/index.tsx` — 身份源列表页（表格、空状态、操作按钮）
- [x] 8.3 新增创建/编辑 Sheet 表单 — 类型选择器动态切换 OIDC/LDAP 字段，Zod 校验
- [x] 8.4 实现 Toggle、测试连接按钮、删除确认 AlertDialog
- [x] 8.5 新增 `web/src/apps/identity/pages/sso-callback.tsx` — SSO 回调页面
- [x] 8.6 修改 `web/src/apps/registry.ts` — 添加 `import './identity/module'`

## 9. ~~前端内核：登录页增强~~ （取消 — 登录页保持原样，SSO 流程不依赖域名检测）

- ~~9.1 修改 `web/src/pages/login/index.tsx`~~
- ~~9.2 实现三模式切换~~

## 10. 集成验证

- [x] 10.1 full edition 启动 — 验证身份源 CRUD、Casbin 策略、菜单种子数据
- [x] 10.2 lite edition 启动 — 验证无 identity App 时编译通过，登录页 check-domain 返回 404 优雅降级
- [x] 10.3 验证域名检测 + SSO 按钮展示
- [x] 10.4 验证 ExternalAuthenticator 注入 — 有 App 时 LDAP fallback 生效，无 App 时跳过
