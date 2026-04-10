## 1. 后端模型与加密

- [x] 1.1 在 model.go 中新增 License struct（含所有字段、TableName、ToResponse）和 LicenseResponse struct
- [x] 1.2 在 app.go 的 Models() 中注册 License 模型以启用 AutoMigrate
- [x] 1.3 在 crypto.go 中实现 `Canonicalize(v any) (string, error)` — 递归排序 JSON key
- [x] 1.4 在 crypto.go 中实现 `SignLicense(payload map[string]any, encryptedPrivateKey string, encKey []byte) (string, error)` — 解密私钥 → canonicalize → Ed25519 签名 → base64url
- [x] 1.5 在 crypto.go 中实现 `GenerateActivationCode(payload map[string]any, signature string) (string, error)` — 合并 payload+sig → JSON → base64url
- [x] 1.6 在 crypto.go 中实现 `DecodeActivationCode(code string) (map[string]any, error)` — base64url → JSON → map

## 2. 后端 Repository

- [x] 2.1 新建 license_repository.go，实现 LicenseRepo：Create（事务内）、FindByID（LEFT JOIN product/licensee）、List（支持 productId/licenseeId/status/keyword 筛选 + 分页）、UpdateStatus
- [x] 2.2 在 ProductKeyRepo 中新增 `FindByProductIDAndVersion(productID uint, version int)` 方法（导出 .lic 时需要按版本查密钥）

## 3. 后端 Service

- [x] 3.1 新建 license_service.go，实现 LicenseService
- [x] 3.2 实现 `IssueLicense` 方法：校验 product(published) → 校验 licensee(active) → 获取当前 key → 构建 payload → 签名 → 生成 activationCode → 事务内创建记录
- [x] 3.3 实现 `RevokeLicense` 方法：校验存在 → 校验 status=issued → 更新 status/revokedAt/revokedBy
- [x] 3.4 实现 `GetLicense` 和 `ListLicenses` 方法
- [x] 3.5 实现 `ExportLicFile` 方法：校验 status=issued → 按 keyVersion 查公钥 → 组装 LicFile JSON → 返回 bytes + filename

## 4. 后端 Handler & 路由

- [x] 4.1 新建 license_handler.go，实现 LicenseHandler：Issue、Revoke、Get、List、Export
- [x] 4.2 在 app.go 的 Providers() 中注册 LicenseRepo、LicenseService、LicenseHandler
- [x] 4.3 在 app.go 的 Routes() 中注册 license 路由组（POST / GET / GET :id / PATCH :id/revoke / GET :id/export）
- [x] 4.4 Export endpoint 设置 Content-Disposition header 实现文件下载

## 5. 种子数据

- [x] 5.1 在 seed.go 中新增「许可签发」菜单（permission=license:license:list, path=/license/licenses）
- [x] 5.2 新增按钮权限：签发(license:license:issue)、吊销(license:license:revoke)
- [x] 5.3 新增 admin 角色的 Casbin 策略：licenses 系列 endpoint + 菜单权限

## 6. 前端路由与页面框架

- [x] 6.1 在 module.ts 中注册 `/license/licenses` 和 `/license/licenses/:id` 路由（lazy-load）
- [x] 6.2 新建 pages/licenses/index.tsx — 许可列表页（useListPage hook、筛选器、数据表、状态标签）
- [x] 6.3 新建 pages/licenses/[id].tsx — 许可详情页（基本信息、有效期、约束值展示、签发/吊销信息、操作按钮）

## 7. 前端签发表单

- [x] 7.1 新建 components/issue-license-sheet.tsx — 签发 Sheet 表单
- [x] 7.2 实现商品选择下拉（仅 published）+ 选择后加载套餐列表
- [x] 7.3 实现授权主体选择下拉（仅 active）
- [x] 7.4 实现套餐选择 + 约束值配置（选套餐自动填充，或选"自定义"显示 ConstraintValueForm）
- [x] 7.5 实现注册码、生效日期、过期日期、备注字段
- [x] 7.6 提交逻辑：调用签发 API，成功后关闭 Sheet 并 invalidate 列表

## 8. 前端吊销与导出

- [x] 8.1 实现吊销确认 Dialog + 调用吊销 API
- [x] 8.2 实现 .lic 文件导出（调用 export API，触发浏览器下载）
- [x] 8.3 在列表行操作和详情页中集成吊销和导出功能
