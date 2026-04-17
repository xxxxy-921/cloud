## Why

ITSM 的二级菜单已有 11 项（服务目录、服务定义、表单管理、全部工单、我的工单、我的待办、历史工单、我的审批、优先级管理、SLA 管理、引擎配置），扁平列表缺乏层次感，用户需要逐一扫描才能定位目标。需要为 sidebar 二级菜单引入分组能力，将相关菜单项归类展示。

## What Changes

- 扩展前端 App 注册接口 `AppModule`，支持可选的 `menuGroups` 声明
- Sidebar Tier 2 渲染逻辑增加分组模式：有 `menuGroups` 时按组渲染（带组标签），无则保持现有扁平渲染
- ITSM App 声明三个菜单分组：服务（3项）、工单（5项）、配置（3项）
- 分组标签支持 i18n

## Capabilities

### New Capabilities
- `sidebar-menu-groups`: Sidebar 二级菜单分组渲染能力，包括 AppModule 类型扩展、分组匹配逻辑、分组标签渲染样式、i18n 支持

### Modified Capabilities

（无现有 spec 需要修改，分组是纯视觉层增强，不影响菜单数据模型、权限、路由）

## Impact

- `web/src/apps/registry.ts` — 类型扩展 + 查询函数
- `web/src/components/layout/sidebar.tsx` — Tier 2 渲染逻辑
- `web/src/apps/itsm/module.ts` — 声明 menuGroups
- `web/src/apps/itsm/locales/zh-CN.json` / `en.json` — 分组标签翻译
- 后端零改动，数据库零改动
