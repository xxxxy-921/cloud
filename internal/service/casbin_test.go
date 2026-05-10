package service

import (
	"testing"

	"github.com/samber/do/v2"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
)

func TestCasbinServiceCheckPermissionAndGetEnforcer(t *testing.T) {
	db := newTestDBForAuthService(t)
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})

	enforcer, err := casbinpkg.NewEnforcerWithDB(db)
	if err != nil {
		t.Fatalf("NewEnforcerWithDB: %v", err)
	}
	do.ProvideValue(injector, enforcer)
	do.Provide(injector, NewCasbin)

	svc := do.MustInvoke[*CasbinService](injector)
	if svc.GetEnforcer() != enforcer {
		t.Fatal("expected GetEnforcer to return underlying enforcer")
	}

	if err := svc.SetPoliciesForRole("admin", [][]string{{"admin", "system:user:list", "GET"}}); err != nil {
		t.Fatalf("SetPoliciesForRole: %v", err)
	}
	allowed, err := svc.CheckPermission("admin", "system:user:list", "GET")
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission to be allowed")
	}
}
