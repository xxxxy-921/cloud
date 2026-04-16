## 1. Test Infrastructure

- [x] 1.1 Create `internal/service/user_test.go` with `newTestDB`, `newUserServiceForTest`, and `seedRole` helpers
- [x] 1.2 Ensure `AutoMigrate` covers `User`, `Role`, `SystemConfig`, and `RefreshToken`

## 2. Create & Retrieve Tests

- [x] 2.1 `TestUserServiceCreate_Success`
- [x] 2.2 `TestUserServiceCreate_RejectsDuplicateUsername`
- [x] 2.3 `TestUserServiceCreate_EnforcesPasswordPolicy`
- [x] 2.4 `TestUserServiceGetByID_Success`
- [x] 2.5 `TestUserServiceGetByID_ReturnsNotFoundForMissing`
- [x] 2.6 `TestUserServiceGetByIDWithManager_Success`

## 3. Update Tests

- [x] 3.1 `TestUserServiceUpdate_Success`
- [x] 3.2 `TestUserServiceUpdate_PreventsSelfRoleChange`
- [x] 3.3 `TestUserServiceUpdate_ReturnsNotFoundForMissing`
- [x] 3.4 `TestUserServiceUpdate_DetectsDirectCircularManager`
- [x] 3.5 `TestUserServiceUpdate_DetectsIndirectCircularManager`

## 4. Delete, Reset Password & Unlock Tests

- [x] 4.1 `TestUserServiceDelete_Success` (verify user deleted and tokens revoked)
- [x] 4.2 `TestUserServiceDelete_PreventsSelfDeletion`
- [x] 4.3 `TestUserServiceDelete_ReturnsNotFoundForMissing`
- [x] 4.4 `TestUserServiceResetPassword_Success` (verify password hashed, tokens revoked)
- [x] 4.5 `TestUserServiceResetPassword_EnforcesPasswordPolicy`
- [x] 4.6 `TestUserServiceResetPassword_ReturnsNotFoundForMissing`
- [x] 4.7 `TestUserServiceUnlockUser_Success`
- [x] 4.8 `TestUserServiceUnlockUser_ReturnsNotFoundForMissing`

## 5. Activation, Deactivation & Manager Chain Tests

- [x] 5.1 `TestUserServiceActivate_Success`
- [x] 5.2 `TestUserServiceActivate_ReturnsNotFoundForMissing`
- [x] 5.3 `TestUserServiceDeactivate_Success` (verify tokens revoked)
- [x] 5.4 `TestUserServiceDeactivate_PreventsSelfDeactivation`
- [x] 5.5 `TestUserServiceGetManagerChain_Success`
- [x] 5.6 `TestUserServiceGetManagerChain_ReturnsNotFoundForMissing`
- [x] 5.7 `TestUserServiceGetManagerChain_BreaksOnCycle`
- [x] 5.8 `TestUserServiceGetManagerChain_RespectsMaxDepth`
- [x] 5.9 `TestUserServiceClearManager_Success`
- [x] 5.10 `TestUserServiceClearManager_ReturnsNotFoundForMissing`

## 6. Verification

- [x] 6.1 Run `go test ./internal/service/...` and ensure all tests pass
- [x] 6.2 Fix any compilation issues in service layer caused by test helpers
