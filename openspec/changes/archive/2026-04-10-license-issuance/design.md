## Context

Metis 许可模块已完成基础设施：Product（商品）、Plan（套餐）、ProductKey（密钥）、Licensee（授权主体）四个实体的 CRUD，以及 Ed25519 密钥生成和 AES-GCM 私钥加密（`crypto.go`）。现在需要实现核心的许可签发（License Issuance）功能，将这些基础设施串联起来，生成带密码学签名的许可证。

参考 NekoAdmin 的同功能实现，但需适配 Metis 的 Go + Gin + GORM + samber/do 技术栈。

## Goals / Non-Goals

**Goals:**
- 实现 License 模型和完整的签发/吊销/导出流程
- 在 crypto.go 中扩展 canonicalize、signLicense、generateActivationCode 等函数
- 提供 RESTful API 和前端页面
- 与现有 Product/Licensee/Plan 实体正确关联

**Non-Goals:**
- 不实现客户端 SDK 或验签逻辑（离线验证由客户端自行实现）
- 不实现许可证自动续期或到期通知
- 不修改已有的 Product/Plan/Licensee 实体和 API

## Decisions

### D1: License 表使用 uint FK 关联 Product/Licensee

**选择**: License.ProductID 和 License.LicenseeID 使用 `uint` 类型 FK，与 Metis BaseModel.ID 一致。

**替代方案**: NekoAdmin 使用 UUID string FK（Drizzle/PostgreSQL），但 Metis 全局使用 uint 自增 ID + SQLite。

**理由**: 保持与现有模型一致，减少类型转换。Product/Licensee 删除时 License 记录保留（SET NULL），通过快照字段（planName、constraintValues）保证审计完整性。

### D2: Go 标准库实现 Ed25519 签名

**选择**: 使用 `crypto/ed25519` 直接对 canonical payload 签名，签名结果用 base64url（`encoding/base64.RawURLEncoding`）编码。

**理由**: Go 的 ed25519 包直接操作原始 32 字节公钥/64 字节私钥种子，不需要 DER/PKCS8 包装。NekoAdmin（Node.js）需要 DER 格式是因为 Node crypto API 的要求。但 Metis 已在 crypto.go 中以 raw bytes base64 存储密钥，可以直接使用。

### D3: Canonicalize 使用 encoding/json + sort

**选择**: 用 `json.Marshal` 解析为 `interface{}` 后递归排序 key，再序列化为确定性 JSON 字符串。

**替代方案**: 使用第三方 JSON canonicalization 库（如 JCS/RFC 8785）。

**理由**: NekoAdmin 的 canonicalize 是简单的 key 递归排序，Go 标准库足够实现。避免引入外部依赖。

### D4: ActivationCode = base64url(JSON({...payload, sig}))

**选择**: 与 NekoAdmin 完全一致的编码格式。payload 字段: `v`, `pid`, `reg`, `con`, `iat`, `nbf`, `exp`, `kv`，加上 `sig` 签名字段。

**理由**: 保持与 NekoAdmin 许可格式的兼容性，客户端可以用相同逻辑解码和验签。

### D5: .lic 文件通过 JSON 下载 endpoint 提供

**选择**: `GET /api/v1/license/licenses/:id/export` 返回 JSON 格式的 .lic 文件内容，设置 `Content-Disposition: attachment` header。

**替代方案**: 前端自行组装 .lic 文件内容并下载。

**理由**: 后端统一控制 .lic 格式和权限校验，前端只需触发下载。

### D6: 签发表单使用 Sheet（与项目约定一致）

**选择**: 签发许可用右侧 Sheet 抽屉，包含商品选择、授权主体选择、套餐/自定义约束、注册码、有效期等字段。

**理由**: CLAUDE.md 明确规定"新建/编辑表单统一使用 Sheet（抽屉），不要用 Dialog（弹窗）"。

### D7: 私钥解密复用 crypto.go 现有函数

**选择**: 签名时先用 `decryptAESGCM` 解密 `EncryptedPrivateKey`，再 base64 decode 得到原始私钥字节。

**理由**: crypto.go 已有完整的加解密链路，直接复用。

## Risks / Trade-offs

**[吊销仅 DB 标记] → 已导出的 .lic 文件仍然有效**
与 NekoAdmin 一致的设计取舍。离线场景下无法在线验证吊销状态。如需在线吊销，客户端需要额外的吊销查询 API（不在本次范围）。

**[ActivationCode 可能很长] → 包含完整 payload + 签名**
当 constraintValues 复杂时，activationCode 可能达到数百字节。存储为 TEXT 类型无问题，但不适合手工输入。实际使用场景是 .lic 文件传递。

**[FK SET NULL] → 删除 Product/Licensee 后 License 记录丢失关联**
通过在 License 中快照 planName 和 constraintValues 缓解。列表查询用 LEFT JOIN 容忍 NULL。
