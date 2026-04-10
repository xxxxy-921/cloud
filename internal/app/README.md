# 可插拔 App 架构

## 概念

```
┌──────────────────────────────────┐
│            Kernel（内核）          │  ← 用户/角色/菜单/认证/设置/任务/审计
│         始终存在，不可拔除          │     代码在 internal/ 原位不动
└──────────────┬───────────────────┘
               │
     ┌─────────┼─────────┐
     ▼         ▼         ▼
  App: AI   App: License  ...     ← 可选模块，build tag 控制
```

- **内核** = 现有系统管理功能，代码位置不变
- **App** = 可选业务模块，放在 `internal/app/<name>/` 和 `web/src/apps/<name>/`
- 开发时全功能可用，打包时按需裁剪

## 新建一个 App

### 1. 后端

```go
// internal/app/ai/app.go
package ai

import (
    "github.com/casbin/casbin/v2"
    "github.com/gin-gonic/gin"
    "github.com/samber/do/v2"
    "gorm.io/gorm"

    "metis/internal/app"
    "metis/internal/scheduler"
)

func init() { app.Register(&AIApp{}) }

type AIApp struct{}

func (a *AIApp) Name() string   { return "ai" }
func (a *AIApp) Models() []any  { return []any{&ChatSession{}} }

func (a *AIApp) Seed(db *gorm.DB, enforcer *casbin.Enforcer) error {
    // 注册菜单 + Casbin 策略
    return nil
}

func (a *AIApp) Providers(i do.Injector) {
    do.Provide(i, NewChatService)
}

func (a *AIApp) Routes(api *gin.RouterGroup) {
    g := api.Group("/ai")
    g.POST("/chat", handleChat)
}

func (a *AIApp) Tasks() []scheduler.TaskDef { return nil }
```

### 2. 注册到 edition 文件

```go
// cmd/server/edition_full.go
import _ "metis/internal/app/ai"
```

### 3. 前端

```typescript
// web/src/apps/ai/module.ts
import { lazy } from "react"
import { registerApp } from "@/apps/registry"

registerApp({
  name: "ai",
  routes: [
    {
      path: "ai/chat",
      Component: lazy(() => import("./pages/chat")),
    },
  ],
})
```

然后在 `web/src/apps/registry.ts` 底部加一行：

```typescript
import './ai/module'
```

完成。`make dev` 即可看到新模块。

## 构建

```bash
make build                                      # 全功能（默认）
make build EDITION=edition_lite APPS=system      # 仅内核
make build APPS=system,ai                        # 内核 + AI（前端裁剪，后端需对应 edition）
```

| 参数 | 作用 | 默认值 |
|------|------|--------|
| `EDITION` | Go build tag，控制后端编译哪些 App | 空（全部编译） |
| `APPS` | 前端模块列表，控制 registry.ts 生成 | 空（全量 registry） |

### 自定义 edition

如果需要新的版本组合（如 system + ai），创建新的 edition 文件：

```go
// cmd/server/edition_system_ai.go
//go:build edition_system_ai

package main

import _ "metis/internal/app/ai"
// license 不导入
```

然后 `make build EDITION=edition_system_ai APPS=system,ai`。

## App 接口

```go
type App interface {
    Name() string                                        // 唯一标识
    Models() []any                                       // GORM AutoMigrate
    Seed(db *gorm.DB, enforcer *casbin.Enforcer) error   // 菜单 + 策略
    Providers(i do.Injector)                             // IOC 注册
    Routes(api *gin.RouterGroup)                         // 路由（已带 JWT+Casbin 中间件）
    Tasks() []scheduler.TaskDef                          // 定时任务，无则返回 nil
}
```

启动时 main.go 对每个注册的 App 依次调用：`Models → Providers → Seed → Routes → Tasks`。

## App 引用内核服务

App 可以通过 IOC 容器直接引用内核的 service：

```go
func NewChatService(i do.Injector) (*ChatService, error) {
    userSvc := do.MustInvoke[*service.UserService](i)
    return &ChatService{userSvc: userSvc}, nil
}
```

## 文件结构

```
internal/app/
  app.go                    # App 接口 + Register/All
  ai/                       # 示例：AI App
    app.go                  #   实现 App 接口 + init() 注册
    handler.go
    service.go
    model.go

cmd/server/
  main.go                   # 内核 + App 引导循环
  edition_full.go           # 默认全功能 (//go:build !edition_lite)
  edition_lite.go           # 精简版 (//go:build edition_lite)

web/src/apps/
  registry.ts               # registerApp / getAppRoutes（git 中为全量版）
  ai/                       # 示例：AI App 前端
    module.ts               #   调用 registerApp() 注册路由
    pages/

scripts/
  gen-registry.sh           # 构建时根据 APPS 生成 registry.ts
```
