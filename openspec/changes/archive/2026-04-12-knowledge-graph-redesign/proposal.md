## Why

当前知识图谱编译算法存在严重质量问题：大量节点只有 title/summary 没有实质 content（空壳节点），边解析时自动创建连 summary 都没有的幽灵节点。根本原因是 prompt 主动鼓励创建空节点（"Create nodes even for concepts that don't have enough content"），且代码在关系目标不存在时自动创建空壳。此外，MAP 阶段对超过 8000 字的来源直接截断，导致长文档（PDF 书籍等）的知识大量丢失。

参考 Karpathy LLM Wiki 模式（2026 年广泛采用的知识编译范式），核心洞察是：每个知识节点应该是一篇完整的、可独立阅读的 Wiki 文章，而非图谱碎片。

## What Changes

- **重写编译 prompt（Map + Reduce）**：强制要求每个节点输出完整文章，禁止 content 为 null，提高最低内容门槛
- **消灭幽灵节点创建逻辑**：边解析时目标节点不存在则跳过，不再自动创建空壳
- **简化边类型**：从 4 种（related/contradicts/extends/part_of）简化为 2 种（related/contradicts）
- **移除 index 节点**：不再在图中生成 index 类型节点，元数据由程序构建
- **长文档三阶段处理（Scan → Gather → Write）**：替代当前 8000 字截断，支持 PDF 书籍等超长来源
- **新增编译配置项**：`targetContentLength`（默认 4000）、`minContentLength`（默认 200）、`maxChunkSize`（默认自动）
- **调整 LLM 输出结构**：`related` 字段简化为 `references`（纯概念名列表）+ 可选 `contradicts` 列表

## Capabilities

### New Capabilities
- `knowledge-compile-long-doc`: 长文档三阶段编译策略（Scan → Gather → Write），支持超出单次 LLM 上下文窗口的来源（PDF 书籍、超长报告）

### Modified Capabilities
- `ai-knowledge`: 编译流程重构 — 节点必须有实质 content、边类型简化为 2 种、移除 index 节点、新增编译配置项

## Impact

- **后端**：`internal/app/ai/knowledge_compile_service.go`（主要改动）、`knowledge_model.go`（CompileConfig、边类型常量）、`knowledge_graph_repository.go`（移除 index 节点相关查询）
- **前端**：`knowledge-graph-view.tsx`（移除 index 节点展示、适配 2 种边类型）、知识库设置面板（新增编译配置项 UI）
- **数据兼容**：已有空壳节点不自动删除，下次重编译时会被有内容的节点覆盖。已有 extends/part_of 边在前端统一显示为 related
- **API**：无破坏性变更，编译配置项通过已有的 PUT /api/v1/ai/knowledge-bases/:id 接口传递
