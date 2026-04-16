## ADDED Requirements

### Requirement: Event 节点圆形渲染
系统 SHALL 将 Start 节点渲染为细线圆形（绿色 #22c55e），End 节点渲染为粗线圆形（红色 #ef4444），Timer 节点渲染为双线圆形+时钟图标（靛蓝 #6366f1），Signal 节点渲染为双线圆形+闪电图标（靛蓝 #6366f1）。

#### Scenario: Start 节点渲染
- **WHEN** 画布上存在 nodeType="start" 的节点
- **THEN** 渲染为细线圆形，填充色为绿色半透明，内部显示 Play 图标，仅有 source handle（底部）

#### Scenario: End 节点渲染
- **WHEN** 画布上存在 nodeType="end" 的节点
- **THEN** 渲染为粗线（3px）圆形，填充色为红色半透明，内部显示 Square 图标，仅有 target handle（顶部）

#### Scenario: Timer 节点渲染
- **WHEN** 画布上存在 nodeType="timer" 的节点
- **THEN** 渲染为双线圆形，内部显示时钟图标，同时具有 target handle（顶部）和 source handle（底部）

#### Scenario: Signal 节点渲染
- **WHEN** 画布上存在 nodeType="signal" 的节点
- **THEN** 渲染为双线圆形，内部显示闪电图标，同时具有 target handle（顶部）和 source handle（底部）

### Requirement: Task 节点卡片式渲染
系统 SHALL 将 Task 类节点渲染为差异化圆角矩形卡片，顶部显示图标+标题行，下方嵌入摘要信息。各类型颜色：Form(蓝 #3b82f6), Approve(琥珀 #f59e0b), Process(紫 #8b5cf6), Action(青 #06b6d4), Script(灰蓝 #64748b), Notify(粉 #ec4899)。

#### Scenario: Form 节点显示表单摘要
- **WHEN** 画布上存在 nodeType="form" 且绑定了 FormDefinition 的节点
- **THEN** 卡片下方显示表单名称和字段数量摘要（如 "用户申请表 · 8 字段"）

#### Scenario: Approve 节点显示参与人
- **WHEN** 画布上存在 nodeType="approve" 且配置了 participants 的节点
- **THEN** 卡片下方显示首个参与人名称 + 剩余数量（如 "张三 +2"），并显示审批模式 badge（单签/会签/依次）

#### Scenario: Action 节点显示动作摘要
- **WHEN** 画布上存在 nodeType="action" 且配置了 actionId 的节点
- **THEN** 卡片下方显示关联的 ServiceAction 名称和 HTTP 方法（如 "创建工单 · POST"）

#### Scenario: Notify 节点显示通道类型
- **WHEN** 画布上存在 nodeType="notify" 且配置了 channelType 的节点
- **THEN** 卡片下方显示通知通道类型（如 "邮件" 或 "站内信"）

#### Scenario: 未配置的 Task 节点
- **WHEN** Task 类节点未配置必要属性（如无 participants、无 formSchema）
- **THEN** 卡片下方显示灰色提示文字（如 "未配置参与人"）

### Requirement: Gateway 节点符号化渲染
系统 SHALL 将 Gateway 节点渲染为菱形+内部符号：Exclusive(✕), Parallel(✛), Inclusive(○)。菱形下方显示标签文字。

#### Scenario: Exclusive 网关渲染
- **WHEN** 画布上存在 nodeType="exclusive" 的节点
- **THEN** 渲染为橙色菱形，内部显示 ✕ 符号，下方显示标签

#### Scenario: Parallel 网关渲染
- **WHEN** 画布上存在 nodeType="parallel" 的节点
- **THEN** 渲染为蓝绿菱形，内部显示 ✛ 符号，具有多个 source handle

#### Scenario: Inclusive 网关渲染
- **WHEN** 画布上存在 nodeType="inclusive" 的节点
- **THEN** 渲染为黄色菱形，内部显示 ○ 符号

### Requirement: Subprocess 节点渲染
系统 SHALL 将 Subprocess 节点渲染为粗边框（3px）矩形，底部显示 [+] 折叠展开按钮。

#### Scenario: Subprocess 折叠态
- **WHEN** subprocess 节点处于折叠状态
- **THEN** 显示粗边框矩形 + 标题 + 底部 [+] 按钮，点击 [+] 展开显示子流程缩略图

#### Scenario: Subprocess 展开态
- **WHEN** subprocess 节点处于展开状态
- **THEN** 矩形尺寸扩大，内部显示子流程节点缩略图（只读），底部 [-] 按钮可收起

### Requirement: 选中节点视觉反馈
系统 SHALL 在节点被选中时显示 primary 颜色的 ring 高亮。

#### Scenario: 单击选中节点
- **WHEN** 用户单击任意节点
- **THEN** 该节点显示 2px primary ring，其余节点取消 ring

### Requirement: 自定义 Edge 渲染
系统 SHALL 使用自定义 Edge 组件渲染连线，支持在路径上显示条件摘要或 outcome 标签。

#### Scenario: Gateway 出边显示条件
- **WHEN** exclusive gateway 的出边配置了 condition
- **THEN** 连线中点显示条件摘要 badge（如 "优先级 = 高"）

#### Scenario: Approve 出边显示 outcome
- **WHEN** approve 节点的出边配置了 outcome="approved" 或 "rejected"
- **THEN** 连线中点显示绿色 "approved" 或红色 "rejected" badge

#### Scenario: Default 边标记
- **WHEN** 出边标记为 isDefault=true
- **THEN** 连线中点显示斜杠标记（/）表示默认路径
