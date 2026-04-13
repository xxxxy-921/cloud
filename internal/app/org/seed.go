package org

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"

	"metis/internal/model"
)

func seedOrg(db *gorm.DB, enforcer *casbin.Enforcer) error {
	// 1. Seed menus: 组织管理 directory
	var orgDir model.Menu
	if err := db.Where("permission = ?", "org").First(&orgDir).Error; err != nil {
		orgDir = model.Menu{
			Name:       "组织管理",
			Type:       model.MenuTypeDirectory,
			Icon:       "Users",
			Permission: "org",
			Sort:       350,
		}
		if err := db.Create(&orgDir).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", orgDir.Name, "permission", orgDir.Permission)
	}

	// 部门管理 menu
	var deptMenu model.Menu
	if err := db.Where("permission = ?", "org:department:list").First(&deptMenu).Error; err != nil {
		deptMenu = model.Menu{
			ParentID:   &orgDir.ID,
			Name:       "部门管理",
			Type:       model.MenuTypeMenu,
			Path:       "/org/departments",
			Icon:       "Network",
			Permission: "org:department:list",
			Sort:       0,
		}
		if err := db.Create(&deptMenu).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", deptMenu.Name, "permission", deptMenu.Permission)
	}

	deptButtons := []model.Menu{
		{Name: "新增部门", Type: model.MenuTypeButton, Permission: "org:department:create", Sort: 0},
		{Name: "编辑部门", Type: model.MenuTypeButton, Permission: "org:department:update", Sort: 1},
		{Name: "删除部门", Type: model.MenuTypeButton, Permission: "org:department:delete", Sort: 2},
	}
	for _, btn := range deptButtons {
		var existing model.Menu
		if err := db.Where("permission = ?", btn.Permission).First(&existing).Error; err != nil {
			btn.ParentID = &deptMenu.ID
			if err := db.Create(&btn).Error; err != nil {
				slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
				continue
			}
			slog.Info("seed: created menu", "name", btn.Name, "permission", btn.Permission)
		}
	}

	// 岗位管理 menu
	var posMenu model.Menu
	if err := db.Where("permission = ?", "org:position:list").First(&posMenu).Error; err != nil {
		posMenu = model.Menu{
			ParentID:   &orgDir.ID,
			Name:       "岗位管理",
			Type:       model.MenuTypeMenu,
			Path:       "/org/positions",
			Icon:       "Briefcase",
			Permission: "org:position:list",
			Sort:       1,
		}
		if err := db.Create(&posMenu).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", posMenu.Name, "permission", posMenu.Permission)
	}

	posButtons := []model.Menu{
		{Name: "新增岗位", Type: model.MenuTypeButton, Permission: "org:position:create", Sort: 0},
		{Name: "编辑岗位", Type: model.MenuTypeButton, Permission: "org:position:update", Sort: 1},
		{Name: "删除岗位", Type: model.MenuTypeButton, Permission: "org:position:delete", Sort: 2},
	}
	for _, btn := range posButtons {
		var existing model.Menu
		if err := db.Where("permission = ?", btn.Permission).First(&existing).Error; err != nil {
			btn.ParentID = &posMenu.ID
			if err := db.Create(&btn).Error; err != nil {
				slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
				continue
			}
			slog.Info("seed: created menu", "name", btn.Name, "permission", btn.Permission)
		}
	}

	// 人员分配 menu
	var assignMenu model.Menu
	if err := db.Where("permission = ?", "org:assignment:list").First(&assignMenu).Error; err != nil {
		assignMenu = model.Menu{
			ParentID:   &orgDir.ID,
			Name:       "人员分配",
			Type:       model.MenuTypeMenu,
			Path:       "/org/assignments",
			Icon:       "UserCog",
			Permission: "org:assignment:list",
			Sort:       2,
		}
		if err := db.Create(&assignMenu).Error; err != nil {
			return err
		}
		slog.Info("seed: created menu", "name", assignMenu.Name, "permission", assignMenu.Permission)
	}

	assignButtons := []model.Menu{
		{Name: "分配岗位", Type: model.MenuTypeButton, Permission: "org:assignment:create", Sort: 0},
		{Name: "编辑分配", Type: model.MenuTypeButton, Permission: "org:assignment:update", Sort: 1},
		{Name: "移除分配", Type: model.MenuTypeButton, Permission: "org:assignment:delete", Sort: 2},
	}
	for _, btn := range assignButtons {
		var existing model.Menu
		if err := db.Where("permission = ?", btn.Permission).First(&existing).Error; err != nil {
			btn.ParentID = &assignMenu.ID
			if err := db.Create(&btn).Error; err != nil {
				slog.Error("seed: failed to create button menu", "permission", btn.Permission, "error", err)
				continue
			}
			slog.Info("seed: created menu", "name", btn.Name, "permission", btn.Permission)
		}
	}

	// 2. Seed Casbin policies for admin role
	policies := [][]string{
		// Departments
		{"admin", "/api/v1/org/departments", "POST"},
		{"admin", "/api/v1/org/departments", "GET"},
		{"admin", "/api/v1/org/departments/tree", "GET"},
		{"admin", "/api/v1/org/departments/:id", "GET"},
		{"admin", "/api/v1/org/departments/:id", "PUT"},
		{"admin", "/api/v1/org/departments/:id", "DELETE"},
		// Positions
		{"admin", "/api/v1/org/positions", "POST"},
		{"admin", "/api/v1/org/positions", "GET"},
		{"admin", "/api/v1/org/positions/:id", "GET"},
		{"admin", "/api/v1/org/positions/:id", "PUT"},
		{"admin", "/api/v1/org/positions/:id", "DELETE"},
		// Assignments
		{"admin", "/api/v1/org/users/:id/positions", "GET"},
		{"admin", "/api/v1/org/users/:id/positions", "POST"},
		{"admin", "/api/v1/org/users/:id/positions/:assignmentId", "PUT"},
		{"admin", "/api/v1/org/users/:id/positions/:assignmentId", "DELETE"},
		{"admin", "/api/v1/org/users/:id/positions/:assignmentId/primary", "PUT"},
		{"admin", "/api/v1/org/users", "GET"},
	}

	menuPerms := [][]string{
		{"admin", "org", "read"},
		{"admin", "org:department:list", "read"},
		{"admin", "org:department:create", "read"},
		{"admin", "org:department:update", "read"},
		{"admin", "org:department:delete", "read"},
		{"admin", "org:position:list", "read"},
		{"admin", "org:position:create", "read"},
		{"admin", "org:position:update", "read"},
		{"admin", "org:position:delete", "read"},
		{"admin", "org:assignment:list", "read"},
		{"admin", "org:assignment:create", "read"},
		{"admin", "org:assignment:update", "read"},
		{"admin", "org:assignment:delete", "read"},
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
