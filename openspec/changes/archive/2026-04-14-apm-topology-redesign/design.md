## Context

当前服务拓扑是 apm-pro-upgrade 中基于 `@xyflow/react` + `dagre` 的实现。矩形卡片节点、Bezier 边、点击跳转。需要升级为 Datadog 风格：圆形图标节点、侧板交互、多维着色、搜索过滤。

现有文件结构：
- `components/topology/service-node.tsx` — 节点组件
- `components/topology/service-edge.tsx` — 边组件
- `components/topology/edge-tooltip.tsx` — 边 tooltip（可删除）
- `components/service-map.tsx` — 主地图组件
- `pages/topology/index.tsx` — 页面

后端 API 无需改动，复用：`fetchTopology`（nodes + edges）、`fetchServiceDetail`（指标 + 操作列表）、`fetchTimeseries`（图表数据）。

## Goals / Non-Goals

**Goals:**
- 圆形图标节点 + 健康环，视觉上达到 Datadog Service Map 水准
- 点击节点弹出右侧详情面板（保持拓扑上下文），含关键指标和操作列表
- 边极简化：移除常驻标签，hover 展示详情
- 支持 Color by 切换（Error Rate / Latency / Throughput）
- 支持服务名搜索 + 仅错误过滤

**Non-Goals:**
- 不做标签分组/聚类布局（当前服务数量不需要）
- 不做实时 WebSocket 推送（继续用 polling/手动刷新）
- 不改后端 API

## Decisions

### D1: 圆形节点用 SVG 还是 HTML div

**选择：HTML div + CSS**
- 圆形用 `rounded-full` + 固定宽高即可
- 图标用 lucide-react，已在项目中
- 健康环用 `ring-*` + `border-*` 组合
- 替代方案：SVG 自绘 → 更灵活但复杂度高，不值得

### D2: 侧板实现方式

**选择：页面内右侧面板（fixed width），不用 Sheet 组件**
- Sheet（抽屉）会遮挡拓扑图，违背"保持上下文"原则
- 用 `flex` 布局：左侧拓扑图（`flex-1`）+ 右侧面板（`w-[380px]`，有节点选中时显示）
- 面板内容复用 `fetchServiceDetail` + `fetchTimeseries` 的数据
- 面板底部放"查看详情"按钮跳转到完整 Service Detail 页

### D3: Color by 状态管理

**选择：页面级 state，通过 props 传递到节点/边**
- `colorMode: "errorRate" | "latency" | "throughput"` state 在 topology page
- 传给 ServiceMap → 节点/边组件根据 colorMode 决定颜色映射
- 不需要全局 store，仅限拓扑页面使用

### D4: 搜索过滤实现

**选择：前端过滤，不匹配节点降低 opacity**
- 不从 ReactFlow 中移除节点（保持拓扑完整性）
- 非匹配节点 `opacity: 0.15`，匹配节点正常显示
- 非匹配边同步降低 opacity
- 用 `useMemo` 计算过滤后的 node/edge 样式

### D5: 节点标签位置

**选择：名称在圆形下方，不在内部**
- 圆形直径 56px，内部仅放图标
- 服务名在下方居中，`text-[11px]`
- 减少节点视觉重量，降低噪音

## Risks / Trade-offs

- **[圆形节点间距]** 圆形比矩形占空间少但 dagre 仍按矩形算间距 → 增大 `nodesep`/`ranksep` 补偿
- **[侧板挤压拓扑图]** 面板出现时拓扑区域缩窄 → ReactFlow `fitView` 会自动适应，加 transition 过渡
- **[Color by 数据来源]** Latency/Throughput 的数据在 `TopologyNode` 中目前没有 → 需要从 `fetchTopology` 的 nodes 或额外的 service list 获取。当前 `TopologyNode` 只有 `requestCount` 和 `errorRate`。对于 latency，可以从 services API 获取或者在拓扑 API 中增加字段。**暂定：在前端额外 fetch service list 补充 latency 数据**
