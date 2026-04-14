## Why

当前 APM 服务拓扑图是 apm-pro-upgrade 中快速搭建的矩形卡片 + ReactFlow 方案，视觉效果和交互深度远不及 Datadog 级别。核心问题：errorRate 阈值错误导致全红、矩形节点信息密度过大、点击直接跳转丢失全局上下文、边标签噪音大、缺少搜索/过滤/着色切换等高级功能。

## What Changes

- **圆形图标节点**：替换矩形卡片为圆形节点 + 中心服务类型图标 + 外围健康环 + 下方标签，节点只承载最小信息
- **点击弹侧板**：点击节点不再跳转离开，而是在右侧弹出服务详情面板（关键指标、操作列表、跳转链接），保持拓扑全局上下文
- **边极简化**：移除边上的常驻文字标签，仅保留 hover tooltip 展示 throughput/latency/error；保留流量方向粒子动画
- **Color by 切换器**：支持按 Error Rate / Latency P95 / Throughput 切换节点/边的着色维度
- **搜索过滤**：服务名搜索框 + "仅错误" toggle，非匹配节点降低透明度
- **健康图例**：底部 legend 展示颜色含义，随 color-by 模式切换

## Capabilities

### New Capabilities
- `topology-circular-nodes`: 圆形图标节点组件，按服务类型显示图标、外围健康环、下方标签
- `topology-detail-panel`: 点击节点弹出的右侧详情面板，展示服务关键指标、操作列表、跳转入口
- `topology-color-modes`: 拓扑图多维着色切换器（Error Rate / Latency / Throughput）
- `topology-search-filter`: 拓扑图搜索和过滤功能（服务名搜索、仅错误 toggle、匹配高亮）

### Modified Capabilities

（无已有 spec 变更）

## Impact

- **前端文件**：`components/topology/` 下节点、边、面板组件重写；`components/service-map.tsx` 主组件重构；`pages/topology/index.tsx` 页面布局调整
- **API**：无新增后端 API，复用已有 `fetchTopology`、`fetchServiceDetail`、`fetchTimeseries`
- **依赖**：无新增依赖，继续使用 `@xyflow/react` + `dagre` + `recharts` + `lucide-react`
- **i18n**：`locales/en.json` 和 `zh-CN.json` 新增 topology 相关翻译 key
