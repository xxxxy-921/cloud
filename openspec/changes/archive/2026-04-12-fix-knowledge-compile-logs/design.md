## Context

`knowledge_compile_service.go` 中 `HandleCompile` 函数处理知识图谱编译流程。当前存在两处日志缺失：

1. **第234-239行**：`writeCompileOutput` 返回错误时，仅更新 KB 状态后返回，未输出控制台日志
2. **第192-197行、208-213行**：`s.logRepo.Create()` 调用未检查错误，写入失败被静默忽略

## Goals / Non-Goals

**Goals:**
- 在 `writeCompileOutput` 失败时输出结构化错误日志（含 kb_id 和错误详情）
- 捕获 `KnowledgeLog` 创建失败的错误并输出警告日志

**Non-Goals:**
- 不修改业务逻辑
- 不修改 API 接口
- 不添加新的日志存储机制

## Decisions

### 日志级别选择
- `writeCompileOutput` 失败：使用 `slog.Error`（错误级别，需要运维关注）
- `KnowledgeLog` 创建失败：使用 `slog.Error`（虽然是辅助功能，但丢失审计日志属于严重问题）

### 日志字段规范
- 统一使用 `"kb_id"` 字段标识知识库
- 错误信息直接使用 `err.Error()`

## Risks / Trade-offs

无显著风险。本变更仅增加日志输出，不改变现有控制流。
