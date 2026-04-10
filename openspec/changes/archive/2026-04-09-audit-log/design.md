## Context

Metis 当前缺乏审计追踪能力。现有的请求日志中间件 (`middleware/logger.go`) 仅记录 HTTP 请求的 method/path/status/latency，不包含用户身份和操作语义。当需要排查安全事件或追溯操作历史时，无法获取有效信息。

参考 NekoAdmin 的双表设计（auth_log + audit_log），本次将两者统一为单表，通过 `category` 字段区分，在前端用 Tab 分开展示。

现有基础设施：
- IOC 容器 (samber/do) 用于依赖注入
- GORM + SQLite 数据层，BaseModel 提供通用字段
- Casbin RBAC + JWT 中间件提供身份上下文
- Scheduler 引擎支持定时任务注册
- SystemConfig 表 + SettingsService 提供 K/V 配置
- 前端 useListPage hook 封装分页 + React Query

## Goals / Non-Goals

**Goals:**
- 统一审计日志表，支持 auth / operation / application 三种 category
- auth 类事件自动记录（登录成功/失败/登出）
- operation 类事件半自动记录（Handler 声明 + 审计中间件收集）
- application 类提供 `AuditService.Log()` 接入点，模型预留但本期不做前端 Tab
- 按 category 独立配置日志保留天数，定时清理
- 前端只读查看，支持按 category 筛选和分页

**Non-Goals:**
- 不做操作前后的 diff 对比（summary 文本足够）
- 不做实时推送 / WebSocket 日志流
- 不做日志导出（CSV/Excel）
- 不做 application Tab 的前端展示（模型预留）
- 不做审计日志的编辑/删除 API（只读 + 定时清理）

## Decisions

### D1: 单表设计 vs 分表

**选择**: 单表 `audit_logs`，通过 `category` 字段区分类型。

**替代方案**: 两张表（auth_log + audit_log），如 NekoAdmin 所做。

**理由**:
- SQLite 环境下 JOIN 开销大，单表查询更高效
- 字段差异小（auth 多 user_agent/reason，operation 多 resource_id），用 nullable 字段即可
- 统一模型简化整个层次（一套 model/repo/service/handler）
- `category + created_at` 联合索引保证 Tab 查询性能
- 未来 application 类事件零成本接入

### D2: 不嵌入 BaseModel

**选择**: AuditLog 不嵌入 BaseModel，自定义 ID + CreatedAt，无 UpdatedAt / DeletedAt。

**理由**: 审计日志是 append-only 的，不需要更新和软删除。清理任务直接硬删除过期记录。

### D3: Operation 类捕获方式 — Handler 声明 + 中间件收集

**选择**: Handler 通过 `c.Set("audit_*")` 声明审计元数据，审计中间件在 `c.Next()` 后读取并异步写入。

**替代方案 A**: 纯中间件自动记录所有写操作 — 粒度太粗，不知道操作语义。
**替代方案 B**: Service 层手动调用 — 太分散，容易遗漏。

**理由**:
- Handler 知道操作语义（"创建用户 xxx"），可以提供精确的 summary
- 中间件统一收集，避免每个 handler 都写 auditService.Log() 调用
- 仅 2xx 响应才记录，失败操作自动跳过
- 异步写入（goroutine），不阻塞响应

**约定的 context key**:
- `audit_action`: string — 操作标识，如 "user.create"
- `audit_resource`: string — 资源类型，如 "user"
- `audit_resource_id`: string — 资源 ID（可选）
- `audit_summary`: string — 人类可读摘要

### D4: Auth 类捕获方式 — Auth handler 直接调用

**选择**: 在 auth handler 的登录/登出/密码修改等方法中直接调用 `AuditService.Log()`。

**理由**: Auth 事件的上下文（成功/失败、用户名、reason）只有 auth handler 知道。不适合用通用中间件。

### D5: Detail JSON 字段

**选择**: `detail` 字段为 nullable text，存 JSON 字符串。

**理由**: 预留扩展能力，不同 category 的额外数据差异较大。前端第一期不解析 detail，留给未来展开行或详情面板。

### D6: Level 字段

**选择**: 增加 `level` 字段 (info/warn/error)，默认 info。

**理由**: 当前 auth 和 operation 基本都是 info，但 application 类（任务失败 = error，登录失败可标为 warn）需要 level 区分。第一期前端不基于 level 筛选，但模型预留。

### D7: 保留策略按 category 分开

**选择**: SystemConfig 中存三个 key，分别配置各 category 的保留天数。

- `audit.retention_days_auth` = 90（默认）
- `audit.retention_days_operation` = 365（默认）
- `audit.retention_days_application` = 30（默认）

清理任务每天凌晨 3 点执行，按各 category 独立清理。

### D8: 审计中间件的注册范围

**选择**: 审计中间件只注册在需要审计的路由组上（写操作路由），不全局注册。

**理由**: GET 请求不需要审计（只读操作），只有 POST/PUT/DELETE 的管理操作需要。Handler 通过是否设置 `audit_action` 来决定是否产生审计记录，所以即使全局注册也不会有误记录，但限制范围可以减少不必要的 context 检查开销。

## Risks / Trade-offs

- **[异步写入丢失]** → goroutine 写入如果服务突然崩溃，最后一条日志可能丢失。可接受，审计日志不是事务级保证。
- **[SQLite 写入竞争]** → 高并发下审计日志写入可能与业务写入竞争 SQLite 写锁。SQLite WAL 模式下读写分离，单次 INSERT 很快，风险较低。
- **[Summary 手动维护]** → 每个 handler 需要手动构建 summary 文本，可能遗漏或不一致。通过代码审查和约定规范来缓解。
- **[单表数据量]** → 长期运行后 audit_logs 表可能较大。定时清理 + 合理索引缓解。SQLite 单表百万级记录仍然高效。
