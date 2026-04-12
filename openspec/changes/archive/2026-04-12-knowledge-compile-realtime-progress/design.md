## Context

当前知识库编译流程 (`KnowledgeCompileService.HandleCompile`) 是一个长时间运行的异步任务，包含多个阶段：
1. 准备来源文档
2. 调用 LLM 分析（30-60秒）
3. 解析并写入知识图谱节点
4. 生成向量索引
5. 质量检查

用户点击"编译"后，只能看到 `compile_status = "compiling"`，无法了解具体进行到哪一步，已完成多少条目。这导致用户体验差，需要手动刷新页面查看状态变化。

## Goals / Non-Goals

**Goals:**
- 提供编译各阶段的实时进度（来源读取、节点写入、索引生成）
- 显示每个阶段完成的实际条目数（如"已创建 23/78 个节点"）
- 前端自动轮询更新，无需用户手动刷新
- 进度数据持久化到数据库，页面刷新后仍可看到当前进度

**Non-Goals:**
- 不使用 WebSocket 或 SSE（保持简单轮询方案）
- 不改变编译逻辑，只添加进度追踪
- 不提供预估完成时间

## Decisions

### 1. 进度数据存储在 JSON 字段
- **决策**: 使用 `knowledge_bases.compile_progress` JSON 字段存储完整进度信息
- **理由**: 避免添加多个零散字段，JSON 可以灵活包含各阶段信息
- **替代方案**: 单独字段 `sources_done`, `nodes_done` 等 - 过于僵化

### 2. 轮询而非 SSE
- **决策**: 前端使用 2 秒间隔的短轮询
- **理由**: 实现简单，编译任务不是高频场景，2 秒延迟可接受
- **替代方案**: SSE 实时推送 - 增加架构复杂度，收益有限

### 3. 在 Service 层更新进度
- **决策**: 在 `KnowledgeCompileService` 的每个阶段主动更新进度
- **理由**: 编译逻辑集中在此处，可以最准确地知道当前进度
- **注意**: 需要控制更新频率，避免过于频繁的数据库写入

### 4. 进度数据结构
```json
{
  "stage": "writing_nodes",
  "sources": {"total": 5, "done": 5},
  "nodes": {"total": 78, "done": 23},
  "embeddings": {"total": 78, "done": 0},
  "current_item": "正在创建节点: Claude API"
}
```

## Risks / Trade-offs

- **[Risk]** 频繁更新进度可能导致数据库写入压力
  - **Mitigation**: 只在关键节点更新（如每完成 5% 的节点），不是每个条目都写库

- **[Risk]** LLM 调用阶段无法提供进度（原子操作）
  - **Mitigation**: 该阶段显示"AI 分析中，请稍候..."，让用户知道系统在工作

- **[Risk]** 并发编译时进度数据竞争
  - **Mitigation**: 每个 KB 同一时间只能有一个编译任务运行（由 scheduler 保证）

## Migration Plan

1. 数据库迁移：添加 `compile_progress` JSON 字段
2. 后端修改：在编译服务中插入进度更新
3. 后端新增：进度查询 API
4. 前端新增：进度组件和轮询逻辑
5. 部署：无中断部署，新字段有默认值

## Open Questions

- 进度更新频率如何设置最合理？（暂定每 5% 或每 5 秒更新一次）
