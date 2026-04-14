## ADDED Requirements

### Requirement: Service name search
拓扑页面 SHALL 提供搜索框，实时过滤服务节点。

#### Scenario: Type search query
- **WHEN** 用户输入 "pay"
- **THEN** 名称包含 "pay" 的节点正常显示，其余节点降低透明度至 0.15

#### Scenario: Clear search
- **WHEN** 用户清空搜索框
- **THEN** 所有节点恢复正常透明度

### Requirement: Error-only filter toggle
拓扑页面 SHALL 提供"仅错误"开关，一键筛出 errorRate > 0 的服务。

#### Scenario: Enable error-only filter
- **WHEN** 用户开启"仅错误" toggle
- **THEN** 仅 errorRate > 0 的服务节点正常显示，其余降低透明度

#### Scenario: Combined search and error filter
- **WHEN** 搜索框有值且"仅错误"开启
- **THEN** 同时满足两个条件的节点正常显示

### Requirement: Edge opacity follows node filter
当节点被过滤降低透明度时，与其关联的边 SHALL 同步降低透明度。

#### Scenario: Filtered node edges
- **WHEN** 某节点因搜索/过滤被降低透明度
- **THEN** 连接该节点的边也降低透明度至 0.15
- **AND** 两端均为匹配节点的边保持正常透明度
