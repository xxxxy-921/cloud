## ADDED Requirements

### Requirement: Plan CRUD
系统 SHALL 支持创建、读取、更新、删除套餐（Plan）。Plan 绑定到一个 Product，包含字段：Name（名称，必填，同一 Product 下唯一，最长 128 字符）、ConstraintValues（约束值，JSON，基于 Product 的 ConstraintSchema 定义）、IsDefault（是否默认套餐）、SortOrder（排序序号）。

#### Scenario: 创建套餐
- **WHEN** 用户为商品创建套餐，提供 name 和 constraintValues
- **THEN** 系统校验 constraintValues 与商品的 ConstraintSchema 一致性，创建套餐记录

#### Scenario: 套餐名称唯一
- **WHEN** 用户为同一商品创建名称已存在的套餐
- **THEN** 系统返回 400 错误，提示名称已存在

#### Scenario: 更新套餐
- **WHEN** 用户更新套餐的 name、constraintValues 或 sortOrder
- **THEN** 系统校验后更新对应字段

#### Scenario: 删除套餐
- **WHEN** 用户删除套餐
- **THEN** 系统软删除该套餐（设置 DeletedAt）

### Requirement: 默认套餐管理
系统 SHALL 支持将某个套餐设为默认。每个商品最多有一个默认套餐。设为默认时 SHALL 在事务中先清除同商品下其他套餐的默认标记。

#### Scenario: 设为默认套餐
- **WHEN** 用户将某套餐设为默认
- **THEN** 该套餐 isDefault=true，同商品下原默认套餐（如有）isDefault 变为 false

#### Scenario: 取消默认套餐
- **WHEN** 用户取消某套餐的默认标记
- **THEN** 该套餐 isDefault=false，商品无默认套餐

### Requirement: ConstraintValues 校验
系统 SHALL 在创建和更新套餐时校验 constraintValues 与商品 ConstraintSchema 的一致性。每个 module key 对应 schema 中的模块，每个 feature value 的类型须匹配 schema 定义（number 在 min/max 范围内，enum 值在 options 中，multiSelect 值均在 options 中）。

#### Scenario: 校验通过
- **WHEN** 提交的 constraintValues 中所有值均符合 ConstraintSchema 定义
- **THEN** 操作正常执行

#### Scenario: 值超出范围
- **WHEN** number 类型 feature 的值小于 min 或大于 max
- **THEN** 系统返回 400 错误，指明哪个 module/feature 值不合法

#### Scenario: 未知 module key
- **WHEN** constraintValues 包含 ConstraintSchema 中不存在的 module key
- **THEN** 系统返回 400 错误，提示未知的模块

### Requirement: Plan API 路由
系统 SHALL 在商品路由下提供套餐管理 API，所有接口 SHALL 受 JWT + Casbin 保护。

#### Scenario: API 路由清单
- **WHEN** license app 注册路由
- **THEN** 以下路由可用：
  - `POST /api/v1/license/products/:id/plans` — 创建套餐
  - `PUT /api/v1/license/plans/:id` — 编辑套餐
  - `DELETE /api/v1/license/plans/:id` — 删除套餐
  - `PATCH /api/v1/license/plans/:id/default` — 设为/取消默认
