## Why

许可管理模块目前只有商品（Product）、套餐（Plan）和密钥（ProductKey）管理。缺少"授权主体"（Licensee）——即被授权使用软件的客户/组织。没有授权主体，后续的许可签发（License Issuance）无法进行，因为许可需要绑定"签给谁"。

## What Changes

- 新增 Licensee（授权主体）实体：包含名称、自动生成的唯一代码（`LS-xxxx`）、联系信息、企业信息、备注
- 支持 active / archived 两种状态，可归档和恢复，无硬删除
- 后端完整 CRUD + 状态管理 API（handler → service → repository）
- 前端列表页 + Drawer 表单（新建/编辑），支持搜索、状态筛选、分页
- Seed 数据：菜单项、Casbin 策略
- 许可管理侧边栏新增"授权主体"菜单入口

## Capabilities

### New Capabilities
- `license-licensee`: 授权主体后端模型、服务、仓库、API 端点、状态管理、种子数据
- `license-licensee-ui`: 授权主体前端列表页、Drawer 表单、搜索筛选、状态操作

### Modified Capabilities

（无现有 spec 需要修改）

## Impact

- 后端：`internal/app/license/` 新增 Licensee 相关的 model、repository、service、handler 代码，修改 `app.go` 注册新模型/Provider/路由，修改 `seed.go` 添加菜单和策略
- 前端：`web/src/apps/license/` 新增 licensees 页面和组件，修改 `module.ts` 注册新路由
- API：新增 `/api/v1/license/licensees` 系列端点
- 数据库：新增 `license_licensees` 表（SQLite，GORM AutoMigrate）
