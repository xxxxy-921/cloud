## Context

Metis ITSM 模块已有完整的 VPN 开通申请 BDD 测试，覆盖经典引擎和智能引擎两种模式。VPN 场景是 2-way 路由（网络管理员 / 安全管理员），基于 exclusive gateway 条件或 AI 决策。

现在需要新增**生产服务器临时访问申请**BDD，参考 bklite-cloud 的 `server_access_branch_decision` 设计。该场景是 3-way 路由（运维 / 网络 / 安全管理员），仅使用智能引擎，由 AI 根据访问目的在运行时判断路由分支。

现有基础设施（`bddContext`、`testOrgService`、`testAgentProvider`、`testUserProvider`、common steps）可直接复用。

## Goals / Non-Goals

**Goals:**
- 验证智能引擎在 3-way 分支场景下的路由决策准确性
- 验证模糊语义输入（boundary case）下 AI 的判断能力
- 验证审批责任边界（错误审批人无法认领/审批）
- 新增的通用 step definitions 可被后续 BDD 场景复用

**Non-Goals:**
- 不覆盖经典引擎（3-way 路由的访问目的是自然语言，不适合枚举下拉做 gateway 条件）
- 不覆盖服务台 Agent 全链路（draft_prepare / draft_confirm 流程）
- 不修改 SmartEngine 核心逻辑
- 不新增 seed 数据（BDD 测试使用独立的 in-memory DB）

## Decisions

### Decision 1: Workflow JSON 由 LLM 生成，而非静态 fixture

**选择**: LLM 生成（与 VPN BDD 一致）

**替代方案**: bklite-cloud 使用硬编码的 `_workflow_json()` 静态 fixture

**理由**: LLM 生成更真实，能同时验证 workflow 生成和执行两个环节。Metis 已有 `generateVPNWorkflow` 基础设施可复用，只需替换协作规范即可。生成的 workflow 没有 gateway 条件（所有边的 routing_conditions 为空），路由完全由智能引擎运行时决策。

### Decision 2: 协作规范明确指定"不要让申请人选择审批类别"

**选择**: 协作规范中显式声明由 AI 判断路由

**理由**: 与 bklite 对齐。这确保 LLM 生成的 workflow 不会引入枚举下拉字段，而是保留自然语言访问目的让 AI 在运行时判断。

### Decision 3: 4 组 case payload 覆盖 3 条分支 + 1 个边界 case

**选择**: ops / network / security / boundary_security

**理由**: 前 3 个覆盖每条分支的 happy path，boundary_security 测试模糊输入（"异常访问核查+证据保全"应判定为安全而非运维），与 bklite 完全对齐。

### Decision 4: 责任边界验证复用 TicketAssignment 的 position/department 检查

**选择**: 通过 `TicketAssignment.PositionID` + `DepartmentID` + `testOrgService.FindUsersByPositionAndDepartment` 判断是否有权认领

**替代方案**: 模拟 HTTP API 的权限校验

**理由**: BDD 测试直接操作引擎层，不经过 HTTP handler。通过检查 assignment 的岗位/部门归属判断可见性，再通过尝试 claim/progress 操作验证越权失败，与 bklite 的 `is_user_eligible_for_assignment` + `claim_work_item` 异常测试对齐。

### Decision 5: 新增 step definitions 放在独立文件，通用 steps 提取到 common

**选择**: `steps_server_access_test.go` 放专属 steps，`当前审批分配到岗位`、`当前审批仅对 X 可见`、`X 认领/审批当前工单应失败` 等通用断言放入 `steps_common_test.go`

**理由**: 这些断言步骤不绑定具体服务类型，后续其他 BDD 场景（如数据库备份申请）也能复用。

## Risks / Trade-offs

- **[LLM 生成不确定性]** → LLM 可能生成不符合预期的 workflow 结构。缓解：`ValidateWorkflow` 验证 + 最多 3 次重试，与 VPN 一致
- **[AI 路由判断不稳定]** → 边界语义 case 的 AI 判断可能不一致。缓解：协作规范中明确归类规则，temperature 设低 (0.2)，测试仅断言岗位而非具体用户
- **[测试耗时]** → 每个 scenario 都需要 LLM 调用（生成 workflow + 决策循环）。缓解：workflow 生成结果可在 support 文件中缓存（同一 collaboration spec 不重复生成）
