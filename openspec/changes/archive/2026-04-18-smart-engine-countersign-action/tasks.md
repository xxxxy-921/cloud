## 1. Data Model

- [x] 1.1 Add `ActivityGroupID string` field to `TicketActivity` in `model_ticket.go`
- [x] 1.2 Add `ActivityGroupID` to `activityModel` in `engine/classic.go` to keep struct in sync

## 2. DecisionPlan 并签支持

- [x] 2.1 Add `ExecutionMode string` field to `DecisionPlan` struct in `engine/smart.go`
- [x] 2.2 Update `executeDecisionPlan()` to handle `execution_mode: "parallel"` — generate UUID group ID, create parallel activities with shared `activity_group_id`, set `current_activity_id` to first activity

## 3. Progress 汇聚检查

- [x] 3.1 Add convergence check in `SmartEngine.Progress()` — after marking activity completed, check if activity has `activity_group_id`; if so, count incomplete siblings; only trigger next decision cycle when all siblings complete

## 4. Action 元调用工具

- [x] 4.1 Add `actionExecutor *ActionExecutor` field to `decisionToolContext` struct
- [x] 4.2 Implement `toolExecuteAction()` — load ServiceAction by ID, call `ActionExecutor.Execute()` synchronously, record execution result, return success/failure to Agent
- [x] 4.3 Register `decision.execute_action` in `allDecisionTools()` (now 8 tools)
- [x] 4.4 Inject `actionExecutor` into `decisionToolContext` in `agenticDecision()`

## 5. ticket_context 增强

- [x] 5.1 Add `parallel_groups` to `toolTicketContext()` — query active parallel groups (activities with non-empty `activity_group_id` and status != completed), aggregate by group_id showing total/completed/pending

## 6. Prompt 更新

- [x] 6.1 Update `agenticToolGuidance` — add `decision.execute_action` tool description, update recommended reasoning steps to guide Agent to use tool call for action execution
- [x] 6.2 Update `agenticOutputFormat` — add `execution_mode` field to JSON template and field description

## 7. BDD 并签场景

- [x] 7.1 Create `features/multi_role_countersign.feature` — 2 LLM-driven scenarios: (1) all approve → converge → complete, (2) partial approve blocks convergence
- [x] 7.2 Create `countersign_support_test.go` — service publish helper with collaboration spec requiring parallel countersign (2 roles), workflow generation via LLM
- [x] 7.3 Create `steps_countersign_test.go` — step definitions for countersign BDD: parallel activity assertions, per-role approve, convergence checks
- [x] 7.4 Register countersign steps in `bdd_test.go`

## 8. BDD Action 元调用场景

- [x] 8.1 Update `features/db_backup_whitelist_action_flow.feature` — simplify scenarios to validate Agent uses `decision.execute_action` tool call (fewer manual decision cycles)
- [x] 8.2 Update `steps_db_backup_test.go` — adapt step definitions for action meta-call model
- [x] 8.3 Verify existing BDD scenarios (vpn_smart, server_access, vpn_deterministic) still pass
