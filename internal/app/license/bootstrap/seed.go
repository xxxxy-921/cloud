package bootstrap

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"

	"metis/internal/model"
)

func SeedLicense(db *gorm.DB, enforcer *casbin.Enforcer) error {
	seedMenu := func(parentID *uint, name string, menuType model.MenuType, path, icon, permission string, sort int) (*model.Menu, error) {
		var menu model.Menu
		if err := db.Unscoped().Where("permission = ?", permission).First(&menu).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
			menu = model.Menu{
				ParentID:   parentID,
				Name:       name,
				Type:       menuType,
				Path:       path,
				Icon:       icon,
				Permission: permission,
				Sort:       sort,
			}
			if err := db.Create(&menu).Error; err != nil {
				return nil, err
			}
			slog.Info("seed: created menu", "name", menu.Name, "permission", menu.Permission)
			return &menu, nil
		}
		if menu.DeletedAt.Valid || menu.Name != name || menu.Type != menuType || menu.Path != path || menu.Icon != icon || menu.Sort != sort || menu.ParentID != nil && parentID == nil || menu.ParentID == nil && parentID != nil || (menu.ParentID != nil && parentID != nil && *menu.ParentID != *parentID) {
			if err := db.Unscoped().Model(&menu).Updates(map[string]any{
				"parent_id":  parentID,
				"name":       name,
				"type":       menuType,
				"path":       path,
				"icon":       icon,
				"sort":       sort,
				"deleted_at": nil,
			}).Error; err != nil {
				return nil, err
			}
		}
		return &menu, nil
	}

	seedButton := func(parentID uint, name, permission string, sort int) error {
		var menu model.Menu
		if err := db.Unscoped().Where("permission = ?", permission).First(&menu).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			menu = model.Menu{
				ParentID:   &parentID,
				Name:       name,
				Type:       model.MenuTypeButton,
				Permission: permission,
				Sort:       sort,
			}
			if err := db.Create(&menu).Error; err != nil {
				return err
			}
			slog.Info("seed: created menu", "name", menu.Name, "permission", menu.Permission)
			return nil
		}
		if menu.DeletedAt.Valid || menu.Name != name || menu.Type != model.MenuTypeButton || menu.Sort != sort || menu.ParentID == nil || *menu.ParentID != parentID {
			return db.Unscoped().Model(&menu).Updates(map[string]any{
				"parent_id":  parentID,
				"name":       name,
				"type":       model.MenuTypeButton,
				"sort":       sort,
				"deleted_at": nil,
			}).Error
		}
		return nil
	}

	// 1. Seed menus: 「许可管理」directory + 「商品管理」menu
	licenseDir, err := seedMenu(nil, "许可管理", model.MenuTypeDirectory, "", "KeyRound", "license", 200)
	if err != nil {
		return err
	}

	productMenu, err := seedMenu(&licenseDir.ID, "商品管理", model.MenuTypeMenu, "/license/products", "Package", "license:product:list", 0)
	if err != nil {
		return err
	}

	// Seed button permissions under product menu
	buttons := []model.Menu{
		{Name: "新增商品", Type: model.MenuTypeButton, Permission: "license:product:create", Sort: 0},
		{Name: "编辑商品", Type: model.MenuTypeButton, Permission: "license:product:update", Sort: 1},
		{Name: "管理套餐", Type: model.MenuTypeButton, Permission: "license:plan:manage", Sort: 2},
		{Name: "管理密钥", Type: model.MenuTypeButton, Permission: "license:key:manage", Sort: 3},
	}
	for _, btn := range buttons {
		if err := seedButton(productMenu.ID, btn.Name, btn.Permission, btn.Sort); err != nil {
			slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
			continue
		}
	}

	// 3. Seed 「授权主体」menu under 「许可管理」
	licenseeMenu, err := seedMenu(&licenseDir.ID, "授权主体", model.MenuTypeMenu, "/license/licensees", "Building2", "license:licensee:list", 1)
	if err != nil {
		return err
	}

	// Seed button permissions under licensee menu
	licenseeButtons := []model.Menu{
		{Name: "新增授权主体", Type: model.MenuTypeButton, Permission: "license:licensee:create", Sort: 0},
		{Name: "编辑授权主体", Type: model.MenuTypeButton, Permission: "license:licensee:update", Sort: 1},
		{Name: "归档授权主体", Type: model.MenuTypeButton, Permission: "license:licensee:archive", Sort: 2},
	}
	for _, btn := range licenseeButtons {
		if err := seedButton(licenseeMenu.ID, btn.Name, btn.Permission, btn.Sort); err != nil {
			slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
			continue
		}
	}

	// 3. Seed 「许可签发」menu under 「许可管理」
	licenseMenu, err := seedMenu(&licenseDir.ID, "许可签发", model.MenuTypeMenu, "/license/licenses", "FileBadge", "license:license:list", 2)
	if err != nil {
		return err
	}

	// Seed button permissions under license menu
	licenseButtons := []model.Menu{
		{Name: "签发许可", Type: model.MenuTypeButton, Permission: "license:license:issue", Sort: 0},
		{Name: "吊销许可", Type: model.MenuTypeButton, Permission: "license:license:revoke", Sort: 1},
		{Name: "续期许可", Type: model.MenuTypeButton, Permission: "license:license:renew", Sort: 2},
		{Name: "升级许可", Type: model.MenuTypeButton, Permission: "license:license:upgrade", Sort: 3},
		{Name: "暂停许可", Type: model.MenuTypeButton, Permission: "license:license:suspend", Sort: 4},
		{Name: "恢复许可", Type: model.MenuTypeButton, Permission: "license:license:reactivate", Sort: 5},
	}
	for _, btn := range licenseButtons {
		if err := seedButton(licenseMenu.ID, btn.Name, btn.Permission, btn.Sort); err != nil {
			slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
			continue
		}
	}

	// 4. Seed 「注册码管理」menu under 「许可管理」
	regMenu, err := seedMenu(&licenseDir.ID, "注册码管理", model.MenuTypeMenu, "/license/registrations", "Ticket", "license:registration:list", 3)
	if err != nil {
		return err
	}

	regButtons := []model.Menu{
		{Name: "新增注册码", Type: model.MenuTypeButton, Permission: "license:registration:create", Sort: 0},
		{Name: "自动生成注册码", Type: model.MenuTypeButton, Permission: "license:registration:generate", Sort: 1},
	}
	for _, btn := range regButtons {
		if err := seedButton(regMenu.ID, btn.Name, btn.Permission, btn.Sort); err != nil {
			slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
			continue
		}
	}

	// 2. Seed Casbin policies for admin role
	policies := [][]string{
		// Products
		{"admin", "/api/v1/license/products", "GET"},
		{"admin", "/api/v1/license/products", "POST"},
		{"admin", "/api/v1/license/products/:id", "GET"},
		{"admin", "/api/v1/license/products/:id", "PUT"},
		{"admin", "/api/v1/license/products/:id/schema", "PUT"},
		{"admin", "/api/v1/license/products/:id/status", "PATCH"},
		{"admin", "/api/v1/license/products/:id/rotate-key", "POST"},
		{"admin", "/api/v1/license/products/:id/rotate-key-impact", "GET"},
		{"admin", "/api/v1/license/products/:id/bulk-reissue", "POST"},
		{"admin", "/api/v1/license/products/:id/public-key", "GET"},
		// Plans
		{"admin", "/api/v1/license/products/:id/plans", "POST"},
		{"admin", "/api/v1/license/plans/:id", "PUT"},
		{"admin", "/api/v1/license/plans/:id", "DELETE"},
		{"admin", "/api/v1/license/plans/:id/default", "PATCH"},
		// Licensees
		{"admin", "/api/v1/license/licensees", "GET"},
		{"admin", "/api/v1/license/licensees", "POST"},
		{"admin", "/api/v1/license/licensees/:id", "GET"},
		{"admin", "/api/v1/license/licensees/:id", "PUT"},
		{"admin", "/api/v1/license/licensees/:id/status", "PATCH"},
		// Licenses
		{"admin", "/api/v1/license/licenses", "GET"},
		{"admin", "/api/v1/license/licenses", "POST"},
		{"admin", "/api/v1/license/licenses/:id", "GET"},
		{"admin", "/api/v1/license/licenses/:id/renew", "POST"},
		{"admin", "/api/v1/license/licenses/:id/upgrade", "POST"},
		{"admin", "/api/v1/license/licenses/:id/suspend", "POST"},
		{"admin", "/api/v1/license/licenses/:id/reactivate", "POST"},
		{"admin", "/api/v1/license/licenses/:id/revoke", "PATCH"},
		{"admin", "/api/v1/license/licenses/:id/export", "GET"},
		// Registrations
		{"admin", "/api/v1/license/registrations", "GET"},
		{"admin", "/api/v1/license/registrations", "POST"},
		{"admin", "/api/v1/license/registrations/generate", "POST"},
	}

	// Also add menu permissions
	menuPerms := [][]string{
		{"admin", "license", "read"},
		{"admin", "license:product:list", "read"},
		{"admin", "license:product:create", "read"},
		{"admin", "license:product:update", "read"},
		{"admin", "license:plan:manage", "read"},
		{"admin", "license:key:manage", "read"},
		{"admin", "license:licensee:list", "read"},
		{"admin", "license:licensee:create", "read"},
		{"admin", "license:licensee:update", "read"},
		{"admin", "license:licensee:archive", "read"},
		{"admin", "license:license:list", "read"},
		{"admin", "license:license:issue", "read"},
		{"admin", "license:license:revoke", "read"},
		{"admin", "license:license:renew", "read"},
		{"admin", "license:license:upgrade", "read"},
		{"admin", "license:license:suspend", "read"},
		{"admin", "license:license:reactivate", "read"},
		{"admin", "license:registration:list", "read"},
		{"admin", "license:registration:create", "read"},
		{"admin", "license:registration:generate", "read"},
	}

	allPolicies := append(policies, menuPerms...)
	if enforcer == nil {
		return nil
	}
	for _, p := range allPolicies {
		if has, _ := enforcer.HasPolicy(p); !has {
			if _, err := enforcer.AddPolicy(p); err != nil {
				slog.Error("seed: failed to add policy", "policy", p, "error", err)
			}
		}
	}

	return nil
}
