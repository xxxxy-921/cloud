## 1. Model 和加密工具

- [x] 1.1 创建 `internal/app/license/model.go` — 定义 Product、Plan、ProductKey 结构体及 Response 类型、TableName、ConstraintSchema/Feature 类型
- [x] 1.2 创建 `internal/app/license/crypto.go` — Ed25519 密钥对生成、AES-256-GCM 加密/解密私钥、获取加密密钥（LICENSE_KEY_SECRET > JWT_SECRET 派生）

## 2. Repository 层

- [x] 2.1 创建 `internal/app/license/repository.go` — ProductRepo：Create/FindByID/FindByCode/List(分页+搜索+状态筛选)/Update/UpdateStatus，PlanRepo：Create/FindByID/ListByProductID/Update/Delete/SetDefault/ClearDefault，ProductKeyRepo：Create/FindCurrentByProductID/RevokeByProductID

## 3. Service 层

- [x] 3.1 创建 `internal/app/license/service.go` — ProductService：CreateProduct（含自动生成密钥对）、GetProduct、ListProducts、UpdateProduct、UpdateConstraintSchema（含 schema 校验）、UpdateStatus（状态机校验）、RotateKey、GetPublicKey
- [x] 3.2 在 service.go 中实现 PlanService：CreatePlan（含 ConstraintValues 校验）、UpdatePlan、DeletePlan、SetDefaultPlan

## 4. Handler 层

- [x] 4.1 创建 `internal/app/license/handler.go` — ProductHandler：Create/List/Get/Update/UpdateSchema/UpdateStatus/RotateKey/GetPublicKey，请求结构体和审计日志 context 设置
- [x] 4.2 在 handler.go 中实现 PlanHandler：Create/Update/Delete/SetDefault

## 5. App 注册和 Seed

- [x] 5.1 创建 `internal/app/license/app.go` — LicenseApp 实现 App 接口 + init() 注册，Models/Providers/Routes/Tasks
- [x] 5.2 创建 `internal/app/license/seed.go` — 菜单种子（「许可管理」一级目录 + 「商品管理」子菜单）+ admin 角色 Casbin 策略
- [x] 5.3 在 `cmd/server/edition_full.go` 中 import `_ "metis/internal/app/license"`
- [x] 5.4 在 Casbin 白名单中添加需要的公开路由（如有），确认 `/api/v1/license/*` 路由受 JWT+Casbin 保护

## 6. 前端 — 商品列表页

- [x] 6.1 创建 `web/src/apps/license/module.ts` — registerApp() 注册路由（/license/products 和 /license/products/:id）
- [x] 6.2 在 `web/src/apps/registry.ts` 中 import license 模块
- [x] 6.3 创建 `web/src/apps/license/pages/products/index.tsx` — 商品列表页：表格（名称/编码/状态Badge/套餐数/创建时间/操作）、搜索、状态筛选、创建按钮
- [x] 6.4 创建商品创建/编辑 Sheet 表单组件（名称、编码、描述字段，React Hook Form + Zod 校验）

## 7. 前端 — 商品详情页

- [x] 7.1 创建 `web/src/apps/license/pages/products/[id].tsx` — 商品详情页骨架，Tabs 布局（基本信息/套餐/约束/密钥）
- [x] 7.2 实现基本信息 Tab — 展示商品信息 + 编辑按钮 + 状态操作按钮（根据状态机显示可用操作）
- [x] 7.3 实现套餐管理 Tab — 套餐列表 + 创建/编辑 Sheet 表单（动态渲染 ConstraintValues 编辑器） + 删除确认 + 设为默认
- [x] 7.4 实现 ConstraintSchema 编辑器组件 — 模块增删、特性增删、类型切换、属性配置（number: min/max/default, enum: options）
- [x] 7.5 实现密钥管理 Tab — 展示当前密钥版本和公钥 + 密钥轮转按钮（AlertDialog 确认）

## 8. 联调和验证

- [ ] 8.1 运行 `make dev` + `make web-dev`，验证后端启动 seed 正确创建菜单和策略
- [ ] 8.2 验证商品 CRUD 全流程（创建 → 列表 → 详情 → 编辑 → 状态切换）
- [ ] 8.3 验证 ConstraintSchema 编辑器保存和回显
- [ ] 8.4 验证套餐管理全流程（创建 → 编辑 → 设为默认 → 删除）
- [ ] 8.5 验证密钥轮转（轮转后版本递增，旧密钥标记撤销）
