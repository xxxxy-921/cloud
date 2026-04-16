## ADDED Requirements

### Requirement: extractJSON 单元测试覆盖
`extractJSON` 函数 SHALL 有以下场景的单元测试覆盖，确保从各种 LLM 输出格式中正确提取 JSON。

#### Scenario: 裸 JSON 提取
- **WHEN** LLM 输出是纯 JSON 字符串 `{"nodes":[],"edges":[]}`
- **THEN** `extractJSON` 返回等价的 `json.RawMessage` 且无 error

#### Scenario: Markdown 代码块提取
- **WHEN** LLM 输出包裹在 ` ```json ... ``` ` 代码块中
- **THEN** `extractJSON` 正确提取代码块内的 JSON 且无 error

#### Scenario: 文字包裹 JSON 提取
- **WHEN** LLM 输出为 `Here is the workflow: {"nodes":...,"edges":...} Hope this helps!`
- **THEN** `extractJSON` 通过 first `{` to last `}` 策略提取 JSON 且无 error

#### Scenario: 无效输入返回 error
- **WHEN** LLM 输出不包含任何合法 JSON（纯文字）
- **THEN** `extractJSON` 返回 non-nil error

### Requirement: buildUserMessage 单元测试覆盖
`buildUserMessage` 函数 SHALL 有测试覆盖，确保 prompt 构建逻辑正确组装协作规范、动作上下文和重试错误反馈。

#### Scenario: 基础 prompt 构建
- **WHEN** 输入仅包含 collaborationSpec，无 actionsContext，无 prevErrors
- **THEN** 输出包含协作规范文本，不包含"可用动作"和"上一次生成"字样

#### Scenario: 带 actions 上下文的 prompt
- **WHEN** 输入包含 collaborationSpec 和 actionsContext
- **THEN** 输出同时包含协作规范和动作列表

#### Scenario: 带重试错误反馈的 prompt
- **WHEN** 输入包含 prevErrors（ValidationError 列表）
- **THEN** 输出包含"上一次生成的工作流存在以下问题"字样和具体错误信息

### Requirement: buildActionsContext 单元测试覆盖
`buildActionsContext` 函数 SHALL 有测试验证 action 列表格式化输出。

#### Scenario: 多个 action 格式化
- **WHEN** 输入包含多个 ServiceAction（含 Name、Code、Description）
- **THEN** 输出包含每个 action 的名称、code 和描述，格式为 markdown 列表

### Requirement: LLM 集成测试验证结构化提取
系统 SHALL 提供通过环境变量门控的 LLM 集成测试，验证 LLM 能将自然语言协作规范转换为合法工作流 JSON。

#### Scenario: 环境变量未设置时跳过
- **WHEN** `LLM_TEST_BASE_URL`、`LLM_TEST_API_KEY`、`LLM_TEST_MODEL` 中任一未设置
- **THEN** 测试 `t.Skip` 而非 fail

#### Scenario: 简单线性工作流提取
- **WHEN** 协作规范为简单线性流程（开始 → 提交表单 → 审批 → 结束）
- **THEN** LLM 响应经 `extractJSON` 提取后为合法 JSON
- **AND** 经 `ValidateWorkflow` 校验后无 error 级别错误
- **AND** 包含恰好 1 个 start 节点和至少 1 个 end 节点

#### Scenario: 分支工作流提取
- **WHEN** 协作规范包含条件分支（如：审批通过走 A 路径，拒绝走 B 路径）
- **THEN** LLM 响应经提取和校验后包含 exclusive 网关节点
- **AND** exclusive 节点至少有 2 条出边
- **AND** 无 error 级别校验错误

### Requirement: 测试凭据安全隔离
测试 LLM 凭据 SHALL 存放在 `.env.test` 文件中，且该文件 MUST 被 `.gitignore` 排除。

#### Scenario: .env.test 被 gitignore
- **WHEN** `.gitignore` 包含 `.env.test` 条目
- **THEN** `git status` 不会将 `.env.test` 显示为 untracked 文件

### Requirement: Makefile 集成测试入口
Makefile SHALL 提供 `test-llm` target，从 `.env.test` 加载凭据并运行 LLM 集成测试。

#### Scenario: make test-llm 运行 LLM 测试
- **WHEN** 执行 `make test-llm` 且 `.env.test` 存在且配置正确
- **THEN** 仅运行 `TestLLM*` 前缀的测试，timeout 为 120s

### Requirement: BDD 基础设施占位
系统 SHALL 提供 BDD 测试基础设施文件，包含空框架和标记，为后续 BDD 用例做准备。

#### Scenario: BDD 文件存在但无实际用例
- **WHEN** 查看 BDD 测试文件
- **THEN** 文件存在，包含 package 声明和必要 import，但无 Test 函数（或仅一个空的示例框架）
