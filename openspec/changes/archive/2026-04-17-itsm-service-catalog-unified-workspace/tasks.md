## 1. i18n Keys

- [x] 1.1 Add new locale keys to `web/src/apps/itsm/locales/zh-CN.json` and `en.json`: workspace title, "全部" label, group section labels, card status text, guide card text, create sheet labels, empty state text
- [x] 1.2 Remove obsolete locale keys for the old catalogs page title and services table-specific labels (search placeholder, allCatalogs, column headers)

## 2. Service Card Component

- [x] 2.1 Create `web/src/apps/itsm/components/service-card.tsx`: engine type brand color mapping (Smart=violet, Classic=sky), 3px top stripe, 2-char Avatar, service name + `⋯` DropdownMenu, engine type Badge, bottom divider with status dot + relative time
- [x] 2.2 Implement card hover effect (`border-primary/20 shadow-md -translate-y-0.5`), click navigation to `/itsm/services/:id`, and `[data-action-zone]` on `⋯` menu to prevent click-through
- [x] 2.3 Create guide card component: dashed border card with `+` icon and "添加服务" text, click opens create Sheet

## 3. Group Section Navigation Panel

- [x] 3.1 Create `web/src/apps/itsm/components/catalog-nav-panel.tsx`: "全部" top item with total count Badge, root categories as section headers (`text-xs font-medium uppercase tracking-wide text-muted-foreground`), child items as selectable nav items with service count Badges
- [x] 3.2 Implement child item selection
- [x] 3.3 Implement root section header hover `⋯` DropdownMenu
- [x] 3.4 Implement bottom `[+ 新建目录]` dashed button

## 4. Unified Workspace Page

- [x] 4.1 Rewrite `web/src/apps/itsm/pages/services/index.tsx`
- [x] 4.2 Implement URL state sync
- [x] 4.3 Implement "全部" view
- [x] 4.4 Implement filtered view
- [x] 4.5 Implement empty state for no services in current catalog selection
- [x] 4.6 Integrate create service Sheet

## 5. Route & Menu Cleanup

- [x] 5.1 Remove `pages/catalogs/index.tsx` and `pages/services/create/index.tsx`
- [x] 5.2 Update `web/src/apps/itsm/module.ts`: remove `/itsm/catalogs` route, remove `/itsm/services/create` route, add redirect from `/itsm/catalogs` to `/itsm/services`
- [x] 5.3 Update ITSM seed (`internal/app/itsm/seed.go`): remove "服务目录" menu item from seed sync, keep "服务定义" menu unchanged
