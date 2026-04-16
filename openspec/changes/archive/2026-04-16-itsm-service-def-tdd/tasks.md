## 1. 基础设施

- [x] 1.1 `.gitignore` 添加 `.env.test` 条目
- [x] 1.2 创建 `.env.test.example` 示例文件（含占位符，可提交）
- [x] 1.3 `Makefile` 添加 `test-llm` target

## 2. 单元测试 (Layer 1)

- [x] 2.1 创建 `workflow_generate_test.go`，编写 `TestExtractJSON_BareJSON`
- [x] 2.2 编写 `TestExtractJSON_MarkdownCodeBlock`
- [x] 2.3 编写 `TestExtractJSON_TextWrapped`
- [x] 2.4 编写 `TestExtractJSON_Invalid`
- [x] 2.5 编写 `TestBuildUserMessage_Basic`
- [x] 2.6 编写 `TestBuildUserMessage_WithActions`
- [x] 2.7 编写 `TestBuildUserMessage_WithPrevErrors`
- [x] 2.8 编写 `TestBuildActionsContext`
- [x] 2.9 运行单元测试确认全部通过

## 3. LLM 集成测试 (Layer 2)

- [x] 3.1 编写 `requireLLMEnv` 辅助函数（环境变量门控 + t.Skip）
- [x] 3.2 编写 `TestLLMExtract_SimpleWorkflow`（线性流程：开始→表单→审批→结束）
- [x] 3.3 编写 `TestLLMExtract_BranchWorkflow`（分支流程：含排他网关）
- [x] 3.4 使用 `.env.test` 配置运行集成测试确认通过

## 4. BDD 基础设施

- [x] 4.1 创建 `workflow_generate_bdd_test.go` 空框架（package 声明 + import + 注释说明）
