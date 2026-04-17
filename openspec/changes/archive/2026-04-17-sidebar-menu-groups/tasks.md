## 1. App Registry 扩展

- [x] 1.1 在 `web/src/apps/registry.ts` 中新增 `MenuGroup` 接口（`label: string`, `items: string[]`）和 `AppModule.menuGroups?: MenuGroup[]` 字段
- [x] 1.2 在 `web/src/apps/registry.ts` 中新增 `getMenuGroups(appName: string): MenuGroup[] | undefined` 查询函数

## 2. Sidebar 分组渲染

- [x] 2.1 在 `web/src/components/layout/sidebar.tsx` 中，Tier 2 区域根据 `getMenuGroups(activeApp.permission)` 判断是否启用分组模式
- [x] 2.2 实现分组渲染逻辑：按 `menuGroups` 顺序遍历，用 `item.permission` 匹配分桶，未匹配项追加到末尾
- [x] 2.3 渲染分组标签：`text-[11px] font-semibold tracking-wider text-muted-foreground/60 px-3 uppercase`，首组无额外 top spacing，后续组 `pt-3`
- [x] 2.4 分组标签使用 i18n key `menu.group.{appName}.{label}`，fallback 到 raw label

## 3. ITSM 分组声明

- [x] 3.1 在 `web/src/apps/itsm/module.ts` 的 `registerApp()` 中添加 `menuGroups` 声明（service / ticket / config 三组）
- [x] 3.2 在 `web/src/apps/itsm/locales/zh-CN.json` 添加分组标签翻译（服务、工单、配置）
- [x] 3.3 在 `web/src/apps/itsm/locales/en.json` 添加分组标签翻译（Service、Ticket、Config）

## 4. 验证

- [x] 4.1 验证 ITSM sidebar 显示三个分组，各组内菜单项顺序正确
- [x] 4.2 验证无 menuGroups 的 App（如系统设置）sidebar 渲染不变
- [x] 4.3 验证 `bun run lint` 通过
