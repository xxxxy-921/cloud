## Context

Sidebar 使用两层导航：Tier 1 是 App 图标轨（12px 宽），Tier 2 是当前 App 的菜单列表（160px 宽）。Tier 2 当前是扁平渲染，所有子菜单按 sort 顺序排列，无分组。

ITSM App 已有 11 个二级菜单项，涵盖服务管理、工单操作、系统配置三类功能。扁平列表缺乏视觉层次，用户定位困难。

相关文件：
- `web/src/apps/registry.ts` — AppModule 类型定义 + 注册函数
- `web/src/components/layout/sidebar.tsx` — Sidebar 组件，Tier 2 渲染在 L154-172
- `web/src/apps/itsm/module.ts` — ITSM App 注册
- `web/src/stores/menu.ts` — MenuItem 类型，menuTree store

## Goals / Non-Goals

**Goals:**
- Sidebar Tier 2 支持可选的菜单分组渲染
- 无分组声明的 App 保持现有扁平渲染，零影响
- ITSM 按逻辑分成三组：服务、工单、配置
- 分组标签支持 i18n

**Non-Goals:**
- 不修改后端 Menu 模型或数据库
- 不修改 Seed 数据
- 不做分组的后台管理界面
- 不处理 Sidebar 收起态的分组标签（当前收起态 Tier 2 宽度为 0，不可见）
- 不为其他 App（AI、系统等）添加分组——仅 ITSM

## Decisions

### D1: 分组信息由各 App 在 `registerApp()` 中声明

**选择**: 扩展 `AppModule` 接口，增加可选 `menuGroups` 字段

**替代方案**:
- (a) sidebar.tsx 集中维护分组映射 → 违反 App 插拔原则，ITSM 卸载后残留配置
- (b) 后端 Menu 模型增加 group 字段 → 改动面过大，分组是视觉层关注点

**理由**: 分组信息与 App 生命周期绑定，App 注册时声明、卸载时消失，符合插拔架构。

### D2: 用 `permission` 字段做菜单匹配键

**选择**: `menuGroups.items` 数组存 permission 值（如 `"itsm:catalog:list"`），sidebar 渲染时用 `item.permission` 匹配

**替代方案**: 用 `path` 匹配 → path 可能变更（如 URL 重构），permission 更稳定且全局唯一

**理由**: permission 是 Menu 的唯一索引，已在 sidebar 中用于 i18n key 查找，复用无额外成本。

### D3: registry 暴露 `getMenuGroups(appName)` 查询函数

**选择**: sidebar 通过 appName 查询分组配置，而非将 menuGroups 传入 NavApp 数据结构

**理由**: NavApp 是从 menuTree 构建的（后端数据），menuGroups 是前端配置，二者来源不同。sidebar 按需查询更清晰，也避免在 buildNavApps 中做不必要的数据合并。匹配方式：`activeApp.permission`（如 `"itsm"`）对应 `appModule.name`（如 `"itsm"`）。

### D4: 分组标签视觉样式

```
text-[11px] font-semibold tracking-wider text-muted-foreground/60 px-3 uppercase
```

- 首组 `pt-0`，后续组 `pt-3` 增加分隔
- 中文环境 uppercase 无效果，英文环境自动大写
- 标签不可点击，纯视觉分隔

### D5: 未匹配项处理

不在任何 group 中的菜单项追加到末尾，无分组标签。这是防御性设计——如果 seed 新增了菜单但 menuGroups 未更新，菜单仍然可见。

## Risks / Trade-offs

- **[分组配置与 seed 数据不同步]** → 未匹配项 fallback 到末尾无标签区域，功能不丢失。开发者新增 ITSM 菜单时需同步更新 module.ts 的 menuGroups。
- **[Tier 2 面板宽度 160px 对分组标签的限制]** → 标签应保持简短（2-4 字），当前三个组（服务/工单/配置）均满足。
