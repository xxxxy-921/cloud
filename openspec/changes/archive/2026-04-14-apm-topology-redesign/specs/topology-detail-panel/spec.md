## ADDED Requirements

### Requirement: Right-side detail panel on node click
点击拓扑节点 SHALL 在右侧弹出 380px 宽的详情面板，拓扑图保持可见并自适应剩余宽度。

#### Scenario: Click node opens panel
- **WHEN** 用户点击服务节点
- **THEN** 右侧面板滑入，显示该服务的详情；拓扑图区域缩窄但内容自适应

#### Scenario: Click another node switches panel
- **WHEN** 面板已打开，用户点击另一个节点
- **THEN** 面板内容切换为新节点的服务详情

#### Scenario: Close panel
- **WHEN** 用户点击面板关闭按钮或按 Escape
- **THEN** 面板收起，拓扑图恢复全宽

### Requirement: Panel shows service key metrics
面板 SHALL 展示选中服务的关键指标卡片：Request Count、Avg Latency、P95、Error Rate。

#### Scenario: Service with normal metrics
- **WHEN** 面板显示一个 healthy 服务
- **THEN** 4 个指标卡片均正常显示，errorRate 用绿色

#### Scenario: Service with high error rate
- **WHEN** 面板显示一个 errorRate > 5% 的服务
- **THEN** errorRate 指标卡片用红色高亮

### Requirement: Panel shows operations list
面板 SHALL 展示该服务的操作列表（spanName、requestCount、avgDuration、errorRate），点击操作行跳转到 Trace Explorer 并预填 service + operation 过滤。

#### Scenario: Click operation row
- **WHEN** 用户点击操作列表中的某一行
- **THEN** 导航到 `/apm/traces?service=X&operation=Y&start=...&end=...`

### Requirement: Panel has navigation link
面板底部 SHALL 有"查看完整详情"按钮，跳转到 `/apm/services/:name`。

#### Scenario: Click view details
- **WHEN** 用户点击"查看完整详情"
- **THEN** 导航到该服务的完整 Service Detail 页面，携带当前时间范围参数
