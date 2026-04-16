## ADDED Requirements

### Requirement: 参与人类型选择
系统 SHALL 提供统一的参与人选择器组件，支持 user（搜索用户）、position（选择岗位）、department（选择部门）、requester_manager（固定选项）四种参与人类型。

#### Scenario: 添加用户类型参与人
- **WHEN** 用户选择参与人类型为 "user" 并输入搜索关键字
- **THEN** 系统调用 `GET /api/v1/users?keyword=xxx` 展示匹配的用户列表，用户点击后添加到参与人列表，显示用户名和头像

#### Scenario: 添加岗位类型参与人
- **WHEN** 用户选择参与人类型为 "position"
- **THEN** 系统调用 `GET /api/v1/org/positions` 展示岗位下拉列表，选择后添加到参与人列表，显示岗位名称

#### Scenario: 添加部门类型参与人
- **WHEN** 用户选择参与人类型为 "department"
- **THEN** 系统调用 `GET /api/v1/org/departments/tree` 展示部门树，选择后添加到参与人列表，显示部门名称

#### Scenario: 添加申请人上级参与人
- **WHEN** 用户选择参与人类型为 "requester_manager"
- **THEN** 系统直接添加一条 "申请人上级" 参与人记录（无需额外选择），运行时自动解析

### Requirement: 参与人列表管理
系统 SHALL 在属性面板中显示已配置的参与人列表，支持添加、删除、排序操作。

#### Scenario: 显示已有参与人
- **WHEN** 节点已配置 participants 数组
- **THEN** 面板显示参与人列表，每项显示类型图标 + 名称，支持拖拽排序

#### Scenario: 删除参与人
- **WHEN** 用户点击参与人列表项的删除按钮
- **THEN** 从 participants 数组中移除该项，节点数据同步更新

#### Scenario: 多参与人节点摘要
- **WHEN** 节点配置了多个参与人
- **THEN** 节点卡片摘要显示首个参与人名称 + 剩余数量（如 "张三 +2 人"）

### Requirement: 参与人选择器在多种节点类型中复用
系统 SHALL 在 form、approve、process 三种节点类型的属性面板中复用同一参与人选择器组件。

#### Scenario: Form 节点配置参与人
- **WHEN** 用户选中 form 节点并打开属性面板
- **THEN** 面板中显示参与人选择器区域

#### Scenario: Approve 节点配置参与人
- **WHEN** 用户选中 approve 节点并打开属性面板
- **THEN** 面板中显示参与人选择器区域，并在其下方显示审批模式选择（单签/会签/依次）
