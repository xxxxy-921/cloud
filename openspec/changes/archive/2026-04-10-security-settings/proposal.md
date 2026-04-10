## Why

Metis 目前缺乏常见的安全管控能力——没有密码复杂度校验、登录失败不锁定、无 2FA、不支持自助注册、登录无验证码。这些是企业级管理系统的标配功能，需要一次性补齐，统一在安全设置页面中管理。

## What Changes

- 新增**密码策略**：最小长度、大小写/数字/特殊字符要求、密码过期天数、管理员强制重置
- 新增**登录锁定**：失败次数累计、自动锁定时长、管理员手动解锁
- 新增**TOTP 两步验证**：用户可选启用、管理员可强制全员启用、恢复码机制
- 新增**用户注册**：管理员可开关自助注册、配置默认角色
- 新增**登录验证码**：图形验证码（base64Captcha），内存存储，预留 Turnstile/reCAPTCHA 扩展位
- 增强**会话管理**：refresh token 有效期从硬编码 7 天改为可配置
- 扩展**安全设置页面**：从当前 2 个配置项扩展为 6 个分组（密码策略、登录安全、会话管理、两步验证、注册设置、审计日志）
- User model 新增字段：PasswordChangedAt、ForcePasswordReset、FailedLoginAttempts、LockedUntil、TwoFactorEnabled
- 新增 TwoFactorSecret 表存储 TOTP 密钥和恢复码
- Login 流程变更：锁定检查 → 验证码校验 → 密码验证 → 2FA 验证 → 签发 token

## Capabilities

### New Capabilities
- `password-policy`: 密码复杂度验证规则、密码过期检查中间件、管理员强制重置
- `login-lockout`: 登录失败计数、账户自动锁定与解锁、管理员手动解锁
- `totp-two-factor`: TOTP 密钥生成与验证、恢复码、2FA 设置页面、登录时 2FA 验证步骤
- `user-registration`: 自助注册开关、注册端点与页面、默认角色分配
- `login-captcha`: 图形验证码生成（base64Captcha）、内存存储与 TTL、登录时验证码校验

### Modified Capabilities
- `user-auth`: Login 流程增加锁定检查、验证码校验、2FA 步骤；JWT claims 新增 passwordChangedAt/forcePasswordReset；Change password 更新 PasswordChangedAt
- `user-auth-frontend`: Login 页面增加验证码组件和注册链接；新增 /register 和 /2fa 路由；API 拦截器处理 409 PASSWORD_EXPIRED 跳转改密页
- `session-management`: Refresh token 有效期从硬编码改为读取 security.session_timeout_minutes 配置
- `typed-settings-api`: Security settings 扩展 13 个新配置字段
- `settings-page`: 安全设置 tab 从 1 个 card 扩展为 6 个分组

## Impact

- **后端**：internal/service/auth.go（登录流程大改）、internal/pkg/token/（JWT claims 扩展 + 密码验证）、internal/model/user.go（5 个新字段）、新增 model/two_factor_secret.go、新增 service/captcha.go、internal/middleware/（新增密码过期中间件）、internal/seed/（新增 13 个 SystemConfig 默认值）
- **前端**：web/src/pages/login（验证码 + 注册链接）、新增 pages/register、新增 pages/2fa、pages/settings/security-card 大改、lib/api.ts（409 拦截）、stores/auth（2FA 状态）
- **新依赖**：Go - `github.com/pquerna/otp`（TOTP）、`github.com/mojocn/base64Captcha`（图形验证码）；前端 - `react-qr-code`（2FA QR 码）
- **数据库**：User 表新增 5 列、新建 two_factor_secrets 表、SystemConfig 新增 13 行种子数据
- **API**：新增 POST /api/v1/auth/register、GET /api/v1/captcha、POST /api/v1/auth/2fa/setup、POST /api/v1/auth/2fa/verify、POST /api/v1/auth/2fa/confirm、DELETE /api/v1/auth/2fa、POST /api/v1/auth/2fa/login
