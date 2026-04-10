package seed

import "metis/internal/model"

var BuiltinRoles = []model.Role{
	{
		Name:        "管理员",
		Code:        model.RoleAdmin,
		Description: "系统管理员，拥有全部权限",
		Sort:        0,
		IsSystem:    true,
	},
	{
		Name:        "普通用户",
		Code:        model.RoleUser,
		Description: "普通用户，拥有基本权限",
		Sort:        1,
		IsSystem:    true,
	},
}
