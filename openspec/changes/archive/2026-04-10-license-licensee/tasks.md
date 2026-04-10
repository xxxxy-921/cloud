## 1. 后端模型与仓库

- [x] 1.1 在 `internal/app/license/model.go` 中添加 Licensee 模型（含 BusinessInfo 结构体）、LicenseeResponse、ToResponse 方法、状态常量
- [x] 1.2 在 `internal/app/license/model.go` 中添加 code 生成函数 `generateLicenseeCode()`，使用 `crypto/rand` 生成 `LS-` + 12 位随机字母数字
- [x] 1.3 新建 `internal/app/license/licensee_repository.go`，实现 LicenseeRepo：Create / FindByID / List（keyword+status 过滤+分页）/ Update / UpdateStatus / ExistsByName

## 2. 后端服务层

- [x] 2.1 新建 `internal/app/license/licensee_service.go`，实现 LicenseeService：CreateLicensee（含 code 自动生成+重试）、GetLicensee、ListLicensees、UpdateLicensee（名称唯一性校验）、UpdateLicenseeStatus（状态机校验）

## 3. 后端 Handler 与路由

- [x] 3.1 新建 `internal/app/license/licensee_handler.go`，实现 LicenseeHandler：Create / List / Get / Update / UpdateStatus，含请求结构体定义和审计字段设置
- [x] 3.2 修改 `internal/app/license/app.go`：Models() 添加 Licensee、Providers() 注册 LicenseeRepo + LicenseeService + LicenseeHandler、Routes() 注册 `/license/licensees` 系列端点

## 4. 种子数据

- [x] 4.1 修改 `internal/app/license/seed.go`：在"许可管理"目录下添加"授权主体"菜单（path=/license/licensees, permission=license:licensee:list）及按钮权限（create/update/archive）
- [x] 4.2 在 seed.go 中添加 admin 角色的 Casbin 策略：所有 licensee API 端点 + 菜单权限

## 5. 前端列表页

- [x] 5.1 新建 `web/src/apps/license/pages/licensees/index.tsx`，使用 `useListPage` 实现分页列表，列：名称、代码（font-mono）、联系人、状态（Badge）、创建时间、操作
- [x] 5.2 实现搜索框（keyword 过滤 name/code）和状态筛选下拉框（全部/活跃/已归档）
- [x] 5.3 实现行操作：编辑按钮、归档/恢复按钮（含确认对话框），受权限控制

## 6. 前端 Drawer 表单

- [x] 6.1 新建 `web/src/apps/license/components/licensee-sheet.tsx`，使用 Sheet + React Hook Form + Zod 实现新建/编辑表单
- [x] 6.2 表单分区：基本信息（名称必填、备注）、联系信息（联系人/电话/邮箱）、企业信息可折叠区域（地址/税号/开户行/银行账号/SWIFT/IBAN）
- [x] 6.3 编辑模式下在表单顶部只读展示 code（font-mono + 复制按钮）

## 7. 前端路由与集成

- [x] 7.1 修改 `web/src/apps/license/module.ts`，注册 `/license/licensees` 路由（lazy-loaded）
- [x] 7.2 在 `web/src/lib/api.ts` 或就近位置添加 licensee 相关的 API 调用函数
