## Why

当前 Metis 仅支持用户名+密码登录。作为管理系统，需要支持 OAuth 第三方登录（GitHub、Google），降低用户注册/登录门槛，同时为后续扩展微信、企业 SSO 等认证源打好基础。

## What Changes

- 新增 `auth_providers` 表，存储 OAuth 认证源配置（ClientID/Secret、Scopes 等），管理员可在后台启用/禁用
- 新增 `user_connections` 表，记录用户与外部 OAuth 身份的绑定关系（provider + externalID）
- 升级 `users` 表结构：Username 和 Password 改为可选字段，支持纯 OAuth 用户（无密码）
- 新增 OAuth 授权/回调 API 端点，实现标准 OAuth2 Authorization Code 流程
- 首次 OAuth 登录自动创建本地用户（JIT provisioning），用户名自动生成（如 `github_12345`）
- 邮箱冲突时拒绝自动关联，提示用户先用密码登录后手动绑定
- 已登录用户可在个人设置页查看/绑定/解绑外部账号
- 前端登录页动态展示已启用的 OAuth 登录按钮
- 不持久化 OAuth 平台的 access_token/refresh_token，仅存储身份标识信息
- 首期只支持 GitHub 和 Google，架构预留扩展其他 provider

## Capabilities

### New Capabilities
- `auth-providers`: OAuth 认证源配置管理、OAuth 授权/回调流程、用户外部身份绑定（user_connections），以及前端 OAuth 登录 UI 和账号关联管理

### Modified Capabilities
- `user-auth`: User 模型升级（Username/Password 可选），新增 OAuth 相关公开端点（providers 列表、授权跳转、回调处理），登录逻辑兼容 OAuth 用户
- `user-auth-frontend`: 登录页增加 OAuth 按钮区域，新增 OAuth 回调前端路由，auth store 支持 OAuth 登录态
- `user-management`: 用户列表展示登录方式标识（本地/GitHub/Google），用户详情显示已绑定的外部账号

## Impact

- **后端新增依赖**: `golang.org/x/oauth2` 及其 GitHub/Google endpoint 子包
- **数据库**: 新增 `auth_providers`、`user_connections` 两张表；`users` 表 Username/Password 字段约束变更
- **API**: 新增 4 个公开端点（providers 列表、授权 URL、回调处理、OAuth 登录）+ 3 个已认证端点（连接列表、绑定、解绑）+ 3 个管理端点（认证源 CRUD）
- **前端**: 登录页改造、新增 OAuth 回调路由页、设置页增加账号关联卡片
- **Seed**: 新增 auth_providers 相关 Casbin 策略
- **安全**: OAuth state 参数 CSRF 防护（内存 Map + TTL）；ClientSecret 明文存储但 API 响应中隐藏
