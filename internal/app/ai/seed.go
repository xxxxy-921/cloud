package ai

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"

	"metis/internal/model"
)

func seedAI(db *gorm.DB, enforcer *casbin.Enforcer) error {
	// 1. AI 管理目录
	var aiDir model.Menu
	if err := db.Where("permission = ?", "ai").First(&aiDir).Error; err != nil {
		aiDir = model.Menu{
			Name:       "AI 管理",
			Type:       model.MenuTypeDirectory,
			Icon:       "Brain",
			Permission: "ai",
			Sort:       100,
		}
		if err := db.Create(&aiDir).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", aiDir.Name, "permission", aiDir.Permission)
	}

	// 2. 供应商管理菜单（含 inline 模型管理）
	var providerMenu model.Menu
	if err := db.Where("permission = ?", "ai:provider:list").First(&providerMenu).Error; err != nil {
		providerMenu = model.Menu{
			ParentID:   &aiDir.ID,
			Name:       "供应商管理",
			Type:       model.MenuTypeMenu,
			Path:       "/ai/providers",
			Icon:       "Server",
			Permission: "ai:provider:list",
			Sort:       0,
		}
		if err := db.Create(&providerMenu).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", providerMenu.Name, "permission", providerMenu.Permission)
	}

	buttons := []model.Menu{
		{Name: "新增供应商", Type: model.MenuTypeButton, Permission: "ai:provider:create", Sort: 0},
		{Name: "编辑供应商", Type: model.MenuTypeButton, Permission: "ai:provider:update", Sort: 1},
		{Name: "删除供应商", Type: model.MenuTypeButton, Permission: "ai:provider:delete", Sort: 2},
		{Name: "连通测试", Type: model.MenuTypeButton, Permission: "ai:provider:test", Sort: 3},
		{Name: "新增模型", Type: model.MenuTypeButton, Permission: "ai:model:create", Sort: 4},
		{Name: "编辑模型", Type: model.MenuTypeButton, Permission: "ai:model:update", Sort: 5},
		{Name: "删除模型", Type: model.MenuTypeButton, Permission: "ai:model:delete", Sort: 6},
		{Name: "设为默认", Type: model.MenuTypeButton, Permission: "ai:model:default", Sort: 7},
		{Name: "同步模型", Type: model.MenuTypeButton, Permission: "ai:model:sync", Sort: 8},
	}
	for _, btn := range buttons {
		var existing model.Menu
		if err := db.Where("permission = ?", btn.Permission).First(&existing).Error; err != nil {
			btn.ParentID = &providerMenu.ID
			if err := db.Create(&btn).Error; err != nil {
				slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
				continue
			}
			slog.Info("seed: created menu", "name", btn.Name, "permission", btn.Permission)
		}
	}

	// 3. Casbin policies
	policies := [][]string{
		// Providers
		{"admin", "/api/v1/ai/providers", "GET"},
		{"admin", "/api/v1/ai/providers", "POST"},
		{"admin", "/api/v1/ai/providers/:id", "GET"},
		{"admin", "/api/v1/ai/providers/:id", "PUT"},
		{"admin", "/api/v1/ai/providers/:id", "DELETE"},
		{"admin", "/api/v1/ai/providers/:id/test", "POST"},
		{"admin", "/api/v1/ai/providers/:id/sync-models", "POST"},
		// Models
		{"admin", "/api/v1/ai/models", "GET"},
		{"admin", "/api/v1/ai/models", "POST"},
		{"admin", "/api/v1/ai/models/:id", "GET"},
		{"admin", "/api/v1/ai/models/:id", "PUT"},
		{"admin", "/api/v1/ai/models/:id", "DELETE"},
		{"admin", "/api/v1/ai/models/:id/default", "PATCH"},
	}

	menuPerms := [][]string{
		{"admin", "ai", "read"},
		{"admin", "ai:provider:list", "read"},
		{"admin", "ai:provider:create", "read"},
		{"admin", "ai:provider:update", "read"},
		{"admin", "ai:provider:delete", "read"},
		{"admin", "ai:provider:test", "read"},
		{"admin", "ai:model:create", "read"},
		{"admin", "ai:model:update", "read"},
		{"admin", "ai:model:delete", "read"},
		{"admin", "ai:model:default", "read"},
		{"admin", "ai:model:sync", "read"},
	}

	allPolicies := append(policies, menuPerms...)
	for _, p := range allPolicies {
		if has, _ := enforcer.HasPolicy(p); !has {
			if _, err := enforcer.AddPolicy(p); err != nil {
				slog.Error("seed: failed to add policy", "policy", p, "error", err)
			}
		}
	}

	return nil
}
