## ADDED Requirements

### Requirement: 自动布局
系统 SHALL 提供一键自动排版功能，使用 dagre 算法将节点按 TB（上到下）方向自动排列。

#### Scenario: 点击自动布局按钮
- **WHEN** 用户点击工具栏中的 "自动布局" 按钮
- **THEN** 系统使用 dagre 计算所有节点的最优位置，并动画过渡到新位置

#### Scenario: 不同节点尺寸的布局
- **WHEN** 画布中存在不同类型的节点（event 圆形较小、task 卡片较大、gateway 菱形中等）
- **THEN** dagre 布局为每种节点类型使用预设的宽高，确保节点间距合理

#### Scenario: 布局后可手动微调
- **WHEN** 自动布局完成后
- **THEN** 用户仍可自由拖拽节点调整位置

### Requirement: Undo/Redo
系统 SHALL 支持 Ctrl+Z 撤销和 Ctrl+Shift+Z 重做操作。

#### Scenario: 撤销添加节点
- **WHEN** 用户拖入一个新节点后按 Ctrl+Z
- **THEN** 新节点被移除，画布恢复到添加前的状态

#### Scenario: 撤销删除节点
- **WHEN** 用户删除一个节点后按 Ctrl+Z
- **THEN** 被删除的节点及其关联边恢复

#### Scenario: 重做已撤销操作
- **WHEN** 用户撤销后按 Ctrl+Shift+Z
- **THEN** 之前撤销的操作重新应用

#### Scenario: 连续拖拽合并
- **WHEN** 用户快速连续拖拽同一节点
- **THEN** 300ms 内的拖拽操作合并为一次 undo 记录

### Requirement: Copy/Paste
系统 SHALL 支持 Ctrl+C / Ctrl+V 复制粘贴选中的节点和边。

#### Scenario: 复制单个节点
- **WHEN** 用户选中一个节点并按 Ctrl+C，然后按 Ctrl+V
- **THEN** 在原节点右下方偏移位置创建一个副本，生成新的 nodeId，保留原始 data

#### Scenario: 复制多个节点及连线
- **WHEN** 用户框选多个节点和连线后按 Ctrl+C，然后按 Ctrl+V
- **THEN** 所有选中的节点和它们之间的连线被复制，新节点生成新 ID，连线关系保持一致

#### Scenario: Start/End 节点不可复制
- **WHEN** 用户选中 start 或 end 节点并尝试复制
- **THEN** start 和 end 节点不被包含在复制结果中（流程只能有一个 start 和一个 end）

### Requirement: 右键上下文菜单
系统 SHALL 在节点、边、画布空白处右键时显示上下文菜单。

#### Scenario: 节点右键菜单
- **WHEN** 用户右键点击一个非 start/end 节点
- **THEN** 显示菜单项：复制、删除、查看属性

#### Scenario: 画布右键菜单
- **WHEN** 用户右键点击画布空白处
- **THEN** 显示菜单项：粘贴（如果剪贴板有内容）、自动布局、全选

#### Scenario: 边右键菜单
- **WHEN** 用户右键点击一条连线
- **THEN** 显示菜单项：编辑条件（如果源为 gateway）、删除

### Requirement: 键盘快捷键
系统 SHALL 支持常用键盘快捷键。

#### Scenario: Delete/Backspace 删除选中
- **WHEN** 用户选中节点或边后按 Delete 或 Backspace
- **THEN** 选中的元素（非 start/end 节点）被删除

#### Scenario: Ctrl+A 全选
- **WHEN** 用户按 Ctrl+A
- **THEN** 画布中所有节点和边被选中

#### Scenario: 方向键微调
- **WHEN** 用户选中节点后按方向键
- **THEN** 选中的节点向对应方向移动 10px（按住 Shift 移动 1px 精细调整）

### Requirement: 校验错误高亮
系统 SHALL 在校验发现错误时，在对应节点上显示红色边框和悬浮提示气泡。

#### Scenario: 节点校验错误高亮
- **WHEN** validationErrors 中包含 nodeId 指向某节点的错误
- **THEN** 该节点边框变为红色（destructive），悬浮时显示错误信息 tooltip

#### Scenario: 边校验错误高亮
- **WHEN** validationErrors 中包含 edgeId 指向某边的错误
- **THEN** 该连线颜色变为红色，悬浮时显示错误信息 tooltip

#### Scenario: 错误列表面板移除
- **WHEN** 校验高亮功能启用后
- **THEN** 原有的底部左侧错误列表面板移除，改为节点/边上的内联高亮
