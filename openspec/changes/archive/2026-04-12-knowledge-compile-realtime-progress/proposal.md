## Why

当前知识图谱的编译流程缺乏实时反馈。用户点击"编译"后只能看到"编译中"状态，无法了解具体进度，导致焦虑等待。用户需要手动刷新页面才能看到状态变化，体验很差。需要添加真实的进度追踪，让用户实时了解编译的每个阶段和完成的条目数。

## What Changes

- 后端：在 `KnowledgeCompileService` 中添加细粒度的进度追踪，实时更新到数据库
- 后端：新增 API `GET /api/v1/ai/knowledge-bases/:id/progress` 获取实时进度
- 前端：新增实时进度条组件，显示来源处理、节点创建、索引生成的真实数字进度
- 前端：在知识库详情页添加自动轮询机制，编译期间每 2 秒刷新进度

## Capabilities

### New Capabilities
- `kb-compile-progress`: 知识库编译实时进度追踪。包括编译各阶段（读取来源、调用LLM、写入图谱、生成索引）的实时进度显示，以及每个阶段完成的条目数统计。

### Modified Capabilities
- 无（仅在现有编译流程中插入进度更新，不改变编译逻辑）

## Impact

- **Backend**: `internal/app/ai/knowledge_compile_service.go` - 插入进度更新点
- **Backend**: `internal/app/ai/knowledge_base_handler.go` - 新增进度查询 API
- **Backend**: `internal/app/ai/knowledge_model.go` - 新增进度字段
- **Frontend**: `web/src/apps/ai/pages/knowledge/[id].tsx` - 添加轮询逻辑
- **Frontend**: 新增 `web/src/apps/ai/pages/knowledge/components/compile-progress.tsx`
- **Database**: `knowledge_bases` 表新增 `compile_progress` JSON 字段存储实时进度
