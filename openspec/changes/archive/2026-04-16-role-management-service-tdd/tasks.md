## 1. Test Infrastructure

- [x] 1.1 Create `internal/service/role_test.go` with `newTestDB`, `newRoleServiceForTest`, `seedRole`, and `seedUser` helpers
- [x] 1.2 Ensure `AutoMigrate` covers `Role`, `RoleDeptScope`, and `User` (Casbin table created automatically by gormadapter)

## 2. Create & Retrieve Tests

- [x] 2.1 `TestRoleServiceCreate_Success`
- [x] 2.2 `TestRoleServiceCreate_RejectsDuplicateCode`
- [x] 2.3 `TestRoleServiceGetByID_Success`
- [x] 2.4 `TestRoleServiceGetByID_ReturnsNotFoundForMissing`
- [x] 2.5 `TestRoleServiceGetByIDWithDeptScope_Success`

## 3. Update Tests

- [x] 3.1 `TestRoleServiceUpdate_Success`
- [x] 3.2 `TestRoleServiceUpdate_RejectsDuplicateCode`
- [x] 3.3 `TestRoleServiceUpdate_PreventsSystemRoleCodeChange`
- [x] 3.4 `TestRoleServiceUpdate_MigratesCasbinPoliciesOnCodeChange`
- [x] 3.5 `TestRoleServiceUpdate_ReturnsNotFoundForMissing`

## 4. Data Scope Tests

- [x] 4.1 `TestRoleServiceUpdateDataScope_Success`
- [x] 4.2 `TestRoleServiceUpdateDataScope_ClearsDeptIDsWhenNotCustom`
- [x] 4.3 `TestRoleServiceUpdateDataScope_RejectsInvalidScope`
- [x] 4.4 `TestRoleServiceUpdateDataScope_PreventsAdminScopeChange`
- [x] 4.5 `TestRoleServiceUpdateDataScope_ReturnsNotFoundForMissing`

## 5. Delete Tests

- [x] 5.1 `TestRoleServiceDelete_Success` (verify role deleted, Casbin policies cleaned, RoleDeptScope cleared)
- [x] 5.2 `TestRoleServiceDelete_PreventsSystemRoleDeletion`
- [x] 5.3 `TestRoleServiceDelete_PreventsDeletionWhenUsersAssigned`
- [x] 5.4 `TestRoleServiceDelete_ReturnsNotFoundForMissing`

## 6. Verification

- [x] 6.1 Run `go test ./internal/service/...` and ensure all tests pass
- [x] 6.2 Fix any compilation issues in service layer caused by test helpers
