## Why

Agent LLM calls fail with "未指定模型名称，模型名称不能为空" (400 Bad Request) because the executor is not setting the `Model` field in `llm.ChatRequest`. The `AgentExecuteConfig` only has `ModelID` (uint, the database ID), but not the actual model name (`AIModel.ModelID`) that the LLM API requires.

## What Changes

- Add `ModelName` field to `AgentExecuteConfig` struct in `internal/app/ai/executor.go`
- Populate `ModelName` from `AIModel.ModelID` in `gateway.buildExecuteConfig()`
- Pass `ModelName` to `llm.ChatRequest.Model` in both `executor_react.go` and `executor_plan.go`

## Capabilities

### New Capabilities
<!-- None - this is a bug fix -->

### Modified Capabilities
<!-- None - internal implementation fix, no behavior change -->

## Impact

- Affected files:
  - `internal/app/ai/executor.go` - add ModelName field
  - `internal/app/ai/gateway.go` - populate ModelName
  - `internal/app/ai/executor_react.go` - use ModelName in ChatRequest
  - `internal/app/ai/executor_plan.go` - use ModelName in ChatRequest
- No API changes
- No database migrations needed
