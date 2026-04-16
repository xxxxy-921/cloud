## 1. Test Infrastructure

- [ ] 1.1 Create `internal/service/menu_test.go` with `newTestDBForMenu`, `newMenuServiceForTest`, and `seedMenu` helpers using in-memory SQLite and real Casbin enforcer.

## 2. Tree Retrieval Tests

- [ ] 2.1 Implement `TestMenuServiceGetTree_Sorted` to verify full tree with mixed sort values.
- [ ] 2.2 Implement `TestMenuServiceGetUserTree_AdminGetsFullTree` to verify admin bypass.
- [ ] 2.3 Implement `TestMenuServiceGetUserTree_RoleSeesOnlyPermittedMenus` to verify Casbin filtering.
- [ ] 2.4 Implement `TestMenuServiceGetUserTree_ParentDirectoryRetained` to verify ancestor inclusion when descendants are permitted.
- [ ] 2.5 Implement `TestMenuServiceGetUserTree_HiddenMenusIncluded` to verify IsHidden does not affect access.

## 3. Permission List Tests

- [ ] 3.1 Implement `TestMenuServiceGetUserPermissions_ReturnsPermissions` to verify non-empty permission strings are returned.

## 4. CRUD Tests

- [ ] 4.1 Implement `TestMenuServiceCreate_Success` and `TestMenuServiceCreate_RootDirectory`.
- [ ] 4.2 Implement `TestMenuServiceUpdate_Success`, `TestMenuServiceUpdate_Parent`, and `TestMenuServiceUpdate_NotFound`.
- [ ] 4.3 Implement `TestMenuServiceReorderMenus_Success`.
- [ ] 4.4 Implement `TestMenuServiceDelete_LeafMenu`, `TestMenuServiceDelete_PreventsChildren`, and `TestMenuServiceDelete_NotFound`.

## 5. Verification

- [ ] 5.1 Run `go test ./internal/service/ -run TestMenuService -v` and ensure all tests pass.
- [ ] 5.2 Run `go test ./...` to confirm no regressions.
