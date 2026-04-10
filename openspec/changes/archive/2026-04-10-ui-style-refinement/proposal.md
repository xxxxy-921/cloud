## Why

登录页已升级为精致克制的 2026 设计风格，但系统内页（列表、表单、卡片、Badge 等）仍停留在 shadcn/ui 默认模板风格：主色过饱和、阴影偏重、输入框和分页等组件与登录页割裂。需要以登录页为基准，统一全系统的视觉品味——降饱和度、减阴影、轻边框——达到 Apple/Notion 级别的克制感。

## What Changes

- **主色降饱和**：`--color-primary` 从 `oklch(0.488 0.185 264)` 降至 ~`oklch(0.50 0.13 264)`，所有使用 primary 的按钮、Badge、focus ring 同步变得沉稳
- **边框减重**：全局 `--color-border` 透明度提升，减少视觉噪音
- **阴影清理**：Card 组件移除 `shadow-sm`，仅靠 border 界定边界
- **Input 统一**：内页 Input 对齐登录页质感（高度、圆角、focus ring 宽度）
- **Badge 柔化**：默认 Badge 变体改用更淡的色调，不再用纯 primary 底色
- **分页简化**：DataTablePagination 去掉 `border-dashed`，简化为干净的文字+按钮
- **Sidebar 高亮柔化**：`--color-sidebar-accent` 降饱和度，高亮更含蓄

## Capabilities

### New Capabilities

_无新增功能，本次为纯视觉升级。_

### Modified Capabilities

- `theme-system`: 调整 oklch color tokens（primary、border、sidebar-accent 饱和度与明度），影响全局视觉表现
- `shared-ui-patterns`: DataTablePagination 样式简化，去除虚线边框

## Impact

- **CSS tokens**: `web/src/styles/globals.css` — 4-5 个 CSS 变量值调整
- **UI 组件**: `web/src/components/ui/` — Input、Card、Badge、DataTable、Table 共 4-5 个文件微调
- **页面代码**: 无需改动（组件层改完，页面自动跟随）
- **布局组件**: 无需改动（Sidebar、TopNav 跟随 token 自动生效）
- **Breaking changes**: 无
