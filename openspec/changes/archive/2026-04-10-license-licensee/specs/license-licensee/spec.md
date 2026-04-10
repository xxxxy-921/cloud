## ADDED Requirements

### Requirement: Licensee data model
系统 SHALL 提供 Licensee（授权主体）实体，存储在 `license_licensees` 表中，包含以下字段：
- `id` (uint, PK) — 继承 BaseModel
- `name` (varchar 128, NOT NULL, UNIQUE) — 主体名称
- `code` (varchar 64, NOT NULL, UNIQUE) — 自动生成的唯一代码，格式 `LS-{12位随机字母数字}`
- `contact_name` (varchar 64) — 联系人姓名
- `contact_phone` (varchar 32) — 联系人电话
- `contact_email` (varchar 128) — 联系人邮箱
- `business_info` (TEXT, JSON) — 企业信息 JSON，结构：`{address?, taxId?, bankName?, bankAccount?, swift?, iban?}`
- `notes` (TEXT) — 备注
- `status` (varchar 16, NOT NULL, default "active") — 状态：`active` 或 `archived`
- `created_at`, `updated_at`, `deleted_at` — 继承 BaseModel

#### Scenario: Licensee 表自动迁移
- **WHEN** 应用启动时 LicenseApp.Models() 返回 Licensee 模型
- **THEN** GORM AutoMigrate SHALL 创建或更新 `license_licensees` 表

### Requirement: Licensee 唯一代码自动生成
创建 Licensee 时系统 SHALL 自动生成格式为 `LS-{12位随机字母数字}` 的唯一代码，使用 `crypto/rand` 生成。如遇唯一约束冲突 SHALL 重试，最多 3 次。

#### Scenario: 创建时自动生成 code
- **WHEN** 创建 Licensee 请求中不包含 code 字段
- **THEN** 系统 SHALL 自动生成 `LS-` 前缀 + 12 位随机字母数字的代码

#### Scenario: code 碰撞重试
- **WHEN** 生成的 code 与已有记录冲突
- **THEN** 系统 SHALL 重新生成，最多重试 3 次；超过 3 次 SHALL 返回错误

### Requirement: Licensee 名称唯一性
系统 SHALL 保证 Licensee 名称在所有未软删除记录中唯一（数据库 UNIQUE INDEX）。

#### Scenario: 创建重名 Licensee
- **WHEN** 创建 Licensee 的名称已被占用
- **THEN** 系统 SHALL 返回 400 错误，提示"主体名称已存在"

#### Scenario: 更新为重名
- **WHEN** 更新 Licensee 名称为另一已存在的名称
- **THEN** 系统 SHALL 返回 400 错误，提示"主体名称已存在"

### Requirement: Licensee CRUD API
系统 SHALL 提供以下 RESTful API 端点，均在 JWT + Casbin 中间件保护下：

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/license/licensees` | 创建授权主体 |
| GET | `/api/v1/license/licensees` | 列表查询（分页+搜索+状态筛选） |
| GET | `/api/v1/license/licensees/:id` | 获取单个详情 |
| PUT | `/api/v1/license/licensees/:id` | 更新授权主体 |
| PATCH | `/api/v1/license/licensees/:id/status` | 变更状态 |

#### Scenario: 创建授权主体
- **WHEN** POST `/api/v1/license/licensees` 携带 `{name, contactName?, contactPhone?, contactEmail?, businessInfo?, notes?}`
- **THEN** 系统 SHALL 创建记录，自动生成 code，返回 `{code:0, data: LicenseeResponse}`

#### Scenario: 列表查询
- **WHEN** GET `/api/v1/license/licensees?keyword=xxx&status=active&page=1&pageSize=20`
- **THEN** 系统 SHALL 返回分页结果 `{items: [], total: N}`，keyword 模糊匹配 name 和 code，默认排除 archived 状态（除非显式指定 `status=archived` 或 `status=all`）

#### Scenario: 获取单个详情
- **WHEN** GET `/api/v1/license/licensees/:id`
- **THEN** 系统 SHALL 返回完整的 LicenseeResponse，包含 businessInfo 解析后的结构

#### Scenario: 更新授权主体
- **WHEN** PUT `/api/v1/license/licensees/:id` 携带可更新字段
- **THEN** 系统 SHALL 更新 name、contactName、contactPhone、contactEmail、businessInfo、notes 字段，code 和 status 不可通过此接口修改

#### Scenario: 获取不存在的记录
- **WHEN** GET/PUT/PATCH 指定的 id 不存在
- **THEN** 系统 SHALL 返回 404 错误

### Requirement: Licensee 状态管理
系统 SHALL 支持 Licensee 在 `active` 和 `archived` 之间切换。

#### Scenario: 归档授权主体
- **WHEN** PATCH `/api/v1/license/licensees/:id/status` 携带 `{status: "archived"}`，且当前状态为 `active`
- **THEN** 系统 SHALL 将状态更新为 `archived`

#### Scenario: 恢复授权主体
- **WHEN** PATCH `/api/v1/license/licensees/:id/status` 携带 `{status: "active"}`，且当前状态为 `archived`
- **THEN** 系统 SHALL 将状态更新为 `active`

#### Scenario: 相同状态转换
- **WHEN** 请求的目标状态与当前状态相同
- **THEN** 系统 SHALL 返回 400 错误，提示状态无需变更

### Requirement: Licensee Response 类型
系统 SHALL 提供 `LicenseeResponse` 结构用于 API 响应，包含所有可公开字段：id, name, code, contactName, contactPhone, contactEmail, businessInfo, notes, status, createdAt, updatedAt。

#### Scenario: ToResponse 转换
- **WHEN** Licensee 模型需要返回给前端
- **THEN** SHALL 调用 `ToResponse()` 方法转换为 `LicenseeResponse`

### Requirement: Licensee 种子数据
系统 SHALL 在启动时自动创建授权主体相关的菜单和 Casbin 策略。

#### Scenario: 菜单种子
- **WHEN** 应用启动且 `license:licensee:list` 菜单不存在
- **THEN** 系统 SHALL 在"许可管理"目录下创建"授权主体"菜单，路径 `/license/licensees`，以及按钮权限：`license:licensee:create`、`license:licensee:update`、`license:licensee:archive`

#### Scenario: Casbin 策略种子
- **WHEN** 应用启动且 admin 角色缺少 licensee 相关策略
- **THEN** 系统 SHALL 为 admin 角色添加所有 licensee API 端点的 Casbin 策略及菜单权限策略

### Requirement: Licensee 审计日志
所有 Licensee 的写操作（创建、更新、状态变更）SHALL 通过 Gin context 设置审计字段，由审计中间件自动记录。

#### Scenario: 创建操作审计
- **WHEN** 成功创建 Licensee
- **THEN** 审计日志 SHALL 记录 action=`create`、resource=`licensee`、resource_id=新记录ID

#### Scenario: 状态变更审计
- **WHEN** 成功变更 Licensee 状态
- **THEN** 审计日志 SHALL 记录 action=`archive` 或 `unarchive`、resource=`licensee`、resource_id=记录ID
