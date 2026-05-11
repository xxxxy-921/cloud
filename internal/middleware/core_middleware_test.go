package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"metis/internal/app"
	"metis/internal/model"
	"metis/internal/pkg/token"
)

type stubOrgResolver struct {
	deptIDs []uint
	err     error
}

func (s *stubOrgResolver) GetUserDeptScope(userID uint, includeSubDepts bool) ([]uint, error) {
	return s.deptIDs, s.err
}
func (s *stubOrgResolver) GetUserPositionIDs(userID uint) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) GetUserDepartmentIDs(userID uint) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) GetUserPositions(userID uint) ([]app.OrgPosition, error) { return nil, nil }
func (s *stubOrgResolver) GetUserDepartment(userID uint) (*app.OrgDepartment, error) { return nil, nil }
func (s *stubOrgResolver) QueryContext(username, deptCode, positionCode string, includeInactive bool) (*app.OrgContextResult, error) {
	return nil, nil
}
func (s *stubOrgResolver) FindUsersByPositionCode(posCode string) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) FindUsersByDepartmentCode(deptCode string) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) FindUsersByPositionAndDepartment(posCode, deptCode string) ([]uint, error) {
	return nil, nil
}
func (s *stubOrgResolver) FindUsersByPositionID(positionID uint) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) FindUsersByDepartmentID(departmentID uint) ([]uint, error) { return nil, nil }
func (s *stubOrgResolver) FindManagerByUserID(userID uint) (uint, error) { return 0, nil }

func newMiddlewareRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middlewares...)
	r.GET("/api/v1/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"userId":              c.MustGet("userId"),
			"userRole":            c.MustGet("userRole"),
			"tokenJTI":            c.MustGet("tokenJTI"),
			"passwordChangedAt":   c.MustGet("passwordChangedAt"),
			"forcePasswordReset":  c.MustGet("forcePasswordReset"),
			"deptScopeWasPresent": c.Keys["deptScope"] != nil,
		})
	})
	return r
}

func newCasbinEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(`
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
`)
	if err != nil {
		t.Fatalf("NewModelFromString returned error: %v", err)
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("NewEnforcer returned error: %v", err)
	}
	return e
}

func TestJWTAuthAcceptsBearerAndQueryToken(t *testing.T) {
	secret := []byte("secret")
	changedAt := time.Now().Add(-time.Minute).UTC()
	tokenString, claims, err := token.GenerateAccessToken(7, "admin", secret, token.WithPasswordMeta(&changedAt, true))
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	r := newMiddlewareRouter(JWTAuth(secret, token.NewBlacklist()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"userId":7`) || !strings.Contains(w.Body.String(), claims.ID) {
		t.Fatalf("expected JWT claims in response, got %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected?token="+tokenString, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for query token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTAuthRejectsInvalidInputsAndBlacklist(t *testing.T) {
	secret := []byte("secret")
	r := newMiddlewareRouter(JWTAuth(secret, token.NewBlacklist()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "invalid authorization format") {
		t.Fatalf("expected invalid authorization format, got %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "missing authorization") {
		t.Fatalf("expected missing authorization, got %d %s", w.Code, w.Body.String())
	}

	tokenString, claims, err := token.GenerateAccessToken(1, "user", secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	blacklist := token.NewBlacklist()
	blacklist.Add(claims.ID, time.Now().Add(time.Minute))
	r = newMiddlewareRouter(JWTAuth(secret, blacklist))
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "session terminated") {
		t.Fatalf("expected terminated session, got %d %s", w.Code, w.Body.String())
	}

	expiredClaims := token.TokenClaims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			Issuer:    "metis",
			Subject:   "1",
			ID:        "expired-jti",
		},
	}
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w = httptest.NewRecorder()
	r = newMiddlewareRouter(JWTAuth(secret, token.NewBlacklist()))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "token expired") {
		t.Fatalf("expected token expired, got %d %s", w.Code, w.Body.String())
	}
}

func TestCasbinAuthWhitelistAndPermissionChecks(t *testing.T) {
	e := newCasbinEnforcer(t)
	if _, err := e.AddPolicy("admin", "/api/v1/protected", http.MethodGet); err != nil {
		t.Fatalf("AddPolicy returned error: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userRole", "admin")
	})
	r.Use(CasbinAuth(e))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/site-info", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/site-info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected whitelist route to pass, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected protected route for admin, got %d %s", w.Code, w.Body.String())
	}

	r = gin.New()
	r.Use(CasbinAuth(e))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected missing role to be forbidden, got %d", w.Code)
	}

	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("userRole", "user") })
	r.Use(CasbinAuth(e))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected insufficient permission to be forbidden, got %d", w.Code)
	}

	r = gin.New()
	r.Use(CasbinAuth(e))
	r.PUT("/api/v1/itsm/tickets/:id/withdraw", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodPut, "/api/v1/itsm/tickets/42/withdraw", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected keymatch whitelist route to pass, got %d", w.Code)
	}
}

func TestDataScopeMiddlewareInjectsExpectedScopes(t *testing.T) {
	cases := []struct {
		name       string
		resolver   app.OrgResolver
		scope      string
		custom     []uint
		userID     uint
		roleCode   string
		scopeErr   bool
		wantNil    bool
		wantLength int
	}{
		{name: "resolver missing means all", resolver: nil, userID: 7, roleCode: "admin", wantNil: true},
		{name: "missing role means all", resolver: &stubOrgResolver{}, userID: 7, wantNil: true},
		{name: "zero user means all", resolver: &stubOrgResolver{}, roleCode: "admin", wantNil: true},
		{name: "scope lookup error degrades to all", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scopeErr: true, wantNil: true},
		{name: "self becomes empty slice", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scope: "self", wantLength: 0},
		{name: "dept comes from resolver", resolver: &stubOrgResolver{deptIDs: []uint{2, 3}}, userID: 7, roleCode: "admin", scope: "dept", wantLength: 2},
		{name: "dept empty becomes self-only marker", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scope: "dept", wantLength: 0},
		{name: "dept and sub uses resolver", resolver: &stubOrgResolver{deptIDs: []uint{5, 6, 7}}, userID: 7, roleCode: "admin", scope: "dept_and_sub", wantLength: 3},
		{name: "dept and sub resolver error becomes empty", resolver: &stubOrgResolver{err: http.ErrHandlerTimeout}, userID: 7, roleCode: "admin", scope: "dept_and_sub", wantLength: 0},
		{name: "custom passes through", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scope: "custom", custom: []uint{9}, wantLength: 1},
		{name: "custom empty becomes empty slice", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scope: "custom", wantLength: 0},
		{name: "unknown falls back to nil", resolver: &stubOrgResolver{}, userID: 7, roleCode: "admin", scope: "weird", wantNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tc.userID != 0 {
					c.Set("userId", tc.userID)
				}
				if tc.roleCode != "" {
					c.Set("userRole", tc.roleCode)
				}
			})
			r.Use(DataScopeMiddleware(tc.resolver, func(roleCode string) (model.DataScope, []uint, error) {
				if tc.scopeErr {
					return "", nil, http.ErrHandlerTimeout
				}
				return model.DataScope(tc.scope), tc.custom, nil
			}))
			r.GET("/scope", func(c *gin.Context) {
				val, _ := c.Get("deptScope")
				scope, _ := val.(*[]uint)
				if tc.wantNil {
					if scope != nil {
						t.Fatalf("expected nil dept scope, got %+v", *scope)
					}
					c.Status(http.StatusNoContent)
					return
				}
				if scope == nil {
					t.Fatal("expected non-nil dept scope")
				}
				if len(*scope) != tc.wantLength {
					t.Fatalf("expected len %d, got %+v", tc.wantLength, *scope)
				}
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/scope", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusNoContent {
				t.Fatalf("expected 204, got %d", w.Code)
			}
		})
	}
}

func TestDataScopeMiddlewareGracefulDegradation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(7))
		c.Set("userRole", "admin")
	})
	r.Use(DataScopeMiddleware(&stubOrgResolver{err: assertErr{}}, func(roleCode string) (model.DataScope, []uint, error) {
		return model.DataScopeDept, nil, assertErr{}
	}))
	r.GET("/scope", func(c *gin.Context) {
		val, _ := c.Get("deptScope")
		if scope, _ := val.(*[]uint); scope != nil {
			t.Fatalf("expected nil scope on lookup failure, got %+v", *scope)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/scope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(0))
		c.Set("userRole", "admin")
	})
	r.Use(DataScopeMiddleware(&stubOrgResolver{deptIDs: []uint{1}}, func(roleCode string) (model.DataScope, []uint, error) {
		return model.DataScopeDept, nil, nil
	}))
	r.GET("/scope-zero", func(c *gin.Context) {
		val, _ := c.Get("deptScope")
		if scope, _ := val.(*[]uint); scope != nil {
			t.Fatalf("expected nil scope when userID is zero, got %+v", *scope)
		}
		c.Status(http.StatusNoContent)
	})
	req = httptest.NewRequest(http.MethodGet, "/scope-zero", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for zero userID path, got %d", w.Code)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

func TestPasswordExpiryAndRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("passwordChangedAt", time.Now().Add(-48*time.Hour).Unix())
		c.Set("forcePasswordReset", false)
	})
	r.Use(PasswordExpiry(func() int { return 1 }))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected expired password conflict, got %d", w.Code)
	}

	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("forcePasswordReset", true)
	})
	r.Use(PasswordExpiry(func() int { return 0 }))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected force reset conflict, got %d", w.Code)
	}

	r = gin.New()
	r.Use(PasswordExpiry(func() int { return 1 }))
	r.GET("/api/v1/auth/password", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/password", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected whitelist path to pass, got %d", w.Code)
	}

	r = gin.New()
	r.Use(PasswordExpiry(func() int { return 1 }))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected missing passwordChangedAt to pass, got %d", w.Code)
	}

	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("passwordChangedAt", time.Now().Add(-12*time.Hour).Unix())
	})
	r.Use(PasswordExpiry(func() int { return 1 }))
	r.GET("/api/v1/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected recent password to pass, got %d", w.Code)
	}

	r = gin.New()
	r.Use(PasswordExpiry(func() int { return 1 }))
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected non-api route to bypass expiry middleware, got %d", w.Code)
	}

	r = gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	req = httptest.NewRequest(http.MethodGet, "/panic", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "internal server error") {
		t.Fatalf("expected recovery to intercept panic, got %d %s", w.Code, w.Body.String())
	}
}

func TestLoggerAndAbortJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger())
	r.GET("/api/v1/notifications/unread-count", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/other", func(c *gin.Context) { abortJSON(c, http.StatusTeapot, "teapot") })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected silent path to pass, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/other?q=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTeapot || !strings.Contains(w.Body.String(), "teapot") {
		t.Fatalf("expected abortJSON payload, got %d %s", w.Code, w.Body.String())
	}
}
