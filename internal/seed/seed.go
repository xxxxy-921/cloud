package seed

import (
	"fmt"
	"log/slog"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"

	"metis/internal/model"
)

type Result struct {
	RolesCreated   int
	RolesSkipped   int
	MenusCreated   int
	MenusSkipped   int
	PoliciesAdded  int
}

func Run(db *gorm.DB, enforcer *casbin.Enforcer) (*Result, error) {
	r := &Result{}

	// 1. Seed roles
	roleMap := make(map[string]*model.Role) // code -> role
	for _, seed := range BuiltinRoles {
		var existing model.Role
		if err := db.Where("code = ?", seed.Code).First(&existing).Error; err == nil {
			r.RolesSkipped++
			roleMap[existing.Code] = &existing
			continue
		}
		role := seed // copy
		if err := db.Create(&role).Error; err != nil {
			return nil, fmt.Errorf("create role %s: %w", seed.Code, err)
		}
		r.RolesCreated++
		roleMap[role.Code] = &role
		slog.Info("seed: created role", "code", role.Code)
	}

	// 2. Seed menus
	seedMenuTree(db, BuiltinMenus, nil, r)

	// 3. Migrate existing users
	MigrateUserRoles(db, roleMap)

	// 4. Seed Casbin policies
	seedPolicies(db, enforcer, roleMap, r)

	// 5. Seed default system configs
	seedDefaultConfigs(db)

	// 6. Seed default auth providers
	seedAuthProviders(db)

	return r, nil
}

func seedMenuTree(db *gorm.DB, menus []MenuSeed, parentID *uint, r *Result) {
	for _, seed := range menus {
		var existing model.Menu
		if seed.Permission != "" {
			if err := db.Where("permission = ?", seed.Permission).First(&existing).Error; err == nil {
				r.MenusSkipped++
				seedMenuTree(db, seed.Children, &existing.ID, r)
				continue
			}
		}

		menu := seed.Menu
		menu.ParentID = parentID
		if err := db.Create(&menu).Error; err != nil {
			slog.Error("seed: failed to create menu", "name", seed.Name, "error", err)
			continue
		}
		r.MenusCreated++
		slog.Info("seed: created menu", "name", menu.Name, "permission", menu.Permission)

		seedMenuTree(db, seed.Children, &menu.ID, r)
	}
}

func seedPolicies(db *gorm.DB, enforcer *casbin.Enforcer, roleMap map[string]*model.Role, r *Result) {
	// Collect all menu permissions for admin
	var allMenus []model.Menu
	db.Find(&allMenus)

	// Admin: all menu permissions + all API policies
	var adminPolicies [][]string
	for _, m := range allMenus {
		if m.Permission != "" {
			adminPolicies = append(adminPolicies, []string{"admin", m.Permission, "read"})
		}
	}
	for _, p := range AdminAPIPolicies {
		adminPolicies = append(adminPolicies, []string{"admin", p[0], p[1]})
	}

	// Clear existing admin policies and re-add
	enforcer.RemoveFilteredPolicy(0, "admin")
	if len(adminPolicies) > 0 {
		added, _ := enforcer.AddPolicies(adminPolicies)
		if added {
			r.PoliciesAdded += len(adminPolicies)
		}
	}

	// User: basic permissions
	var userPolicies [][]string
	for _, p := range UserAPIPolicies {
		userPolicies = append(userPolicies, []string{"user", p[0], p[1]})
	}

	enforcer.RemoveFilteredPolicy(0, "user")
	if len(userPolicies) > 0 {
		added, _ := enforcer.AddPolicies(userPolicies)
		if added {
			r.PoliciesAdded += len(userPolicies)
		}
	}
}

// defaultConfigs are seeded if they don't already exist.
var defaultConfigs = []model.SystemConfig{
	{Key: "scheduler.history_retention_days", Value: "30", Remark: "任务执行历史保留天数，0 表示永不清理"},
	{Key: "security.max_concurrent_sessions", Value: "5", Remark: "每用户最大并发会话数，0 表示不限制"},
	{Key: "audit.retention_days_auth", Value: "90", Remark: "登录活动日志保留天数，0 表示永不清理"},
	{Key: "audit.retention_days_operation", Value: "365", Remark: "操作记录日志保留天数，0 表示永不清理"},
	{Key: "audit.retention_days_application", Value: "30", Remark: "系统事件日志保留天数，0 表示永不清理"},
	{Key: "security.password_min_length", Value: "8", Remark: "密码最小长度"},
	{Key: "security.password_require_upper", Value: "false", Remark: "密码是否要求大写字母"},
	{Key: "security.password_require_lower", Value: "false", Remark: "密码是否要求小写字母"},
	{Key: "security.password_require_number", Value: "false", Remark: "密码是否要求数字"},
	{Key: "security.password_require_special", Value: "false", Remark: "密码是否要求特殊字符"},
	{Key: "security.password_expiry_days", Value: "0", Remark: "密码过期天数，0 表示永不过期"},
	{Key: "security.login_max_attempts", Value: "5", Remark: "最大登录失败次数，0 表示不限制"},
	{Key: "security.login_lockout_minutes", Value: "30", Remark: "登录锁定时长（分钟）"},
	{Key: "security.session_timeout_minutes", Value: "10080", Remark: "会话超时时间（分钟），默认 7 天"},
	{Key: "security.require_two_factor", Value: "false", Remark: "是否强制所有用户启用两步验证"},
	{Key: "security.registration_open", Value: "false", Remark: "是否开放用户注册"},
	{Key: "security.default_role_code", Value: "", Remark: "注册用户默认角色代码"},
	{Key: "security.captcha_provider", Value: "none", Remark: "验证码类型：none, image"},
}

func seedDefaultConfigs(db *gorm.DB) {
	for _, cfg := range defaultConfigs {
		var existing model.SystemConfig
		if err := db.Where("`key` = ?", cfg.Key).First(&existing).Error; err == nil {
			continue // already exists, skip
		}
		if err := db.Create(&cfg).Error; err != nil {
			slog.Error("seed: failed to create config", "key", cfg.Key, "error", err)
			continue
		}
		slog.Info("seed: created config", "key", cfg.Key)
	}
}

var defaultAuthProviders = []model.AuthProvider{
	{ProviderKey: "github", DisplayName: "GitHub", SortOrder: 1},
	{ProviderKey: "google", DisplayName: "Google", SortOrder: 2},
}

func seedAuthProviders(db *gorm.DB) {
	for _, p := range defaultAuthProviders {
		var existing model.AuthProvider
		if err := db.Where("provider_key = ?", p.ProviderKey).First(&existing).Error; err == nil {
			continue
		}
		if err := db.Create(&p).Error; err != nil {
			slog.Error("seed: failed to create auth provider", "key", p.ProviderKey, "error", err)
			continue
		}
		slog.Info("seed: created auth provider", "key", p.ProviderKey)
	}
}
