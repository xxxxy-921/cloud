# Repository Instructions

- 默认用中文与用户交流。
- `AGENTS.md` 是指向 `CLAUDE.md` 的软链接；更新说明时改 `CLAUDE.md`，不要删除或重建链接。
- `openspec/` 是规格工件，`support-files/refer/` 是参考代码；除非用户明确要求，否则不要修改。
- 做 UI 改动先读 `DESIGN.md`；其中的工作台基线和组件约定已经落地，稳定变化后再同步更新它。

## Commands

- 前端只用 Bun，目录在 `web/`；安装依赖用 `make web-install`，不要改用 `npm`/`pnpm`/`yarn`。
- 本地联调优先用 `make dev`：它运行 `cmd/dev`，自动找空闲 `3000+/8000+` 端口对，启动 `go run -tags dev ./cmd/server` 和 `bun run dev`，并把 Vite `/api` 代理到本次后端端口。
- 根目录有 `.env.dev` 时，`make dev` 会先跑 `make seed-dev`；`make seed-dev` 需要 `.env.dev`，会生成或复用 `config.yml`、初始化 DB、执行 App Seed，并补齐开发用 AI/ITSM 配置。
- 只跑前端用 `make web-dev`；默认代理到 `http://localhost:8080`，需要别的后端地址时用 `VITE_API_TARGET=http://localhost:<port> make web-dev`。
- 快速验证常用命令：`go build -tags dev ./cmd/server`、`make build-sidecar`、`cd web && bun run lint`、`cd web && bun run build`。
- 前端测试存在且走 Bun，例如 `cd web && bun test src/components/chat-workspace/message-timeline.test.ts`；`web/tsconfig.app.json` 会排除 `src/**/*.test.*`，所以 `bun run build` 不会帮你检查测试文件。
- 后端常规测试用 `go test ./...`；聚焦包或单测直接用 `go test ./path/to/pkg -run TestName`。ITSM 的 `make test-llm`、`make test-bdd`、`make test-bdd-vpn` 都要求根目录 `.env.test`。

## Architecture

- 这是 Go 后端 + Vite/React 前端的单仓库：后端入口 `cmd/server/main.go`，前端入口 `web/src/main.tsx`；生产构建把 `web/dist` 嵌进 Go 二进制，开发态由 `cmd/dev` 同时拉起前后端。
- 可插拔 App 的真实接口以 `internal/app/app.go` 为准；当前 `Seed` 签名是 `Seed(db, enforcer, install bool)`，不要照抄 `internal/app/README.md` 的旧说明。
- 新增 App 至少同步三处：`internal/app/<name>/`、`cmd/server/edition_*.go`、`web/src/apps/<name>/module.ts`；前端是否生效还取决于 `web/src/apps/_bootstrap.ts` 的副作用导入。
- `cmd/server/edition_full.go` 的空白导入顺序会影响 `app.Register()` 顺序，进而影响启动时的 `Models -> Providers -> Seed -> Routes -> Tasks` 顺序；有跨 App 依赖时不要随意重排。
- `handler.Register()` 返回的 `/api/v1` 路由组已经串好 `JWT -> PasswordExpiry -> Casbin -> DataScope -> Audit`；App 路由默认挂在这条鉴权链后面。

## Gotchas

- `scripts/gen-registry.sh` 会重写 `web/src/apps/_bootstrap.ts`；过滤构建时还会重写 `web/tsconfig.app.json`。新增或删除前端 App 时同时更新脚本里的 `ALL_APPS`；如果脚本或过滤构建留下脏改动，用 `make web-full-registry` 恢复全量状态。
- 后端响应统一是 `R{code,message,data}`，不是裸 JSON；前端优先复用 `web/src/lib/api.ts`，它已经处理了 token 刷新和并发 401 排队。
- 运行时配置主要来自 `config.yml` 和数据库里的 `SystemConfig`，不是通用 `.env`；`.env.dev` 只给 dev bootstrap 用，`.env.test` 只给 LLM/BDD 测试用。
- 某个 API 如果只要求“已登录”而不要求细粒度权限，除了注册路由，还要同步更新 `internal/middleware/casbin.go` 的 `casbinWhitelist`、前缀白名单或 `KeyMatch` 规则，否则会被 Casbin 拦住。
- React Compiler 已在 `web/vite.config.ts` 开启；前端改动避免破坏 hooks 顺序、避免在 effect 里同步 `setState`、避免在 render 中读写 `ref.current`。
- `DESIGN.md` 里的后台工作台模式是当前基线；最容易猜错的一条是新建/编辑表单优先用 `Sheet`，不要新开 `Dialog` 模式。
