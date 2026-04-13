## 1. 后端：增量 Assignment API

- [x] 1.1 在 `assignment_repository.go` 中新增 `AddPosition(userPosition)` 方法（含 `(UserID, DepartmentID)` 去重校验）
- [x] 1.2 在 `assignment_repository.go` 中新增 `RemovePosition(assignmentId, userID)` 方法（验证归属后删除，若删除的是主岗则自动提升下一个）
- [x] 1.3 在 `assignment_repository.go` 中新增 `UpdatePosition(assignmentId, userID, fields)` 方法（支持修改 positionId、isPrimary）
- [x] 1.4 在 `assignment_service.go` 中新增 `AddUserPosition(userID, deptID, posID, isPrimary)` — 含去重校验、自动主岗处理
- [x] 1.5 在 `assignment_service.go` 中新增 `RemoveUserPosition(userID, assignmentID)` — 含主岗自动继承逻辑
- [x] 1.6 在 `assignment_service.go` 中新增 `UpdateUserPosition(userID, assignmentID, posID, isPrimary)` — 修改单条分配
- [x] 1.7 在 `assignment_handler.go` 中新增 `POST /org/users/:id/positions` 端点
- [x] 1.8 在 `assignment_handler.go` 中新增 `DELETE /org/users/:id/positions/:assignmentId` 端点
- [x] 1.9 在 `assignment_handler.go` 中新增 `PUT /org/users/:id/positions/:assignmentId` 端点（修改岗位/主岗）
- [x] 1.10 删除旧的全量替换 `PUT /org/users/:id/positions` handler 及对应的 `ReplaceUserPositions` repository 方法
- [x] 1.11 在 `seed.go` 中补充新端点的 Casbin 策略

## 2. 后端：部门树成员计数

- [x] 2.1 在 `assignment_repository.go` 中新增 `CountByDepartments()` 方法 — `SELECT department_id, COUNT(*) FROM user_positions GROUP BY department_id`
- [x] 2.2 修改 `DepartmentService.Tree()` 方法，调用 `CountByDepartments()` 并在 `DepartmentTreeNode` 中填充 `MemberCount` 字段
- [x] 2.3 在 `DepartmentTreeNode` 响应结构体中新增 `MemberCount int` 字段

## 3. 后端：Scope Helper 过滤停用部门

- [x] 3.1 修改 `GetUserDepartmentScope()` 中的 BFS 遍历，跳过 `IsActive = false` 的部门节点（停用部门及其子树不纳入 scope）
- [x] 3.2 修改 `GetSubDepartmentIDs()` 同步过滤停用部门

## 4. 前端：部门树面板增强

- [x] 4.1 左侧部门树节点显示成员计数 Badge（从 tree API 的 `memberCount` 字段读取）
- [x] 4.2 添加部门搜索输入框，实时过滤匹配节点并保留祖先链
- [x] 4.3 选中部门增加视觉反馈（accent left border + 背景高亮）

## 5. 前端：成员表增强

- [x] 5.1 成员列改为双行显示：头像 + 用户名（第一行）、邮箱（第二行，muted 色）
- [x] 5.2 新增"类型"列，用 Badge 区分主岗（★ 标识 + accent 色）和兼岗
- [x] 5.3 新增"分配时间"列，显示 `createdAt` 格式化日期
- [x] 5.4 操作列改为 DropdownMenu：设为主岗（仅兼岗显示）、变更岗位、查看组织信息、从部门移除

## 6. 前端：分配成员 Sheet 改造

- [x] 6.1 用户选择改为可搜索下拉（输入时查询 `/api/v1/users?keyword=...`），显示头像+名称+邮箱
- [x] 6.2 已在当前部门的用户在下拉中灰显并标注"(已分配)"
- [x] 6.3 提交改为调用 `POST /api/v1/org/users/:id/positions` 增量 API
- [x] 6.4 成功后 invalidate 成员列表 + 部门树（更新计数）

## 7. 前端：变更岗位 Sheet

- [x] 7.1 新建 ChangePositionSheet 组件，接收当前 assignmentId 和 positionId
- [x] 7.2 Sheet 内展示岗位 Select（预选当前岗位），提交调用 `PUT /api/v1/org/users/:id/positions/:assignmentId`

## 8. 前端：用户组织信息 Sheet

- [x] 8.1 新建 UserOrgSheet 组件，接收 userId，调用 `GET /api/v1/org/users/:id/positions`
- [x] 8.2 Sheet 内展示用户基本信息 + 所有分配列表（部门 / 岗位 / 主岗 Badge）

## 9. 前端：空状态与细节

- [x] 9.1 未选择部门时右侧显示空状态（图标 + "请在左侧选择一个部门"）
- [x] 9.2 选中部门无成员时显示空状态（图标 + "该部门暂无成员" + "分配成员"按钮）
- [x] 9.3 移除成员操作增加 AlertDialog 确认
- [x] 9.4 更新 zh-CN 和 en 国际化文件，补充新增的翻译 key

## 10. 清理与验证

- [x] 10.1 删除前端旧的全量替换调用逻辑（原 `PUT /users/:id/positions` 拼接逻辑）
- [x] 10.2 验证 Casbin 策略覆盖所有新端点
- [ ] 10.3 手动测试完整流程：添加成员、设主岗、变更岗位、查看组织信息、移除成员、部门搜索过滤
