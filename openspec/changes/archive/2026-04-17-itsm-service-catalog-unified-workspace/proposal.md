## Why

ITSM 服务目录管理分散在两个独立页面：`/itsm/catalogs`（目录树管理）和 `/itsm/services`（服务列表表格）。用户需要在一个页面管理目录层级，再切换到另一个页面用下拉框重新找回刚才的目录来查看服务——心智模型割裂。同时服务列表使用纯表格展示，缺乏视觉辨识度，而服务数量通常 < 30，更适合卡片网格。

## What Changes

- 将 `/itsm/catalogs` 和 `/itsm/services` 合并为一个一体化工作区页面（路由 `/itsm/services`）
- 左侧：Group Section 导航面板，root 目录作为分组标题（不可选中），child 目录作为可选中导航项，支持目录 CRUD 操作内联在导航中
- 右侧：服务卡片网格（`auto-fill, minmax(340px, 1fr)`），全量加载不分页
- 卡片按引擎类型（Smart=violet / Classic=sky）映射品牌色条 + 首字母 Avatar
- "全部"视图按 root 分组展示；选中具体 child 时平铺展示
- 废除独立的 `/itsm/catalogs` 页面，目录管理完全内联
- 废除 `/itsm/services/create` 独立页面，创建服务改用 Sheet（与项目约定一致）

## Capabilities

### New Capabilities

- `itsm-unified-catalog-workspace`: 一体化服务目录工作区——Group Section 导航 + 服务卡片网格 + 内联目录管理

### Modified Capabilities

- `itsm-service-definition-ui`: 服务列表从表格改为卡片网格，创建流程从独立页面改为 Sheet，移除独立列表页

## Impact

- **前端路由**: `/itsm/catalogs` 废除，`/itsm/services` 重写为一体化工作区，`/itsm/services/create` 废除
- **菜单 Seed**: 移除"服务目录"菜单项，"服务定义"菜单项路由不变但指向新工作区
- **API**: 无后端 API 变更，仅前端数据获取策略调整（服务列表改为 pageSize=100 全量加载）
- **前端文件**: `pages/catalogs/index.tsx` 删除，`pages/services/index.tsx` 重写，`pages/services/create/index.tsx` 删除
