## ADDED Requirements

### Requirement: 商品列表页
系统 SHALL 提供商品列表页面，展示所有商品的表格视图。表格列：名称、编码、状态（Badge）、套餐数量、创建时间、操作。支持按关键词搜索（名称/编码）和按状态筛选。列表使用 useListPage hook 实现分页和 React Query 缓存。

#### Scenario: 查看商品列表
- **WHEN** 用户导航到商品管理页面
- **THEN** 展示商品表格，默认按创建时间降序排列

#### Scenario: 搜索和筛选
- **WHEN** 用户输入关键词或选择状态筛选
- **THEN** 表格实时过滤，分页重置到第一页

#### Scenario: 创建商品入口
- **WHEN** 用户点击「创建商品」按钮
- **THEN** 打开 Sheet 抽屉，展示商品创建表单（名称、编码、描述）

### Requirement: 商品详情页
系统 SHALL 提供商品详情页面，使用 Tabs 组织信息：基本信息、套餐管理、约束定义、密钥管理。

#### Scenario: 基本信息 Tab
- **WHEN** 用户查看基本信息 Tab
- **THEN** 展示商品名称、编码、描述、状态，提供编辑按钮（打开 Sheet 编辑表单）和状态操作按钮（发布/下架/归档/恢复，根据当前状态显示可用操作）

#### Scenario: 套餐管理 Tab
- **WHEN** 用户查看套餐管理 Tab
- **THEN** 展示该商品下所有套餐列表（名称、是否默认、排序），提供创建/编辑/删除/设为默认操作

#### Scenario: 约束定义 Tab
- **WHEN** 用户查看约束定义 Tab
- **THEN** 展示 ConstraintSchema 可视化编辑器

#### Scenario: 密钥管理 Tab
- **WHEN** 用户查看密钥管理 Tab
- **THEN** 展示当前密钥版本和公钥，提供密钥轮转按钮（需二次确认）

### Requirement: ConstraintSchema 可视化编辑器
系统 SHALL 提供可视化编辑器，用于编辑商品的 ConstraintSchema。编辑器支持：添加/删除模块（Module）、编辑模块 key 和 label、添加/删除特性（Feature）、配置特性类型和属性。

#### Scenario: 添加模块
- **WHEN** 用户点击「添加模块」
- **THEN** 在列表末尾新增空模块项，包含 key 和 label 输入框

#### Scenario: 添加特性
- **WHEN** 用户在某模块下点击「添加特性」
- **THEN** 新增空特性行，包含 key、label 输入框和 type 下拉选择

#### Scenario: 配置 number 类型特性
- **WHEN** 用户选择特性类型为 number
- **THEN** 展示 min、max、default 数字输入框

#### Scenario: 配置 enum 类型特性
- **WHEN** 用户选择特性类型为 enum
- **THEN** 展示 options 列表编辑器和 default 下拉选择

#### Scenario: 保存约束定义
- **WHEN** 用户点击「保存」
- **THEN** 将编辑器状态序列化为 JSON 并调用 PUT /schema API，成功后刷新数据

### Requirement: 套餐表单
系统 SHALL 提供套餐创建/编辑表单（Sheet 抽屉）。表单包含：名称输入、基于商品 ConstraintSchema 动态渲染的约束值编辑器（每个 module 一个区块，每个 feature 根据 type 渲染对应控件）。

#### Scenario: 新建套餐
- **WHEN** 用户点击「创建套餐」并填写表单
- **THEN** 各 module 以折叠区块展示，feature 根据类型渲染：number → 数字输入框、enum → 下拉选择、multiSelect → 多选框组

#### Scenario: 编辑套餐
- **WHEN** 用户编辑已有套餐
- **THEN** 表单预填现有 constraintValues 值

### Requirement: 前端路由和菜单
系统 SHALL 在 license app 的 module.ts 中注册路由：`/license/products`（列表）和 `/license/products/:id`（详情）。对应菜单项由后端 seed 创建，前端通过 menu store 动态渲染。

#### Scenario: 路由注册
- **WHEN** license 前端模块加载
- **THEN** `/license/products` 和 `/license/products/:id` 路由注册到应用路由表

#### Scenario: 权限控制
- **WHEN** 用户无商品管理菜单权限
- **THEN** 菜单中不显示「商品管理」，直接访问路由被 PermissionGuard 拦截
