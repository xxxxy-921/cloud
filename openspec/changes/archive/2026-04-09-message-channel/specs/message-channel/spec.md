## ADDED Requirements

### Requirement: MessageChannel 数据模型
系统 SHALL 维护 message_channel 表，包含以下字段：name（通道名称）、type（通道类型，如 "email"）、config（JSON 文本，存储通道配置）、enabled（启用状态）。嵌入 BaseModel 提供 id、created_at、updated_at、deleted_at。

#### Scenario: 创建邮件通道
- **WHEN** 管理员提交通道创建请求，type 为 "email"，config 包含 host/port/secure/username/password/from
- **THEN** 系统 SHALL 创建记录并返回通道 ID

#### Scenario: 类型创建后不可变
- **WHEN** 管理员尝试更新已有通道的 type 字段
- **THEN** 系统 SHALL 忽略 type 字段的变更，保持原值

### Requirement: 通道 CRUD 操作
系统 SHALL 提供完整的通道管理 API：列表查询（分页+搜索）、单条查询、创建、更新、删除。

#### Scenario: 列表查询
- **WHEN** 管理员请求 GET /api/v1/channels，传入 page、pageSize、keyword 参数
- **THEN** 系统 SHALL 返回分页结果，config 中的 password 字段 SHALL 脱敏为 "******"

#### Scenario: 单条查询
- **WHEN** 管理员请求 GET /api/v1/channels/:id
- **THEN** 系统 SHALL 返回通道详情，password 字段脱敏

#### Scenario: 创建通道
- **WHEN** 管理员请求 POST /api/v1/channels，body 包含 name、type、config
- **THEN** 系统 SHALL 校验必填字段后创建记录

#### Scenario: 更新通道
- **WHEN** 管理员请求 PUT /api/v1/channels/:id，config 中 password 值为 "******"
- **THEN** 系统 SHALL 保留数据库中原始 password 值不变，仅更新其他字段

#### Scenario: 删除通道
- **WHEN** 管理员请求 DELETE /api/v1/channels/:id
- **THEN** 系统 SHALL 软删除该通道记录

### Requirement: 切换通道启用状态
系统 SHALL 提供独立的启用/禁用切换接口。

#### Scenario: 切换启用状态
- **WHEN** 管理员请求 PUT /api/v1/channels/:id/toggle
- **THEN** 系统 SHALL 翻转 enabled 字段并返回更新后的状态

### Requirement: 测试连接
系统 SHALL 提供测试通道配置可用性的接口。

#### Scenario: 测试邮件连接成功
- **WHEN** 管理员请求 POST /api/v1/channels/:id/test，且 SMTP 配置有效
- **THEN** 系统 SHALL 返回 `{"success": true}`

#### Scenario: 测试邮件连接失败
- **WHEN** 管理员请求 POST /api/v1/channels/:id/test，且 SMTP 配置无效
- **THEN** 系统 SHALL 返回 `{"success": false, "error": "错误信息"}`

### Requirement: 发送测试邮件
系统 SHALL 提供发送测试邮件的接口，验证通道端到端可用。

#### Scenario: 发送测试邮件成功
- **WHEN** 管理员请求 POST /api/v1/channels/:id/send-test，body 包含 to、subject、body
- **THEN** 系统 SHALL 通过该通道发送邮件并返回成功结果

#### Scenario: 发送测试邮件到禁用通道
- **WHEN** 管理员请求发送测试邮件到一个 enabled=false 的通道
- **THEN** 系统 SHALL 仍然允许发送（测试不受启用状态限制）

### Requirement: Driver 接口抽象
系统 SHALL 定义 ChannelDriver 接口，包含 Send(config, payload) 和 Test(config) 方法。通过注册表模式按 type 查找 Driver 实例。

#### Scenario: 获取已注册的 Driver
- **WHEN** 系统以 type="email" 查找 Driver
- **THEN** SHALL 返回 EmailDriver 实例

#### Scenario: 获取未注册的 Driver
- **WHEN** 系统以不支持的 type 查找 Driver
- **THEN** SHALL 返回错误 "unsupported channel type: xxx"

### Requirement: EmailDriver 实现
EmailDriver SHALL 使用 SMTP 协议发送邮件，支持 TLS 加密连接。

#### Scenario: 发送 HTML 邮件
- **WHEN** payload 包含 HTML 字段
- **THEN** EmailDriver SHALL 构建 multipart MIME 消息，包含纯文本和 HTML 两个部分

#### Scenario: 发送纯文本邮件
- **WHEN** payload 仅包含 Body 字段（无 HTML）
- **THEN** EmailDriver SHALL 发送纯文本邮件

### Requirement: 通道管理前端页面
系统 SHALL 提供消息通道管理页面，路径为 /channels。

#### Scenario: 查看通道列表
- **WHEN** 管理员访问 /channels 页面
- **THEN** SHALL 展示 DataTable，包含名称、类型（Badge）、启用状态（Switch）、操作按钮

#### Scenario: 新建通道
- **WHEN** 管理员点击"新建通道"按钮
- **THEN** SHALL 弹出 Dialog，选择类型后动态渲染对应的 config 表单字段

#### Scenario: 编辑通道
- **WHEN** 管理员点击某通道的"编辑"按钮
- **THEN** SHALL 弹出 Dialog，类型字段禁用不可更改，password 字段显示占位符

#### Scenario: 测试连接交互
- **WHEN** 管理员在编辑 Dialog 中点击"测试连接"
- **THEN** SHALL 调用 test API 并以 Toast 展示成功或失败结果

#### Scenario: 发送测试邮件交互
- **WHEN** 管理员在编辑 Dialog 中点击"发送测试邮件"
- **THEN** SHALL 弹出输入框要求填写收件人地址，发送后 Toast 反馈结果

### Requirement: 权限控制
通道管理 SHALL 受 Casbin RBAC 保护，默认仅 admin 角色可访问。

#### Scenario: admin 访问通道管理
- **WHEN** admin 角色用户请求通道相关 API
- **THEN** Casbin SHALL 放行

#### Scenario: 普通用户访问通道管理
- **WHEN** user 角色用户请求通道相关 API
- **THEN** Casbin SHALL 拒绝并返回 403
