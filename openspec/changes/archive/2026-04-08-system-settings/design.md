## Context

Metis 后端已有 `system_config` 表（key-value 存储）和完整 CRUD API（`/api/v1/config`）。前端有通用配置管理页。TopNav 当前硬编码显示 "Metis"。导航配置集中在单个 `lib/nav.ts` 文件中。

参考 NekoAdmin：同一张 system_setting 表支撑两种访问——通用配置表格（开发者）和专用设置页面（管理员）。Logo 以 base64 data URL 存库。

## Goals / Non-Goals

**Goals:**
- 提供管理员友好的系统设置页面（名称 + Logo）
- TopNav 动态展示站点名称和 Logo
- 导航配置模块化，按 App 拆分文件
- 复用现有 system_config 基础设施，零迁移

**Non-Goals:**
- 安全策略设置（密码策略、登录锁定等）— 后续 change
- 文件存储服务 — Logo 用 base64 入库，不引入独立存储
- 多主题/暗色模式切换
- 国际化

## Decisions

### D1: 存储策略 → 复用 system_config + key 前缀约定

**选择**：在现有 `system_config` 表中用 `system.app_name`、`system.logo` 作为 key
**替代**：添加 `group` 字段做正式分组；新建 `site_settings` 表
**理由**：零迁移，不改模型。key 前缀已经足够区分用途，体量小不需要正式分组机制。

### D2: Logo 存储 → base64 data URL 入库

**选择**：`data:image/png;base64,...` 完整 data URL 存入 value 字段
**替代**：存文件到磁盘目录；引入 Blob 存储
**理由**：与 NekoAdmin 一致，2MB 上限可控。单二进制部署不需要额外数据目录。后端提供解码端点返回二进制图片。

### D3: API 设计 → 独立 site-info 端点

**选择**：新增 `/api/v1/site-info` 系列端点，handler 内部调用 SystemConfig repo
**替代**：直接用现有 `/api/v1/config` 端点读写特定 key
**理由**：专用端点可以做 Logo 文件上传/下载处理，不污染通用配置 API 语义。Logo 端点返回二进制图片（Content-Type: image/*），前端直接 `<img src="...">` 使用。

API 设计：
```
GET    /api/v1/site-info        → { appName: string, hasLogo: bool }
PUT    /api/v1/site-info        → body: { appName: string } → 更新名称
GET    /api/v1/site-info/logo   → 二进制图片 (解码 base64)
PUT    /api/v1/site-info/logo   → body: { data: "data:image/..." } → 上传 Logo
DELETE /api/v1/site-info/logo   → 移除 Logo
```

### D4: 导航模块化 → 按 App 拆文件

**选择**：`lib/nav/` 目录，每个 App 一个文件，index.ts 汇总
**替代**：页面自注册模式（每个 pages/*/nav.ts 声明自己的导航）
**理由**：对 Metis 体量足够，一个 App 文件看全部子菜单是优势。页面自注册适合插件化架构，此处过度设计。

结构：
```
lib/nav/
  types.ts     → AppDef, NavItemDef, breadcrumbLabels 类型和工具
  home.ts      → homeApp 定义
  system.ts    → systemApp 定义（含 config + settings 两个 item）
  index.ts     → 导出 apps[]、findActiveApp()、breadcrumbLabels
```

### D5: 前端设置页 → Card 分区表单

**选择**：`/settings` 页面，两个 Card 区域：系统名称（Input + 保存按钮）、Logo（预览 + 上传 + 移除）
**替代**：Sheet 抽屉表单（与配置页一致）；单个大表单
**理由**：设置页是独立页面不是列表操作，Card 分区比 Sheet 更合适。每个设置项独立保存，不需要"全部保存"。

## Risks / Trade-offs

- **[base64 Logo 影响配置列表查询]** → /api/v1/config 列表会返回 system.logo 的大 value。可接受：列表页低频访问，2MB 上限。后续可考虑列表 API 支持排除特定 key。
- **[key 命名无强制约束]** → 用户可能通过通用配置页修改 system.* key 导致设置页异常。可接受：当前无权限系统，后续加权限可限制。
- **[Logo 上传无裁剪/缩放]** → 用户需自行准备合适尺寸图片。可接受：MVP 阶段，后续可加前端裁剪。
