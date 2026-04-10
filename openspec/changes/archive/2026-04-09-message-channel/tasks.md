## 1. 数据模型与 Driver 抽象

- [x] 1.1 创建 `internal/model/message_channel.go`：MessageChannel 结构体（嵌入 BaseModel）、MessageChannelResponse（脱敏响应）、ToResponse 方法
- [x] 1.2 在 `internal/database/database.go` 的 AutoMigrate 中注册 MessageChannel 模型
- [x] 1.3 创建 `internal/channel/driver.go`：定义 Payload 结构体、Driver 接口（Send/Test）、GetDriver 注册表函数
- [x] 1.4 创建 `internal/channel/email.go`：EmailDriver 实现，使用 net/smtp + crypto/tls 发送邮件和测试连接

## 2. 后端 Repository + Service + Handler

- [x] 2.1 创建 `internal/repository/message_channel.go`：CRUD 操作（Create/FindByID/List/Update/Delete/ToggleEnabled），列表查询支持分页和关键字搜索，返回脱敏 config
- [x] 2.2 创建 `internal/service/message_channel.go`：业务逻辑层，封装 CRUD + TestChannel + SendTest，更新时处理密码脱敏值保留逻辑
- [x] 2.3 创建 `internal/handler/message_channel.go`：HTTP handler，注册路由 `/api/v1/channels/*`
- [x] 2.4 更新 `internal/handler/handler.go`：Handler 结构体新增 channel 字段，New 函数注入依赖，Register 注册路由
- [x] 2.5 更新 `cmd/server/main.go`：IOC 容器注册 MessageChannel 的 repo/service provider

## 3. 种子数据（菜单 + Casbin 策略）

- [x] 3.1 更新 `internal/seed/menus.go`：在系统管理下新增"消息通道"菜单及按钮权限（list/create/update/delete）
- [x] 3.2 更新 `internal/seed/policies.go`：AdminAPIPolicies 追加 channels 相关 API 策略

## 4. 前端页面

- [x] 4.1 创建 `web/src/pages/channels/index.tsx`：通道列表页，DataTable 展示，Switch 切换启用状态
- [x] 4.2 创建 `web/src/pages/channels/channel-form-dialog.tsx`：新建/编辑 Dialog，类型选择后动态渲染 config 表单
- [x] 4.3 创建 `web/src/pages/channels/channel-types.ts`：前端通道类型元数据定义（CHANNEL_TYPES），邮件类型的 config schema
- [x] 4.4 创建 `web/src/pages/channels/send-test-dialog.tsx`：发送测试邮件对话框

## 5. 前端路由与导航

- [x] 5.1 更新前端路由配置：新增 `/channels` 路由，lazy 加载，PermissionGuard 检查 `system:channel:list`
- [x] 5.2 更新 `web/src/lib/nav/` 导航配置：系统管理下新增"消息通道"导航项（由 seed menus 驱动，已在 3.1 完成）
