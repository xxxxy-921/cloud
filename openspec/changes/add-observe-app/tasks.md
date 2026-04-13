## 1. 后端 App 骨架

- [ ] 1.1 创建 `internal/app/observe/` 目录，新建 `app.go` 实现 App 接口（Name/Models/Seed/Providers/Routes/Tasks）
- [ ] 1.2 在 `cmd/server/edition_full.go` 新增 `_ "metis/internal/app/observe"` import

## 2. Token 数据模型与 Token 工具函数

- [ ] 2.1 新建 `internal/app/observe/model.go`，定义 `IntegrationToken` struct（含 user_id、org_id *uint、scope、name、token_hash、token_prefix、last_used_at、revoked_at、created_at）
- [ ] 2.2 新建 `internal/app/observe/token.go`，实现 `GenerateIntegrationToken()`（格式 `itk_<32hex>`）和 `ValidateIntegrationToken()` 函数

## 3. Repository 层

- [ ] 3.1 新建 `internal/app/observe/repo.go`，实现 `IntegrationTokenRepo`：`Create`、`ListByUserID`（仅未撤销）、`FindByPrefix`（用于验证）、`Revoke`（写 revoked_at）、`UpdateLastUsed`

## 4. Service 层

- [ ] 4.1 新建 `internal/app/observe/service.go`，实现 `IntegrationTokenService`：`Create`（含数量上限检查）、`List`、`Revoke`（含主动清除缓存）
- [ ] 4.2 在 Service 中实现验证缓存逻辑：内存 map + TTL 60s，`Verify(raw string) (*VerifyResult, error)` 方法，先查缓存再 bcrypt，撤销时清除缓存条目

## 5. Handler 层

- [ ] 5.1 新建 `internal/app/observe/handler.go`，实现 `IntegrationTokenHandler`：`Create`、`List`、`Revoke` 三个 HTTP handler，使用标准 `handler.OK` / `handler.Fail` 响应格式
- [ ] 5.2 新建 `internal/app/observe/auth_handler.go`，实现 `AuthHandler.Verify`：从 Authorization header 提取 Token，调用 Service.Verify，成功则写入 `X-Metis-User-Id`、`X-Metis-Token-Id`、`X-Metis-Scope` response header，返回 200；失败返回 401

## 6. 中间件与路由注册

- [ ] 6.1 新建 `internal/app/observe/middleware.go`，实现 `IntegrationTokenMiddleware`（模式同 `node/middleware.go`）
- [ ] 6.2 在 `app.go` 的 `Routes()` 方法中注册受 JWT+Casbin 保护的 CRUD 路由（`/api/v1/observe/tokens`），并通过 `do.MustInvoke[*gin.Engine]` 注册独立的 ForwardAuth 路由组 `/api/v1/observe/auth`

## 7. Seed 数据

- [ ] 7.1 新建 `internal/app/observe/seed.go`，创建菜单（Integrations 目录 + Integration Catalog 子菜单 + API Tokens 子菜单）及按钮权限（token:create/list/revoke）
- [ ] 7.2 在 seed 中添加 admin 角色的 Casbin 策略（API 路由权限 + 菜单权限）
- [ ] 7.3 在 seed 中通过 `db.FirstOrCreate` 写入 SystemConfig key `observe.otel_endpoint`（默认空字符串）

## 8. IOC 注册

- [ ] 8.1 在 `app.go` 的 `Providers()` 方法中注册所有依赖：`do.Provide` 注册 Repo → Service → Handler → AuthHandler

## 9. 前端 App 骨架

- [ ] 9.1 创建 `web/src/apps/observe/` 目录结构：`module.ts`、`locales/zh-CN.json`、`locales/en.json`、`data/integrations.ts`
- [ ] 9.2 在 `web/src/apps/_bootstrap.ts` 中新增 `import "./observe/module"`

## 10. 集成数据定义

- [ ] 10.1 在 `data/integrations.ts` 中定义集成模板类型和 11 个集成配置（APM×4 + Metrics×4 + Logs×3），每个包含 slug、name、icon、category、dockerComposeSnippet、binarySnippet

## 11. Integration Catalog 页面

- [ ] 11.1 新建 `pages/integrations/index.tsx`，实现卡片网格布局、搜索框过滤、分类 Tab 切换
- [ ] 11.2 新建 Integration 卡片组件，DataDog 风格（白色背景、细边框、hover 效果、图标+名称+标签）
- [ ] 11.3 新建 `pages/integrations/[slug].tsx`，实现三步引导页：Token 选择区块（下拉 + Endpoint 展示）、配置片段区块（Docker/Binary Tab + 一键复制）、验证区块

## 12. API Tokens 管理页面

- [ ] 12.1 新建 `pages/tokens/index.tsx`，实现 Token 列表（卡片式，含前缀/名称/scope/时间）、空态、数量上限提示
- [ ] 12.2 实现新建 Token Sheet：name 表单 → 提交 → 明文一次性展示区域（含警告文案 + 一键复制 + 关闭前确认）
- [ ] 12.3 实现撤销 Token 二次确认 Dialog（含 Token 名称和断开警告文案）

## 13. 前端 API 集成

- [ ] 13.1 在 `lib/api.ts` 或 observe app 内新增 API 调用函数：`createToken`、`listTokens`、`revokeToken`、`getOtelEndpoint`
- [ ] 13.2 在详情页中实现 Token 选择后的配置片段动态替换逻辑（`{{TOKEN}}` → 实际值，`{{ENDPOINT}}` → endpoint 值）

## 14. 路由注册

- [ ] 14.1 在 `module.ts` 中调用 `registerApp()` 注册路由：`observe/integrations`（index + `[slug]`）和 `observe/tokens`（index）

## 15. 构建验证

- [ ] 15.1 运行 `go build -tags dev ./cmd/server/` 确认后端编译无误
- [ ] 15.2 运行 `cd web && bun run lint` 确认前端 ESLint 无报错
