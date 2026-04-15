## MODIFIED Requirements

### Requirement: 基础信息 Tab

基础信息 Tab SHALL 展示服务定义的核心属性表单，支持编辑和保存。当服务为 smart 类型时，协作规范区域 SHALL 提供「解析工作流」按钮。

#### Scenario: 表单字段展示
- **WHEN** 管理员进入基础信息 Tab
- **THEN** 系统 SHALL 展示以下字段：服务名称（text）、服务编码（text，创建后只读）、描述（textarea）、所属分类（下拉选择二级分类）、SLA 模板（下拉选择，可选）、引擎类型（下拉选择，创建后只读）、状态（开关）。当 engine_type 为 "smart" 时，额外展示协作规范（CollaborationSpec，多行文本编辑器）

#### Scenario: 保存修改
- **WHEN** 管理员修改表单字段并点击保存
- **THEN** 系统 SHALL 调用 `PUT /api/v1/itsm/services/:id` 保存修改，显示成功提示

#### Scenario: 智能模式协作规范编辑
- **WHEN** 服务为 smart 类型
- **THEN** 系统 SHALL 展示协作规范（CollaborationSpec）多行文本编辑区域，高度至少 8 行

#### Scenario: 解析工作流按钮
- **WHEN** 服务为 smart 类型且协作规范不为空
- **THEN** 系统 SHALL 在协作规范区域下方展示「解析工作流」按钮

#### Scenario: 协作规范为空时禁用解析
- **WHEN** 服务为 smart 类型但协作规范为空
- **THEN** 系统 SHALL 将「解析工作流」按钮置为 disabled 状态，tooltip 提示 "请先填写协作规范"

#### Scenario: 点击解析工作流
- **WHEN** 管理员点击「解析工作流」按钮
- **THEN** 系统 SHALL 调用 `POST /api/v1/itsm/workflows/generate`，传入当前协作规范和已配置的动作列表（从动作 Tab 读取）。按钮进入 loading 状态，显示 "正在解析..."

#### Scenario: 解析成功
- **WHEN** 解析 API 返回成功
- **THEN** 系统 SHALL 将返回的 workflow_json 更新到当前服务定义的 workflowJSON 状态中，自动切换到工作流 Tab 展示结果，显示成功提示 "工作流已生成"

#### Scenario: 解析失败
- **WHEN** 解析 API 返回错误
- **THEN** 系统 SHALL 显示错误提示（包含后端返回的错误信息），按钮恢复可点击状态

#### Scenario: 引擎未配置提示
- **WHEN** 解析 API 返回 400 错误 "工作流解析引擎未配置"
- **THEN** 系统 SHALL 显示提示 "请先前往引擎配置页面配置工作流解析引擎"，提供跳转到 `/itsm/engine-config` 的链接

#### Scenario: 经典模式隐藏智能字段
- **WHEN** 服务为 classic 类型
- **THEN** 系统 SHALL 隐藏协作规范字段和解析按钮

### Requirement: 工作流 Tab 只读查看器

工作流 Tab SHALL 使用 ReactFlow 渲染服务定义的 workflowJSON。Smart 类型服务支持点选节点查看配置详情；Classic 类型和无工作流数据时保持只读。支持平移和缩放。

#### Scenario: 有工作流数据时渲染
- **WHEN** 管理员切换到工作流 Tab，且服务的 workflowJSON 不为空
- **THEN** 系统 SHALL 使用 ReactFlow 渲染工作流图，节点不可拖拽（nodesDraggable=false）、不可连线（nodesConnectable=false），支持画布平移和缩放

#### Scenario: 节点点选查看详情（Smart 模式）
- **WHEN** 管理员在 Smart 类型服务的工作流 Tab 中点击某个节点
- **THEN** 系统 SHALL 在右侧面板展示该节点的配置详情：activity_kind、参与人类型与值、表单字段列表、关联的 action_code（如有）

#### Scenario: 无工作流数据时空状态
- **WHEN** 管理员切换到工作流 Tab，且服务的 workflowJSON 为空或 null
- **THEN** 系统 SHALL 展示空状态提示："尚未生成工作流，请在基础信息 Tab 填写协作规范后点击「解析工作流」"

#### Scenario: 按需加载
- **WHEN** 工作流 Tab 组件加载
- **THEN** 系统 SHALL 通过 React.lazy 动态导入 @xyflow/react，不影响列表页和其他 Tab 的首屏加载
