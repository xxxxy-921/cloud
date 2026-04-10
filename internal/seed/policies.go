package seed

// AdminAPIPolicies defines API endpoint permissions for the admin role.
// Format: [obj, act] — sub ("admin") is added by the seeder.
var AdminAPIPolicies = [][]string{
	// Users
	{"/api/v1/users", "GET"},
	{"/api/v1/users", "POST"},
	{"/api/v1/users/:id", "GET"},
	{"/api/v1/users/:id", "PUT"},
	{"/api/v1/users/:id", "DELETE"},
	{"/api/v1/users/:id/reset-password", "POST"},
	{"/api/v1/users/:id/activate", "POST"},
	{"/api/v1/users/:id/deactivate", "POST"},
	// Roles
	{"/api/v1/roles", "GET"},
	{"/api/v1/roles", "POST"},
	{"/api/v1/roles/:id", "GET"},
	{"/api/v1/roles/:id", "PUT"},
	{"/api/v1/roles/:id", "DELETE"},
	{"/api/v1/roles/:id/permissions", "GET"},
	{"/api/v1/roles/:id/permissions", "PUT"},
	// Menus
	{"/api/v1/menus/tree", "GET"},
	{"/api/v1/menus", "POST"},
	{"/api/v1/menus/sort", "PUT"},
	{"/api/v1/menus/:id", "PUT"},
	{"/api/v1/menus/:id", "DELETE"},
	// Config → replaced by typed settings
	{"/api/v1/settings/security", "GET"},
	{"/api/v1/settings/security", "PUT"},
	{"/api/v1/settings/scheduler", "GET"},
	{"/api/v1/settings/scheduler", "PUT"},
	// Sessions
	{"/api/v1/sessions", "GET"},
	{"/api/v1/sessions/:id", "DELETE"},
	// Site info
	{"/api/v1/site-info", "PUT"},
	{"/api/v1/site-info/logo", "PUT"},
	{"/api/v1/site-info/logo", "DELETE"},
	// Tasks
	{"/api/v1/tasks", "GET"},
	{"/api/v1/tasks/stats", "GET"},
	{"/api/v1/tasks/:name", "GET"},
	{"/api/v1/tasks/:name/executions", "GET"},
	{"/api/v1/tasks/:name/pause", "POST"},
	{"/api/v1/tasks/:name/resume", "POST"},
	{"/api/v1/tasks/:name/trigger", "POST"},
	// Announcements
	{"/api/v1/announcements", "GET"},
	{"/api/v1/announcements", "POST"},
	{"/api/v1/announcements/:id", "PUT"},
	{"/api/v1/announcements/:id", "DELETE"},
	// Channels
	{"/api/v1/channels", "GET"},
	{"/api/v1/channels", "POST"},
	{"/api/v1/channels/:id", "GET"},
	{"/api/v1/channels/:id", "PUT"},
	{"/api/v1/channels/:id", "DELETE"},
	{"/api/v1/channels/:id/toggle", "PUT"},
	{"/api/v1/channels/:id/test", "POST"},
	{"/api/v1/channels/:id/send-test", "POST"},
	// Auth providers
	{"/api/v1/admin/auth-providers", "GET"},
	{"/api/v1/admin/auth-providers/:key", "PUT"},
	{"/api/v1/admin/auth-providers/:key/toggle", "PATCH"},
	// Audit logs
	{"/api/v1/audit-logs", "GET"},
	// Identity sources
	{"/api/v1/identity-sources", "GET"},
	{"/api/v1/identity-sources", "POST"},
	{"/api/v1/identity-sources/:id", "PUT"},
	{"/api/v1/identity-sources/:id", "DELETE"},
	{"/api/v1/identity-sources/:id/toggle", "PATCH"},
	{"/api/v1/identity-sources/:id/test", "POST"},
}

// UserAPIPolicies defines basic API permissions for the user role.
var UserAPIPolicies = [][]string{}
