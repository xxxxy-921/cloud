## Why

Metis 前端目前是 Vite 8 + React 19 的空白脚手架（默认 counter demo），没有 UI 框架、路由、状态管理或布局系统。需要搭建完整的前端架构骨架，包括 shadcn/ui 组件库、ActiveBar 二级导航布局、主题系统和路由框架，为后续业务页面（用户认证、系统配置管理等）提供基础。

## What Changes

- 引入 Tailwind CSS v4 + shadcn/ui 作为 UI 基础
- 引入 React Router 7 实现 SPA 路由
- 引入 Zustand 管理客户端状态
- 引入 TanStack Query 管理服务端状态（API 缓存、失效）
- 实现 ActiveBar 二级导航布局：Icon Rail (48px) + Nav Panel (160px) + TopNav (56px)，参考 NekoAdmin 模式
- 建立 oklch 色彩空间主题系统，融合 bklite-cloud（字体、圆角）和 NekoAdmin（毛玻璃、交互）的设计语言
- 抽屉（Sheet）作为所有表单操作的统一交互模式，不使用弹窗
- 搭建页面骨架：首页（占位）、系统配置页（对接已有 /api/v1/config API）
- 建立前端目录约定：pages/、components/layout/、stores/、hooks/、lib/

## Capabilities

### New Capabilities
- `theme-system`：oklch CSS 变量主题、毛玻璃效果、Plus Jakarta Sans 字体、统一圆角/阴影/动效规范
- `activebar-layout`：TopNav + Icon Rail + Nav Panel 二级导航布局，导航配置数据驱动，Sidebar 折叠态
- `frontend-routing`：React Router 7 路由配置、布局嵌套、面包屑、404 页面
- `frontend-state`：Zustand 客户端状态 + TanStack Query 服务端状态、fetch 封装、统一错误处理
- `config-page`：系统配置管理页面，对接 /api/v1/config CRUD API，使用 Sheet 抽屉编辑

### Modified Capabilities
- `web-embed`：Vite 构建产物路径不变，但 vite.config.ts 新增 Tailwind CSS 插件配置

## Impact

- **新增前端依赖**：tailwindcss, @tailwindcss/vite, react-router, zustand, @tanstack/react-query, react-hook-form, zod, lucide-react, clsx, tailwind-merge
- **shadcn/ui**：通过 CLI 安装组件源码到 `web/src/components/ui/`
- **目录变更**：新增 `web/src/{pages,components/layout,stores,hooks,lib,styles}`
- **现有文件修改**：`web/vite.config.ts`（加 Tailwind 插件）、`web/src/main.tsx`（加 Router Provider）、`web/src/App.tsx`（重写为路由配置）
- **已有 API 对接**：直接使用 `/api/v1/config` 接口，无后端改动
