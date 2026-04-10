## 1. Go 模块和依赖初始化

- [x] 1.1 执行 `go get` 安装核心依赖：gin, gorm, glebarez/sqlite, samber/do/v2
- [x] 1.2 创建目录结构：`cmd/server/`、`internal/{database,model,repository,service,handler,middleware}`

## 2. 基础模型层

- [x] 2.1 创建 `internal/model/base.go`：定义 BaseModel（ID uint PK, CreatedAt, UpdatedAt, DeletedAt）及 JSON tags
- [x] 2.2 创建 `internal/model/system_config.go`：定义 SystemConfig 结构体（Key PK, Value, Remark, CreatedAt, UpdatedAt）

## 3. 数据库初始化

- [x] 3.1 创建 `internal/database/database.go`：GORM 初始化函数，环境变量驱动 SQLite/PostgreSQL 切换，注册为 samber/do Provider
- [x] 3.2 在数据库初始化中实现 AutoMigrate，注册所有模型
- [x] 3.3 实现 Shutdown 接口，关闭数据库连接

## 4. SystemConfig 数据访问层

- [x] 4.1 创建 `internal/repository/system_config.go`：SysConfigRepo 结构体，注册为 samber/do Provider
- [x] 4.2 实现方法：Get(key)、Set(key, value, remark)、List()、Delete(key)

## 5. SystemConfig 业务层

- [x] 5.1 创建 `internal/service/system_config.go`：SysConfigService 结构体，注册为 samber/do Provider
- [x] 5.2 实现业务方法，包裹 repository 调用

## 6. 中间件

- [x] 6.1 创建 `internal/middleware/logger.go`：基于 slog 的请求日志中间件（method, path, status, latency）
- [x] 6.2 创建 `internal/middleware/recovery.go`：panic 恢复中间件，记录错误并返回 500

## 7. HTTP Handler 层

- [x] 7.1 创建 `internal/handler/handler.go`：Handler 聚合结构体，注册为 samber/do Provider，Register 方法绑定路由到 Gin
- [x] 7.2 创建 `internal/handler/system_config.go`：实现 GET /api/v1/config（列表）、GET /api/v1/config/:key（查询）、PUT /api/v1/config（创建/更新）、DELETE /api/v1/config/:key（删除）
- [x] 7.3 统一 API 响应格式（成功/错误 JSON 结构）

## 8. 前端静态资源嵌入

- [x] 8.1 创建 `embed.go`：`//go:embed web/dist/*` 嵌入前端构建产物，导出 embed.FS
- [x] 8.2 在 Handler 中注册静态文件服务：从嵌入 FS 服务静态文件，SPA fallback 到 index.html，API 路由优先
- [x] 8.3 修改 `web/vite.config.ts`：添加 `server.proxy` 配置，`/api` → `http://localhost:8080`

## 9. 应用启动入口

- [x] 9.1 创建 `cmd/server/main.go`：初始化 samber/do 容器，注册所有 Provider，创建 Gin 引擎，注册中间件和路由
- [x] 9.2 实现优雅退出：`injector.ShutdownOnSignals(SIGTERM, SIGINT)`
- [x] 9.3 支持 `SERVER_PORT` 环境变量配置监听端口，默认 8080

## 10. 构建和开发命令

- [x] 10.1 更新 Makefile：添加 `dev`（启动 Go server）、`build`（前端构建 + Go 编译单二进制）、`run`（运行编译后的二进制）命令
- [x] 10.2 更新 `.gitignore`：添加 `*.db`、`web/dist/`、编译产物等
