package license

import (
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app/license/testutil"
	"metis/internal/model"
)

func TestLicenseAppProvidesRoutesModelsAndTasks(t *testing.T) {
	db := testutil.SetupTestDB(t)
	injector := do.New()
	do.ProvideValue(injector, db)
	do.ProvideValue[[]byte](injector, []byte("test-jwt-secret"))
	do.ProvideNamedValue(injector, "licenseKeySecret", []byte("test-license-secret"))

	app := &LicenseApp{}
	if app.Name() != "license" {
		t.Fatalf("Name = %q, want license", app.Name())
	}
	if len(app.Models()) != 6 {
		t.Fatalf("Models len = %d, want 6", len(app.Models()))
	}

	app.Providers(injector)

	router := gin.New()
	api := router.Group("/api/v1")
	app.Routes(api)
	if len(router.Routes()) != 31 {
		t.Fatalf("route count = %d, want 31", len(router.Routes()))
	}

	tasks := app.Tasks()
	if len(tasks) != 2 {
		t.Fatalf("tasks len = %d, want 2", len(tasks))
	}
	if tasks[0].Name != "license-expired-check" || tasks[1].Name != "license-registration-cleanup" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	for _, task := range tasks {
		if err := task.Handler(nil, nil); err != nil {
			t.Fatalf("task %s handler: %v", task.Name, err)
		}
	}
}

func TestLicenseAppSeedRegistersMenusAndPolicies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:license-app-seed?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}

	m, err := casbinmodel.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`)
	if err != nil {
		t.Fatalf("new casbin model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}

	app := &LicenseApp{}
	if err := app.Seed(db, enforcer, false); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	var count int64
	if err := db.Model(&model.Menu{}).Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if count == 0 {
		t.Fatal("expected seed to create license menus")
	}

	allowed, err := enforcer.Enforce("admin", "/api/v1/license/products/:id/public-key", "GET")
	if err != nil {
		t.Fatalf("enforce seeded API policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected public key API policy to be seeded")
	}
}
