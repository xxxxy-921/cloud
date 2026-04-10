package seed

import (
	"log/slog"

	"gorm.io/gorm"

	"metis/internal/model"
)

// MigrateUserRoles migrates users from the old Role string field to the new RoleID FK.
func MigrateUserRoles(db *gorm.DB, roleMap map[string]*model.Role) {
	// Find users with RoleID=0 (not yet migrated)
	var users []model.User
	db.Where("role_id = 0 OR role_id IS NULL").Find(&users)

	if len(users) == 0 {
		return
	}

	slog.Info("seed: migrating user roles", "count", len(users))

	for _, u := range users {
		// Try to find role by the old code field
		// Since we removed the Role string field from the model,
		// we need to read it raw from the database
		var roleStr string
		row := db.Model(&model.User{}).Select("role").Where("id = ?", u.ID).Row()
		if row != nil {
			_ = row.Scan(&roleStr)
		}

		if roleStr == "" {
			roleStr = model.RoleUser // default
		}

		if role, ok := roleMap[roleStr]; ok {
			db.Model(&model.User{}).Where("id = ?", u.ID).Update("role_id", role.ID)
			slog.Info("seed: migrated user role", "userId", u.ID, "role", roleStr, "roleId", role.ID)
		}
	}
}
