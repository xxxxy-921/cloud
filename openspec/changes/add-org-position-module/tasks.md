## 1. Backend App Scaffold

- [x] 1.1 Create `internal/app/org/` directory with `app.go` implementing `app.App` interface
- [x] 1.2 Create edition import in `cmd/server/edition_full.go` for `org` app
- [x] 1.3 Verify `go build -tags dev ./cmd/server/` compiles after scaffold

## 2. Data Models

- [x] 2.1 Create `internal/app/org/model.go` with `Department`, `Position`, `UserPosition` structs and JSON helper types
- [x] 2.2 Register models in `OrgApp.Models()`
- [x] 2.3 Verify GORM auto-migration creates `departments`, `positions`, `user_positions` tables on startup

## 3. Repositories

- [x] 3.1 Create `department_repository.go` with CRUD + tree query + "has children/members" check
- [x] 3.2 Create `position_repository.go` with CRUD + paginated list + "in use" check
- [x] 3.3 Create `assignment_repository.go` with CRUD for `UserPosition` + primary position enforcement + scope queries
- [x] 3.4 Wire repositories into IOC via `OrgApp.Providers()`

## 4. Services

- [x] 4.1 Create `department_service.go` with business logic and validation
- [x] 4.2 Create `position_service.go` with business logic and validation
- [x] 4.3 Create `assignment_service.go` with batch update, primary position enforcement, and scope helpers
- [x] 4.4 Wire services into IOC via `OrgApp.Providers()`

## 5. Handlers & Routes

- [x] 5.1 Create `department_handler.go` with endpoints: `POST/GET /departments`, `GET/PUT/DELETE /departments/:id`, `GET /departments/tree`
- [x] 5.2 Create `position_handler.go` with endpoints: `POST/GET /positions`, `GET/PUT/DELETE /positions/:id`
- [x] 5.3 Create `assignment_handler.go` with endpoints: `GET/PUT /users/:id/positions`, `GET /users` (org-enhanced list)
- [x] 5.4 Mount all routes under `/api/v1/org` in `OrgApp.Routes()`
- [x] 5.5 Register IOC providers for handlers

## 6. Seed & Permissions

- [x] 6.1 Create `seed.go` with menu tree: 组织管理目录 + 部门管理/岗位管理/人员分配菜单 + 对应按钮权限
- [x] 6.2 Seed Casbin policies for `admin` role covering `/api/v1/org/**` CRUD and menu permissions
- [x] 6.3 Verify menu appears in admin user-tree after fresh install / restart sync

## 7. Frontend Scaffold

- [x] 7.1 Create `web/src/apps/org/` directory structure (`module.ts`, `pages/`, `components/`, `locales/`)
- [x] 7.2 Add `import "./org/module"` to `web/src/apps/_bootstrap.ts`
- [x] 7.3 Add `zh-CN.json` and `en.json` locale files for org module

## 8. Frontend Pages

- [x] 8.1 Implement `/org/departments` page with tree table, Sheet form, and delete confirmation
- [x] 8.2 Implement `/org/positions` page with DataTable, Sheet form, and delete confirmation
- [x] 8.3 Implement `/org/assignments` page with department tree (left) + member list (right), add/remove member, primary toggle
- [x] 8.4 Register routes in `module.ts` matching backend API paths

## 9. Wiring & Verification

- [x] 9.1 Run `make dev` and `make web-dev`, verify no compilation errors
- [ ] 9.2 Verify admin can navigate to 组织管理 → 部门管理 and create a department
- [ ] 9.3 Verify admin can create a position, then assign it to a user in 人员分配
- [ ] 9.4 Verify that deleting an in-use department or position returns a proper error toast
