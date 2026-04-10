## 1. 添加 shadcn 组件

- [x] 1.1 运行 `bunx shadcn@latest add dropdown-menu select` 添加缺失的 shadcn 组件

## 2. Sheet a11y 修复

- [x] 2.1 为 `role-sheet.tsx` 添加 `SheetDescription`（可用 VisuallyHidden 包裹）
- [x] 2.2 为 `user-sheet.tsx` 添加 `SheetDescription`
- [x] 2.3 为 `config-sheet.tsx` 添加 `SheetDescription`
- [x] 2.4 为 `menu-sheet.tsx` 添加 `SheetDescription`
- [x] 2.5 为 `permission-dialog.tsx` 添加 `SheetDescription`（利用已有的 "已选 X/Y 项" 文本）

## 3. TopNav dropdown 替换

- [x] 3.1 重写 `top-nav.tsx` 的用户菜单：移除手工 `useState/useRef/useEffect` dropdown，替换为 shadcn `DropdownMenu` + `DropdownMenuContent` + `DropdownMenuItem`

## 4. Login 页面修复

- [x] 4.1 修复 `login/index.tsx`：将渲染期 `navigate()` 调用替换为 `<Navigate to="/" replace />`

## 5. 原生 select → shadcn Select

- [x] 5.1 替换 `menu-sheet.tsx` 的 parentId 和 type 两个原生 `<select>` 为 shadcn `Select`
- [x] 5.2 替换 `user-sheet.tsx` 的 roleId 原生 `<select>` 为 shadcn `Select`
- [x] 5.3 检查并删除 `lib/utils.ts` 中不再使用的 `nativeSelectClass`

## 6. 表格操作列改造

- [x] 6.1 重写 `users/index.tsx` 操作列：替换为 `DropdownMenu`（MoreHorizontal 触发器），包含编辑、停用/启用、separator、删除（AlertDialog 确认）
- [x] 6.2 重写 `roles/index.tsx` 操作列：替换为 `DropdownMenu`（MoreHorizontal 触发器），包含权限、编辑、separator、删除（系统角色 disabled）

## 7. 共享代码提取

- [x] 7.1 在 `lib/api.ts` 中定义共享 `SiteInfo` 接口，从 `top-nav.tsx` 和 `settings/index.tsx` 中移除重复定义并 import
- [x] 7.2 创建 `hooks/use-list-page.ts`，提取分页搜索逻辑为 `useListPage<T>` hook
- [x] 7.3 重构 `users/index.tsx` 使用 `useListPage` hook
- [x] 7.4 重构 `roles/index.tsx` 使用 `useListPage` hook

## 8. 空状态 & 杂项

- [x] 8.1 改进所有表格的空状态：添加图标 + 引导文案（用户、角色、配置页面）
- [x] 8.2 为 `role-sheet.tsx` 添加取消按钮
