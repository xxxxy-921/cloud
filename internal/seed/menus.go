package seed

import "metis/internal/model"

// Helper to create a uint pointer.
func uintPtr(v uint) *uint { return &v }

// BuiltinMenus defines the default menu tree.
// ParentID will be resolved dynamically during seeding based on parent Permission.
type MenuSeed struct {
	model.Menu
	ParentPermission string // used to resolve ParentID during seeding
	Children         []MenuSeed
}

var BuiltinMenus = []MenuSeed{
	{
		Menu: model.Menu{
			Name: "首页", Type: model.MenuTypeMenu,
			Path: "/", Icon: "Home", Permission: "home", Sort: 0,
		},
		Children: nil,
	},
	{
		Menu: model.Menu{
			Name: "系统管理", Type: model.MenuTypeDirectory,
			Icon: "Settings", Permission: "system", Sort: 100,
		},
		Children: []MenuSeed{
			{
				Menu: model.Menu{
					Name: "用户管理", Type: model.MenuTypeMenu,
					Path: "/users", Icon: "Users", Permission: "system:user:list", Sort: 0,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增用户", Type: model.MenuTypeButton, Permission: "system:user:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑用户", Type: model.MenuTypeButton, Permission: "system:user:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除用户", Type: model.MenuTypeButton, Permission: "system:user:delete", Sort: 2}},
					{Menu: model.Menu{Name: "重置密码", Type: model.MenuTypeButton, Permission: "system:user:reset-password", Sort: 3}},
				},
			},
			{
				Menu: model.Menu{
					Name: "角色管理", Type: model.MenuTypeMenu,
					Path: "/roles", Icon: "Shield", Permission: "system:role:list", Sort: 1,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增角色", Type: model.MenuTypeButton, Permission: "system:role:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑角色", Type: model.MenuTypeButton, Permission: "system:role:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除角色", Type: model.MenuTypeButton, Permission: "system:role:delete", Sort: 2}},
					{Menu: model.Menu{Name: "分配权限", Type: model.MenuTypeButton, Permission: "system:role:assign", Sort: 3}},
				},
			},
			{
				Menu: model.Menu{
					Name: "菜单管理", Type: model.MenuTypeMenu,
					Path: "/menus", Icon: "Menu", Permission: "system:menu:list", Sort: 2,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增菜单", Type: model.MenuTypeButton, Permission: "system:menu:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑菜单", Type: model.MenuTypeButton, Permission: "system:menu:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除菜单", Type: model.MenuTypeButton, Permission: "system:menu:delete", Sort: 2}},
				},
			},
			{
				Menu: model.Menu{
					Name: "会话管理", Type: model.MenuTypeMenu,
					Path: "/sessions", Icon: "Monitor", Permission: "system:session:list", Sort: 3,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "踢出会话", Type: model.MenuTypeButton, Permission: "system:session:delete", Sort: 0}},
				},
			},
			{
				Menu: model.Menu{
					Name: "系统设置", Type: model.MenuTypeMenu,
					Path: "/settings", Icon: "Wrench", Permission: "system:settings:list", Sort: 4,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "编辑设置", Type: model.MenuTypeButton, Permission: "system:settings:update", Sort: 0}},
				},
			},
			{
				Menu: model.Menu{
					Name: "任务中心", Type: model.MenuTypeMenu,
					Path: "/tasks", Icon: "Clock", Permission: "system:task:list", Sort: 5,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "暂停任务", Type: model.MenuTypeButton, Permission: "system:task:pause", Sort: 0}},
					{Menu: model.Menu{Name: "恢复任务", Type: model.MenuTypeButton, Permission: "system:task:resume", Sort: 1}},
					{Menu: model.Menu{Name: "触发任务", Type: model.MenuTypeButton, Permission: "system:task:trigger", Sort: 2}},
				},
			},
			{
				Menu: model.Menu{
					Name: "公告管理", Type: model.MenuTypeMenu,
					Path: "/announcements", Icon: "Megaphone", Permission: "system:announcement:list", Sort: 6,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增公告", Type: model.MenuTypeButton, Permission: "system:announcement:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑公告", Type: model.MenuTypeButton, Permission: "system:announcement:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除公告", Type: model.MenuTypeButton, Permission: "system:announcement:delete", Sort: 2}},
				},
			},
			{
				Menu: model.Menu{
					Name: "消息通道", Type: model.MenuTypeMenu,
					Path: "/channels", Icon: "Mail", Permission: "system:channel:list", Sort: 7,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增通道", Type: model.MenuTypeButton, Permission: "system:channel:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑通道", Type: model.MenuTypeButton, Permission: "system:channel:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除通道", Type: model.MenuTypeButton, Permission: "system:channel:delete", Sort: 2}},
				},
			},
			{
				Menu: model.Menu{
					Name: "认证源", Type: model.MenuTypeMenu,
					Path: "/auth-providers", Icon: "KeyRound", Permission: "system:auth-provider:list", Sort: 8,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "编辑认证源", Type: model.MenuTypeButton, Permission: "system:auth-provider:update", Sort: 0}},
				},
			},
			{
				Menu: model.Menu{
					Name: "审计日志", Type: model.MenuTypeMenu,
					Path: "/audit-logs", Icon: "ClipboardList", Permission: "system:audit-log:list", Sort: 9,
				},
			},
			{
				Menu: model.Menu{
					Name: "身份源管理", Type: model.MenuTypeMenu,
					Path: "/identity-sources", Icon: "Fingerprint", Permission: "system:identity-source:list", Sort: 10,
				},
				Children: []MenuSeed{
					{Menu: model.Menu{Name: "新增身份源", Type: model.MenuTypeButton, Permission: "system:identity-source:create", Sort: 0}},
					{Menu: model.Menu{Name: "编辑身份源", Type: model.MenuTypeButton, Permission: "system:identity-source:update", Sort: 1}},
					{Menu: model.Menu{Name: "删除身份源", Type: model.MenuTypeButton, Permission: "system:identity-source:delete", Sort: 2}},
				},
			},
		},
	},
}
