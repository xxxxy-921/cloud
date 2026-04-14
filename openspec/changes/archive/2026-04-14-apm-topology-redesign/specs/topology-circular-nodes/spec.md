## ADDED Requirements

### Requirement: Circular icon node with health ring
每个服务节点 SHALL 渲染为圆形（56px 直径），中心显示服务类型图标（lucide-react），外围显示健康色环。服务名称 SHALL 显示在圆形下方居中。

#### Scenario: Healthy service node
- **WHEN** 节点 errorRate < 1%
- **THEN** 外围环为绿色（emerald），图标背景为绿色淡色

#### Scenario: Warning service node
- **WHEN** 节点 errorRate 在 1% - 5% 之间
- **THEN** 外围环为黄色（amber），图标背景为黄色淡色

#### Scenario: Critical service node
- **WHEN** 节点 errorRate > 5%
- **THEN** 外围环为红色（red），图标背景为红色淡色

### Requirement: Service type icon mapping
节点 SHALL 根据服务名推断图标类型。

#### Scenario: Gateway service
- **WHEN** 服务名匹配 gateway/api-gw/nginx/ingress/proxy
- **THEN** 显示 Globe 图标

#### Scenario: Payment service
- **WHEN** 服务名匹配 pay/billing/stripe/checkout
- **THEN** 显示 CreditCard 图标

#### Scenario: Default service
- **WHEN** 服务名不匹配任何已知模式
- **THEN** 显示 Server 图标

### Requirement: Node label below circle
服务名 SHALL 显示在圆形节点下方，text-[11px]，最大宽度截断。节点内部 SHALL 不显示任何文字。

#### Scenario: Long service name
- **WHEN** 服务名超过 20 字符
- **THEN** 文字截断并显示省略号

### Requirement: Selected node visual feedback
选中的节点 SHALL 有明显的视觉区分。

#### Scenario: Node selected
- **WHEN** 用户点击某节点
- **THEN** 该节点显示 primary 色 ring，其余节点保持正常
