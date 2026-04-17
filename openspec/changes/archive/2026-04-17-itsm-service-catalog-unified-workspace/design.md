## Context

当前 ITSM 服务目录管理由两个独立页面承担：

- `/itsm/catalogs` — Split-pane 布局，左侧 root 目录导航列表 + 右侧子目录表格。目录 CRUD 通过 Sheet 完成。
- `/itsm/services` — 标准表格 + 工具栏（关键词搜索 + 目录下拉筛选 + 分页），操作列提供查看/编辑/删除。

两页共用 `fetchCatalogTree()` API 获取目录树。服务列表通过 `useListPage` 做服务端分页（pageSize=20）。

目录树结构为严格两层：root（parentId=null）→ children。服务只挂载在 child（叶子）节点上。

## Goals / Non-Goals

**Goals:**

- 将目录导航与服务浏览合并到一个页面，消除割裂感
- 服务展示从表格升级为卡片网格，提升视觉辨识度
- 目录管理内联到左侧导航面板，无需独立页面
- 服务创建改为 Sheet，与项目表单容器约定一致

**Non-Goals:**

- 不改动后端 API（目录树、服务列表、服务 CRUD 接口保持不变）
- 不改动服务详情页 (`/itsm/services/:id`)
- 不改动 Forms / Priorities / SLA 等辅助实体页面
- 不增加关键词搜索（服务数量少，目录导航已足够）

## Decisions

### D1: 左侧导航使用 Group Section 而非可折叠树

**选择**: Root 目录作为分组标题（`text-xs uppercase tracking-wide text-muted-foreground`），Child 目录作为可选中导航项。

**替代方案**: 可折叠树（root 可展开/收起）。

**理由**: 目录树严格两层，root 数量通常 3-5 个、每个 root 下 2-4 个 child，全部展开无滚动压力。Group Section 零状态管理（无 open/closed），所有目录一目了然。Root 是纯分组标签不可选中（服务只挂叶子），交互模型更简单。

### D2: "全部"视图按 Root 分组展示

**选择**: 选中"全部"时，按 root 分组渲染卡片网格，每组一个标题 + 该 root 下所有 children 的服务合并展示。选中具体 child 时平铺展示。

**替代方案**: 全部铺平不分组。

**理由**: 分组保留信息层次，一眼看出哪个大类有多少服务。客户端已有 catalog tree 数据，child → root 反向映射零成本。

### D3: 卡片品牌色按引擎类型映射

**选择**: Smart Engine → violet 色系（AI 感），Classic Engine → sky 色系（流程感）。顶部 3px 色条 + 首字母 Avatar 背景色。

**理由**: 引擎类型是服务最重要的区分维度，决定了后续配置流程完全不同。颜色作为前注意加工，识别速度远快于阅读 Badge 文字。

```
Smart:   stripe bg-violet-500, avatar bg-violet-50 text-violet-700
Classic: stripe bg-sky-500,    avatar bg-sky-50 text-sky-700
```

### D4: 全量加载不分页

**选择**: 服务列表请求 `pageSize=100`，一次性加载所有服务。

**替代方案**: 保留分页。

**理由**: DESIGN.md 明确要求卡片网格全量加载（`pageSize=100，不分页`）。服务数量通常 < 30，分页在卡片网格下翻页后空间位置重排，用户失去空间记忆。

### D5: 服务创建改为 Sheet

**选择**: 废除 `/itsm/services/create` 独立页面，创建服务改用右侧 Sheet。

**理由**: 项目约定「新建/编辑表单统一使用 Sheet」。创建字段少（name, code, catalogId, engineType, description），Sheet 完全够用。创建成功后导航到详情页继续配置。

### D6: URL 保存选中目录状态

**选择**: 选中目录时更新 URL query `?catalog=<childId>`，支持分享和浏览器后退。

**理由**: 用户从详情页返回时应恢复之前的目录筛选状态。URL state 是最简单可靠的方案，不需要额外的全局状态。

## Risks / Trade-offs

- **菜单 Seed 变更**: 需移除"服务目录"菜单项，菜单 seed 是 idempotent 的但需要处理已存在的旧菜单记录 → 在 seed sync 中检测并删除旧菜单
- **旧路由兼容**: 用户可能 bookmark 了 `/itsm/catalogs` → 前端路由配置中保留 redirect 到 `/itsm/services`
- **卡片数量 > 20 的降级**: 如果某个目录下服务特别多，卡片网格可能需要滚动较长 → 服务通常不会超过 30，且目录筛选后数量更少，可接受
