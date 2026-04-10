## Context

Metis 的侧边栏使用后端 seed 数据驱动的菜单树。当前有三个顶级菜单项：首页（sort=0）、系统管理（sort=100）、许可管理（sort=200）。首页仅显示欢迎文案，无功能价值。系统管理作为基础设施模块排在业务模块（许可管理）前面，不符合用户预期。

前端登录后硬编码 `navigate("/")`，`/` 对应首页 index route。侧边栏 `findActiveNavApp` 已有 fallback 到第一个 app 的逻辑。

## Goals / Non-Goals

**Goals:**
- 移除首页，减少一次无意义的点击
- 系统管理菜单永远排在最后，业务 App 优先展示
- 登录/访问根路径时自动落地到第一个可用菜单

**Non-Goals:**
- 不做数据库迁移（用户会删 DB 重建）
- 不改变菜单树 API 结构或权限模型
- 不新增任何页面或组件

## Decisions

### D1: 系统管理 sort 值用 9999

**选择**: sort=9999
**备选**: sort=-1（放最后用负数标记）、动态排序（代码层面把 system 排到最后）
**理由**: 9999 简单直观，所有正常 App 用 100~9000 区间即可排在系统管理前面。无需额外排序逻辑。

### D2: 首页菜单和 Casbin 策略直接从 seed 中删除

**选择**: 删除 seed 中的 home 菜单项和对应的 user 角色 home 策略
**理由**: 首页不再存在，保留 seed 数据无意义。用户确认会删 DB 重来。

### D3: 根路径 `/` 使用 React Router Navigate 组件重定向

**选择**: 在 App.tsx 的 index route 放一个 `<DefaultRedirect />` 组件，从 menu store 读取第一个可用菜单路径并 `<Navigate to={path} replace />`
**备选**: 在 login 页面计算目标路径（仅解决登录场景，不解决直接访问 `/` 的场景）
**理由**: 统一处理所有到达 `/` 的场景（登录跳转、直接访问、刷新）。login 页面的 `navigate("/")` 不需要改——它会到达 `/`，然后被重定向。

### D4: DefaultRedirect fallback 到第一个菜单路径

**选择**: 从 menu store 的菜单树中取第一个 directory 的第一个 menu 子项的 path
**fallback**: 如果菜单树为空（理论上不会），fallback 到 `/users`
**理由**: 对齐侧边栏 `findActiveNavApp` 已有的 fallback 逻辑，保证 URL 和侧边栏高亮一致。

## Risks / Trade-offs

- **[菜单树为空时 fallback]** → 使用 `/users` 作为硬编码 fallback，因为用户管理是内核必有的页面
- **[首页组件残留]** → 整个 `pages/home/` 目录删除，无残留风险
- **[面包屑 "首页" 引用]** → 需检查面包屑是否有硬编码 "首页"，如有则移除
