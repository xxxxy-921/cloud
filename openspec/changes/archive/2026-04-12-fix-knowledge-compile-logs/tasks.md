## 1. 添加 writeCompileOutput 错误日志

- [x] 1.1 在 `internal/app/ai/knowledge_compile_service.go` 第234-239行添加 slog.Error 日志
- [x] 1.2 验证日志格式包含 kb_id 和 error 字段

## 2. 检查 KnowledgeLog 创建错误

- [x] 2.1 修改第192-197行的 `s.logRepo.Create` 调用，检查并记录错误
- [x] 2.2 修改第208-213行的 `s.logRepo.Create` 调用，检查并记录错误

## 3. 编译验证

- [x] 3.1 运行 `go build -tags dev ./cmd/server/` 确保无编译错误
