## 1. 后端 site-info API

- [x] 1.1 创建 `internal/handler/site_info.go`：实现 GetSiteInfo（读取 system.app_name + 检查 system.logo 是否存在）
- [x] 1.2 实现 UpdateSiteInfo：验证 appName 非空，upsert system.app_name
- [x] 1.3 实现 GetLogo：读取 system.logo，解析 base64 data URL，返回二进制图片 + Content-Type
- [x] 1.4 实现 UploadLogo：验证 data URL 格式（data:image/*;base64,...）、解码后大小 ≤ 2MB，upsert system.logo
- [x] 1.5 实现 DeleteLogo：删除 system.logo 配置项
- [x] 1.6 在路由注册中添加 `/api/v1/site-info` 系列端点

## 2. 导航模块化重构

- [x] 2.1 创建 `src/lib/nav/types.ts`：将 AppDef、NavItemDef 类型和 findActiveApp、breadcrumbLabels 移入
- [x] 2.2 创建 `src/lib/nav/home.ts`：homeApp 定义
- [x] 2.3 创建 `src/lib/nav/system.ts`：systemApp 定义，包含"系统配置"和"系统设置"两个 item
- [x] 2.4 创建 `src/lib/nav/index.ts`：导入所有 app，导出 apps[]、findActiveApp()、breadcrumbLabels
- [x] 2.5 更新所有引用 `@/lib/nav` 的文件，确保导入路径兼容（sidebar、header、App.tsx）
- [x] 2.6 删除旧的 `src/lib/nav.ts` 文件

## 3. 系统设置前端页面

- [x] 3.1 创建 `src/pages/settings/index.tsx`：设置页面布局，包含"基本信息"和"系统 Logo"两个 Card
- [x] 3.2 创建 `src/pages/settings/site-name-card.tsx`：系统名称表单（Input + 保存按钮），React Hook Form + Zod 校验
- [x] 3.3 创建 `src/pages/settings/logo-card.tsx`：Logo 上传卡片（预览 + 文件选择 + 移除），2MB 前端校验
- [x] 3.4 在 App.tsx 路由配置中添加 /settings 路由（lazy import）

## 4. TopNav 动态站点信息

- [x] 4.1 修改 TopNav 组件：使用 TanStack Query 从 /api/v1/site-info 读取名称和 Logo 状态
- [x] 4.2 TopNav 展示：有 Logo 时显示 `<img src="/api/v1/site-info/logo">` + 名称，无 Logo 时仅显示名称

## 5. 验证

- [x] 5.1 验证 `bun run build` 前端构建成功
- [x] 5.2 验证 `make build` 完整链路通过
