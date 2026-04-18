## 1. 共享步骤提升

- [x] 1.1 将 `whenSmartEngineDecisionCycleUntilComplete` 方法及其 step 注册从 `steps_db_backup_test.go` 移动到 `steps_common_test.go`，在 `registerCommonSteps` 中注册
- [x] 1.2 从 `steps_db_backup_test.go` 中移除该方法和注册行，确保 db-backup 不再重复注册
- [x] 1.3 运行 `go build -tags dev ./cmd/server/` 确认编译通过

## 2. Boss Support 文件

- [x] 2.1 创建 `boss_support_test.go`：定义 `bossCollaborationSpec` 常量（基于 seed.go Boss 规范，强化串签顺序和完成条件措辞）
- [x] 2.2 定义 `bossCasePayload` 结构体和 `bossCasePayloads`（2 组 case payload：requester-1 和 requester-2，包含 subject、category、risk_level、expected_completion、change_start、change_end、impact_scope、rollback_requirement、impact_module、resource_items 明细表格数组）
- [x] 2.3 实现 `generateBossWorkflow()`：复用 generateVPNWorkflow 模式，替换协作规范为 Boss 版本
- [x] 2.4 实现 `publishBossSmartService()`：创建 ServiceCatalog + Priority + Agent + ServiceDefinition（无 ServiceAction）

## 3. Boss Step Definitions

- [x] 3.1 创建 `steps_boss_test.go`，实现 `registerBossSteps()`
- [x] 3.2 实现 Given step `已定义高风险变更协同申请协作规范`
- [x] 3.3 实现 Given step `已基于协作规范发布高风险变更协同申请服务（智能引擎）`：调用 publishBossSmartService
- [x] 3.4 实现 Given step `"<username>" 已创建高风险变更工单，场景为 "<case_key>"`：根据 case_key 选择 payload，创建 Ticket
- [x] 3.5 实现 Given step `"<username>" 已创建高风险变更工单 "<alias>"，场景为 "<case_key>"`：支持并行工单别名
- [x] 3.6 实现 Then step `工单的表单数据中包含完整的 resource_items 明细表格`：断言 ticket.FormData 中 resource_items 数组存在且字段值完整
- [x] 3.7 实现 Then step `工单 "<ticket_ref>" 的审批记录与工单 "<other_ref>" 完全隔离`：断言两张工单的 TicketAssignment 记录互不包含对方的 ticket_id

## 4. Feature 文件

- [x] 4.1 创建 `features/boss_serial_approval.feature`：Background（系统初始化、参与人注册含 serial-reviewer + ops-approver + ops_admin 岗位、协作规范、服务发布）+ 4 个 Scenario

## 5. 注册与验证

- [x] 5.1 在 `bdd_test.go` 的 `initializeScenario` 中注册 `registerBossSteps(sc, bc)`
- [x] 5.2 运行 `go build -tags dev ./cmd/server/` 确认编译通过
- [x] 5.3 运行 `make test-bdd` 验证全部 4 个 scenario 通过
