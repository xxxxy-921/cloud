# Capability: app-registry

## Purpose
Defines the App interface and global registry for pluggable application modules, enabling optional business modules to self-register at both backend and frontend layers.

## Requirements

### Requirement: App 接口定义
系统 SHALL 在 `internal/app/app.go` 中定义 `App` 接口，包含以下方法：
- `Name() string` — 返回 App 唯一标识（kebab-case）
- `Models() []any` — 返回需要 AutoMigrate 的 GORM model 列表
- `Seed(db *gorm.DB, enforcer casbin.IEnforcer) error` — 执行 App 的种子数据（菜单、Casbin 策略、默认配置）
- `Providers(i *do.Injector)` — 向 IOC 容器注册 App 的 repository/service/handler
- `Routes(api *gin.RouterGroup)` — 在 `/api/v1` 路由组下注册 App 的 API 路由
- `Tasks() []scheduler.TaskDefinition` — 返回 App 的定时任务定义列表，无任务时返回 nil

#### Scenario: 接口定义可编译
- **WHEN** 开发者引入 `internal/app` 包
- **THEN** App 接口 SHALL 可用，且不依赖任何具体业务模块

### Requirement: 全局 App 注册表
系统 SHALL 提供包级函数 `Register(a App)` 和 `All() []App`，用于 App 的注册和检索。注册表 SHALL 使用包级 slice 变量存储。

#### Scenario: App 通过 init() 自注册
- **WHEN** 一个 App 包被导入（blank import）
- **THEN** 该包的 `init()` 函数 SHALL 调用 `app.Register()` 将自身注册到全局注册表

#### Scenario: 获取所有已注册 App
- **WHEN** main.go 调用 `app.All()`
- **THEN** SHALL 返回所有已注册 App 的列表，顺序为注册顺序

### Requirement: 前端模块注册机制
前端 SHALL 在 `web/src/apps/registry.ts` 中提供 `AppModule` 类型和 `registerApp()`、`getAppRoutes()` 函数。每个 App 模块 SHALL 在 `web/src/apps/<name>/module.ts` 中调用 `registerApp()` 注册自己的路由。

#### Scenario: App 模块注册路由
- **WHEN** 一个 App 模块的 `module.ts` 被导入
- **THEN** 该模块 SHALL 调用 `registerApp()` 注册自己的路由定义（path + lazy component）

#### Scenario: 获取所有 App 路由
- **WHEN** App.tsx 调用 `getAppRoutes()`
- **THEN** SHALL 返回所有已注册模块的路由定义的扁平数组

### Requirement: App 目录结构约定
每个可选 App 的后端代码 SHALL 放在 `internal/app/<name>/` 目录下，前端代码 SHALL 放在 `web/src/apps/<name>/` 目录下。App 包 SHALL 至少包含一个实现 App 接口的类型和一个调用 `app.Register()` 的 `init()` 函数。

#### Scenario: 新建 App 模块
- **WHEN** 开发者在 `internal/app/myapp/` 下创建 `app.go` 并实现 App 接口
- **THEN** 该 App SHALL 可通过 blank import 注册到系统中
