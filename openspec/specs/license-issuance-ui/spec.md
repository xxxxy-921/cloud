# Capability: license-issuance-ui

## Purpose
许可签发前端能力 — 提供许可列表页、签发表单（Sheet）、详情页、导出 .lic 文件、吊销操作等 UI 功能。

## Requirements

### Requirement: License list page
系统 SHALL 在「许可管理」目录下提供「许可签发」菜单页面，路径为 `/license/licenses`。

页面 MUST 包含：
- 搜索框：按 planName 和 registrationCode 搜索
- 筛选器：按商品（productId）、授权主体（licenseeId）、状态（issued/revoked）筛选
- 数据表：显示 planName、商品名称、授权主体名称、状态、生效时间、过期时间、签发时间
- 状态标签：issued 显示为"已签发"（绿色），revoked 显示为"已吊销"（红色）
- 签发按钮：权限 `license:license:issue`
- 行操作：查看详情、吊销（仅 issued 状态）、导出 .lic（仅 issued 状态）
- 标准分页

#### Scenario: Empty state
- **WHEN** 没有任何许可记录
- **THEN** 页面 MUST 显示空状态提示

#### Scenario: Filter by status
- **WHEN** 用户选择状态筛选为 "已签发"
- **THEN** 列表 MUST 仅显示 status=issued 的记录

### Requirement: Issue license form
系统 SHALL 提供许可签发表单，使用右侧 Sheet（抽屉）展示。

表单字段：
1. **商品选择** (required) — 下拉选择已发布（published）商品
2. **授权主体选择** (required) — 下拉选择活跃（active）授权主体
3. **套餐选择** (optional) — 选择商品下的套餐，或选"自定义"手动配置约束
4. **约束值配置** — 选择套餐后自动填充，自定义时手动配置（模块开关 + 功能值）
5. **注册码** (required) — 文本输入
6. **生效日期** (required) — 日期选择器
7. **过期日期** (optional) — 日期选择器，留空表示永久有效
8. **备注** (optional) — 文本域

#### Scenario: Select product loads plans
- **WHEN** 用户选择一个商品
- **THEN** 表单 MUST 加载该商品下的套餐列表和约束定义

#### Scenario: Select plan fills constraints
- **WHEN** 用户选择一个套餐
- **THEN** 约束值配置 MUST 自动填充该套餐的 constraintValues

#### Scenario: Custom constraints
- **WHEN** 用户选择"自定义"套餐
- **THEN** 表单 MUST 显示约束值编辑器，根据商品 constraintSchema 渲染模块和功能配置

#### Scenario: Successful submission
- **WHEN** 用户填写完整表单并提交
- **THEN** 系统 MUST 调用签发 API，成功后关闭 Sheet 并刷新列表

### Requirement: License detail page
系统 SHALL 提供许可详情页面，路径为 `/license/licenses/:id`。

页面 MUST 包含以下信息区块：
- **基本信息**: 状态、商品名称/代码、授权主体名称/代码、套餐名称、注册码
- **有效期**: 生效时间、过期时间（永久有效显示为"永久"）
- **约束值**: 以结构化方式展示 constraintValues（模块和功能值）
- **签发信息**: 签发人、签发时间、密钥版本
- **吊销信息** (仅 revoked): 吊销人、吊销时间
- **操作按钮**: 导出 .lic（仅 issued）、吊销（仅 issued，需确认弹窗）

#### Scenario: View issued license
- **WHEN** 查看状态为 issued 的许可详情
- **THEN** 页面 MUST 显示"导出"和"吊销"操作按钮

#### Scenario: View revoked license
- **WHEN** 查看状态为 revoked 的许可详情
- **THEN** 页面 MUST 显示吊销信息，不显示"导出"和"吊销"按钮

### Requirement: Export .lic file from UI
系统 SHALL 支持从列表行操作和详情页导出 .lic 文件。

#### Scenario: Download .lic file
- **WHEN** 用户点击导出按钮
- **THEN** 浏览器 MUST 下载文件，文件名格式为 `<productCode>_<YYYYMMDD>.lic`

### Requirement: Revoke license from UI
系统 SHALL 支持从列表行操作和详情页吊销许可。

#### Scenario: Revoke confirmation
- **WHEN** 用户点击吊销按钮
- **THEN** 系统 MUST 显示确认弹窗（Dialog），确认后调用吊销 API

#### Scenario: Revoke success
- **WHEN** 吊销成功
- **THEN** 列表 MUST 刷新，详情页 MUST 更新状态显示

### Requirement: License seed data
系统 SHALL 在 seed 阶段创建许可签发相关的菜单和权限：
- 菜单：「许可签发」挂载在「许可管理」目录下，permission=`license:license:list`，path=`/license/licenses`
- 按钮权限：签发 (`license:license:issue`)、吊销 (`license:license:revoke`)
- Casbin 策略：admin 角色对 `/api/v1/license/licenses` 系列 endpoint 的访问权限

#### Scenario: Idempotent seed
- **WHEN** seed 多次执行
- **THEN** 不 MUST 产生重复的菜单或策略记录

### Requirement: Frontend route registration
系统 SHALL 在 license app 的 `module.ts` 中注册许可签发相关路由：
- `/license/licenses` — 许可列表页
- `/license/licenses/:id` — 许可详情页

路由 MUST 使用 lazy-load 方式加载页面组件。

#### Scenario: Route accessible
- **WHEN** 用户导航到 `/license/licenses`
- **THEN** 系统 MUST 渲染许可列表页
