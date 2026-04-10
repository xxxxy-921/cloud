package database

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"

	"metis/internal/model"
)

// DB wraps gorm.DB to implement do.Shutdowner.
type DB struct {
	*gorm.DB
}

func New(i do.Injector) (*DB, error) {
	driver := os.Getenv("DB_DRIVER")
	dsn := os.Getenv("DB_DSN")

	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		return nil, fmt.Errorf("postgres driver not yet installed; run: go get gorm.io/driver/postgres")
	default:
		if dsn == "" {
			dsn = "metis.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
		}
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// OpenTelemetry: auto-trace all DB queries (noop when OTel is disabled)
	if err := db.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables())); err != nil {
		return nil, fmt.Errorf("otelgorm plugin failed: %w", err)
	}

	if err := db.AutoMigrate(
		&model.SystemConfig{},
		&model.Role{},
		&model.Menu{},
		&model.User{},
		&model.RefreshToken{},
		&model.AuthProvider{},
		&model.UserConnection{},
		&model.TwoFactorSecret{},
		&model.TaskState{},
		&model.TaskExecution{},
		&model.Notification{},
		&model.NotificationRead{},
		&model.MessageChannel{},
		&model.AuditLog{},
		&model.IdentitySource{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	slog.Info("database initialized", "driver", driver, "dsn", dsn)
	return &DB{DB: db}, nil
}

// Shutdown implements do.ShutdownerWithError for graceful cleanup.
func (d *DB) Shutdown() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	slog.Info("closing database connection")
	return sqlDB.Close()
}
