## Why

Metis 需要一个许可管理系统，用于管理软件产品的授权许可颁发。第一步是建设「商品管理」能力——定义可授权的软件产品、功能维度（ConstraintSchema）、套餐（Plan）以及用于离线验证的 Ed25519 签名密钥对。这是整个许可管理链路的基础，后续的授权主体管理和许可证颁发都依赖商品定义。

## What Changes

- 新增 `license` 可插拔 App，遵循 Metis App 接口（Model → Repo → Service → Handler → Seed）
- 新增三个数据模型：`Product`（商品）、`Plan`（套餐）、`ProductKey`（签名密钥对），表名带 `license_` 前缀
- Product 支持自定义 ConstraintSchema（JSON），定义功能模块和可授权维度（number / enum / multiSelect）
- Plan 绑定到 Product，预设 ConstraintValues 组合，支持设为默认、排序
- ProductKey 使用 Ed25519 算法，私钥 AES-GCM 加密存储，支持版本轮转
- Product 状态机：`unpublished ↔ published ↔ archived`
- 新增 `LICENSE_KEY_SECRET` 环境变量用于私钥加密
- 前端新增商品管理页面，包含 ConstraintSchema 可视化编辑器和套餐管理

## Capabilities

### New Capabilities
- `license-product`: 商品 CRUD、状态机、ConstraintSchema 定义、Ed25519 密钥对生成与轮转
- `license-plan`: 套餐管理——绑定商品、预设约束值、默认套餐、排序
- `license-product-ui`: 商品管理前端——列表/详情页、ConstraintSchema 编辑器、套餐管理 Tab、密钥管理

### Modified Capabilities
<!-- 无现有能力需要修改 -->

## Impact

- **后端**：新增 `internal/app/license/` 目录，包含完整的 model/repo/service/handler/seed
- **前端**：新增 `web/src/apps/license/` 模块，注册路由和页面
- **数据库**：AutoMigrate 新增 `license_products`、`license_plans`、`license_product_keys` 三张表
- **Casbin 策略**：seed 阶段为 admin 角色添加商品管理相关的 API 和菜单权限
- **环境变量**：新增 `LICENSE_KEY_SECRET`（可选，缺省时从 JWT_SECRET 派生）
- **Edition**：需在 `cmd/server/edition_full.go` 中 import `license` app
