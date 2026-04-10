## 1. 数据模型 & 数据库

- [x] 1.1 创建 `internal/model/auth_provider.go` — AuthProvider 模型（ProviderKey, DisplayName, Enabled, ClientID, ClientSecret, Scopes, CallbackURL, SortOrder）+ ToResponse() 隐藏 Secret
- [x] 1.2 创建 `internal/model/user_connection.go` — UserConnection 模型（UserID, Provider, ExternalID, ExternalName, ExternalEmail, AvatarURL）+ 唯一约束
- [x] 1.3 修改 `internal/model/user.go` — Username/Password 字段去掉 required binding，添加 HasPassword() 辅助方法，UserResponse 增加 HasPassword 和 Connections 字段
- [x] 1.4 修改 `internal/database/database.go` — AutoMigrate 注册 AuthProvider 和 UserConnection
- [x] 1.5 修改 `internal/seed/seed.go` — 添加 auth_providers 种子数据（github/google 默认记录，Enabled=false）

## 2. Repository 层

- [x] 2.1 创建 `internal/repository/auth_provider.go` — AuthProvider CRUD（FindByKey, FindAllEnabled, FindAll, Update）
- [x] 2.2 创建 `internal/repository/user_connection.go` — UserConnection CRUD（FindByUserID, FindByProviderAndExternalID, FindByUserAndProvider, Create, Delete）

## 3. OAuth 核心逻辑

- [x] 3.1 创建 `internal/pkg/oauth/provider.go` — OAuthProvider 接口定义 + OAuthUserInfo 结构体
- [x] 3.2 创建 `internal/pkg/oauth/github.go` — GitHub OAuth 实现（golang.org/x/oauth2，GetAuthURL, ExchangeCode 获取用户信息）
- [x] 3.3 创建 `internal/pkg/oauth/google.go` — Google OAuth 实现（golang.org/x/oauth2/google，GetAuthURL, ExchangeCode 获取用户信息）
- [x] 3.4 创建 `internal/pkg/oauth/state.go` — OAuth state 管理器（sync.Map + TTL 过期清理 goroutine）

## 4. Service 层

- [x] 4.1 创建 `internal/service/auth_provider.go` — AuthProvider 服务（ListEnabled, ListAll, Update, Toggle, BuildOAuthProvider 工厂方法）
- [x] 4.2 创建 `internal/service/user_connection.go` — UserConnection 服务（ListByUser, Bind, Unbind 含最后登录方式校验）
- [x] 4.3 修改 `internal/service/auth.go` — 新增 OAuthLogin 方法（查找/创建用户 + 连接，邮箱冲突检测，签发 TokenPair），GetMe 返回 connections + hasPassword

## 5. Handler 层 & 路由

- [x] 5.1 创建 `internal/handler/auth_provider.go` — 管理端点（GET/PUT/PATCH admin auth-providers）
- [x] 5.2 修改 `internal/handler/auth.go` — 新增公开端点（GET providers, GET oauth/:provider, POST oauth/callback）+ 已认证端点（GET/POST/DELETE connections）
- [x] 5.3 修改 `internal/handler/handler.go` — 注册新路由
- [x] 5.4 修改 `internal/seed/policies.go` — 添加 auth-providers 管理端点的 Casbin 策略

## 6. IOC 注册 & 依赖

- [x] 6.1 添加 `golang.org/x/oauth2` 依赖（go get）
- [x] 6.2 修改 `cmd/server/main.go` — 注册 AuthProvider/UserConnection 的 repository、service，注册 OAuth state manager

## 7. 前端 — 登录页改造

- [x] 7.1 修改 `web/src/pages/login/index.tsx` — 添加 OAuth 按钮区域（动态加载 providers API），点击跳转授权 URL
- [x] 7.2 创建 `web/src/pages/oauth/callback.tsx` — OAuth 回调页（提取 code+state，调后端 API，存 TokenPair，跳转首页）
- [x] 7.3 修改 `web/src/stores/auth.ts` — 新增 oauthLogin 方法（存 TokenPair + fetchUser）
- [x] 7.4 修改 `web/src/App.tsx` — 添加 /oauth/callback 路由（公开，无需 AuthGuard）

## 8. 前端 — 设置页账号关联

- [x] 8.1 创建 `web/src/pages/settings/connections-card.tsx` — 账号关联卡片（显示已绑定列表、绑定/解绑按钮）
- [x] 8.2 修改 `web/src/pages/settings/index.tsx` — 引入 ConnectionsCard 组件

## 9. 前端 — 用户管理增强

- [x] 9.1 修改 `web/src/pages/users/index.tsx` — 用户列表表格新增"登录方式"列（图标展示密码/GitHub/Google）
