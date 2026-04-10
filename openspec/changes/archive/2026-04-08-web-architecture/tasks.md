## 1. 前端依赖安装

- [x] 1.1 安装 Tailwind CSS v4 + Vite 插件：`bun add -D tailwindcss @tailwindcss/vite`
- [x] 1.2 安装 shadcn/ui 依赖：`bun add clsx tailwind-merge lucide-react`，运行 `npx shadcn@latest init` 初始化
- [x] 1.3 安装路由和状态管理：`bun add react-router zustand @tanstack/react-query`
- [x] 1.4 安装表单工具：`bun add react-hook-form zod @hookform/resolvers`

## 2. Tailwind CSS + 主题配置

- [x] 2.1 修改 `vite.config.ts`：添加 `@tailwindcss/vite` 插件
- [x] 2.2 创建 `src/styles/globals.css`：Tailwind 导入 + oklch CSS 变量（--background, --foreground, --primary, --accent, --border, --sidebar 等）+ Plus Jakarta Sans 字体声明
- [x] 2.3 更新 `src/main.tsx`：导入 globals.css 替换原有样式
- [x] 2.4 创建 `src/lib/utils.ts`：cn() 工具函数（clsx + tailwind-merge）

## 3. shadcn/ui 组件安装

- [x] 3.1 通过 shadcn CLI 添加基础组件：button, sheet, table, form, input, label, alert-dialog, separator, tooltip, badge
- [x] 3.2 验证组件在 oklch 主题变量下正常渲染

## 4. 路由框架搭建

- [x] 4.1 创建 `src/pages/home/index.tsx`：首页占位组件
- [x] 4.2 创建 `src/pages/not-found.tsx`：404 页面
- [x] 4.3 重写 `src/App.tsx`：使用 createBrowserRouter 配置路由，DashboardLayout 包裹受保护路由
- [x] 4.4 更新 `src/main.tsx`：RouterProvider + QueryClientProvider 包裹 App

## 5. ActiveBar 布局组件

- [x] 5.1 创建 `src/lib/nav.ts`：AppDef 和 NavItemDef 类型定义 + 初始导航配置（home、system 两个 app）
- [x] 5.2 创建 `src/components/layout/top-nav.tsx`：TopNav 组件（Logo + 应用名 + 右侧操作区），毛玻璃背景
- [x] 5.3 创建 `src/components/layout/sidebar.tsx`：Sidebar 组件包含 Icon Rail（w-12）+ Nav Panel（w-40），pathname 驱动 active 状态
- [x] 5.4 创建 `src/components/layout/header.tsx`：Header 组件（面包屑导航），根据 pathname 生成层级
- [x] 5.5 创建 `src/components/layout/dashboard-layout.tsx`：组合 TopNav + Sidebar + Header + Outlet，flex 布局

## 6. 状态管理基础

- [x] 6.1 创建 `src/lib/api.ts`：fetch 封装，统一处理 R{code, message, data} 响应格式，错误抛出
- [x] 6.2 创建 `src/stores/ui.ts`：Zustand store 管理 sidebar 折叠状态等 UI 状态

## 7. 系统配置页

- [x] 7.1 创建 `src/pages/config/index.tsx`：配置列表页，使用 TanStack Query 获取 /api/v1/config，shadcn Table 展示
- [x] 7.2 创建 `src/pages/config/config-sheet.tsx`：Sheet 抽屉表单（新建/编辑），React Hook Form + Zod 校验，提交调用 PUT /api/v1/config
- [x] 7.3 在配置列表中实现删除操作：AlertDialog 确认 → DELETE /api/v1/config/:key → invalidate query

## 8. 清理和集成

- [x] 8.1 删除原有 App.css、index.css、assets/ 等默认 Vite 脚手架文件
- [x] 8.2 验证 `bun run build` 构建成功
- [x] 8.3 验证 `make build` 完整链路：前端构建 → Go embed → 单二进制运行 → 页面可访问
