## Why

系统需要通过外部通道（邮件、企业微信等）向用户发送消息通知的能力。当前只有站内通知（Notification），缺少向外部推送的手段。本次先支持邮件通道，为后续扩展其他通道类型（企业微信、钉钉等）建立 Driver 抽象基础。

## What Changes

- 新增 `message_channel` 数据表，存储通道配置（名称、类型、JSON 配置、启用状态）
- 新增 ChannelDriver 接口抽象，实现 EmailDriver（SMTP 发送 + 连接测试）
- 新增后端 CRUD API：创建、查询、更新、删除、切换启用、测试连接
- 新增前端管理页面：通道列表、新建/编辑表单、测试连接交互
- Config 不加密，明文 JSON 存储
- 不与现有 Notification 系统联动

## Capabilities

### New Capabilities
- `message-channel`: 消息通道管理，包括 CRUD、Driver 抽象、邮件发送与连接测试

### Modified Capabilities
- `nav-modules`: 需要在系统管理菜单下新增"消息通道"菜单项

## Impact

- **后端新增文件**: model、repository、service、handler 各一个
- **前端新增页面**: `web/src/pages/channels/` 列表 + 表单页
- **路由注册**: 新增 `/api/v1/channels/*` 路由组
- **数据库**: AutoMigrate 新增 MessageChannel 模型
- **IOC 容器**: main.go 注册新的 repo/service/handler provider
- **依赖**: Go 侧需要 SMTP 库（`net/smtp` 或 `gomail`）
- **Casbin**: 新增 message_channel 资源的 RBAC 策略
- **种子数据**: seed 中追加菜单项和 Casbin 策略
