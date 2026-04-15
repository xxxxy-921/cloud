## 1. AI Agent 模型扩展

- [x] 1.1 Agent 模型新增 `code` 字段（string，unique，可为空）和 `type` 字段扩展支持 `internal` 值，运行 AutoMigrate
- [x] 1.2 Agent Repository 新增 `GetByCode(code string) (*Agent, error)` 方法
- [x] 1.3 Agent Service / Handler 修改：创建 internal Agent 时校验 code 必填且唯一；列表 API 默认排除 internal 类型（除非显式 `type=internal`）
- [x] 1.4 验证现有 Agent 功能不受影响（assistant/coding 创建、列表、更新、删除）

## 2. ITSM 引擎配置后端

- [x] 2.1 在 `internal/app/itsm/` 下新增 `engine_config_handler.go` 和 `engine_config_service.go`
- [x] 2.2 实现 `GET /api/v1/itsm/engine/config`：读取 `itsm.generator` 和 `itsm.runtime` Agent（通过 code 查找），读取 `itsm.engine.*` SystemConfig，组装为聚合 JSON 响应
- [x] 2.3 实现 `PUT /api/v1/itsm/engine/config`：解析请求体，分别更新两个 Agent 的 model_id + temperature，更新 SystemConfig 的 decision_mode、max_retries、timeout_seconds、reasoning_log
- [x] 2.4 在 ITSM App 的 `Routes()` 中注册引擎配置路由（挂载到已有的 JWT+Casbin 中间件链下）

## 3. ITSM Seed 引擎默认配置

- [x] 3.1 ITSM Seed 新增 `itsm.generator` internal Agent 创建（code、name、type=internal、temperature=0.3、system_prompt=内置解析提示词）
- [x] 3.2 ITSM Seed 新增 `itsm.runtime` internal Agent 创建（code、name、type=internal、temperature=0.1、system_prompt=内置决策提示词）
- [x] 3.3 ITSM Seed 新增 SystemConfig 默认值写入（decision_mode、max_retries、timeout_seconds、reasoning_log），使用 idempotent 模式
- [x] 3.4 ITSM Seed 注册「引擎配置」菜单项（路由 `/itsm/engine-config`）和 Casbin 权限策略

## 4. 工作流解析后端

- [x] 4.1 在 `internal/app/itsm/` 下新增 `workflow_generate_handler.go` 和 `workflow_generate_service.go`
- [x] 4.2 实现 `POST /api/v1/itsm/workflows/generate`：读取 `itsm.generator` Agent → 构建 LLM Client → 组装 prompt（system_prompt + 协作规范 + 可用动作）→ 调用 LLM
- [x] 4.3 实现 workflow_json 结构校验（nodes/edges 字段完整性、activity_kind 枚举值）
- [x] 4.4 实现 workflow_json 拓扑校验（唯一起始/结束节点、路径连通性、无孤立节点）
- [x] 4.5 实现重试逻辑：LLM 失败或校验失败时重试（读取 max_retries 配置），校验失败时将错误注入下次 prompt
- [x] 4.6 编写内置解析约束提示词（Generator System Prompt）：定义 JSON 输出格式、节点类型约束、参与人类型约束等
- [x] 4.7 在 ITSM App 的 `Routes()` 中注册工作流解析路由

## 5. ITSM 引擎配置前端

- [x] 5.1 新增 `web/src/apps/itsm/pages/engine-config/` 页面组件，三个配置区块（解析引擎、决策引擎、通用设置）
- [x] 5.2 实现 Provider 下拉 → Model 下拉联动（调用 AI 模块 Provider list + Model list API）
- [x] 5.3 实现决策模式 Radio 组件（direct_first / ai_only）和推理日志 Select 组件（full / summary / off）
- [x] 5.4 实现保存逻辑（PUT 聚合 API）和加载逻辑（GET 聚合 API）
- [x] 5.5 处理未配置 Provider 的引导状态（空 Provider 列表时展示跳转提示）
- [x] 5.6 在 ITSM 前端模块注册引擎配置路由和侧边栏菜单项

## 6. 服务定义 UI 增强

- [x] 6.1 基础信息 Tab：协作规范区域下方新增「解析工作流」按钮，绑定 `POST /api/v1/itsm/workflows/generate` 调用
- [x] 6.2 实现按钮状态管理：协作规范为空时 disabled、解析中 loading、引擎未配置时提示跳转
- [x] 6.3 解析成功后将 workflow_json 写入服务定义状态，自动切换到工作流 Tab
- [x] 6.4 工作流 Tab：升级为支持节点点选，右侧面板展示节点配置详情（activity_kind、参与人、表单字段、action_code）
- [x] 6.5 工作流 Tab：无数据时展示引导文案，引导用户回到基础信息 Tab 填写协作规范
