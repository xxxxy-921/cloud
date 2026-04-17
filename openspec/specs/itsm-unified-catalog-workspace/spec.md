## ADDED Requirements

### Requirement: 一体化工作区页面布局

系统 SHALL 在路由 `/itsm/services` 提供一体化服务目录工作区页面，采用左右分栏布局：左侧为固定宽度（w-64）的目录导航面板，右侧为服务卡片网格区域。

#### Scenario: 页面加载
- **WHEN** 管理员进入 `/itsm/services` 页面
- **THEN** 系统 SHALL 并行加载目录树（`GET /api/v1/itsm/catalogs/tree`）和服务列表（`GET /api/v1/itsm/services?pageSize=100`），左侧展示目录导航，右侧展示服务卡片网格

#### Scenario: URL 恢复目录选中状态
- **WHEN** URL 包含 query 参数 `?catalog=<childId>`
- **THEN** 系统 SHALL 自动选中对应的 child 目录项，右侧仅展示该目录下的服务

#### Scenario: 无 catalog 参数时默认全部
- **WHEN** URL 不包含 catalog 参数
- **THEN** 系统 SHALL 默认选中"全部"，右侧按 root 分组展示所有服务

### Requirement: Group Section 目录导航面板

左侧导航面板 SHALL 采用 Group Section 模式：顶部为"全部"特殊导航项，下方按 root 目录分组，child 目录作为可选中导航项。Root 目录作为纯分组标题不可选中。

#### Scenario: 全部导航项
- **WHEN** 页面加载完成
- **THEN** 系统 SHALL 在导航面板顶部展示"全部"导航项，右侧显示服务总数 Badge，点击后右侧显示所有服务（按 root 分组）

#### Scenario: Root 分组标题
- **WHEN** 页面加载完成
- **THEN** 系统 SHALL 将每个 root 目录渲染为分组标题（`text-xs font-medium uppercase tracking-wide text-muted-foreground`），不可点击选中。Hover 时在标题右侧显示 `⋯` 菜单按钮

#### Scenario: Root 菜单操作
- **WHEN** 管理员点击 root 分组标题的 `⋯` 菜单
- **THEN** 系统 SHALL 展示 DropdownMenu，包含：编辑目录、添加子目录、删除目录（text-destructive，需确认）

#### Scenario: Child 导航项
- **WHEN** 页面加载完成
- **THEN** 系统 SHALL 在每个 root 分组下渲染其 children 为可选中导航项，缩进显示，右侧显示该目录下的服务数 Badge

#### Scenario: 选中 Child 导航项
- **WHEN** 管理员点击某个 child 导航项
- **THEN** 系统 SHALL 高亮该项（左边缘 2px primary 色条 + bg-accent），更新 URL query 为 `?catalog=<childId>`，右侧仅展示该目录下的服务卡片（平铺，不分组）

#### Scenario: 新建目录入口
- **WHEN** 页面加载完成
- **THEN** 系统 SHALL 在导航面板底部显示虚线按钮 `[+ 新建目录]`（`border-dashed border-muted-foreground/25`），点击后打开 Sheet 创建 root 目录

#### Scenario: 编辑目录
- **WHEN** 管理员通过 root `⋯` 菜单选择"编辑目录"，或 hover child 项时点击编辑图标
- **THEN** 系统 SHALL 打开 Sheet，预填充目录信息（name、code、icon、description），支持修改并保存

#### Scenario: 删除目录确认
- **WHEN** 管理员通过菜单选择"删除目录"
- **THEN** 系统 SHALL 弹出 AlertDialog 确认删除，确认后调用 `DELETE /api/v1/itsm/catalogs/:id`

### Requirement: 服务卡片网格

右侧区域 SHALL 以卡片网格（`grid auto-fill minmax(340px, 1fr) gap-4`）展示服务定义，全量加载不分页。

#### Scenario: 卡片结构
- **WHEN** 存在服务定义
- **THEN** 系统 SHALL 将每个服务渲染为卡片，结构为：顶部 3px 品牌色条（Smart=`bg-violet-500`，Classic=`bg-sky-500`）→ 头部区域（首字母 Avatar + 服务名称 + `⋯` 菜单）→ 中部 chips（引擎类型 Badge）→ 底部分割线后状态圆点 + 相对时间

#### Scenario: 卡片首字母 Avatar
- **WHEN** 渲染服务卡片
- **THEN** 系统 SHALL 取服务名称前两个字符作为 Avatar 文字，背景色按引擎类型映射（Smart=`bg-violet-50 text-violet-700`，Classic=`bg-sky-50 text-sky-700`）

#### Scenario: 卡片交互效果
- **WHEN** 管理员 hover 服务卡片
- **THEN** 系统 SHALL 应用 `border-primary/20 shadow-md -translate-y-0.5` 过渡效果

#### Scenario: 点击卡片导航到详情
- **WHEN** 管理员点击服务卡片（非操作区域）
- **THEN** 系统 SHALL 导航到 `/itsm/services/:id` 详情页

#### Scenario: 卡片 `⋯` 菜单
- **WHEN** 管理员点击卡片右上角的 `⋯` 按钮
- **THEN** 系统 SHALL 展示 DropdownMenu，包含：编辑（导航到详情页）、删除（text-destructive，需确认）。点击 `⋯` 不触发卡片导航

#### Scenario: "全部"视图按 root 分组
- **WHEN** 当前选中"全部"导航项
- **THEN** 系统 SHALL 按 root 目录分组展示，每组标题为 root name（`text-sm font-medium text-muted-foreground`），下方为该 root 所有 children 的服务合并的卡片网格

#### Scenario: 引导卡片
- **WHEN** 当前目录下存在服务且管理员有创建权限
- **THEN** 系统 SHALL 在卡片网格末尾展示虚线引导卡片（`border-2 border-dashed border-muted-foreground/20 bg-muted/20`），内含 `+` 图标和"添加服务"文字，点击后打开创建服务 Sheet

#### Scenario: 空状态
- **WHEN** 当前目录下没有任何服务
- **THEN** 系统 SHALL 展示居中空状态：图标（`h-12 w-12 text-muted-foreground/40`）+ 标题 + 描述 + "新建服务"按钮

### Requirement: 创建服务 Sheet

管理员 SHALL 通过 Sheet（右侧抽屉）创建新服务定义，Sheet 内包含基础必填字段。

#### Scenario: 打开创建 Sheet
- **WHEN** 管理员点击引导卡片或顶部"新建服务"按钮
- **THEN** 系统 SHALL 打开 Sheet，表单包含：服务名称（必填）、服务编码（必填）、所属分类（必填，下拉选择 child 目录，若当前已选中某目录则预填）、引擎类型（必填，默认 smart）、描述（可选）

#### Scenario: 创建成功
- **WHEN** 管理员提交表单且 API 返回成功
- **THEN** 系统 SHALL 关闭 Sheet，显示成功提示，刷新服务列表，并导航到新服务的详情页 `/itsm/services/:id`

#### Scenario: 编码冲突
- **WHEN** 提交的编码已存在
- **THEN** 系统 SHALL 显示错误提示"服务编码已存在"
