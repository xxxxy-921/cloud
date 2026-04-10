## 1. Design Tokens 调整

- [x] 1.1 修改 `globals.css` 中 `--color-primary` 为 `oklch(0.50 0.13 264)`
- [x] 1.2 修改 `--color-ring` 跟随新 primary 值
- [x] 1.3 修改 `--color-border` 和 `--color-input` 为 `oklch(0.91 0.005 250)`
- [x] 1.4 修改 `--color-sidebar-accent` 为 `oklch(0.95 0.02 264)`

## 2. 基础组件微调

- [x] 2.1 Card 组件：移除 `shadow-sm`，仅保留 border
- [x] 2.2 Input 组件：高度改为 `h-[2.375rem]`，圆角改为 `rounded-lg`，focus ring 改为 `ring-[2px]`
- [x] 2.3 Badge 组件：default variant 改为 `bg-primary/10 text-primary`（淡底深字）
- [x] 2.4 Table 组件：表头背景从 `bg-muted/20` 微调为 `bg-muted/15`，行 border 从 `border-border/60` 调为 `border-border/40`

## 3. DataTable 组件

- [x] 3.1 DataTablePagination：去掉 `border border-dashed bg-muted/10`，改为 `pt-4` 无边框样式
- [x] 3.2 DataTableCard：移除 `shadow-sm`，与 Card 统一

## 4. 视觉验证

- [x] 4.1 浏览器检查登录页样式未被 token 改动破坏
- [x] 4.2 浏览器检查用户列表页：按钮、Badge、表格、分页的视觉效果
- [x] 4.3 浏览器检查 Sidebar 高亮色是否更含蓄
- [x] 4.4 运行 `cd web && bun run lint` 确认无 lint 错误
