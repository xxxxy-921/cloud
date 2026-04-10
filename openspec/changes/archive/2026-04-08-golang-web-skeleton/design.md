## Context

Metis 是一个全新项目，当前只有前端（Vite 8 + React 19 + TypeScript 6 + React Compiler，使用 bun 管理）。Go 后端从零开始搭建，`go.mod` 已初始化为 `module metis`，Go 1.24.3。

前端构建产物需要嵌入 Go 二进制实现单文件部署。开发模式下前端通过 Vite proxy 连接 Go Server，两者独立运行。

用户明确要求：不允许过度设计，非必要不引入。`refer/` 目录是用户的代码参考，不可修改。

## Goals / Non-Goals

**Goals:**
- 建立清晰的分层目录结构（cmd / internal / model / repository / service / handler / middleware）
- GORM + 纯 Go SQLite 驱动（无 CGO），支持环境变量切换 PostgreSQL
- samber/do v2 作为 IOC 容器，管理依赖注入和优雅退出
- slog（标准库）作为日志方案
- Gin 中间件链作为 AOP 实现（请求日志、异常恢复）
- K/V 系统配置表（SystemConfig）作为第一个业务表
- `//go:embed` 嵌入前端 dist，单二进制运行
- Vite proxy 开发模式
- Makefile 统一构建命令

**Non-Goals:**
- 权限/认证体系（明确排除）
- 微服务架构、gRPC、消息队列
- 配置文件系统（首阶段用环境变量 + K/V 表）
- 前端路由/页面开发（只处理嵌入和代理）
- 单元测试框架搭建（骨架先行）
- Docker / CI/CD 配置

## Decisions

### D1: Web 框架 → Gin

**选择**: `github.com/gin-gonic/gin`
**替代方案**: Echo（生态略小）、Chi（需要自己组合 binding/validation）、stdlib net/http（缺少中间件生态）
**理由**: 2026 年 Go Web 框架事实标准，生态最完整，中间件链天然实现 AOP，内置 request binding 和 validation，学习成本最低。

### D2: ORM → GORM + 纯 Go SQLite

**选择**: `gorm.io/gorm` + `github.com/glebarez/sqlite`（无 CGO）
**替代方案**: Ent（代码生成过重）、Bun（生态较小）、SQLC（SQL-first 不适合快速原型）
**理由**: GORM 是 Go ORM 事实标准，AutoMigrate 对骨架期友好。纯 Go SQLite 驱动免去 CGO 交叉编译问题，未来切 PostgreSQL 只需换 dialector。

### D3: 日志 → slog（标准库）

**选择**: `log/slog`（Go 1.21+ 标准库）
**替代方案**: zap（性能更好但外部依赖）、zerolog（同理）
**理由**: 零外部依赖，结构化日志，完美符合"非必要不引入"原则。以后需要 JSON 输出或接 ELK 直接换 Handler。

### D4: IOC → samber/do v2

**选择**: `github.com/samber/do/v2`
**替代方案**: google/wire（需要代码生成工具）、uber/fx（过重）、手动 DI（无优雅退出和 HealthCheck）
**理由**: 轻量泛型 DI 容器，惰性初始化，内置 `ShutdownOnSignals` 优雅退出，`Override` 方便测试 mock，几乎零学习成本。

### D5: AOP → Gin 中间件

**选择**: 不引入独立 AOP 框架，使用 Gin 中间件链
**理由**: Go 没有原生 AOP，Web 场景下中间件链（日志、Recovery、RequestID）就是 AOP 的最佳实践。引入专门的 AOP 库是过度设计。

### D6: 静态资源嵌入 → //go:embed

**选择**: Go 标准库 `embed` 包
**理由**: 标准库方案，零依赖。`web/dist/` 构建产物通过 `//go:embed` 指令嵌入到二进制。生产模式从内嵌 FS 服务静态文件，开发模式走 Vite proxy。

### D7: 数据库切换策略 → 环境变量

**选择**: `DB_DRIVER` + `DB_DSN` 环境变量控制
**理由**: 零配置默认 SQLite，设环境变量切 PostgreSQL。不引入配置文件解析库（如 viper），配置统一走 K/V 系统表 + 环境变量。

## Risks / Trade-offs

- **[纯 Go SQLite 性能低于 CGO 版]** → 对于骨架和中小规模场景完全够用；高并发场景已有 PostgreSQL 切换路径
- **[GORM AutoMigrate 不适合生产]** → 首阶段可用，后续引入 migration 工具（goose/atlas）时再替换
- **[samber/do 相对小众]** → API 稳定（v2 SemVer），最坏情况可退回手动 DI，成本极低
- **[SQLite WAL 模式并发限制]** → 单写多读足够骨架阶段，PostgreSQL 是逃生通道
