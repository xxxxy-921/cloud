## 1. 基础设施：Model 变更 + 配置种子

- [x] 1.1 User model 新增字段：PasswordChangedAt (*time.Time), ForcePasswordReset (bool), FailedLoginAttempts (int), LockedUntil (*time.Time), TwoFactorEnabled (bool)；更新 ToResponse 方法
- [x] 1.2 新建 TwoFactorSecret model（ID, UserID unique, Secret, BackupCodes, CreatedAt, UpdatedAt）；注册到 AutoMigrate
- [x] 1.3 seed.go 新增 13 个 SystemConfig 默认值（password_min_length, password_require_upper, password_require_lower, password_require_number, password_require_special, password_expiry_days, login_max_attempts, login_lockout_minutes, session_timeout_minutes, require_two_factor, registration_open, default_role_code, captcha_provider）
- [x] 1.4 新增 Go 依赖：`github.com/pquerna/otp` + `github.com/mojocn/base64Captcha`

## 2. 密码策略

- [x] 2.1 实现 ValidatePassword(plain, policy) 函数，放在 internal/pkg/token/ 或新建 internal/pkg/security/
- [x] 2.2 SettingsService 新增 GetPasswordPolicy() 方法，从 SystemConfig 读取策略
- [x] 2.3 在 UserService.Create、AuthService.ChangePassword 中接入密码策略校验
- [x] 2.4 JWT claims 扩展：生成 token 时写入 passwordChangedAt 和 forcePasswordReset
- [x] 2.5 新增密码过期检查 middleware
- [x] 2.6 在 main.go 中间件链中注册 PasswordExpiryMiddleware

## 3. 登录锁定

- [x] 3.1 UserRepository 新增 IncrementFailedAttempts(userID) 原子更新方法
- [x] 3.2 UserRepository 新增 LockUser(userID, duration) 和 UnlockUser(userID) 方法
- [x] 3.3 AuthService.Login 流程重构：密码验证前检查 LockedUntil，失败后调用 IncrementFailedAttempts，到阈值触发锁定
- [x] 3.4 UserHandler.Update 支持管理员解锁操作（重置 FailedLoginAttempts 和 LockedUntil）

## 4. 验证码

- [x] 4.1 新建 internal/service/captcha.go：CaptchaStore（sync.Map + TTL 清理 goroutine）+ Generate() + Verify()
- [x] 4.2 新建 GET /api/v1/captcha 端点（CaptchaHandler），加入 JWT/Casbin 白名单
- [x] 4.3 AuthService.Login 中接入验证码校验（读 X-Captcha-Id/X-Captcha-Answer header），位于锁定检查之后、密码验证之前

## 5. 会话超时配置化

- [x] 5.1 修改 token.GenerateRefreshToken：从 SettingsService 读取 session_timeout_minutes 替代硬编码 7 天
- [x] 5.2 SettingsService 新增 GetSessionTimeoutMinutes() 方法

## 6. TOTP 两步验证

- [x] 6.1 新建 TwoFactorSecretRepository（CRUD，按 UserID 查询）
- [x] 6.2 新建 TwoFactorService：Setup（生成 secret + QR URI）、Confirm（验证 TOTP + 保存 + 生成恢复码）、Verify（校验 TOTP 或恢复码）、Disable（删除记录）
- [x] 6.3 新建 TwoFactorHandler：POST /2fa/setup、POST /2fa/confirm、DELETE /2fa、POST /2fa/login
- [x] 6.4 实现 twoFactorToken 签发（5 分钟有效，purpose="2fa"的 JWT）
- [x] 6.5 AuthService.Login 中接入 2FA：TwoFactorEnabled 时返回 202 + twoFactorToken
- [x] 6.6 2FA 相关路由加入 Casbin 白名单（/api/v1/auth/2fa/*）
- [x] 6.7 实现 require_two_factor 强制模式：登录响应附加 requireTwoFactorSetup 标记

## 7. 用户注册

- [x] 7.1 新建 POST /api/v1/auth/register 端点（AuthHandler 或新 RegisterHandler）
- [x] 7.2 实现注册逻辑：检查 registration_open → 密码策略校验 → 创建用户 → 分配默认角色 → 签发 token
- [x] 7.3 新建 GET /api/v1/auth/registration-status 端点（公开，返回 registrationOpen）
- [x] 7.4 注册相关路由加入 JWT/Casbin 白名单

## 8. Settings API 扩展

- [x] 8.1 扩展 SecuritySettings struct：新增 13 个字段
- [x] 8.2 更新 SettingsService.GetSecuritySettings / UpdateSecuritySettings 读写全部 16 个配置
- [x] 8.3 新增输入验证：passwordMinLength >= 1、captchaProvider 枚举校验、数值 >= 0

## 9. 前端：安全设置页面

- [x] 9.1 重构 security-card.tsx 为 6 个分组卡片（密码策略 / 登录安全 / 会话管理 / 两步验证 / 注册设置 / 日志保留）
- [x] 9.2 角色下拉选项从 GET /api/v1/roles 获取，用于默认角色配置
- [x] 9.3 表单验证（Zod schema）+ 保存调用 PUT /api/v1/settings/security

## 10. 前端：登录页增强

- [x] 10.1 Login 页面集成验证码组件（条件渲染图形验证码 + 刷新按钮）
- [x] 10.2 Login 页面添加注册链接（条件渲染，调用 registration-status API）
- [x] 10.3 Login 表单提交时附带 X-Captcha-Id/X-Captcha-Answer headers
- [x] 10.4 处理 202 (needsTwoFactor) 响应：保存 twoFactorToken 跳转 /2fa
- [x] 10.5 处理 423 (locked) 响应：显示锁定剩余时间

## 11. 前端：新页面

- [x] 11.1 新建 /register 注册页面（表单 + 密码确认 + 错误显示 + 注册关闭提示）
- [x] 11.2 新建 /2fa 两步验证页面（TOTP 输入 / 恢复码切换 + twoFactorToken 传递）
- [x] 11.3 用户设置/Profile 区域新增 2FA 设置面板（启用流程：QR 码 → 确认码 → 恢复码展示；关闭流程）
- [x] 11.4 前端新增依赖：react-qr-code

## 12. 前端：拦截器与路由守卫

- [x] 12.1 api.ts 拦截器新增 409 PASSWORD_EXPIRED 处理：跳转改密页 + 提示
- [x] 12.2 AuthGuard 新增 requireTwoFactorSetup 检查：限制导航至 2FA 设置页
- [x] 12.3 路由注册：/register、/2fa 添加为公开路由（无 AuthGuard）
