package bootstrap

import (
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/model"
)

type noopAdapter struct{}

func (noopAdapter) LoadPolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) SavePolicy(casbinmodel.Model) error                        { return nil }
func (noopAdapter) AddPolicy(string, string, []string) error                  { return nil }
func (noopAdapter) RemovePolicy(string, string, []string) error               { return nil }
func (noopAdapter) RemoveFilteredPolicy(string, string, int, ...string) error { return nil }

var _ persist.Adapter = (*noopAdapter)(nil)

func newSeedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Menu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}
	return db
}

func newSeedEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
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
		t.Fatalf("model: %v", err)
	}
	e, err := casbin.NewEnforcer(m, &noopAdapter{})
	if err != nil {
		t.Fatalf("enforcer: %v", err)
	}
	return e
}

func TestSeedLicenseIsIdempotentAndRegistersPolicies(t *testing.T) {
	db := newSeedDB(t)
	enforcer := newSeedEnforcer(t)

	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var count int64
	if err := db.Model(&model.Menu{}).Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if count != 20 {
		t.Fatalf("menu count = %d, want 20", count)
	}

	var productMenu model.Menu
	if err := db.Where("permission = ?", "license:product:list").First(&productMenu).Error; err != nil {
		t.Fatalf("find product menu: %v", err)
	}
	if productMenu.Path != "/license/products" {
		t.Fatalf("product menu path = %q", productMenu.Path)
	}

	allowed, err := enforcer.Enforce("admin", "/api/v1/license/licenses/:id/renew", "POST")
	if err != nil {
		t.Fatalf("enforce renew policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected renew API policy to be seeded")
	}

	allowed, err = enforcer.Enforce("admin", "license:registration:generate", "read")
	if err != nil {
		t.Fatalf("enforce registration menu policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected registration menu permission to be seeded")
	}
}

func TestSeedLicense_RestoresAndRepairsMenusAndButtonsInPlace(t *testing.T) {
	db := newSeedDB(t)
	enforcer := newSeedEnforcer(t)

	licenseDir := model.Menu{
		Name:       "旧许可",
		Type:       model.MenuTypeMenu,
		Icon:       "Legacy",
		Permission: "license",
		Sort:       999,
	}
	if err := db.Create(&licenseDir).Error; err != nil {
		t.Fatalf("create legacy license dir: %v", err)
	}
	licenseDirID := licenseDir.ID
	if err := db.Delete(&licenseDir).Error; err != nil {
		t.Fatalf("soft delete legacy license dir: %v", err)
	}

	wrongParent := model.Menu{
		Name:       "错误父菜单",
		Type:       model.MenuTypeDirectory,
		Permission: "license:wrong-parent",
	}
	if err := db.Create(&wrongParent).Error; err != nil {
		t.Fatalf("create wrong parent: %v", err)
	}

	productMenu := model.Menu{
		ParentID:   &wrongParent.ID,
		Name:       "旧商品",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/products",
		Icon:       "LegacyPkg",
		Permission: "license:product:list",
		Sort:       77,
	}
	if err := db.Create(&productMenu).Error; err != nil {
		t.Fatalf("create drifted product menu: %v", err)
	}
	productMenuID := productMenu.ID
	if err := db.Delete(&productMenu).Error; err != nil {
		t.Fatalf("soft delete drifted product menu: %v", err)
	}

	productButton := model.Menu{
		ParentID:   &wrongParent.ID,
		Name:       "旧新增商品",
		Type:       model.MenuTypeMenu,
		Permission: "license:product:create",
		Sort:       99,
	}
	if err := db.Create(&productButton).Error; err != nil {
		t.Fatalf("create drifted product button: %v", err)
	}
	productButtonID := productButton.ID
	if err := db.Delete(&productButton).Error; err != nil {
		t.Fatalf("soft delete drifted product button: %v", err)
	}

	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("seed license: %v", err)
	}

	var restoredDir model.Menu
	if err := db.Where("permission = ?", "license").First(&restoredDir).Error; err != nil {
		t.Fatalf("find restored license dir: %v", err)
	}
	if restoredDir.ID != licenseDirID {
		t.Fatalf("expected license dir restored in place, got original=%d restored=%d", licenseDirID, restoredDir.ID)
	}
	if restoredDir.Name != "许可管理" || restoredDir.Type != model.MenuTypeDirectory || restoredDir.Icon != "KeyRound" || restoredDir.Sort != 200 {
		t.Fatalf("unexpected restored license dir: %+v", restoredDir)
	}

	var restoredProductMenu model.Menu
	if err := db.Where("permission = ?", "license:product:list").First(&restoredProductMenu).Error; err != nil {
		t.Fatalf("find restored product menu: %v", err)
	}
	if restoredProductMenu.ID != productMenuID {
		t.Fatalf("expected product menu restored in place, got original=%d restored=%d", productMenuID, restoredProductMenu.ID)
	}
	if restoredProductMenu.ParentID == nil || *restoredProductMenu.ParentID != restoredDir.ID {
		t.Fatalf("expected product menu parent %d, got %v", restoredDir.ID, restoredProductMenu.ParentID)
	}
	if restoredProductMenu.Name != "商品管理" || restoredProductMenu.Type != model.MenuTypeMenu || restoredProductMenu.Path != "/license/products" || restoredProductMenu.Icon != "Package" || restoredProductMenu.Sort != 0 {
		t.Fatalf("unexpected restored product menu: %+v", restoredProductMenu)
	}

	var restoredButton model.Menu
	if err := db.Where("permission = ?", "license:product:create").First(&restoredButton).Error; err != nil {
		t.Fatalf("find restored product button: %v", err)
	}
	if restoredButton.ID != productButtonID {
		t.Fatalf("expected product button restored in place, got original=%d restored=%d", productButtonID, restoredButton.ID)
	}
	if restoredButton.ParentID == nil || *restoredButton.ParentID != restoredProductMenu.ID {
		t.Fatalf("expected product button parent %d, got %v", restoredProductMenu.ID, restoredButton.ParentID)
	}
	if restoredButton.Name != "新增商品" || restoredButton.Type != model.MenuTypeButton || restoredButton.Sort != 0 {
		t.Fatalf("unexpected restored product button: %+v", restoredButton)
	}
}

func TestSeedLicense_RestoresLicenseeLicenseAndRegistrationMenus(t *testing.T) {
	db := newSeedDB(t)
	enforcer := newSeedEnforcer(t)

	root := model.Menu{
		Name:       "许可管理",
		Type:       model.MenuTypeDirectory,
		Permission: "license",
	}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root menu: %v", err)
	}

	licenseeMenu := model.Menu{
		ParentID:   &root.ID,
		Name:       "旧主体",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/licensees",
		Icon:       "LegacyUser",
		Permission: "license:licensee:list",
		Sort:       99,
	}
	if err := db.Create(&licenseeMenu).Error; err != nil {
		t.Fatalf("create licensee menu: %v", err)
	}
	licenseeButton := model.Menu{
		ParentID:   &licenseeMenu.ID,
		Name:       "旧归档",
		Type:       model.MenuTypeMenu,
		Permission: "license:licensee:archive",
		Sort:       88,
	}
	if err := db.Create(&licenseeButton).Error; err != nil {
		t.Fatalf("create licensee button: %v", err)
	}
	if err := db.Delete(&licenseeButton).Error; err != nil {
		t.Fatalf("soft delete licensee button: %v", err)
	}

	licenseMenu := model.Menu{
		ParentID:   &root.ID,
		Name:       "旧许可",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/licenses",
		Icon:       "LegacyLicense",
		Permission: "license:license:list",
		Sort:       88,
	}
	if err := db.Create(&licenseMenu).Error; err != nil {
		t.Fatalf("create license menu: %v", err)
	}
	if err := db.Delete(&licenseMenu).Error; err != nil {
		t.Fatalf("soft delete license menu: %v", err)
	}

	regMenu := model.Menu{
		ParentID:   &root.ID,
		Name:       "旧注册码",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/registrations",
		Icon:       "LegacyReg",
		Permission: "license:registration:list",
		Sort:       77,
	}
	if err := db.Create(&regMenu).Error; err != nil {
		t.Fatalf("create reg menu: %v", err)
	}
	regButton := model.Menu{
		ParentID:   &regMenu.ID,
		Name:       "旧生成注册码",
		Type:       model.MenuTypeMenu,
		Permission: "license:registration:generate",
		Sort:       66,
	}
	if err := db.Create(&regButton).Error; err != nil {
		t.Fatalf("create reg button: %v", err)
	}
	if err := db.Delete(&regButton).Error; err != nil {
		t.Fatalf("soft delete reg button: %v", err)
	}

	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("seed license: %v", err)
	}

	assertMenu := func(permission, name, path, icon string, menuType model.MenuType, sort int) model.Menu {
		t.Helper()
		var menu model.Menu
		if err := db.Where("permission = ?", permission).First(&menu).Error; err != nil {
			t.Fatalf("find menu %s: %v", permission, err)
		}
		if menu.Name != name || menu.Type != menuType || menu.Path != path || menu.Icon != icon || menu.Sort != sort {
			t.Fatalf("unexpected menu %s: %+v", permission, menu)
		}
		return menu
	}

	restoredLicensee := assertMenu("license:licensee:list", "授权主体", "/license/licensees", "Building2", model.MenuTypeMenu, 1)
	restoredLicense := assertMenu("license:license:list", "许可签发", "/license/licenses", "FileBadge", model.MenuTypeMenu, 2)
	restoredReg := assertMenu("license:registration:list", "注册码管理", "/license/registrations", "Ticket", model.MenuTypeMenu, 3)

	var restoredArchive model.Menu
	if err := db.Where("permission = ?", "license:licensee:archive").First(&restoredArchive).Error; err != nil {
		t.Fatalf("find restored archive button: %v", err)
	}
	if restoredArchive.ParentID == nil || *restoredArchive.ParentID != restoredLicensee.ID || restoredArchive.Type != model.MenuTypeButton || restoredArchive.Sort != 2 {
		t.Fatalf("unexpected restored archive button: %+v", restoredArchive)
	}

	var restoredGenerate model.Menu
	if err := db.Where("permission = ?", "license:registration:generate").First(&restoredGenerate).Error; err != nil {
		t.Fatalf("find restored generate button: %v", err)
	}
	if restoredGenerate.ParentID == nil || *restoredGenerate.ParentID != restoredReg.ID || restoredGenerate.Type != model.MenuTypeButton || restoredGenerate.Sort != 1 {
		t.Fatalf("unexpected restored generate button: %+v", restoredGenerate)
	}

	allowed, err := enforcer.Enforce("admin", "/api/v1/license/registrations/generate", "POST")
	if err != nil {
		t.Fatalf("enforce registration generate policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected registration generate API policy to be seeded")
	}

	allowed, err = enforcer.Enforce("admin", "license:license:reactivate", "read")
	if err != nil {
		t.Fatalf("enforce license reactivate menu policy: %v", err)
	}
	if !allowed {
		t.Fatal("expected license reactivate menu permission to be seeded")
	}

	if restoredLicense.ParentID == nil || *restoredLicense.ParentID != root.ID {
		t.Fatalf("expected license menu under root, got %+v", restoredLicense)
	}
}

func TestSeedLicense_RestoresOperationalButtonsInPlace(t *testing.T) {
	db := newSeedDB(t)
	enforcer := newSeedEnforcer(t)

	root := model.Menu{
		Name:       "许可管理",
		Type:       model.MenuTypeDirectory,
		Permission: "license",
	}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root menu: %v", err)
	}

	licenseMenu := model.Menu{
		ParentID:   &root.ID,
		Name:       "旧许可签发",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/licenses",
		Icon:       "LegacyLicense",
		Permission: "license:license:list",
		Sort:       98,
	}
	if err := db.Create(&licenseMenu).Error; err != nil {
		t.Fatalf("create drifted license menu: %v", err)
	}

	regMenu := model.Menu{
		ParentID:   &root.ID,
		Name:       "旧注册码",
		Type:       model.MenuTypeDirectory,
		Path:       "/legacy/registrations",
		Icon:       "LegacyReg",
		Permission: "license:registration:list",
		Sort:       77,
	}
	if err := db.Create(&regMenu).Error; err != nil {
		t.Fatalf("create drifted reg menu: %v", err)
	}

	wrongParent := model.Menu{
		Name:       "错误父菜单",
		Type:       model.MenuTypeDirectory,
		Permission: "license:wrong-parent-buttons",
	}
	if err := db.Create(&wrongParent).Error; err != nil {
		t.Fatalf("create wrong parent: %v", err)
	}

	buttons := []model.Menu{
		{
			ParentID:   &wrongParent.ID,
			Name:       "旧恢复许可",
			Type:       model.MenuTypeMenu,
			Permission: "license:license:reactivate",
			Sort:       99,
		},
		{
			ParentID:   &wrongParent.ID,
			Name:       "旧升级许可",
			Type:       model.MenuTypeMenu,
			Permission: "license:license:upgrade",
			Sort:       88,
		},
		{
			ParentID:   &wrongParent.ID,
			Name:       "旧新增注册码",
			Type:       model.MenuTypeMenu,
			Permission: "license:registration:create",
			Sort:       66,
		},
	}
	for i := range buttons {
		if err := db.Create(&buttons[i]).Error; err != nil {
			t.Fatalf("create drifted button %s: %v", buttons[i].Permission, err)
		}
		if err := db.Delete(&buttons[i]).Error; err != nil {
			t.Fatalf("soft delete drifted button %s: %v", buttons[i].Permission, err)
		}
	}

	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("seed license: %v", err)
	}

	var restoredLicenseMenu model.Menu
	if err := db.Where("permission = ?", "license:license:list").First(&restoredLicenseMenu).Error; err != nil {
		t.Fatalf("find restored license menu: %v", err)
	}
	if restoredLicenseMenu.ParentID == nil || *restoredLicenseMenu.ParentID != root.ID || restoredLicenseMenu.Name != "许可签发" || restoredLicenseMenu.Path != "/license/licenses" {
		t.Fatalf("unexpected restored license menu: %+v", restoredLicenseMenu)
	}

	var restoredRegMenu model.Menu
	if err := db.Where("permission = ?", "license:registration:list").First(&restoredRegMenu).Error; err != nil {
		t.Fatalf("find restored reg menu: %v", err)
	}
	if restoredRegMenu.ParentID == nil || *restoredRegMenu.ParentID != root.ID || restoredRegMenu.Name != "注册码管理" || restoredRegMenu.Path != "/license/registrations" {
		t.Fatalf("unexpected restored reg menu: %+v", restoredRegMenu)
	}

	assertButton := func(permission, name string, parentID uint, sort int) {
		t.Helper()
		var menu model.Menu
		if err := db.Where("permission = ?", permission).First(&menu).Error; err != nil {
			t.Fatalf("find button %s: %v", permission, err)
		}
		if menu.ParentID == nil || *menu.ParentID != parentID || menu.Name != name || menu.Type != model.MenuTypeButton || menu.Sort != sort {
			t.Fatalf("unexpected button %s: %+v", permission, menu)
		}
	}

	assertButton("license:license:reactivate", "恢复许可", restoredLicenseMenu.ID, 5)
	assertButton("license:license:upgrade", "升级许可", restoredLicenseMenu.ID, 3)
	assertButton("license:registration:create", "新增注册码", restoredRegMenu.ID, 0)

	allowed, err := enforcer.Enforce("admin", "license:license:upgrade", "read")
	if err != nil {
		t.Fatalf("enforce license upgrade menu permission: %v", err)
	}
	if !allowed {
		t.Fatal("expected license upgrade menu permission to be seeded")
	}
}

func TestSeedLicense_RestoresRootDirectoryToTopLevel(t *testing.T) {
	db := newSeedDB(t)
	enforcer := newSeedEnforcer(t)

	parent := model.Menu{
		Name:       "错误上级",
		Type:       model.MenuTypeDirectory,
		Permission: "license:wrong-root-parent",
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create wrong parent: %v", err)
	}

	root := model.Menu{
		ParentID:   &parent.ID,
		Name:       "旧许可管理",
		Type:       model.MenuTypeMenu,
		Path:       "/legacy/license-root",
		Icon:       "LegacyKey",
		Permission: "license",
		Sort:       999,
	}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create drifted root: %v", err)
	}
	rootID := root.ID
	if err := db.Delete(&root).Error; err != nil {
		t.Fatalf("soft delete drifted root: %v", err)
	}

	if err := SeedLicense(db, enforcer); err != nil {
		t.Fatalf("seed license: %v", err)
	}

	var restored model.Menu
	if err := db.Where("permission = ?", "license").First(&restored).Error; err != nil {
		t.Fatalf("find restored root: %v", err)
	}
	if restored.ID != rootID {
		t.Fatalf("expected root restored in place, got original=%d restored=%d", rootID, restored.ID)
	}
	if restored.ParentID != nil {
		t.Fatalf("expected restored root to be top-level, got parent=%v", restored.ParentID)
	}
	if restored.Name != "许可管理" || restored.Type != model.MenuTypeDirectory || restored.Path != "" || restored.Icon != "KeyRound" || restored.Sort != 200 {
		t.Fatalf("unexpected restored root: %+v", restored)
	}

	var productMenu model.Menu
	if err := db.Where("permission = ?", "license:product:list").First(&productMenu).Error; err != nil {
		t.Fatalf("find seeded product menu: %v", err)
	}
	if productMenu.ParentID == nil || *productMenu.ParentID != restored.ID {
		t.Fatalf("expected product menu under restored root %d, got %+v", restored.ID, productMenu.ParentID)
	}
}

func TestSeedLicense_AllowsNilEnforcerWhileStillSeedingMenus(t *testing.T) {
	db := newSeedDB(t)

	if err := SeedLicense(db, nil); err != nil {
		t.Fatalf("SeedLicense with nil enforcer: %v", err)
	}

	var root model.Menu
	if err := db.Where("permission = ?", "license").First(&root).Error; err != nil {
		t.Fatalf("find seeded root: %v", err)
	}
	if root.Name != "许可管理" || root.Type != model.MenuTypeDirectory {
		t.Fatalf("unexpected seeded root: %+v", root)
	}

	var productButton model.Menu
	if err := db.Where("permission = ?", "license:product:create").First(&productButton).Error; err != nil {
		t.Fatalf("find seeded product button: %v", err)
	}
	if productButton.ParentID == nil || productButton.Type != model.MenuTypeButton {
		t.Fatalf("unexpected seeded product button: %+v", productButton)
	}
}
