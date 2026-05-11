package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/pquerna/otp/totp"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/pkg/token"
	"metis/internal/pkg/oauth"
	"metis/internal/repository"
	"metis/internal/service"
	"metis/internal/scheduler"
)

func newSystemHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Role{},
		&model.RoleDeptScope{},
		&model.User{},
		&model.UserConnection{},
		&model.Menu{},
		&model.SystemConfig{},
		&model.AuthProvider{},
		&model.Notification{},
		&model.NotificationRead{},
		&model.RefreshToken{},
		&model.TwoFactorSecret{},
		&model.TaskState{},
		&model.TaskExecution{},
	); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func newSystemInjector(t *testing.T, db *gorm.DB) do.Injector {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.ProvideValue(injector, []byte("test-jwt-secret"))
	do.ProvideValue(injector, token.NewBlacklist())
	enforcer, err := casbinpkg.NewEnforcerWithDB(db)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	do.ProvideValue(injector, enforcer)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewUserConnection)
	do.Provide(injector, repository.NewRole)
	do.Provide(injector, repository.NewMenu)
	do.Provide(injector, repository.NewMessageChannel)
	do.Provide(injector, repository.NewTwoFactorSecret)
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, repository.NewIdentitySource)
	do.Provide(injector, repository.NewNotification)
	do.Provide(injector, repository.NewAuthProvider)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, NewTaskEngineForTest)
	do.ProvideValue(injector, oauth.NewStateManager())
	do.Provide(injector, service.NewCasbin)
	do.Provide(injector, service.NewMenu)
	do.Provide(injector, service.NewRole)
	do.Provide(injector, service.NewUser)
	do.Provide(injector, service.NewUserConnection)
	do.Provide(injector, service.NewCaptcha)
	do.Provide(injector, service.NewAuth)
	do.Provide(injector, service.NewTwoFactor)
	do.Provide(injector, service.NewMessageChannel)
	do.Provide(injector, service.NewAuditLog)
	do.Provide(injector, service.NewIdentitySource)
	do.Provide(injector, service.NewNotification)
	do.Provide(injector, service.NewAuthProvider)
	do.Provide(injector, service.NewSysConfig)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewSession)
	do.Provide(injector, service.NewTask)
	return injector
}

func NewTaskEngineForTest(i do.Injector) (*scheduler.Engine, error) {
	engine, err := scheduler.New(i)
	if err != nil {
		return nil, err
	}
	engine.Register(&scheduler.TaskDef{
		Name:     "test-task",
		Type:     scheduler.TypeScheduled,
		CronExpr: "0 0 * * *",
		Handler:  func(ctx context.Context, payload json.RawMessage) error { return nil },
	})
	return engine, nil
}

func TestSessionHandlerListAndKick(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	role := &model.Role{Name: "用户", Code: "user"}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	user := &model.User{Username: "alice", RoleID: role.ID, IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	active := &model.RefreshToken{Token: "t1", UserID: user.ID, ExpiresAt: nowPlusHour(), AccessTokenJTI: "jti-1"}
	other := &model.RefreshToken{Token: "t2", UserID: user.ID, ExpiresAt: nowPlusHour(), AccessTokenJTI: "jti-2"}
	if err := db.Create(active).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(other).Error; err != nil {
		t.Fatalf("seed other session: %v", err)
	}

	h := &SessionHandler{sessionSvc: do.MustInvoke[*service.SessionService](injector)}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tokenJTI", "jti-1")
	})
	r.GET("/sessions", h.List)
	r.DELETE("/sessions/:id", h.Kick)

	req := httptest.NewRequest(http.MethodGet, "/sessions?page=0&pageSize=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected list success, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sessions/%d", other.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected kick success, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/sessions/invalid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid id bad request, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sessions/%d", active.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected self kick rejection, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/sessions/999999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected missing session 404, got %d: %s", w.Code, w.Body.String())
	}
}

func nowPlusHour() time.Time { return time.Now().Add(time.Hour) }

func TestSettingsHandlers(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	h := &Handler{settingsSvc: do.MustInvoke[*service.SettingsService](injector)}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/settings/security", h.GetSecuritySettings)
	r.PUT("/settings/security", h.UpdateSecuritySettings)
	r.GET("/settings/scheduler", h.GetSchedulerSettings)
	r.PUT("/settings/scheduler", h.UpdateSchedulerSettings)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/settings/security", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected get security success, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/security", bytes.NewBufferString(`{"maxConcurrentSessions":-1}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected validation failure, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/security", bytes.NewBufferString(`{"maxConcurrentSessions":3,"sessionTimeoutMinutes":120}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update security success, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/scheduler", bytes.NewBufferString(`{"historyRetentionDays":-1}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected scheduler validation failure, got %d", w.Code)
	}

	for _, body := range []string{
		`{"historyRetentionDays":7,"auditRetentionDaysAuth":-1,"auditRetentionDaysOperation":60}`,
		`{"historyRetentionDays":7,"auditRetentionDaysAuth":30,"auditRetentionDaysOperation":-1}`,
	} {
		req = httptest.NewRequest(http.MethodPut, "/settings/scheduler", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected scheduler validation failure for %s, got %d body=%s", body, w.Code, w.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/settings/scheduler", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected get scheduler success, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/settings/scheduler", bytes.NewBufferString(`{"historyRetentionDays":7,"auditRetentionDaysAuth":30,"auditRetentionDaysOperation":60}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update scheduler success, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationAnnouncementAndAuthProviderHandlers(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	user := &model.User{Username: "alice", RoleID: 1}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	provider := &model.AuthProvider{ProviderKey: "github", DisplayName: "GitHub"}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	notifSvc := do.MustInvoke[*service.NotificationService](injector)
	if _, err := notifSvc.Send("notice", "system", "hello", "world", model.NotificationTargetAll, nil, nil); err != nil {
		t.Fatalf("seed notification: %v", err)
	}
	announcement, err := notifSvc.CreateAnnouncement("公告", "内容", user.ID)
	if err != nil {
		t.Fatalf("seed announcement: %v", err)
	}

	notificationHandler := &NotificationHandler{notifSvc: notifSvc}
	announcementHandler := &AnnouncementHandler{notifSvc: notifSvc}
	authProviderHandler := &AuthProviderHandler{svc: do.MustInvoke[*service.AuthProviderService](injector)}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
	})
	r.GET("/notifications", notificationHandler.List)
	r.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
	r.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
	r.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
	r.GET("/announcements", announcementHandler.List)
	r.POST("/announcements", announcementHandler.Create)
	r.PUT("/announcements/:id", announcementHandler.Update)
	r.DELETE("/announcements/:id", announcementHandler.Delete)
	r.GET("/providers", authProviderHandler.ListAll)
	r.PUT("/providers/:key", authProviderHandler.Update)
	r.PATCH("/providers/:key/toggle", authProviderHandler.Toggle)

	for _, path := range []string{"/notifications", "/notifications/unread-count", "/announcements", "/providers"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected GET %s success, got %d: %s", path, w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/notifications/%d/read", announcement.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected mark read success, got %d", w.Code)
	}

	unreadBefore := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(unreadBefore, req)
	if unreadBefore.Code != http.StatusOK {
		t.Fatalf("expected unread-count success, got %d", unreadBefore.Code)
	}
	if !bytes.Contains(unreadBefore.Body.Bytes(), []byte(`"count":1`)) {
		t.Fatalf("expected unread count 1 after single read, got %s", unreadBefore.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected mark all read success, got %d", w.Code)
	}

	unreadAfter := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(unreadAfter, req)
	if unreadAfter.Code != http.StatusOK {
		t.Fatalf("expected unread-count success after read-all, got %d", unreadAfter.Code)
	}
	if !bytes.Contains(unreadAfter.Body.Bytes(), []byte(`"count":0`)) {
		t.Fatalf("expected unread count 0 after read-all, got %s", unreadAfter.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/announcements", bytes.NewBufferString(`{"title":"新公告","content":"内容"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected create announcement success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/announcements/%d", announcement.ID), bytes.NewBufferString(`{"title":"改名","content":"改内容"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update announcement success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/announcements/%d", announcement.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete announcement success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/providers/github", bytes.NewBufferString(`{"displayName":"GitHub OAuth"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update provider success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/providers/github/toggle", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected toggle provider success, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnnouncementAndAuthProviderHandlers_ErrorBranches(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	user := &model.User{Username: "alice", RoleID: 1}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.AuthProvider{ProviderKey: "github", DisplayName: "GitHub"}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	announcementHandler := &AnnouncementHandler{notifSvc: do.MustInvoke[*service.NotificationService](injector)}
	authProviderHandler := &AuthProviderHandler{svc: do.MustInvoke[*service.AuthProviderService](injector)}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
	})
	r.POST("/announcements", announcementHandler.Create)
	r.PUT("/announcements/:id", announcementHandler.Update)
	r.DELETE("/announcements/:id", announcementHandler.Delete)
	r.PUT("/providers/:key", authProviderHandler.Update)
	r.PATCH("/providers/:key/toggle", authProviderHandler.Toggle)

	for _, tc := range []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{http.MethodPost, "/announcements", `{}`, http.StatusBadRequest},
		{http.MethodPut, "/announcements/abc", `{"title":"x","content":"y"}`, http.StatusBadRequest},
		{http.MethodPut, "/announcements/999", `{"title":"x","content":"y"}`, http.StatusNotFound},
		{http.MethodDelete, "/announcements/abc", "", http.StatusBadRequest},
		{http.MethodDelete, "/announcements/999", "", http.StatusNotFound},
		{http.MethodPut, "/providers/github", `{"displayName":`, http.StatusBadRequest},
		{http.MethodPut, "/providers/missing", `{"displayName":"x"}`, http.StatusNotFound},
		{http.MethodPatch, "/providers/missing/toggle", "", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("expected %s %s => %d, got %d: %s", tc.method, tc.path, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestAnnouncementHandlers_InternalErrors(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	user := &model.User{Username: "alice", RoleID: 1}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	notifSvc := do.MustInvoke[*service.NotificationService](injector)
	announcement, err := notifSvc.CreateAnnouncement("公告", "内容", user.ID)
	if err != nil {
		t.Fatalf("seed announcement: %v", err)
	}

	announcementHandler := &AnnouncementHandler{notifSvc: notifSvc}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
	})
	r.GET("/announcements", announcementHandler.List)
	r.POST("/announcements", announcementHandler.Create)
	r.PUT("/announcements/:id", announcementHandler.Update)
	r.DELETE("/announcements/:id", announcementHandler.Delete)

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/announcements", ""},
		{http.MethodPost, "/announcements", `{"title":"新公告","content":"内容"}`},
		{http.MethodPut, fmt.Sprintf("/announcements/%d", announcement.ID), `{"title":"改名","content":"改内容"}`},
		{http.MethodDelete, fmt.Sprintf("/announcements/%d", announcement.ID), ""},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected %s %s => 500, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}

func TestTaskHandlerEndpoints(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	taskSvc := do.MustInvoke[*service.TaskService](injector)
	engine := do.MustInvoke[*scheduler.Engine](injector)
	if err := db.Save(&model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive}).Error; err != nil {
		t.Fatalf("seed task state: %v", err)
	}
	if _, err := engine.TriggerTask("test-task"); err != nil {
		t.Fatalf("trigger seed task: %v", err)
	}

	h := &TaskHandler{svc: taskSvc}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tasks", h.ListTasks)
	r.GET("/tasks/stats", h.GetStats)
	r.GET("/tasks/:name", h.GetTask)
	r.GET("/tasks/:name/executions", h.ListExecutions)
	r.POST("/tasks/:name/pause", h.PauseTask)
	r.POST("/tasks/:name/resume", h.ResumeTask)
	r.POST("/tasks/:name/trigger", h.TriggerTask)

	cases := []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodGet, "/tasks", http.StatusOK},
		{http.MethodGet, "/tasks/stats", http.StatusOK},
		{http.MethodGet, "/tasks/test-task", http.StatusOK},
		{http.MethodGet, "/tasks/test-task/executions", http.StatusOK},
		{http.MethodPost, "/tasks/test-task/pause", http.StatusOK},
		{http.MethodPost, "/tasks/test-task/resume", http.StatusOK},
		{http.MethodPost, "/tasks/test-task/trigger", http.StatusOK},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("expected %s %s => %d, got %d: %s", tc.method, tc.path, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestTaskHandlerErrorBranches(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	taskSvc := do.MustInvoke[*service.TaskService](injector)

	h := &TaskHandler{svc: taskSvc}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tasks/:name", h.GetTask)
	r.POST("/tasks/:name/pause", h.PauseTask)
	r.POST("/tasks/:name/resume", h.ResumeTask)
	r.POST("/tasks/:name/trigger", h.TriggerTask)

	for _, tc := range []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodGet, "/tasks/missing-task", http.StatusNotFound},
		{http.MethodPost, "/tasks/missing-task/pause", http.StatusBadRequest},
		{http.MethodPost, "/tasks/missing-task/resume", http.StatusBadRequest},
		{http.MethodPost, "/tasks/missing-task/trigger", http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("expected %s %s => %d, got %d: %s", tc.method, tc.path, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestMenuNotificationAndTaskHandlers_InternalErrors(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)

	menuHandler := &MenuHandler{menuSvc: do.MustInvoke[*service.MenuService](injector)}
	notificationHandler := &NotificationHandler{notifSvc: do.MustInvoke[*service.NotificationService](injector)}
	taskHandler := &TaskHandler{svc: do.MustInvoke[*service.TaskService](injector)}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Set("userRole", model.RoleUser)
	})
	r.GET("/menus/tree", menuHandler.GetTree)
	r.GET("/menus/user-tree", menuHandler.GetUserTree)
	r.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
	r.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
	r.GET("/tasks", taskHandler.ListTasks)
	r.GET("/tasks/stats", taskHandler.GetStats)
	r.GET("/tasks/:name/executions", taskHandler.ListExecutions)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/menus/tree"},
		{http.MethodGet, "/menus/user-tree"},
		{http.MethodGet, "/notifications/unread-count"},
		{http.MethodPut, "/notifications/read-all"},
		{http.MethodGet, "/tasks"},
		{http.MethodGet, "/tasks/test-task/executions"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected %s %s => 500, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/stats", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /tasks/stats => 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoleSettingsAuthProviderAndNotificationHandlers_InternalErrors(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	role := &model.Role{Name: "普通角色", Code: "user", DataScope: model.DataScopeAll}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	roleHandler := &RoleHandler{
		roleSvc:   do.MustInvoke[*service.RoleService](injector),
		casbinSvc: do.MustInvoke[*service.CasbinService](injector),
		menuSvc:   do.MustInvoke[*service.MenuService](injector),
		roleRepo:  do.MustInvoke[*repository.RoleRepo](injector),
	}
	authProviderHandler := &AuthProviderHandler{svc: do.MustInvoke[*service.AuthProviderService](injector)}
	notificationHandler := &NotificationHandler{notifSvc: do.MustInvoke[*service.NotificationService](injector)}
	h := &Handler{settingsSvc: do.MustInvoke[*service.SettingsService](injector)}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
	})
	r.GET("/roles", roleHandler.List)
	r.GET("/roles/:id", roleHandler.Get)
	r.PUT("/roles/:id", roleHandler.Update)
	r.DELETE("/roles/:id", roleHandler.Delete)
	r.PUT("/roles/:id/permissions", roleHandler.SetPermissions)
	r.PUT("/roles/:id/data-scope", roleHandler.UpdateDataScope)
	r.GET("/providers", authProviderHandler.ListAll)
	r.GET("/notifications", notificationHandler.List)
	r.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
	r.PUT("/settings/security", h.UpdateSecuritySettings)
	r.PUT("/settings/scheduler", h.UpdateSchedulerSettings)

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/roles", ""},
		{http.MethodGet, fmt.Sprintf("/roles/%d", role.ID), ""},
		{http.MethodPut, fmt.Sprintf("/roles/%d", role.ID), `{"name":"changed"}`},
		{http.MethodDelete, fmt.Sprintf("/roles/%d", role.ID), ""},
		{http.MethodPut, fmt.Sprintf("/roles/%d/permissions", role.ID), `{"menuIds":[1]}`},
		{http.MethodPut, fmt.Sprintf("/roles/%d/data-scope", role.ID), `{"dataScope":"all"}`},
		{http.MethodGet, "/providers", ""},
		{http.MethodGet, "/notifications", ""},
		{http.MethodPut, "/notifications/1/read", ""},
		{http.MethodPut, "/settings/security", `{"maxConcurrentSessions":1,"sessionTimeoutMinutes":60,"passwordMinLength":8,"captchaProvider":"none"}`},
		{http.MethodPut, "/settings/scheduler", `{"historyRetentionDays":1,"auditRetentionDaysAuth":1,"auditRetentionDaysOperation":1}`},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected %s %s => 500, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}

func TestUserHandlers_InternalErrors(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)

	role := &model.Role{Name: "普通用户", Code: model.RoleUser}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	manager := &model.User{Username: "manager", RoleID: role.ID, IsActive: true}
	if err := db.Create(manager).Error; err != nil {
		t.Fatalf("seed manager: %v", err)
	}
	alice := &model.User{Username: "alice", RoleID: role.ID, IsActive: true}
	if err := db.Create(alice).Error; err != nil {
		t.Fatalf("seed alice: %v", err)
	}

	userHandler := &UserHandler{
		userSvc:  do.MustInvoke[*service.UserService](injector),
		connRepo: do.MustInvoke[*repository.UserConnectionRepo](injector),
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", manager.ID)
	})
	r.GET("/users", userHandler.List)
	r.GET("/users/:id", userHandler.Get)
	r.PUT("/users/:id", userHandler.Update)
	r.POST("/users/:id/activate", userHandler.Activate)
	r.POST("/users/:id/deactivate", userHandler.Deactivate)
	r.POST("/users/:id/unlock", userHandler.Unlock)
	r.GET("/users/:id/manager-chain", userHandler.GetManagerChain)

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/users", ""},
		{http.MethodGet, fmt.Sprintf("/users/%d", alice.ID), ""},
		{http.MethodPut, fmt.Sprintf("/users/%d", alice.ID), `{"phone":"123"}`},
		{http.MethodPost, fmt.Sprintf("/users/%d/activate", alice.ID), ""},
		{http.MethodPost, fmt.Sprintf("/users/%d/deactivate", alice.ID), ""},
		{http.MethodPost, fmt.Sprintf("/users/%d/unlock", alice.ID), ""},
		{http.MethodGet, fmt.Sprintf("/users/%d/manager-chain", alice.ID), ""},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected %s %s => 500, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}

func TestMenuAndNotificationHandlers_ErrorBranches(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)

	menuSvc := do.MustInvoke[*service.MenuService](injector)
	notifSvc := do.MustInvoke[*service.NotificationService](injector)

	parent := &model.Menu{Name: "系统管理", Type: model.MenuTypeDirectory, Permission: "system", Sort: 1}
	if err := menuSvc.Create(parent); err != nil {
		t.Fatalf("create parent menu: %v", err)
	}
	child := &model.Menu{Name: "用户管理", Type: model.MenuTypeMenu, ParentID: &parent.ID, Permission: "system:user:list", Sort: 2}
	if err := menuSvc.Create(child); err != nil {
		t.Fatalf("create child menu: %v", err)
	}

	user := &model.User{Username: "alice", RoleID: 1, IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create notification user: %v", err)
	}

	menuHandler := &MenuHandler{menuSvc: menuSvc}
	notificationHandler := &NotificationHandler{notifSvc: notifSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
	})
	r.POST("/menus", menuHandler.Create)
	r.PUT("/menus/:id", menuHandler.Update)
	r.PUT("/menus/sort", menuHandler.Reorder)
	r.DELETE("/menus/:id", menuHandler.Delete)
	r.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		code   int
	}{
		{"menu create invalid body", http.MethodPost, "/menus", `{}`, http.StatusBadRequest},
		{"menu update invalid id", http.MethodPut, "/menus/abc", `{"name":"x"}`, http.StatusBadRequest},
		{"menu update missing", http.MethodPut, "/menus/9999", `{"name":"x"}`, http.StatusNotFound},
		{"menu reorder invalid body", http.MethodPut, "/menus/sort", `{}`, http.StatusBadRequest},
		{"menu delete invalid id", http.MethodDelete, "/menus/abc", ``, http.StatusBadRequest},
		{"menu delete has children", http.MethodDelete, fmt.Sprintf("/menus/%d", parent.ID), ``, http.StatusBadRequest},
		{"menu delete missing", http.MethodDelete, "/menus/9999", ``, http.StatusNotFound},
		{"notification read invalid id", http.MethodPut, "/notifications/abc/read", ``, http.StatusBadRequest},
		{"notification read missing is idempotent", http.MethodPut, "/notifications/9999/read", ``, http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("%s: expected %d, got %d body=%s", tc.name, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestUserRoleMenuAndTwoFactorHandlers(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)

	roleSvc := do.MustInvoke[*service.RoleService](injector)
	userSvc := do.MustInvoke[*service.UserService](injector)
	menuSvc := do.MustInvoke[*service.MenuService](injector)
	connRepo := do.MustInvoke[*repository.UserConnectionRepo](injector)
	casbinSvc := do.MustInvoke[*service.CasbinService](injector)
	tfSvc := do.MustInvoke[*service.TwoFactorService](injector)
	authSvc := do.MustInvoke[*service.AuthService](injector)
	roleRepo := do.MustInvoke[*repository.RoleRepo](injector)
	jwtSecret := do.MustInvoke[[]byte](injector)

	adminRole := &model.Role{Name: "管理员", Code: "admin", Sort: 1, IsSystem: true, DataScope: model.DataScopeAll}
	if err := db.Create(adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	userRole, err := roleSvc.Create("普通用户", model.RoleUser, "", 2)
	if err != nil {
		t.Fatalf("create user role: %v", err)
	}
	manager, err := userSvc.Create("manager", "Password123!", "manager@example.com", "", adminRole.ID)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	alice, err := userSvc.Create("alice", "Password123!", "alice@example.com", "", userRole.ID)
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	alice.ManagerID = &manager.ID
	if err := db.Save(alice).Error; err != nil {
		t.Fatalf("assign manager: %v", err)
	}
	if err := connRepo.Create(&model.UserConnection{
		UserID:       alice.ID,
		Provider:     "github",
		ExternalID:   "gh-1",
		ExternalName: "Alice",
	}); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	rootMenu := &model.Menu{Name: "系统管理", Type: model.MenuTypeDirectory, Permission: "system", Sort: 1}
	if err := menuSvc.Create(rootMenu); err != nil {
		t.Fatalf("create root menu: %v", err)
	}
	userMenu := &model.Menu{Name: "用户管理", Type: model.MenuTypeMenu, ParentID: &rootMenu.ID, Permission: "system:user:list", Sort: 2}
	if err := menuSvc.Create(userMenu); err != nil {
		t.Fatalf("create user menu: %v", err)
	}
	if err := casbinSvc.SetPoliciesForRole(adminRole.Code, [][]string{{adminRole.Code, "system:user:list", "GET"}}); err != nil {
		t.Fatalf("set admin policies: %v", err)
	}

	userHandler := &UserHandler{userSvc: userSvc, connRepo: connRepo}
	roleHandler := &RoleHandler{roleSvc: roleSvc, casbinSvc: casbinSvc, menuSvc: menuSvc, roleRepo: roleRepo}
	menuHandler := &MenuHandler{menuSvc: menuSvc}
	twoFactorHandler := &TwoFactorHandler{tfSvc: tfSvc, authSvc: authSvc, jwtSecret: jwtSecret}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", manager.ID)
		c.Set("userRole", adminRole.Code)
	})

	r.GET("/users", userHandler.List)
	r.POST("/users", userHandler.Create)
	r.GET("/users/:id", userHandler.Get)
	r.PUT("/users/:id", userHandler.Update)
	r.DELETE("/users/:id", userHandler.Delete)
	r.POST("/users/:id/reset-password", userHandler.ResetPassword)
	r.POST("/users/:id/activate", userHandler.Activate)
	r.POST("/users/:id/deactivate", userHandler.Deactivate)
	r.POST("/users/:id/unlock", userHandler.Unlock)
	r.GET("/users/:id/manager-chain", userHandler.GetManagerChain)

	r.GET("/roles", roleHandler.List)
	r.POST("/roles", roleHandler.Create)
	r.GET("/roles/:id", roleHandler.Get)
	r.PUT("/roles/:id", roleHandler.Update)
	r.DELETE("/roles/:id", roleHandler.Delete)
	r.GET("/roles/:id/permissions", roleHandler.GetPermissions)
	r.PUT("/roles/:id/permissions", roleHandler.SetPermissions)
	r.PUT("/roles/:id/data-scope", roleHandler.UpdateDataScope)

	r.GET("/menus/tree", menuHandler.GetTree)
	r.GET("/menus/user-tree", menuHandler.GetUserTree)
	r.POST("/menus", menuHandler.Create)
	r.PUT("/menus/:id", menuHandler.Update)
	r.PUT("/menus/sort", menuHandler.Reorder)
	r.DELETE("/menus/:id", menuHandler.Delete)

	r.POST("/auth/2fa/setup", twoFactorHandler.Setup)
	r.POST("/auth/2fa/confirm", twoFactorHandler.Confirm)
	r.DELETE("/auth/2fa", twoFactorHandler.Disable)
	r.POST("/auth/2fa/login", twoFactorHandler.Login)

	for _, reqSpec := range []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{http.MethodGet, "/users?page=1&pageSize=10&keyword=ali", "", http.StatusOK},
		{http.MethodGet, fmt.Sprintf("/users/%d", alice.ID), "", http.StatusOK},
		{http.MethodGet, fmt.Sprintf("/users/%d/manager-chain", alice.ID), "", http.StatusOK},
		{http.MethodGet, "/roles?page=1&pageSize=10", "", http.StatusOK},
		{http.MethodGet, fmt.Sprintf("/roles/%d", adminRole.ID), "", http.StatusOK},
		{http.MethodGet, fmt.Sprintf("/roles/%d/permissions", adminRole.ID), "", http.StatusOK},
		{http.MethodGet, "/menus/tree", "", http.StatusOK},
		{http.MethodGet, "/menus/user-tree", "", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(reqSpec.method, reqSpec.path, bytes.NewBufferString(reqSpec.body))
		if reqSpec.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		r.ServeHTTP(w, req)
		if w.Code != reqSpec.code {
			t.Fatalf("expected %s %s => %d, got %d: %s", reqSpec.method, reqSpec.path, reqSpec.code, w.Code, w.Body.String())
		}
	}

	rolePermResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/roles/%d/permissions", adminRole.ID), nil)
	r.ServeHTTP(rolePermResp, req)
	if rolePermResp.Code != http.StatusOK {
		t.Fatalf("expected role permissions success, got %d: %s", rolePermResp.Code, rolePermResp.Body.String())
	}
	if !bytes.Contains(rolePermResp.Body.Bytes(), []byte(`"menuPermissions":["system:user:list"]`)) {
		t.Fatalf("expected menu permission in role permissions payload, got %s", rolePermResp.Body.String())
	}

	w := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(fmt.Sprintf(`{"username":"charlie","password":"Password123!","email":"charlie@example.com","roleId":%d}`, userRole.ID)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected create user success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/users/%d", alice.ID), bytes.NewBufferString(`{"phone":"123","clearManager":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected clear manager success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%d/reset-password", alice.ID), bytes.NewBufferString(`{"password":"NewPassword123!"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected reset password success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%d/deactivate", alice.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected deactivate success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%d/activate", alice.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected activate success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/users/%d/unlock", alice.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected unlock success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/users/%d", alice.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete user success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(`{"name":"编辑","code":"editor","description":"desc","sort":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected create role success, got %d: %s", w.Code, w.Body.String())
	}
	var createdRoleResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &createdRoleResp); err != nil {
		t.Fatalf("unmarshal created role: %v", err)
	}
	roleData := createdRoleResp["data"].(map[string]any)
	editorRoleID := uint(roleData["id"].(float64))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/roles/%d", editorRoleID), bytes.NewBufferString(`{"name":"编辑者"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update role success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/roles/%d/permissions", editorRoleID), bytes.NewBufferString(fmt.Sprintf(`{"menuIds":[%d],"apiPolicies":[{"path":"/api/v1/test","method":"GET"}]}`, userMenu.ID)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected set permissions success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/roles/%d/data-scope", editorRoleID), bytes.NewBufferString(`{"dataScope":"custom","deptIds":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update data scope success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/roles/%d", editorRoleID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete role success, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/menus", bytes.NewBufferString(`{"name":"角色管理","type":"menu","path":"/roles","permission":"system:role:list","sort":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected create menu success, got %d: %s", w.Code, w.Body.String())
	}
	var createdMenuResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &createdMenuResp); err != nil {
		t.Fatalf("unmarshal created menu: %v", err)
	}
	menuData := createdMenuResp["data"].(map[string]any)
	createdMenuID := uint(menuData["id"].(float64))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/menus/%d", createdMenuID), bytes.NewBufferString(`{"icon":"Users"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update menu success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/menus/sort", bytes.NewBufferString(fmt.Sprintf(`{"items":[{"id":%d,"sort":1}]}`, createdMenuID)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected reorder menu success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/menus/%d", createdMenuID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete menu success, got %d: %s", w.Code, w.Body.String())
	}

	setupW := httptest.NewRecorder()
	setupReq := httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
	r.ServeHTTP(setupW, setupReq)
	if setupW.Code != http.StatusOK {
		t.Fatalf("expected 2FA setup success, got %d: %s", setupW.Code, setupW.Body.String())
	}
	var setupResp map[string]any
	if err := json.Unmarshal(setupW.Body.Bytes(), &setupResp); err != nil {
		t.Fatalf("unmarshal setup resp: %v", err)
	}
	setupData := setupResp["data"].(map[string]any)
	secret := setupData["secret"].(string)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/confirm", bytes.NewBufferString(fmt.Sprintf(`{"code":"%s"}`, code)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 2FA confirm success, got %d: %s", w.Code, w.Body.String())
	}
	twoFactorToken, err := token.GenerateTwoFactorToken(manager.ID, jwtSecret)
	if err != nil {
		t.Fatalf("generate 2FA token: %v", err)
	}
	loginCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate login totp code: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/login", bytes.NewBufferString(fmt.Sprintf(`{"twoFactorToken":"%s","code":"%s"}`, twoFactorToken, loginCode)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 2FA login success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/auth/2fa", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 2FA disable success, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserRoleAndTwoFactorHandlers_ErrorPaths(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)

	roleSvc := do.MustInvoke[*service.RoleService](injector)
	userSvc := do.MustInvoke[*service.UserService](injector)
	menuSvc := do.MustInvoke[*service.MenuService](injector)
	connRepo := do.MustInvoke[*repository.UserConnectionRepo](injector)
	casbinSvc := do.MustInvoke[*service.CasbinService](injector)
	tfSvc := do.MustInvoke[*service.TwoFactorService](injector)
	authSvc := do.MustInvoke[*service.AuthService](injector)
	roleRepo := do.MustInvoke[*repository.RoleRepo](injector)
	jwtSecret := do.MustInvoke[[]byte](injector)

	adminRole, err := roleSvc.Create("管理员", "admin", "", 1)
	if err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	adminRole.IsSystem = true
	if err := db.Save(adminRole).Error; err != nil {
		t.Fatalf("mark admin role system: %v", err)
	}
	userRole, err := roleSvc.Create("普通用户", model.RoleUser, "", 2)
	if err != nil {
		t.Fatalf("create user role: %v", err)
	}
	manager, err := userSvc.Create("manager", "Password123!", "manager@example.com", "", adminRole.ID)
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	alice, err := userSvc.Create("alice", "Password123!", "alice@example.com", "", userRole.ID)
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	rootMenu := &model.Menu{Name: "系统管理", Type: model.MenuTypeDirectory, Permission: "system", Sort: 1}
	if err := menuSvc.Create(rootMenu); err != nil {
		t.Fatalf("create root menu: %v", err)
	}
	userMenu := &model.Menu{Name: "用户管理", Type: model.MenuTypeMenu, ParentID: &rootMenu.ID, Permission: "system:user:list", Sort: 2}
	if err := menuSvc.Create(userMenu); err != nil {
		t.Fatalf("create user menu: %v", err)
	}

	userHandler := &UserHandler{userSvc: userSvc, connRepo: connRepo}
	roleHandler := &RoleHandler{roleSvc: roleSvc, casbinSvc: casbinSvc, menuSvc: menuSvc, roleRepo: roleRepo}
	twoFactorHandler := &TwoFactorHandler{tfSvc: tfSvc, authSvc: authSvc, jwtSecret: jwtSecret}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userId", manager.ID)
		c.Set("userRole", adminRole.Code)
	})
	r.POST("/users", userHandler.Create)
	r.GET("/users/:id", userHandler.Get)
	r.PUT("/users/:id", userHandler.Update)
	r.DELETE("/users/:id", userHandler.Delete)
	r.POST("/users/:id/reset-password", userHandler.ResetPassword)
	r.POST("/users/:id/activate", userHandler.Activate)
	r.POST("/users/:id/deactivate", userHandler.Deactivate)
	r.POST("/users/:id/unlock", userHandler.Unlock)
	r.GET("/users/:id/manager-chain", userHandler.GetManagerChain)
	r.POST("/roles", roleHandler.Create)
	r.GET("/roles/:id", roleHandler.Get)
	r.PUT("/roles/:id", roleHandler.Update)
	r.DELETE("/roles/:id", roleHandler.Delete)
	r.GET("/roles/:id/permissions", roleHandler.GetPermissions)
	r.PUT("/roles/:id/permissions", roleHandler.SetPermissions)
	r.PUT("/roles/:id/data-scope", roleHandler.UpdateDataScope)
	r.POST("/auth/2fa/setup", twoFactorHandler.Setup)
	r.POST("/auth/2fa/confirm", twoFactorHandler.Confirm)
	r.DELETE("/auth/2fa", twoFactorHandler.Disable)
	r.POST("/auth/2fa/login", twoFactorHandler.Login)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
		code   int
	}{
		{"user create invalid body", http.MethodPost, "/users", `{}`, http.StatusBadRequest},
		{"user create weak password", http.MethodPost, "/users", fmt.Sprintf(`{"username":"weak","password":"short","email":"weak@example.com","roleId":%d}`, userRole.ID), http.StatusBadRequest},
		{"user get invalid id", http.MethodGet, "/users/abc", ``, http.StatusBadRequest},
		{"user get not found", http.MethodGet, "/users/9999", ``, http.StatusNotFound},
		{"user update invalid body", http.MethodPut, fmt.Sprintf("/users/%d", alice.ID), `{"roleId":"bad"}`, http.StatusBadRequest},
		{"user update self role rejected", http.MethodPut, fmt.Sprintf("/users/%d", manager.ID), fmt.Sprintf(`{"roleId":%d}`, userRole.ID), http.StatusBadRequest},
		{"user update circular manager", http.MethodPut, fmt.Sprintf("/users/%d", alice.ID), fmt.Sprintf(`{"managerId":%d}`, alice.ID), http.StatusBadRequest},
		{"user update missing", http.MethodPut, "/users/9999", `{"phone":"123"}`, http.StatusNotFound},
		{"user clear manager missing", http.MethodPut, "/users/9999", `{"clearManager":true}`, http.StatusNotFound},
		{"user delete self rejected", http.MethodDelete, fmt.Sprintf("/users/%d", manager.ID), ``, http.StatusBadRequest},
		{"user reset invalid body", http.MethodPost, fmt.Sprintf("/users/%d/reset-password", alice.ID), `{}`, http.StatusBadRequest},
		{"user reset weak password", http.MethodPost, fmt.Sprintf("/users/%d/reset-password", alice.ID), `{"password":"short"}`, http.StatusBadRequest},
		{"user reset missing", http.MethodPost, "/users/9999/reset-password", `{"password":"Password123!"}`, http.StatusNotFound},
		{"user activate invalid id", http.MethodPost, "/users/abc/activate", ``, http.StatusBadRequest},
		{"user activate missing", http.MethodPost, "/users/9999/activate", ``, http.StatusNotFound},
		{"user deactivate self", http.MethodPost, fmt.Sprintf("/users/%d/deactivate", manager.ID), ``, http.StatusBadRequest},
		{"user unlock missing", http.MethodPost, "/users/9999/unlock", ``, http.StatusNotFound},
		{"user manager chain missing", http.MethodGet, "/users/9999/manager-chain", ``, http.StatusNotFound},
		{"role create invalid body", http.MethodPost, "/roles", `{}`, http.StatusBadRequest},
		{"role create duplicate code", http.MethodPost, "/roles", `{"name":"重复管理员","code":"admin"}`, http.StatusBadRequest},
		{"role permissions invalid id", http.MethodGet, "/roles/abc/permissions", ``, http.StatusBadRequest},
		{"role get missing", http.MethodGet, "/roles/9999", ``, http.StatusNotFound},
		{"role update invalid body", http.MethodPut, fmt.Sprintf("/roles/%d", adminRole.ID), `{"sort":"bad"}`, http.StatusBadRequest},
		{"role update system code", http.MethodPut, fmt.Sprintf("/roles/%d", adminRole.ID), `{"code":"root"}`, http.StatusBadRequest},
		{"role update missing", http.MethodPut, "/roles/9999", `{"name":"x"}`, http.StatusNotFound},
		{"role delete system role", http.MethodDelete, fmt.Sprintf("/roles/%d", adminRole.ID), ``, http.StatusBadRequest},
		{"role delete has users", http.MethodDelete, fmt.Sprintf("/roles/%d", userRole.ID), ``, http.StatusBadRequest},
		{"role permissions missing", http.MethodGet, "/roles/9999/permissions", ``, http.StatusNotFound},
		{"role set permissions missing role", http.MethodPut, "/roles/9999/permissions", fmt.Sprintf(`{"menuIds":[%d]}`, userMenu.ID), http.StatusNotFound},
		{"role set permissions invalid body", http.MethodPut, fmt.Sprintf("/roles/%d/permissions", userRole.ID), `{"menuIds":"bad"}`, http.StatusBadRequest},
		{"role data scope system role", http.MethodPut, fmt.Sprintf("/roles/%d/data-scope", adminRole.ID), `{"dataScope":"dept"}`, http.StatusBadRequest},
		{"role data scope invalid", http.MethodPut, fmt.Sprintf("/roles/%d/data-scope", userRole.ID), `{"dataScope":"bad"}`, http.StatusBadRequest},
		{"role data scope missing", http.MethodPut, "/roles/9999/data-scope", `{"dataScope":"custom","deptIds":[1]}`, http.StatusNotFound},
		{"2fa confirm invalid body", http.MethodPost, "/auth/2fa/confirm", `{}`, http.StatusBadRequest},
		{"2fa confirm before setup", http.MethodPost, "/auth/2fa/confirm", `{"code":"123456"}`, http.StatusBadRequest},
		{"2fa disable before setup", http.MethodDelete, "/auth/2fa", ``, http.StatusBadRequest},
		{"2fa login invalid body", http.MethodPost, "/auth/2fa/login", `{}`, http.StatusBadRequest},
		{"2fa login invalid token", http.MethodPost, "/auth/2fa/login", `{"twoFactorToken":"bad","code":"123456"}`, http.StatusUnauthorized},
	}{
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("expected %s %s => %d, got %d: %s", tc.method, tc.path, tc.code, w.Code, w.Body.String())
			}
		})
	}

	setupW := httptest.NewRecorder()
	setupReq := httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
	r.ServeHTTP(setupW, setupReq)
	if setupW.Code != http.StatusOK {
		t.Fatalf("expected setup success, got %d: %s", setupW.Code, setupW.Body.String())
	}
	setupW2 := httptest.NewRecorder()
	setupReq2 := httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
	r.ServeHTTP(setupW2, setupReq2)
	if setupW2.Code != http.StatusOK {
		t.Fatalf("expected second setup to replace pending secret, got %d: %s", setupW2.Code, setupW2.Body.String())
	}

	var setupResp map[string]any
	if err := json.Unmarshal(setupW2.Body.Bytes(), &setupResp); err != nil {
		t.Fatalf("unmarshal setup resp: %v", err)
	}
	secret := setupResp["data"].(map[string]any)["secret"].(string)
	invalidCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate invalid code source: %v", err)
	}
	if invalidCode == "000000" {
		invalidCode = "111111"
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/confirm", bytes.NewBufferString(`{"code":"000000"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid 2FA code rejection, got %d: %s", w.Code, w.Body.String())
	}

	validCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate valid code: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/confirm", bytes.NewBufferString(fmt.Sprintf(`{"code":"%s"}`, validCode)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected confirm success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/setup", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected setup already enabled rejection, got %d: %s", w.Code, w.Body.String())
	}

	purposeToken, _, err := token.GenerateAccessToken(manager.ID, adminRole.Code, jwtSecret)
	if err != nil {
		t.Fatalf("generate non-2fa token: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/login", bytes.NewBufferString(fmt.Sprintf(`{"twoFactorToken":"%s","code":"123456"}`, purposeToken)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid token purpose, got %d: %s", w.Code, w.Body.String())
	}

	tfToken, err := token.GenerateTwoFactorToken(manager.ID, jwtSecret)
	if err != nil {
		t.Fatalf("generate 2fa token: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/2fa/login", bytes.NewBufferString(fmt.Sprintf(`{"twoFactorToken":"%s","code":"000000"}`, tfToken)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid 2fa login code, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSiteInfoHandlersAndParseDataURL(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	sysCfg := do.MustInvoke[*service.SysConfigService](injector)
	h := &Handler{sysCfg: sysCfg}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/site-info", h.GetSiteInfo)
	r.PUT("/site-info", h.UpdateSiteInfo)
	r.GET("/site-info/logo", h.GetLogo)
	r.PUT("/site-info/logo", h.UploadLogo)
	r.DELETE("/site-info/logo", h.DeleteLogo)

	req := httptest.NewRequest(http.MethodGet, "/site-info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected get site info success, got %d: %s", w.Code, w.Body.String())
	}

	if err := sysCfg.Set(&model.SystemConfig{Key: keySiteLocale, Value: "en-US", Remark: "locale"}); err != nil {
		t.Fatalf("seed locale: %v", err)
	}
	if err := sysCfg.Set(&model.SystemConfig{Key: keySiteTimezone, Value: "Asia/Shanghai", Remark: "timezone"}); err != nil {
		t.Fatalf("seed timezone: %v", err)
	}

	req = httptest.NewRequest(http.MethodPut, "/site-info", bytes.NewBufferString(`{"appName":"Metis Pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update site info success, got %d: %s", w.Code, w.Body.String())
	}

	logoData := []byte("fake-png")
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(logoData)
	req = httptest.NewRequest(http.MethodPut, "/site-info/logo", bytes.NewBufferString(fmt.Sprintf(`{"data":"%s"}`, dataURL)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected upload logo success, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/site-info/logo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), logoData) {
		t.Fatalf("expected get logo payload, got %d body=%q", w.Code, w.Body.Bytes())
	}

	req = httptest.NewRequest(http.MethodGet, "/site-info", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected get site info with overrides success, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"appName":"Metis Pro"`) || !strings.Contains(body, `"hasLogo":true`) || !strings.Contains(body, `"locale":"en-US"`) || !strings.Contains(body, `"timezone":"Asia/Shanghai"`) {
		t.Fatalf("expected updated site info in response, got %s", body)
	}

	req = httptest.NewRequest(http.MethodDelete, "/site-info/logo", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete logo success, got %d: %s", w.Code, w.Body.String())
	}

	if ct, decoded, ok := parseDataURL(dataURL); !ok || ct != "image/png" || !bytes.Equal(decoded, logoData) {
		t.Fatalf("expected parseDataURL success, got ct=%q ok=%v data=%q", ct, ok, decoded)
	}
	if _, _, ok := parseDataURL("invalid"); ok {
		t.Fatal("expected parseDataURL to reject invalid input")
	}
}

func TestSiteInfoHandlers_ErrorPaths(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	sysCfg := do.MustInvoke[*service.SysConfigService](injector)
	h := &Handler{sysCfg: sysCfg}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/site-info/logo", h.GetLogo)
	r.PUT("/site-info", h.UpdateSiteInfo)
	r.PUT("/site-info/logo", h.UploadLogo)
	r.DELETE("/site-info/logo", h.DeleteLogo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/site-info/logo", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected missing logo 404, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/site-info", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected update site info 400, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/site-info/logo", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected upload logo missing data 400, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/site-info/logo", bytes.NewBufferString(`{"data":"data:text/plain;base64,SGVsbG8="}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected non-image data url rejection, got %d: %s", w.Code, w.Body.String())
	}

	largeData := "data:image/png;base64," + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("a"), maxLogoBytes+1))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/site-info/logo", bytes.NewBufferString(fmt.Sprintf(`{"data":"%s"}`, largeData)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected oversize logo rejection, got %d: %s", w.Code, w.Body.String())
	}

	if err := sysCfg.Set(&model.SystemConfig{Key: keySiteLogo, Value: "data:image/png;base64,@@@", Remark: "bad"}); err != nil {
		t.Fatalf("seed invalid logo data: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/site-info/logo", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected invalid logo data 500, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/site-info/logo", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete existing bad logo success, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/site-info/logo", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected delete missing logo 404, got %d: %s", w.Code, w.Body.String())
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/site-info", bytes.NewBufferString(`{"appName":"Metis Broken"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected update site info 500 after db close, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCaptchaHandlerGenerate(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	settingsSvc := do.MustInvoke[*service.SettingsService](injector)
	captchaSvc := do.MustInvoke[*service.CaptchaService](injector)
	h := &CaptchaHandler{captchaSvc: captchaSvc, settingsSvc: settingsSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/captcha", h.Generate)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/captcha", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("expected disabled captcha response, got %d: %s", w.Code, w.Body.String())
	}

	if err := settingsSvc.UpdateSecuritySettings(service.SecuritySettings{CaptchaProvider: "image"}); err != nil {
		t.Fatalf("enable captcha: %v", err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/captcha", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"enabled":true`) {
		t.Fatalf("expected enabled captcha response, got %d: %s", w.Code, w.Body.String())
	}
}

func TestParseUintParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	if v, ok := ParseUintParam(c, "id"); !ok || v != 7 {
		t.Fatalf("expected parsed uint 7, got v=%d ok=%v", v, ok)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "0"}}
	if _, ok := ParseUintParam(c, "id"); ok {
		t.Fatal("expected zero id to be rejected")
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	if _, ok := ParseUintParam(c, "id"); ok {
		t.Fatal("expected invalid id to be rejected")
	}
}

func TestHandlerNewAndRegister(t *testing.T) {
	db := newSystemHandlerTestDB(t)
	injector := newSystemInjector(t, db)
	h, err := New(injector)
	if err != nil {
		t.Fatalf("New handler returned error: %v", err)
	}

	engine := gin.New()
	enforcer := do.MustInvoke[*casbin.Enforcer](injector)
	blacklist := do.MustInvoke[*token.TokenBlacklist](injector)
	jwtSecret := do.MustInvoke[[]byte](injector)
	group := h.Register(engine, jwtSecret, enforcer, blacklist)
	if group == nil {
		t.Fatal("expected Register to return authed group")
	}

	for _, tc := range []struct {
		path string
		code int
	}{
		{"/api/v1/auth/providers", http.StatusOK},
		{"/api/v1/site-info", http.StatusOK},
		{"/api/v1/notifications", http.StatusUnauthorized},
		{"/api/v1/users", http.StatusUnauthorized},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		engine.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("expected GET %s => %d, got %d: %s", tc.path, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestRegisterStaticServesAssetsAndSpaFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterStatic(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/linear-D0UYRuD7.js", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected static asset success, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/non-existent-route", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<!doctype html") && !strings.Contains(strings.ToLower(w.Body.String()), "<html") {
		t.Fatalf("expected SPA fallback index.html, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected API path to 404, got %d: %s", w.Code, w.Body.String())
	}
}
