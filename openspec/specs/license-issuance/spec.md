# Capability: license-issuance

## Purpose
许可签发与管理后端能力 — 提供 License 数据模型、签名算法（Ed25519 + JSON canonicalization）、签发/吊销/导出/列表/详情 API。

## Requirements

### Requirement: License data model
系统 SHALL 提供 License 数据模型，存储于 `license_licenses` 表，包含以下字段：
- `ProductID` (uint, nullable FK → license_products) — 关联商品
- `LicenseeID` (uint, nullable FK → license_licensees) — 关联授权主体
- `PlanID` (uint, nullable) — 关联套餐（可选）
- `PlanName` (string, required) — 套餐名称快照
- `RegistrationCode` (string, required) — 客户端注册码
- `ConstraintValues` (JSONText) — 功能约束值快照
- `ValidFrom` (time.Time, required) — 生效时间
- `ValidUntil` (*time.Time, nullable) — 过期时间，null 表示永久有效
- `ActivationCode` (text, unique) — base64url 编码的完整许可（payload + 签名）
- `KeyVersion` (int) — 签名使用的密钥版本
- `Signature` (text) — Ed25519 签名（base64url）
- `Status` (string) — `issued` 或 `revoked`
- `IssuedBy` (uint) — 签发操作人 ID
- `RevokedAt` (*time.Time, nullable) — 吊销时间
- `RevokedBy` (*uint, nullable) — 吊销操作人 ID
- `Notes` (text, optional) — 备注

License 模型 SHALL 嵌入 `model.BaseModel` 以获得 ID、CreatedAt、UpdatedAt、DeletedAt 字段。

#### Scenario: License record creation
- **WHEN** 系统创建一条 License 记录
- **THEN** 所有必填字段（ProductID, LicenseeID, PlanName, RegistrationCode, ValidFrom, ActivationCode, KeyVersion, Signature, Status, IssuedBy）MUST 被设置

#### Scenario: License with permanent validity
- **WHEN** ValidUntil 为 null
- **THEN** 该许可被视为永久有效

### Requirement: License payload structure
系统 SHALL 使用以下 JSON payload 结构作为签名输入：
```json
{
  "v": 1,
  "pid": "<product code>",
  "reg": "<registration code>",
  "con": { ... constraint values ... },
  "iat": <unix timestamp issued at>,
  "nbf": <unix timestamp valid from>,
  "exp": <unix timestamp valid until | null>,
  "kv": <key version>
}
```

#### Scenario: Payload construction
- **WHEN** 构建签名 payload
- **THEN** `pid` MUST 使用商品的 Code 字段，`iat` MUST 为签发时刻的 Unix 时间戳，`nbf` MUST 为 ValidFrom 的 Unix 时间戳，`exp` 永久许可时 MUST 为 null

### Requirement: JSON canonicalization
系统 SHALL 实现 JSON canonicalization 函数，递归对所有对象的 key 按字母序排序，确保相同数据产生相同的 JSON 字符串。

#### Scenario: Deterministic serialization
- **WHEN** 对 `{"b":2,"a":1}` 执行 canonicalize
- **THEN** 结果 MUST 为 `{"a":1,"b":2}`

#### Scenario: Nested object canonicalization
- **WHEN** 对 `{"z":{"b":2,"a":1},"a":3}` 执行 canonicalize
- **THEN** 结果 MUST 为 `{"a":3,"z":{"a":1,"b":2}}`

### Requirement: License signing
系统 SHALL 使用 Ed25519 算法对 canonicalized payload 进行签名。签名流程：
1. 用 AES-GCM 解密 ProductKey 的 EncryptedPrivateKey
2. Base64 decode 得到原始 Ed25519 私钥字节
3. 对 canonicalize(payload) 的 UTF-8 字节签名
4. 签名结果用 base64url（无 padding）编码

#### Scenario: Successful signing
- **WHEN** 使用有效私钥对 payload 签名
- **THEN** 生成的签名 MUST 可用对应的公钥验证通过

#### Scenario: Signature verification
- **WHEN** 使用对应公钥对 canonicalize(payload) 和签名进行验证
- **THEN** 验证 MUST 返回 true

### Requirement: Activation code generation
系统 SHALL 将 payload 与签名组合为 activationCode：
1. 构建 JSON 对象：`{...payload, "sig": "<signature>"}`
2. 序列化为 JSON 字符串
3. Base64url 编码（无 padding）

#### Scenario: Activation code encode/decode roundtrip
- **WHEN** 对 payload + signature 生成 activationCode 后再 decode
- **THEN** MUST 还原出原始 payload 和 signature

### Requirement: Issue license
系统 SHALL 提供许可签发功能，通过 `POST /api/v1/license/licenses` 调用。

签发流程：
1. 校验商品 MUST 存在且 status 为 `published`
2. 校验授权主体 MUST 存在且 status 为 `active`
3. 获取商品当前密钥（isCurrent=true）MUST 存在
4. 构建 LicensePayload
5. Canonicalize → Ed25519 签名 → 生成 ActivationCode
6. 在事务内创建 License 记录，status 设为 `issued`

#### Scenario: Successful issuance
- **WHEN** 用户提交有效的签发请求（已发布商品、活跃授权主体、有效注册码）
- **THEN** 系统 MUST 创建 License 记录，status 为 `issued`，activationCode 包含有效签名

#### Scenario: Product not published
- **WHEN** 尝试对未发布商品签发许可
- **THEN** 系统 MUST 返回错误 "只能对已发布商品签发许可"

#### Scenario: Licensee not active
- **WHEN** 尝试对已归档授权主体签发许可
- **THEN** 系统 MUST 返回错误 "授权主体必须为活跃状态"

#### Scenario: No current key
- **WHEN** 商品没有当前有效密钥
- **THEN** 系统 MUST 返回错误 "商品密钥不存在"

### Requirement: Revoke license
系统 SHALL 提供许可吊销功能，通过 `PATCH /api/v1/license/licenses/:id/revoke` 调用。

#### Scenario: Successful revocation
- **WHEN** 对状态为 `issued` 的许可执行吊销
- **THEN** 系统 MUST 将 status 更新为 `revoked`，设置 revokedAt 和 revokedBy

#### Scenario: Already revoked
- **WHEN** 对已吊销的许可再次吊销
- **THEN** 系统 MUST 返回错误 "许可已吊销"

#### Scenario: License not found
- **WHEN** 对不存在的许可 ID 执行吊销
- **THEN** 系统 MUST 返回 404 错误

### Requirement: Export .lic file
系统 SHALL 提供 `.lic` 文件导出功能，通过 `GET /api/v1/license/licenses/:id/export` 调用。

.lic 文件 JSON 结构：
```json
{
  "version": 1,
  "activationCode": "<base64url>",
  "publicKey": "<base64 public key>",
  "meta": {
    "productCode": "...",
    "productName": "...",
    "licenseeName": "...",
    "validFrom": "2026-01-01T00:00:00Z",
    "validUntil": null,
    "issuedAt": "2026-04-10T12:00:00Z"
  }
}
```

响应 MUST 设置 `Content-Type: application/json` 和 `Content-Disposition: attachment; filename="<productCode>_<YYYYMMDD>.lic"`。

#### Scenario: Successful export
- **WHEN** 导出状态为 `issued` 的许可
- **THEN** 系统 MUST 返回包含 activationCode、publicKey 和 meta 信息的 JSON 文件

#### Scenario: Export revoked license
- **WHEN** 尝试导出已吊销的许可
- **THEN** 系统 MUST 返回错误 "已吊销的许可不能导出"

### Requirement: List licenses
系统 SHALL 提供许可列表查询功能，通过 `GET /api/v1/license/licenses` 调用。

支持以下筛选条件：
- `productId` — 按商品筛选
- `licenseeId` — 按授权主体筛选
- `status` — 按状态筛选（issued/revoked）
- `keyword` — 搜索 planName 和 registrationCode

列表 MUST 通过 LEFT JOIN 返回 productName 和 licenseeName 冗余字段。分页使用标准 ListParams。

#### Scenario: List with filters
- **WHEN** 用户按 productId 和 status=issued 筛选
- **THEN** 系统 MUST 仅返回该商品下状态为 issued 的许可记录

#### Scenario: Deleted product/licensee
- **WHEN** 关联的商品或授权主体被删除
- **THEN** 列表中对应的 productName/licenseeName MUST 显示为空，不影响查询

### Requirement: Get license detail
系统 SHALL 提供许可详情查询功能，通过 `GET /api/v1/license/licenses/:id` 调用。

返回 MUST 包含 License 全部字段以及关联的 productName、productCode、licenseeName、licenseeCode。

#### Scenario: Get existing license
- **WHEN** 查询存在的许可 ID
- **THEN** 系统 MUST 返回完整的许可详情

#### Scenario: Get non-existent license
- **WHEN** 查询不存在的许可 ID
- **THEN** 系统 MUST 返回 404 错误
