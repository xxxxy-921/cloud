## Why

Metis 项目需要一个 Go Web 后端骨架。当前仅有前端（Vite 8 + React 19），无任何后端代码。目标是建立一个最小化但生产可用的 Go 骨架，支持单二进制部署（前端静态资源嵌入），开发时前端通过 Vite proxy 连接 Go Server。

## What Changes

- 引入 Go Web 骨架：Gin + GORM + slog + samber/do
- 建立标准分层目录结构：`cmd/server`、`internal/{database,model,repository,service,handler,middleware}`
- GORM 默认使用 SQLite（纯 Go 无 CGO 驱动），支持通过环境变量切换 PostgreSQL
- 定义 BaseModel（ID、时间戳、软删除）作为所有业务表的基础
- 实现 SystemConfig K/V 系统表（增删改查）
- 前端构建产物通过 `//go:embed` 嵌入二进制
- 配置 Vite proxy，开发模式下 `/api/*` 转发到 Go Server
- API 路径统一使用 `/api/v1` 前缀
- 更新 Makefile，支持 `dev`、`build`（单二进制）等命令

## Capabilities

### New Capabilities
- `database`: GORM 数据库初始化，SQLite/PostgreSQL 双驱动切换，AutoMigrate
- `base-model`: BaseModel 定义（ID、CreatedAt、UpdatedAt、DeletedAt），所有业务表的公共基础
- `system-config`: K/V 系统配置表，完整的 CRUD API（/api/v1/config）
- `web-embed`: 前端静态资源 `//go:embed` 嵌入，生产单二进制；开发模式 Vite proxy
- `server-bootstrap`: 应用启动流程，samber/do IOC 容器，Gin 引擎初始化，优雅退出，slog 中间件

### Modified Capabilities

（无已有 spec 需要修改）

## Impact

- **新增依赖**：gin, gorm, glebarez/sqlite, samber/do/v2（gorm/driver/postgres 按需）
- **目录变更**：新增 `cmd/`、`internal/`、`embed.go`
- **构建变更**：Makefile 扩展，新增 Go 编译和单二进制打包命令
- **前端变更**：`vite.config.ts` 添加 proxy 配置
- **go.mod**：从空模块变为包含上述依赖
