## Context

Metis 后端骨架已完成（Gin + GORM + samber/do），前端是 Vite 8 + React 19 + TypeScript 6 + React Compiler 空项目。需要在不引入 Next.js 的前提下搭建 SPA 前端架构。

参考项目：
- **bklite-cloud**：字体（Plus Jakarta Sans + PingFang SC）、Indigo 主色、柔和圆角
- **NekoAdmin**：ActiveBar 二级导航（Icon Rail 48px + Nav Panel 160px）、oklch 色彩、毛玻璃、200ms 呼吸感交互

约束：不使用 Dialog/Modal 做表单操作，统一用 Sheet（抽屉）。

## Goals / Non-Goals

**Goals:**
- Tailwind CSS v4 + shadcn/ui 组件库初始化
- oklch 色彩空间主题变量，毛玻璃 + 大圆角 + 呼吸感动效
- ActiveBar 二级导航：数据驱动，通过 AppDef[] 配置增减模块
- React Router 7 路由 + DashboardLayout 嵌套布局
- Zustand（客户端状态）+ TanStack Query（服务端状态）
- 系统配置页对接已有 /api/v1/config API，验证全链路
- fetch 封装 + 统一错误处理

**Non-Goals:**
- 用户认证/JWT/登录页（下一个 change）
- 用户管理页面（下一个 change）
- 国际化 i18n
- 移动端响应式（先聚焦桌面）
- 暗色模式

## Decisions

### D1: UI 框架 → shadcn/ui + Tailwind CSS v4

**选择**：shadcn/ui（源码复制模式）+ Tailwind CSS v4（原生 CSS @import）
**替代**：Ant Design（bklite-cloud 在用，但偏重）、Radix UI（底层太原始）、headless UI
**理由**：源码可控，与 Tailwind 深度集成，组件按需添加不臃肿。shadcn 的 Sheet 组件天然支持抽屉模式。

### D2: 导航模式 → ActiveBar（Icon Rail + Nav Panel）

**选择**：48px Icon Rail + 160px Nav Panel 二级导航，参考 NekoAdmin
**替代**：传统 240px 单层 Sidebar（bklite-cloud 模式）、顶部 Tab 导航
**理由**：二级导航更好地组织模块，Icon Rail 节省空间，视觉层次更清晰。数据驱动的 AppDef[] 配置，后续加模块只需加配置。

### D3: 色彩空间 → oklch

**选择**：oklch 感知均匀色彩空间
**替代**：hsl（shadcn 默认）、hex
**理由**：oklch 在 2026 已是主流，色彩插值更自然，所有现代浏览器支持。与 NekoAdmin 一致。

### D4: 路由 → React Router 7 配置式

**选择**：React Router 7，配置式路由（createBrowserRouter）
**替代**：TanStack Router（类型安全但生态小）、React Router 文件约定模式
**理由**：最成熟的 SPA 路由方案，配置式对这个规模更清晰可控。

### D5: 状态管理 → Zustand + TanStack Query

**选择**：Zustand（客户端状态）+ TanStack Query（服务端状态）
**替代**：Redux Toolkit（过重）、Jotai（原子化，此项目不需要）、SWR（功能少）
**理由**：2026 标准组合。Zustand 极简处理 UI 状态（sidebar 折叠等），TanStack Query 处理 API 缓存/失效/轮询。职责分离清晰。

### D6: 交互模式 → Sheet 抽屉

**选择**：所有表单操作使用 shadcn Sheet（从右侧滑入）
**替代**：Dialog/Modal、内联展开、新页面
**理由**：用户明确要求抽屉风格。Sheet 内容区域更大，表单不被截断，操作不遮挡列表上下文。AlertDialog 仅用于删除确认。

### D7: 字体 → Plus Jakarta Sans

**选择**：Plus Jakarta Sans + PingFang SC + system-ui
**替代**：Inter（过于普遍）、Geist（Vercel 系列）
**理由**：bklite-cloud 验证过的方案，Latin + CJK 覆盖好，几何感现代字体。

## Risks / Trade-offs

- **[Tailwind v4 + Vite 8 兼容性]** → @tailwindcss/vite 插件已稳定，社区广泛使用
- **[shadcn/ui 组件需要逐个添加]** → 按需添加，避免不使用的组件占体积
- **[oklch 在旧浏览器不支持]** → 2026 年所有主流浏览器均支持，无需 fallback
- **[ActiveBar 在少量菜单时显得空]** → 初期只有 2 个 App，但随业务增长会更有价值
