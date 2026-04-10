## 1. Models & Database

- [x] 1.1 Create `internal/model/user.go` — User struct with Username, Password, Email, Phone, Avatar, Role, IsActive fields, embed BaseModel
- [x] 1.2 Create `internal/model/refresh_token.go` — RefreshToken struct with Token, UserID, ExpiresAt, Revoked fields, embed BaseModel
- [x] 1.3 Update `internal/database/database.go` — add User and RefreshToken to AutoMigrate call

## 2. Token Utilities

- [x] 2.1 Add `golang-jwt/jwt/v5` and `golang.org/x/crypto` dependencies
- [x] 2.2 Create `internal/pkg/token/jwt.go` — TokenClaims struct, GenerateAccessToken(), ParseToken() functions using HS256
- [x] 2.3 Add GenerateRefreshToken() (crypto/rand 32-byte base64url) and HashPassword/CheckPassword (bcrypt) utilities

## 3. Repositories

- [x] 3.1 Create `internal/repository/user.go` — UserRepo with FindByUsername, FindByID, List (search+filter+pagination), Create, Update, Delete
- [x] 3.2 Create `internal/repository/refresh_token.go` — RefreshTokenRepo with Create, FindValid, Revoke, RevokeAllForUser

## 4. Services

- [x] 4.1 Create `internal/service/auth.go` — AuthService with Login, Logout, RefreshTokens, ChangePassword, GetCurrentUser
- [x] 4.2 Create `internal/service/user.go` — UserService with List, GetByID, Create, Update, Delete, ResetPassword, Activate, Deactivate

## 5. Middleware

- [x] 5.1 Create `internal/middleware/jwt.go` — JWTAuth() middleware: extract Bearer token, parse JWT, set userId/userRole in context, handle 401
- [x] 5.2 Create `internal/middleware/role.go` — RequireRole() middleware: check userRole against allowed roles, handle 403

## 6. Handlers & Routes

- [x] 6.1 Create `internal/handler/auth.go` — AuthHandler with Login, Logout, Refresh, GetMe, ChangePassword endpoints
- [x] 6.2 Create `internal/handler/user.go` — UserHandler with List, Create, Get, Update, Delete, ResetPassword, Activate, Deactivate endpoints
- [x] 6.3 Update `internal/handler/handler.go` — reorganize route registration into public/authenticated/admin groups with middleware

## 7. IOC & CLI

- [x] 7.1 Update `cmd/server/main.go` — register all new providers (UserRepo, RefreshTokenRepo, AuthService, UserService, AuthHandler, UserHandler), add JWT_SECRET env var handling
- [x] 7.2 Create CLI subcommand logic — `metis create-admin --username=xxx --password=xxx` subcommand that initializes DB, creates admin user, and exits

## 8. Frontend Auth Infrastructure

- [x] 8.1 Create `web/src/stores/auth.ts` — Zustand auth store with token persistence (localStorage), login/logout/refresh methods, current user state
- [x] 8.2 Update `web/src/lib/api.ts` — add Bearer token interceptor, 401 auto-refresh with concurrent request queuing, redirect to /login on refresh failure

## 9. Frontend Pages

- [x] 9.1 Create `web/src/pages/login/index.tsx` — login form page (full-screen, no DashboardLayout), username/password fields, error display
- [x] 9.2 Create `web/src/pages/users/index.tsx` — user management table with search, pagination, create/edit/delete/activate/deactivate actions
- [x] 9.3 Create `web/src/pages/users/user-sheet.tsx` — user create/edit form sheet
- [x] 9.4 Create password change dialog component

## 10. Frontend Routing & Layout

- [x] 10.1 Update `web/src/App.tsx` — add /login and /users routes, add auth guard (redirect unauthenticated to /login)
- [x] 10.2 Update layout components — add user menu dropdown (username display, change password, logout) in TopNav
- [x] 10.3 Update `web/src/lib/nav.ts` — add "用户管理" nav item under "系统" section with admin-only visibility
