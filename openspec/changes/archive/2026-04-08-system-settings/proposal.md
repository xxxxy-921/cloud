## Why

现有系统配置页（/config）是面向开发者的通用 key-value CRUD 表格，缺少面向管理员的友好设置界面。需要提供系统名称和 Logo 的可视化设置功能，让管理员能自定义应用品牌标识，并在 TopNav 中动态展示。

## What Changes

- 新增"系统设置"页面（/settings），提供系统名称输入和 Logo 上传/预览/移除
- 新增后端 site-info API 端点：获取站点信息、提供 Logo 图片、更新名称、上传/删除 Logo
- 复用现有 `system_config` 表，用 `system.app_name` 和 `system.logo` 约定 key 存储
- Logo 以 base64 data URL 存入数据库，后端提供解码为二进制图片的端点
- TopNav 动态读取站点信息展示名称和 Logo
- 导航配置从单文件拆分为按 App 分文件结构（`lib/nav/`），提升可维护性

## Capabilities

### New Capabilities
- `site-info-api`: 后端站点信息 API — 获取/更新系统名称、上传/获取/删除 Logo
- `settings-page`: 前端系统设置页面 — 系统名称表单 + Logo 上传卡片
- `nav-modules`: 导航配置模块化 — 按 App 拆分文件，中心汇总

### Modified Capabilities
- `activebar-layout`: TopNav 从硬编码"Metis"改为动态读取站点名称和 Logo

## Impact

- **后端新增**：`internal/handler/site_info.go` 处理 site-info 端点，路由注册
- **前端新增**：`src/pages/settings/` 设置页面，`src/lib/nav/` 模块化导航
- **前端修改**：TopNav 组件改为动态读取站点信息，`src/lib/nav.ts` 拆分为目录结构
- **API 新增**：`GET/PUT /api/v1/site-info`、`GET/PUT/DELETE /api/v1/site-info/logo`
- **无数据库迁移**：复用现有 system_config 表，仅约定新的 key 名
