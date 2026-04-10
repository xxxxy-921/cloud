package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/samber/do/v2"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	casbinpkg "metis/internal/casbin"
	"metis/internal/database"
	"metis/internal/handler"
	"metis/internal/middleware"
	"metis/internal/model"
	"metis/internal/pkg/oauth"
	"metis/internal/pkg/token"
	"metis/internal/repository"
	"metis/internal/scheduler"
	"metis/internal/seed"
	"metis/internal/service"
	"metis/internal/telemetry"

	"metis/internal/app"
)

func main() {
	// Load .env (silently skip if not present)
	_ = godotenv.Load()

	// Initialize OpenTelemetry (no-op when OTEL_ENABLED != "true")
	otelShutdown, err := telemetry.Init(context.Background())
	if err != nil {
		slog.Error("opentelemetry init failed", "error", err)
		os.Exit(1)
	}

	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create-admin":
			runCreateAdmin()
			return
		case "seed":
			runSeed()
			return
		}
	}

	// IOC container
	injector := do.New()

	// JWT secret
	jwtSecret := resolveJWTSecret()
	do.ProvideValue(injector, jwtSecret)

	// Token blacklist (in-memory, shared across middleware and services)
	do.ProvideValue(injector, token.NewBlacklist())

	do.Provide(injector, database.New)
	do.Provide(injector, casbinpkg.NewEnforcer)
	do.Provide(injector, repository.NewSysConfig)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRefreshToken)
	do.Provide(injector, repository.NewRole)
	do.Provide(injector, repository.NewMenu)
	do.Provide(injector, repository.NewNotification)
	do.Provide(injector, repository.NewMessageChannel)
	do.Provide(injector, repository.NewAuthProvider)
	do.Provide(injector, repository.NewUserConnection)
	do.Provide(injector, repository.NewAuditLog)
	do.Provide(injector, repository.NewTwoFactorSecret)
	do.Provide(injector, service.NewSysConfig)
	do.Provide(injector, service.NewCasbin)
	do.Provide(injector, service.NewRole)
	do.Provide(injector, service.NewMenu)
	do.Provide(injector, service.NewAuth)
	do.Provide(injector, service.NewUser)
	do.Provide(injector, service.NewNotification)
	do.Provide(injector, service.NewMessageChannel)
	do.Provide(injector, service.NewSession)
	do.Provide(injector, service.NewSettings)
	do.Provide(injector, service.NewAuthProvider)
	do.Provide(injector, service.NewUserConnection)
	do.Provide(injector, service.NewAuditLog)
	do.Provide(injector, service.NewCaptcha)
	do.Provide(injector, service.NewTwoFactor)
	do.Provide(injector, repository.NewIdentitySource)
	do.Provide(injector, service.NewIdentitySource)
	do.ProvideValue(injector, oauth.NewStateManager())
	do.Provide(injector, handler.New)
	do.Provide(injector, scheduler.New)

	// Resolve DB and enforcer early for auto-seed
	db := do.MustInvoke[*database.DB](injector)
	enforcer := do.MustInvoke[*casbin.Enforcer](injector)

	// Auto-seed: keep roles, menus, and Casbin policies in sync on every startup.
	// This is the long-term fix for stale enforcer state after external seed runs.
	if result, err := seed.Run(db.DB, enforcer); err != nil {
		slog.Error("auto-seed failed", "error", err)
		os.Exit(1)
	} else {
		slog.Info("auto-seed complete",
			"roles_created", result.RolesCreated,
			"menus_created", result.MenusCreated,
			"policies_added", result.PoliciesAdded,
		)
	}

	// Boot pluggable Apps
	for _, a := range app.All() {
		if models := a.Models(); len(models) > 0 {
			if err := db.DB.AutoMigrate(models...); err != nil {
				slog.Error("app auto-migrate failed", "app", a.Name(), "error", err)
				os.Exit(1)
			}
		}
		a.Providers(injector)
		if err := a.Seed(db.DB, enforcer); err != nil {
			slog.Error("app seed failed", "app", a.Name(), "error", err)
			os.Exit(1)
		}
	}

	// Resolve handler (triggers lazy init of remaining dependencies)
	h := do.MustInvoke[*handler.Handler](injector)
	blacklist := do.MustInvoke[*token.TokenBlacklist](injector)

	// Initialize scheduler engine
	engine := do.MustInvoke[*scheduler.Engine](injector)
	sysConfigRepo := do.MustInvoke[*repository.SysConfigRepo](injector)

	// Wire builtin cleanup task handler
	scheduler.SetCleanupHandler(
		scheduler.HistoryCleanupTask,
		func(key string) (string, error) {
			cfg, err := sysConfigRepo.Get(key)
			if err != nil {
				return "", err
			}
			return cfg.Value, nil
		},
		engine.GetStore().(*scheduler.GormStore),
	)
	engine.Register(scheduler.HistoryCleanupTask)

	// Wire blacklist cleanup task
	scheduler.SetBlacklistCleanupHandler(scheduler.BlacklistCleanupTask, blacklist.Cleanup)
	engine.Register(scheduler.BlacklistCleanupTask)

	// Wire expired token cleanup task
	refreshTokenRepo := do.MustInvoke[*repository.RefreshTokenRepo](injector)
	scheduler.SetExpiredTokenCleanupHandler(scheduler.ExpiredTokenCleanupTask, refreshTokenRepo.DeleteExpiredTokens)
	engine.Register(scheduler.ExpiredTokenCleanupTask)

	// Wire audit log cleanup task
	auditLogSvc := do.MustInvoke[*service.AuditLogService](injector)
	scheduler.SetAuditLogCleanupHandler(scheduler.AuditLogCleanupTask, auditLogSvc.Cleanup)
	engine.Register(scheduler.AuditLogCleanupTask)

	// Start scheduler (after all task registrations)
	if err := engine.Start(); err != nil {
		slog.Error("scheduler start failed", "error", err)
		os.Exit(1)
	}

	// Gin engine
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(otelgin.Middleware("metis"))
	r.Use(middleware.Logger(), middleware.Recovery())

	authedGroup := h.Register(r, jwtSecret, enforcer, blacklist)

	// Register App routes and tasks
	for _, a := range app.All() {
		a.Routes(authedGroup)
		for _, t := range a.Tasks() {
			engine.Register(&t)
		}
	}

	handler.RegisterStatic(r)

	// Server port
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	// Graceful shutdown: stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	// Flush pending OTel spans
	otelShutdown(ctx)

	// Shutdown IOC container (closes DB, etc.)
	report := injector.Shutdown()
	if errMsg := report.Error(); errMsg != "" {
		slog.Error("injector shutdown error", "error", errMsg)
	}

	slog.Info("server stopped")
}

func resolveJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		return []byte(secret)
	}

	// Generate random secret for this session
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("failed to generate JWT secret", "error", err)
		os.Exit(1)
	}
	slog.Warn("JWT_SECRET not set, using random secret — tokens will be invalidated on restart")
	return b
}

func runCreateAdmin() {
	// Parse flags manually from os.Args
	var username, password string
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if val, ok := parseFlag(arg, "--username"); ok {
			username = val
		} else if val, ok := parseFlag(arg, "--password"); ok {
			password = val
		}
	}

	if username == "" || password == "" {
		slog.Error("usage: metis create-admin --username=<name> --password=<pass>")
		os.Exit(1)
	}

	// Init DB directly (no full IOC needed)
	injector := do.New()
	do.Provide(injector, database.New)
	do.Provide(injector, repository.NewUser)
	do.Provide(injector, repository.NewRole)

	userRepo := do.MustInvoke[*repository.UserRepo](injector)
	roleRepo := do.MustInvoke[*repository.RoleRepo](injector)

	exists, err := userRepo.ExistsByUsername(username)
	if err != nil {
		slog.Error("database error", "error", err)
		os.Exit(1)
	}
	if exists {
		slog.Error("username already exists", "username", username)
		os.Exit(1)
	}

	// Find admin role
	adminRole, err := roleRepo.FindByCode(model.RoleAdmin)
	if err != nil {
		slog.Error("admin role not found — run 'metis seed' first", "error", err)
		os.Exit(1)
	}

	hashedPassword, err := token.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	user := &model.User{
		Username: username,
		Password: hashedPassword,
		RoleID:   adminRole.ID,
		IsActive: true,
	}
	if err := userRepo.Create(user); err != nil {
		slog.Error("failed to create admin", "error", err)
		os.Exit(1)
	}

	slog.Info("admin user created", "username", username, "id", user.ID)

	report := injector.Shutdown()
	if errMsg := report.Error(); errMsg != "" {
		slog.Error("shutdown error", "error", errMsg)
	}
}

func runSeed() {
	injector := do.New()
	do.Provide(injector, database.New)
	do.Provide(injector, casbinpkg.NewEnforcer)

	db := do.MustInvoke[*database.DB](injector)
	enforcer := do.MustInvoke[*casbin.Enforcer](injector)

	result, err := seed.Run(db.DB, enforcer)
	if err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Seed complete:\n")
	fmt.Printf("  Roles: %d created, %d skipped\n", result.RolesCreated, result.RolesSkipped)
	fmt.Printf("  Menus: %d created, %d skipped\n", result.MenusCreated, result.MenusSkipped)
	fmt.Printf("  Policies: %d added\n", result.PoliciesAdded)

	report := injector.Shutdown()
	if errMsg := report.Error(); errMsg != "" {
		slog.Error("shutdown error", "error", errMsg)
	}
}

func parseFlag(arg, name string) (string, bool) {
	prefix := name + "="
	if len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
		return arg[len(prefix):], true
	}
	return "", false
}
