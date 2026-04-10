package app

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/scheduler"
)

// App is the interface that pluggable modules must implement.
type App interface {
	Name() string
	Models() []any
	Seed(db *gorm.DB, enforcer *casbin.Enforcer) error
	Providers(i do.Injector)
	Routes(api *gin.RouterGroup)
	Tasks() []scheduler.TaskDef
}

var apps []App

func Register(a App) { apps = append(apps, a) }
func All() []App     { return apps }
