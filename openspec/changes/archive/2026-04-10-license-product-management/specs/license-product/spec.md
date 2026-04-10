## ADDED Requirements

### Requirement: Product CRUD
系统 SHALL 支持创建、读取、更新商品（Product）。商品包含以下字段：Name（名称，必填，最长 128 字符）、Code（编码，必填，唯一，最长 64 字符，仅允许小写字母、数字、连字符）、Description（描述，可选，TEXT）、Status（状态，默认 unpublished）、ConstraintSchema（约束定义，JSON）。

创建商品时 SHALL 自动生成 Ed25519 密钥对（版本 1，isCurrent=true）。

#### Scenario: 创建商品
- **WHEN** 用户提交商品创建请求（name="NekoMonitor", code="neko-monitor"）
- **THEN** 系统创建商品记录，status 为 unpublished，同时生成 Ed25519 密钥对 v1

#### Scenario: 商品编码唯一性
- **WHEN** 用户创建商品使用已存在的 code
- **THEN** 系统返回 400 错误，提示编码已存在

#### Scenario: 获取商品列表
- **WHEN** 用户请求商品列表，可传入 keyword（搜索名称/编码）、status（筛选状态）、page、pageSize
- **THEN** 系统返回分页结果，包含 items（商品列表，含 planCount）和 total

#### Scenario: 获取商品详情
- **WHEN** 用户请求商品详情
- **THEN** 系统返回商品信息，包含关联的 Plans 列表（按 sortOrder 排序）

#### Scenario: 更新商品基本信息
- **WHEN** 用户更新商品的 name 或 description
- **THEN** 系统更新对应字段，code 不可修改

### Requirement: ConstraintSchema 定义
系统 SHALL 支持为商品定义 ConstraintSchema，描述该商品可授权的功能模块和维度。ConstraintSchema 是一个 ConstraintModule 数组，每个 Module 包含 key（标识）、label（显示名）、features（特性数组）。每个 Feature 包含 key、type（number/enum/multiSelect）、label，以及类型相关属性（number: min/max/default; enum/multiSelect: options/default）。

#### Scenario: 更新 ConstraintSchema
- **WHEN** 用户提交新的 ConstraintSchema JSON
- **THEN** 系统校验 JSON 结构合法性（module key 唯一、feature key 在模块内唯一、type 合法），校验通过后保存

#### Scenario: ConstraintSchema 校验失败
- **WHEN** 用户提交的 ConstraintSchema 包含重复的 module key 或无效的 feature type
- **THEN** 系统返回 400 错误，说明具体的校验失败原因

### Requirement: Product 状态机
系统 SHALL 实现商品状态转换：unpublished → published（发布）、published → unpublished（下架）、unpublished/published → archived（归档）、archived → unpublished（恢复）。不合法的状态转换 SHALL 被拒绝。

#### Scenario: 发布商品
- **WHEN** 商品状态为 unpublished，用户请求发布
- **THEN** 商品状态变为 published

#### Scenario: 下架商品
- **WHEN** 商品状态为 published，用户请求下架
- **THEN** 商品状态变为 unpublished

#### Scenario: 归档商品
- **WHEN** 商品状态为 unpublished 或 published，用户请求归档
- **THEN** 商品状态变为 archived

#### Scenario: 恢复归档商品
- **WHEN** 商品状态为 archived，用户请求恢复
- **THEN** 商品状态变为 unpublished

#### Scenario: 非法状态转换
- **WHEN** 用户请求不合法的状态转换（如 archived → published）
- **THEN** 系统返回 400 错误，提示非法的状态转换

### Requirement: Ed25519 密钥对管理
系统 SHALL 为每个商品维护 Ed25519 签名密钥对。私钥 SHALL 使用 AES-256-GCM 加密后存储，加密密钥来源：优先 `LICENSE_KEY_SECRET` 环境变量，缺省时从 `JWT_SECRET` 做 SHA-256 派生。公钥以 base64 编码明文存储。

#### Scenario: 创建商品自动生成密钥对
- **WHEN** 商品创建成功
- **THEN** 自动生成 Ed25519 密钥对，version=1，isCurrent=true，私钥加密存储

#### Scenario: 密钥轮转
- **WHEN** 用户请求对商品执行密钥轮转
- **THEN** 在事务中：旧密钥 isCurrent 置为 false 并设 revokedAt，生成新密钥 version=旧版本+1、isCurrent=true

#### Scenario: 获取当前公钥
- **WHEN** 用户请求商品的当前公钥
- **THEN** 系统返回 isCurrent=true 的密钥的 publicKey 和 version

#### Scenario: 缺少加密密钥
- **WHEN** LICENSE_KEY_SECRET 和 JWT_SECRET 均未设置
- **THEN** 创建商品时返回 500 错误，提示缺少加密配置

### Requirement: Product API 路由
系统 SHALL 在 `/api/v1/license/products` 下提供商品管理 REST API，所有接口 SHALL 受 JWT + Casbin 保护，操作 SHALL 记录审计日志。

#### Scenario: API 路由清单
- **WHEN** license app 注册路由
- **THEN** 以下路由可用：
  - `POST /api/v1/license/products` — 创建商品
  - `GET /api/v1/license/products` — 商品列表
  - `GET /api/v1/license/products/:id` — 商品详情
  - `PUT /api/v1/license/products/:id` — 编辑基本信息
  - `PUT /api/v1/license/products/:id/schema` — 更新 ConstraintSchema
  - `PATCH /api/v1/license/products/:id/status` — 状态转换
  - `POST /api/v1/license/products/:id/rotate-key` — 密钥轮转
  - `GET /api/v1/license/products/:id/public-key` — 获取当前公钥

### Requirement: Seed 数据
系统 SHALL 在 license app 启动 seed 时注册菜单项和 Casbin 策略。菜单挂在一级目录「许可管理」下，包含「商品管理」子菜单。admin 角色 SHALL 获得所有商品管理 API 的访问权限。

#### Scenario: Seed 菜单
- **WHEN** license app seed 执行
- **THEN** 创建「许可管理」一级菜单目录和「商品管理」子菜单（如不存在）

#### Scenario: Seed Casbin 策略
- **WHEN** license app seed 执行
- **THEN** admin 角色获得所有 `/api/v1/license/products*` 路由的访问策略
