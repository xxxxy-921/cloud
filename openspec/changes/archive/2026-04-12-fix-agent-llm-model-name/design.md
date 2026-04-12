## Context

This is a bug fix for Agent LLM calls failing with "未指定模型名称" error.

The issue: `executor_react.go` and `executor_plan.go` build `llm.ChatRequest` without setting the `Model` field, causing 400 Bad Request from LLM APIs.

## Goals / Non-Goals

**Goals:**
- Fix the missing model name in LLM API calls
- Pass correct model identifier from agent configuration to LLM client

**Non-Goals:**
- No behavior changes
- No API changes
- No new features

## Decisions

- Add `ModelName` field to `AgentExecuteConfig` to carry the actual model identifier (`AIModel.ModelID`) alongside the database ID (`ModelID`)
- Populate `ModelName` in gateway when building execution config
- Use `ModelName` in both React and Plan-and-Execute executors

## Risks / Trade-offs

None - this is a straightforward bug fix.
