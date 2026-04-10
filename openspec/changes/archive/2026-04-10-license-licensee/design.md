## Context

许可管理模块已有 Product / Plan / ProductKey 三层实体，覆盖了"商品定义"和"套餐配置"。下一步是引入"授权主体"（Licensee），即被授权使用软件的客户或组织。参考 NekoAdmin 的同名模块，在 Metis 架构下重新实现。

现有代码结构：
- 后端：`internal/app/license/` — model / repository / service / handler / seed / app.go
- 前端：`web/src/apps/license/` — module.ts + pages + components
- 模式：IOC (samber/do) → handler → service → repository → GORM

## Goals / Non-Goals

**Goals:**
- 实现 Licensee CRUD（创建、查看、编辑、列表）
- 实现 Licensee 状态管理（active ↔ archived）
- 自动生成唯一代码（`LS-` 前缀）
- 联系信息 + 企业信息（JSON 存储）
- 前端列表页 + Drawer 表单，搜索、状态筛选、分页
- 菜单与 Casbin 策略种子数据

**Non-Goals:**
- 不做独立详情页（列表 + Drawer 足够）
- 不做 License 签发（后续变更）
- 不加 `createdBy` 字段（依赖审计日志追溯）
- 不做导入/导出功能

## Decisions

### D1: businessInfo 用 JSON 字符串存储

将联系人之外的企业信息（地址、税号、银行信息等）作为 JSON 存在一个 TEXT 字段中，复用现有的 `JSONText` 类型。

**理由**：Product 的 ConstraintSchema 已用此模式；这些字段不需要被独立查询/索引；保持表结构简洁。

**替代方案**：打平成 6 个独立字段 — 类型更安全但表结构冗长，且这些字段极少用于过滤查询。

### D2: status 字段 + BaseModel 软删除并存

Licensee 有独立的 `status` 字段（`active` / `archived`），同时继承 BaseModel 的 GORM 软删除。

**理由**：归档是业务操作（可恢复），软删除是技术兜底（不同语义）。与 NekoAdmin 保持一致。

### D3: 不加 createdBy 字段

依赖审计日志中间件记录创建者，不在模型上冗余存储。

**理由**：Metis 现有模型（Product/Plan）均不存 createdBy；审计日志已完整记录操作者和操作；减少 handler 层需要从 JWT context 传递 userId 的耦合。

### D4: Code 生成用 crypto/rand

格式 `LS-` + 12 位随机字母数字，用 Go 标准库 `crypto/rand` 生成，碰撞重试最多 3 次。不引入第三方 nanoid 库。

**理由**：12 位字母数字（62^12 ≈ 3.2×10^21）碰撞概率极低；`crypto/rand` 是标准库，无额外依赖。

### D5: 前端只做列表页 + Drawer

不做独立的 `/license/licensees/:id` 详情页。所有查看和编辑在 Drawer 中完成，Drawer 内分区展示基本信息、联系信息、企业信息。

**理由**：保持和现有 Product Sheet 风格一致；授权主体信息量适中，Drawer 足够承载；将来做 License 签发时如需要可再加详情页。

### D6: 联系信息字段放在模型顶层

`contactName` / `contactPhone` / `contactEmail` 作为模型的独立字段（非 JSON），因为这三个字段使用频率高，可能用于列表展示和搜索。

**理由**：列表页需要直接展示联系人；搜索时可能按联系人过滤；与 businessInfo（低频访问）区分存储层级。

## Risks / Trade-offs

- **[JSON 字段迁移]** → 如果将来需要按企业信息字段查询/统计，JSON 存储会增加复杂度。缓解：SQLite 支持 JSON 函数，必要时可用 `json_extract()`。
- **[无 createdBy]** → 查询"谁创建了这个主体"需要查审计日志而非直接关联查询。缓解：审计日志有索引，查询成本可接受；如果将来频繁需要此信息，可考虑加字段。
- **[无详情页]** → 当 Licensee 关联的 License 数量增多后，Drawer 可能不够展示。缓解：这是后续 License 签发变更的范畴，届时再评估。
