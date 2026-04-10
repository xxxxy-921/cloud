## ADDED Requirements

### Requirement: 授权主体列表页
系统 SHALL 在 `/license/licensees` 路由提供授权主体列表页面，使用 `useListPage` hook 实现分页数据加载。

#### Scenario: 列表展示
- **WHEN** 用户访问 `/license/licensees`
- **THEN** 页面 SHALL 展示数据表格，列包含：主体名称、主体代码（等宽字体）、联系人、状态（Badge）、创建时间、操作按钮

#### Scenario: 搜索过滤
- **WHEN** 用户在搜索框输入关键词
- **THEN** 列表 SHALL 按 name 和 code 模糊匹配过滤结果

#### Scenario: 状态筛选
- **WHEN** 用户选择状态筛选（全部/活跃/已归档）
- **THEN** 列表 SHALL 只展示对应状态的记录，默认显示"活跃"

#### Scenario: 空状态
- **WHEN** 无任何授权主体记录
- **THEN** 页面 SHALL 展示空状态提示，引导用户创建

### Requirement: 授权主体 Drawer 表单
系统 SHALL 使用 Sheet（Drawer）组件实现授权主体的新建和编辑表单。

#### Scenario: 新建表单
- **WHEN** 用户点击"新增授权主体"按钮
- **THEN** SHALL 打开右侧 Drawer，展示表单：
  - 基本信息区域：名称（必填）、备注（选填）
  - 联系信息区域：联系人、电话、邮箱
  - 企业信息区域（可折叠）：地址、税号、开户行、银行账号、SWIFT、IBAN

#### Scenario: 编辑表单
- **WHEN** 用户点击列表行的编辑按钮
- **THEN** SHALL 打开 Drawer 并预填充当前数据，name 字段可编辑

#### Scenario: 表单验证
- **WHEN** 用户提交表单但名称为空
- **THEN** SHALL 显示验证错误提示，不发送请求

#### Scenario: 保存成功
- **WHEN** 表单提交成功
- **THEN** Drawer SHALL 关闭，列表 SHALL 刷新，显示 toast 成功提示

### Requirement: 授权主体状态操作
系统 SHALL 在列表行操作中提供归档和恢复功能。

#### Scenario: 归档操作
- **WHEN** 用户对活跃状态的主体点击"归档"
- **THEN** SHALL 弹出确认对话框，确认后调用 PATCH API 将状态改为 archived，列表刷新

#### Scenario: 恢复操作
- **WHEN** 用户对已归档的主体点击"恢复"
- **THEN** SHALL 调用 PATCH API 将状态改为 active，列表刷新

### Requirement: 授权主体前端路由注册
系统 SHALL 在 license 模块的 `module.ts` 中注册 `/license/licensees` 路由，使用 lazy-loaded 导入。

#### Scenario: 路由可访问
- **WHEN** 用户通过侧边栏点击"授权主体"菜单
- **THEN** SHALL 导航到 `/license/licensees` 并加载列表页组件

### Requirement: 授权主体代码展示
列表中的 code 字段 SHALL 使用等宽字体展示，并在 Drawer 编辑模式中以只读方式展示（带复制按钮）。

#### Scenario: 列表中 code 展示
- **WHEN** 列表渲染 code 列
- **THEN** SHALL 使用 `font-mono` 样式展示 `LS-xxxx` 格式代码

#### Scenario: Drawer 中 code 展示
- **WHEN** 编辑模式打开 Drawer
- **THEN** SHALL 在表单顶部以只读方式展示 code，提供复制到剪贴板功能

### Requirement: 授权主体权限控制
前端 SHALL 根据用户权限控制操作按钮的显示。

#### Scenario: 无创建权限
- **WHEN** 用户不具有 `license:licensee:create` 权限
- **THEN** "新增授权主体"按钮 SHALL 不显示

#### Scenario: 无编辑权限
- **WHEN** 用户不具有 `license:licensee:update` 权限
- **THEN** 编辑按钮 SHALL 不显示

#### Scenario: 无归档权限
- **WHEN** 用户不具有 `license:licensee:archive` 权限
- **THEN** 归档/恢复按钮 SHALL 不显示
