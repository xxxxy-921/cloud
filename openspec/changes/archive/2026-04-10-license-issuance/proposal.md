## Why

Metis 的许可模块已实现商品管理（Product）、套餐管理（Plan）、密钥管理（ProductKey）和授权主体管理（Licensee），但缺少核心的"许可签发"功能——即将商品、套餐、授权主体组合起来，生成带密码学签名的许可证并支持导出 `.lic` 文件。这是许可管理闭环的最后一环，没有它前面的基础设施无法产生实际价值。

## What Changes

- 新增 `License` 数据模型，记录许可签发的完整信息（关联商品/授权主体/套餐、注册码、约束值快照、有效期、签名、激活码）
- 新增签发流程：校验商品和授权主体状态 → 构建 payload → Ed25519 签名 → 生成 activationCode → 入库
- 新增吊销功能：将已签发许可标记为 revoked
- 新增 `.lic` 文件导出：包含 activationCode + publicKey + meta 信息，支持客户端离线验签
- 新增加密签名工具函数：JSON canonicalize、signLicense、generateActivationCode
- 新增许可列表页、签发表单、许可详情页（含导出）

## Capabilities

### New Capabilities

- `license-issuance`: 许可签发核心业务——License 模型、签发/吊销/导出的后端逻辑、签名与激活码生成
- `license-issuance-ui`: 许可签发前端——许可列表页、签发表单（Sheet）、详情页、.lic 文件下载

### Modified Capabilities

- `license-product`: 签发时需查询已发布商品及其当前密钥，现有 API 已满足，无需求变更
- `license-licensee`: 签发时需查询活跃授权主体，现有 API 已满足，无需求变更

（无需修改现有 spec）

## Impact

- **后端**：`internal/app/license/` 新增 license model、repository、service、handler；扩展 crypto.go 加入签名/canonicalize 函数
- **前端**：`web/src/apps/license/` 新增许可页面和组件，module.ts 注册新路由
- **API**：新增 `/api/v1/license/licenses` 系列 endpoint（CRUD + revoke + export）
- **数据库**：新增 `license_licenses` 表，通过 GORM AutoMigrate 自动创建
- **种子数据**：seed.go 新增许可签发相关菜单和 Casbin 策略
