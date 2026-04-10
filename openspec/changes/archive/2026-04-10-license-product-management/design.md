## Context

Metis 目前有内核（用户/角色/菜单/认证/设置/任务/审计）和一个 identity app。现在需要新增 license app，第一阶段实现商品管理——这是许可管理链路的基础数据层。

参考了 NekoAdmin 老代码的许可管理业务设计，但实现完全遵循 Metis 的 Go + Gin + GORM + samber/do 架构。

## Goals / Non-Goals

**Goals:**
- 实现完整的商品生命周期管理（CRUD + 状态机）
- 支持动态 ConstraintSchema 定义功能模块和可授权维度
- 支持 Plan 套餐管理，预设约束值组合
- 实现 Ed25519 密钥对生成、加密存储、版本轮转
- 前端提供商品管理页面，包含 ConstraintSchema 可视化编辑器

**Non-Goals:**
- 授权主体（Licensee）管理 — 后续阶段
- 许可证颁发和签名 — 后续阶段
- 许可证在线/离线验证 — 后续阶段
- 商品定价和计费 — 不在此系统范围

## Decisions

### D1: 数据模型采用 JSON 字段存储动态 Schema

**决策**：ConstraintSchema 和 ConstraintValues 使用 `json.RawMessage` 存储在 TEXT 列中。

**备选**：
- EAV 表（Entity-Attribute-Value）— 查询灵活但复杂度高，Join 多
- 独立列 — 无法支持动态功能模块定义

**理由**：ConstraintSchema 是商品特有的维度定义，不同商品结构不同，JSON 是最自然的表达。SQLite 的 JSON 函数在需要时可以做查询。前端直接消费 JSON 结构渲染编辑器。

### D2: Ed25519 私钥 AES-256-GCM 加密存储

**决策**：私钥 base64 编码后用 AES-256-GCM 加密，密钥来源优先级：`LICENSE_KEY_SECRET` > 从 `JWT_SECRET` 派生（SHA-256 hash）。

**备选**：
- 明文存储 — 不安全
- 操作系统 keyring — 不可移植
- 外部 KMS — 过于复杂

**理由**：AES-GCM 提供认证加密，Go 标准库原生支持。环境变量方式与 Metis 现有配置模式一致。从 JWT_SECRET 派生作为 fallback 减少必配项。

### D3: 密钥版本轮转而非替换

**决策**：每次轮转生成新版本密钥（version++），旧密钥标记 `isCurrent=false` 并记录 `revokedAt`，但不删除。

**理由**：已颁发的许可证记录了签名时的 `keyVersion`，保留旧公钥才能验证历史许可证。

### D4: Plan 软删除

**决策**：Plan 使用 BaseModel 的软删除（GORM `DeletedAt`），而非硬删除。

**备选**：硬删除（老代码方案）

**理由**：已颁发的许可证会快照 planName，但保留 Plan 记录便于审计追溯。软删除是 Metis 的默认模式。

### D5: 文件结构遵循单文件分层

**决策**：license app 所有后端代码放在 `internal/app/license/` 下，按 model/repo/service/handler/seed 各一个文件。

```
internal/app/license/
├── app.go           # App 接口实现 + init() 注册
├── model.go         # Product, Plan, ProductKey + Response 类型
├── crypto.go        # Ed25519 密钥生成、加密/解密、签名工具
├── repository.go    # GORM 数据访问
├── service.go       # 业务逻辑（状态机、密钥轮转、Plan 管理）
├── handler.go       # HTTP handlers + 请求/响应结构
└── seed.go          # 菜单 + Casbin 策略种子
```

### D6: API 路由前缀 `/license`

**决策**：所有商品管理 API 挂在 `authed.Group("/license")` 下，如 `/api/v1/license/products`。

**理由**：与 identity app 的 `/identity-sources` 路径风格一致，通过 group 前缀区分 app 域。

### D7: 前端 App 模块结构

**决策**：前端放在 `web/src/apps/license/` 下，通过 `registerApp()` 注册路由。

```
web/src/apps/license/
├── module.ts                    # registerApp() 入口
├── pages/
│   ├── products/
│   │   ├── index.tsx            # 商品列表
│   │   └── [id].tsx             # 商品详情（Tabs: 基本信息/套餐/约束/密钥）
│   └── ...                      # 后续阶段的页面
└── components/
    ├── constraint-editor.tsx     # ConstraintSchema 可视化编辑器
    ├── plan-form.tsx             # 套餐表单（Sheet 抽屉）
    └── ...
```

## Risks / Trade-offs

- **[ConstraintSchema 编辑器前端复杂度高]** → 分步实现：先支持 number/enum 两种类型，multiSelect 稍后增加。编辑器参考老代码的交互模式。
- **[私钥加密增加操作复杂度]** → 加密/解密封装在 `crypto.go` 中，对外暴露简洁 API。缺少 LICENSE_KEY_SECRET 时自动 fallback。
- **[JSON Schema 无法做 DB 级约束校验]** → 在 Service 层做 Go 结构体校验，确保 ConstraintSchema 和 ConstraintValues 的一致性。
- **[SQLite 无 JSON 列类型]** → 使用 TEXT 列 + Go 层序列化/反序列化，不依赖数据库 JSON 函数。
