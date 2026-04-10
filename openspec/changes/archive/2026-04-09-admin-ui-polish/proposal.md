## Why

全面 review 后发现系统管理 UI 存在多处 UX 短板和技术债：手工实现的 dropdown 缺少 a11y、Sheet 组件缺少 description 导致控制台警告、原生 select 与 shadcn 风格割裂、登录页渲染期副作用、表格操作列溢出风险、类型定义重复、分页逻辑重复等。这些都是小改动但数量多，需要统一修复一轮。

## What Changes

- 替换 TopNav 手工 dropdown 为 shadcn `DropdownMenu`，获得 a11y + 键盘导航 + 主题适配
- 所有 Sheet 组件添加 `SheetDescription`（或 `aria-describedby`）消除控制台警告
- 修复 `login/index.tsx` 渲染期 `navigate()` 调用为 `<Navigate>` 组件
- 表格操作列（用户/角色）从多按钮横排改为 `DropdownMenu` 三点菜单，防溢出
- `menu-sheet.tsx` 和 `user-sheet.tsx` 的原生 `<select>` 替换为 shadcn `Select` 组件
- 提取 `SiteInfo` 类型到共享位置，消除 3 处重复定义
- 提取分页列表通用逻辑为 `useListPage` hook，消除 users/roles 页面重复代码
- 空状态改进：表格空状态加图标和引导文案
- 角色编辑 Sheet 添加取消按钮，与其他 Sheet 一致

## Capabilities

### New Capabilities

- `shared-ui-patterns`: 提取复用的 UI 模式（useListPage hook、SiteInfo 类型、空状态组件）

### Modified Capabilities

- `user-auth-frontend`: 修复登录页渲染期 navigate 反模式
- `nav-modules`: TopNav dropdown 替换为 shadcn DropdownMenu
- `user-management`: 操作列改为 DropdownMenu，原生 select 替换
- `activebar-layout`: Sheet 组件 a11y 修复

## Impact

- **前端文件**：约 12-15 个文件改动，均为 `web/src/` 下
- **依赖**：无新增依赖，shadcn `Select` 和 `DropdownMenu` 需要通过 CLI 添加（`npx shadcn@latest add select dropdown-menu`）
- **后端**：无改动
- **Breaking**：无破坏性变更
