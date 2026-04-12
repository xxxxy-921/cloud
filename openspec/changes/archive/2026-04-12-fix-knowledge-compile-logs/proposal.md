## Why

知识图谱编译过程中，当 `writeCompileOutput` 步骤失败时（如 FalkorDB 写入错误），控制台没有输出错误日志，导致运维人员无法排查问题。此外，`KnowledgeLog` 写入失败也被静默忽略，进一步增加了调试难度。

## What Changes

- 在 `writeCompileOutput` 失败处添加 `slog.Error` 日志，包含知识库ID和错误详情
- 检查 `KnowledgeLog` 创建操作的错误，失败时输出警告日志
- 保持现有行为不变，仅增加日志输出，**非破坏性变更**

## Capabilities

### New Capabilities
- （无，本变更仅为日志增强）

### Modified Capabilities
- （无，本变更不涉及 spec 级别的需求变更，仅为实现层面的日志补充）

## Impact

**受影响文件：**
- `internal/app/ai/knowledge_compile_service.go`

**无 API 变更、无数据库变更、无配置变更**
