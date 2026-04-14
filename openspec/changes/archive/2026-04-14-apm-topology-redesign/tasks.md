## 1. 圆形图标节点

- [x] 1.1 重写 `service-node.tsx`：圆形节点（56px）+ 中心图标 + 外围健康色环（绿/黄/红按 errorRate 阈值）+ 下方服务名标签
- [x] 1.2 实现服务类型图标映射：gateway→Globe, payment→CreditCard, inventory→Package, notification→Bell, db→Database, worker→Cpu, 默认→Server
- [x] 1.3 选中态：选中节点显示 primary ring，区别于健康环
- [x] 1.4 更新 `service-map.tsx` 中 `NODE_WIDTH`/`NODE_HEIGHT` 和 dagre 布局参数适配圆形节点

## 2. 边极简化

- [x] 2.1 重写 `service-edge.tsx`：移除常驻文字标签，仅保留流量方向粒子动画 + hover tooltip
- [x] 2.2 hover tooltip 展示 caller→callee、throughput、avg/p95 latency、error rate
- [x] 2.3 边颜色/粗细根据当前 color-by 模式映射
- [x] 2.4 删除不再使用的 `edge-tooltip.tsx`

## 3. 右侧详情面板

- [x] 3.1 新建 `components/topology/detail-panel.tsx`：右侧 380px 面板组件，含关闭按钮和 Escape 关闭
- [x] 3.2 面板顶部：服务名 + 健康 badge + 图标
- [x] 3.3 面板指标卡：4 个 metric cards（Request Count、Avg Latency、P95、Error Rate），数据来自 `fetchServiceDetail`
- [x] 3.4 面板操作列表：表格（spanName、requestCount、avgDuration、errorRate），行可点击跳转到 traces
- [x] 3.5 面板底部："查看完整详情"按钮，跳转 `/apm/services/:name?start=...&end=...`
- [x] 3.6 改造 `pages/topology/index.tsx`：flex 布局，左侧拓扑图（flex-1）+ 右侧面板（选中时显示），`selectedNode` state
- [x] 3.7 改造 `service-map.tsx`：`onNodeClick` 不再 navigate，改为调用 `onSelectNode` 回调

## 4. Color by 切换器

- [x] 4.1 新建 `components/topology/color-mode-select.tsx`：下拉切换器组件（Error Rate / Latency P95 / Throughput）
- [x] 4.2 `pages/topology/index.tsx` 添加 `colorMode` state，传递给 ServiceMap → 节点/边
- [x] 4.3 节点组件根据 colorMode 切换颜色映射逻辑：errorRate 模式用阈值分级，latency 用动态 percentile，throughput 用渐变
- [x] 4.4 边组件同步根据 colorMode 调整颜色
- [x] 4.5 获取 latency 数据：topology page 额外 fetch services list 补充 avgDuration/p95 数据，合并到 node data
- [x] 4.6 底部 Legend 随 colorMode 切换更新标签和色阶

## 5. 搜索过滤

- [x] 5.1 新建 `components/topology/topology-toolbar.tsx`：搜索框 + "仅错误" Switch，放在页面标题栏右侧
- [x] 5.2 `pages/topology/index.tsx` 添加 `searchQuery` 和 `errorOnly` state
- [x] 5.3 实现过滤逻辑：非匹配节点 `opacity: 0.15`，两端均不匹配的边同步降低 opacity
- [x] 5.4 将过滤状态传入 ServiceMap，通过 node/edge className 或 style 控制 opacity

## 6. i18n + 清理

- [x] 6.1 更新 `locales/en.json` 和 `zh-CN.json`：新增 topology detail panel、color mode、search filter 相关翻译 key
- [x] 6.2 删除废弃的 `edge-tooltip.tsx`
- [x] 6.3 运行 `bun run lint` 确认 APM 文件无错误
