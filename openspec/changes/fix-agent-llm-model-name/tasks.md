## 1. Add ModelName field to AgentExecuteConfig

- [ ] 1.1 Add `ModelName string` field to `AgentExecuteConfig` struct in `internal/app/ai/executor.go`

## 2. Populate ModelName in Gateway

- [ ] 2.1 In `gateway.go`, fetch the AIModel by ID and set `ModelName` in `AgentExecuteConfig`
- [ ] 2.2 Pass the model's `ModelID` (string identifier) as `ModelName`

## 3. Use ModelName in Executors

- [ ] 3.1 In `executor_react.go`, set `chatReq.Model = req.AgentConfig.ModelName`
- [ ] 3.2 In `executor_plan.go`, set `Model` field in both `Chat()` and `ChatStream()` calls

## 4. Verify

- [ ] 4.1 Run `go build -tags dev ./cmd/server/` to verify compilation
- [ ] 4.2 Test agent conversation to confirm LLM calls succeed
