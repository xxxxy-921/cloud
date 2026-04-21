## 1. 后端 — 公共工具与参数校验 (B1)

- [x] 1.1 在 `internal/handler/` 新增 `ParseUintParam(c *gin.Context, name string) (uint, bool)` 工具函数，解析失败写 400 并返回 false
- [x] 1.2 替换 `provider_handler.go` 中所有 `strconv.Atoi(c.Param("id"))` 为 `handler.ParseUintParam`
- [x] 1.3 替换 `model_handler.go` 中所有 `strconv.Atoi(c.Param("id"))` 为 `handler.ParseUintParam`

## 2. 后端 — Provider Update 条件性重置 status (B2)

- [x] 2.1 修改 `ProviderService.Update`：对比旧 BaseURL / APIKey 是否变化，仅变化时重置 status 为 inactive
- [x] 2.2 补充单元测试 `TestProviderService_Update_PreservesStatusOnNameChange`

## 3. 后端 — SetDefault 作用域 + 事务 (B3)

- [x] 3.1 在 `model_repository.go` 新增 `ClearDefaultByProviderAndType(tx *gorm.DB, providerID uint, modelType string)` 方法
- [x] 3.2 在 `model_repository.go` 新增 `SetDefaultInTx(tx *gorm.DB, id uint)` 方法
- [x] 3.3 重写 `ModelService.SetDefault` 使用事务包裹，调用新的 provider 维度清除方法
- [x] 3.4 保留 `ClearDefaultByType` 和 `FindDefaultByType` 不变（下游 knowledge_compile_service 依赖）
- [x] 3.5 补充单元测试 `TestModelService_SetDefault_ScopedToProvider` — 验证跨 provider 默认不被清除

## 4. 后端 — 静默错误与日志 (B4)

- [x] 4.1 在 `ProviderHandler.List` 中为 `ModelCountsForProviders` 和 `TypeCountsForProviders` 失败时添加 `slog.Warn` 日志

## 5. 后端 — Anthropic 测试连接动态模型 (B5)

- [x] 5.1 修改 `ProviderHandler.TestConnection`：Anthropic 分支先查该 provider 下已同步的第一个 active 模型 ID，无则 fallback 到硬编码值
- [x] 5.2 将 `testAnthropicConnection` 签名改为接收 `modelID string` 参数

## 6. 后端 — guessModelType 兜底类型 (B6)

- [x] 6.1 在 `model.go` 新增 `ModelTypeOther = "other"` 常量，加入 `ValidModelTypes`
- [x] 6.2 修改 `guessModelType` 未匹配时返回 `ModelTypeOther`

## 7. 后端 — LIKE 转义 (B7)

- [x] 7.1 在 `provider_repository.go` 和 `model_repository.go` 的 List 方法中，对 keyword 进行 `%` 和 `_` 转义后再拼 LIKE

## 8. 前端 — 模型列表 pageSize (F1)

- [x] 8.1 详情页 `[id].tsx` 中模型查询 `pageSize=100` 改为 `pageSize=500`；`TYPE_ORDER` 增加 `"other"` 支持新模型类型

## 9. 前端 — IIFE 重构为组件 (F2)

- [x] 9.1 从 `[id].tsx` 提取 `ModelTypePanel` 为独立组件（同文件内）
- [x] 9.2 将分页状态、搜索状态、权限标志、mutation 回调作为 props 传入

## 10. 前端 — 空状态文案区分 (F3)

- [x] 10.1 修改 `[id].tsx` 中模型面板空状态，区分"无数据"和"搜索无结果"使用不同文案
- [x] 10.2 在 `locales/zh-CN.json` 和 `locales/en.json` 补充 `models.emptySearch` 翻译 key

## 11. 前端 — 相对时间 i18n (F6)

- [x] 11.1 修改 `provider-card.tsx` 的 `formatRelativeTime`，使用 i18n 的 `t()` 输出时间后缀
- [x] 11.2 在 locales 中补充 `providers.timeAgo.justNow` / `minutes` / `hours` / `days` 翻译 key

## 12. 前端 — 详情页返回导航 (F7)

- [x] 12.1 在 `[id].tsx` 的 `Component` 中，`ProviderInfoSection` 上方添加面包屑式返回链接（`← 供应商列表 / 供应商名`）
