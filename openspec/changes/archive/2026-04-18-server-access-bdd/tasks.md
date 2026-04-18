## 1. Support 基础设施

- [x] 1.1 创建 `server_access_support_test.go`：定义 `serverAccessCollaborationSpec` 常量和 `serverAccessCasePayloads`（ops/network/security/boundary_security 4 组 case payload）
- [x] 1.2 实现 `generateServerAccessWorkflow()`：复用 `generateVPNWorkflow` 模式，替换协作规范为 server access 版本
- [x] 1.3 实现 `publishServerAccessSmartService()`：创建 ServiceCatalog + Priority + Agent + ServiceDefinition（engine_type=smart）

## 2. 通用 Step Definitions（扩展 itsm-bdd-infrastructure）

- [x] 2.1 在 `steps_common_test.go` 新增 step `当前审批分配到岗位 "<position_code>"`：断言当前活动的 TicketAssignment 中 position code 匹配
- [x] 2.2 在 `steps_common_test.go` 新增 step `当前审批仅对 "<username>" 可见`：断言仅指定用户在 assignment 的 position_department 可处理人中
- [x] 2.3 在 `steps_common_test.go` 新增 step `"<username>" 认领当前工单应失败`：尝试认领并断言失败
- [x] 2.4 在 `steps_common_test.go` 新增 step `"<username>" 审批当前工单应失败`：尝试审批并断言失败

## 3. Server Access 专属 Step Definitions

- [x] 3.1 创建 `steps_server_access_test.go`，实现 `registerServerAccessSteps()`
- [x] 3.2 实现 Given step `已定义生产服务器临时访问申请协作规范`
- [x] 3.3 实现 Given step `已基于协作规范发布生产服务器临时访问服务（智能引擎）`
- [x] 3.4 实现 Given step `"<username>" 已创建生产服务器访问工单，场景为 "<case_key>"`：根据 case_key 选择 payload，创建 Ticket

## 4. Feature 文件

- [x] 4.1 创建 `features/server_access_branch_decision.feature`：包含 Background + 5 个 Scenario（ops/network/security/boundary_security/责任边界）

## 5. 注册与验证

- [x] 5.1 在 `bdd_test.go` 的 `initializeScenario` 中注册 `registerServerAccessSteps(sc, bc)`
- [x] 5.2 运行 `go build -tags dev ./cmd/server/` 确认编译通过
- [x] 5.3 运行 `make test-bdd` 验证全部 5 个 scenario 通过
