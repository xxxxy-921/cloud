## Context

Metis 管理后台前端已完成核心 CRUD 功能，但 UI 一致性和代码复用存在多处问题。全部改动集中在 `web/src/` 下，无后端变更。项目使用 Vite + React 19 + shadcn/ui + TanStack Query + Zustand。

当前状态：
- TopNav 用户菜单：手工 `useState` + `mousedown` 实现，60 行代码，无 a11y
- 4 个 Sheet 组件缺少 `SheetDescription`，产生控制台警告
- 登录页在渲染阶段直接调用 `navigate()`
- 表格操作列横排 3 个按钮，窄屏有溢出风险
- `menu-sheet` 和 `user-sheet` 使用原生 `<select>`
- `SiteInfo` 接口在 3 处重复定义
- users 和 roles 页面分页逻辑几乎完全相同

## Goals / Non-Goals

**Goals:**
- 消除所有控制台警告（a11y、React 反模式）
- 统一组件风格：全部使用 shadcn/ui 原语，消除手工实现
- 提取重复代码为共享 hook/类型
- 表格操作列防溢出，响应式友好
- 空状态视觉改进

**Non-Goals:**
- 移动端专项适配（不做汉堡菜单、侧栏抽屉等）
- 首页 Dashboard 改造
- 新增功能或 API 变更
- 暗色模式

## Decisions

### D1: TopNav dropdown → shadcn DropdownMenu

**选择**：直接替换为 `<DropdownMenu>` + `<DropdownMenuContent>` + `<DropdownMenuItem>`。

**原因**：
- 项目已引入 shadcn/ui 体系，DropdownMenu 基于 Radix UI，内置 a11y、键盘导航、动画、自动定位
- 替换后代码从 ~60 行减少到 ~20 行
- 自动跟随主题 token，不再需要硬编码 `bg-white`

**替代方案**：Popover + 手动 menu role — 太重，DropdownMenu 直接解决。

### D2: 表格操作列 → DropdownMenu 三点菜单

**选择**：每行操作统一收入 `<DropdownMenu>` 触发按钮为 `MoreHorizontal` 图标。

**结构**：
```
[MoreHorizontal] →
  ├─ 编辑
  ├─ 停用/启用 (仅用户页)
  ├─ 权限 (仅角色页)
  ├─ ── separator ──
  └─ 删除 (红色，系统角色 disabled)
```

**原因**：
- 3-4 个按钮横排在窄列中溢出风险高
- DropdownMenu 统一收纳，响应式安全
- 删除操作用 separator 隔开 + 红色文字，降低误操作

**删除确认**：保留 AlertDialog，从 DropdownMenu 内触发。需注意 Radix 的 portal 嵌套——使用 `<AlertDialog>` 包裹在 DropdownMenuItem 外层，通过 state 控制打开。

### D3: 原生 select → shadcn Select

**选择**：用 `<Select><SelectTrigger><SelectContent><SelectItem>` 替换原生 `<select>` + `nativeSelectClass`。

**涉及文件**：
- `menu-sheet.tsx`：parentId 和 type 两个 select
- `user-sheet.tsx`：roleId 一个 select

**注意**：shadcn Select 的 value 必须是 string，需要做 `String(id)` / `Number(value)` 转换。

### D4: useListPage hook 提取

**选择**：提取到 `web/src/hooks/use-list-page.ts`。

**接口**：
```ts
function useListPage<T>(options: {
  queryKey: string
  endpoint: string
  pageSize?: number
}) => {
  keyword, setKeyword, searchKeyword,
  page, setPage, pageSize, totalPages, total,
  items, isLoading,
  handleSearch,
}
```

**原因**：users 和 roles 的 keyword/searchKeyword/page/handleSearch/分页 UI 逻辑几乎完全重复（~30 行重复代码）。

### D5: SiteInfo 类型提取

**选择**：定义在 `web/src/lib/api.ts` 中（已有 `PaginatedResponse` 等共享类型），从 `top-nav.tsx` 和 `settings/index.tsx` 中 import。

### D6: 空状态组件

**选择**：不创建独立通用组件，直接在各表格的空状态 `<TableCell>` 中加图标 + 引导文字。保持简单，不过度抽象。

## Risks / Trade-offs

- **DropdownMenu 内触发 AlertDialog**：Radix 的 portal 嵌套需要特殊处理。风险低，常见模式是用 state 变量在 DropdownMenu 外部挂载 AlertDialog → 已有社区验证方案
- **shadcn Select 的 value 类型**：必须是 string，和 react-hook-form 的 number 字段需要转换层 → 在 `onValueChange` 中做 `Number()` 转换
- **useListPage hook 的泛型**：需要确保 TanStack Query 的泛型推断不丢失 → 使用 `useQuery<PaginatedResponse<T>>` 保持类型安全
- **`nativeSelectClass` 变为死代码**：替换完 select 后该工具函数可删除。如果其他地方用了需要检查 → 全局搜索确认只在被替换的 2 处使用
