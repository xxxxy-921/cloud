package ai

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app"
	"metis/internal/scheduler"
)

func init() {
	app.Register(&AIApp{})
}

type AIApp struct {
	injector do.Injector
}

func (a *AIApp) Name() string { return "ai" }

func (a *AIApp) Models() []any {
	return []any{&Provider{}, &AIModel{}, &AILog{}}
}

func (a *AIApp) Seed(db *gorm.DB, enforcer *casbin.Enforcer) error {
	return seedAI(db, enforcer)
}

func (a *AIApp) Providers(i do.Injector) {
	a.injector = i
	do.Provide(i, NewProviderRepo)
	do.Provide(i, NewModelRepo)
	do.Provide(i, NewProviderService)
	do.Provide(i, NewModelService)
	do.Provide(i, NewProviderHandler)
	do.Provide(i, NewModelHandler)
}

func (a *AIApp) Routes(api *gin.RouterGroup) {
	providerH := do.MustInvoke[*ProviderHandler](a.injector)
	modelH := do.MustInvoke[*ModelHandler](a.injector)

	providers := api.Group("/ai/providers")
	{
		providers.POST("", providerH.Create)
		providers.GET("", providerH.List)
		providers.GET("/:id", providerH.Get)
		providers.PUT("/:id", providerH.Update)
		providers.DELETE("/:id", providerH.Delete)
		providers.POST("/:id/test", providerH.TestConnection)
		providers.POST("/:id/sync-models", modelH.SyncModels)
	}

	models := api.Group("/ai/models")
	{
		models.POST("", modelH.Create)
		models.GET("", modelH.List)
		models.GET("/:id", modelH.Get)
		models.PUT("/:id", modelH.Update)
		models.DELETE("/:id", modelH.Delete)
		models.PATCH("/:id/default", modelH.SetDefault)
	}
}

func (a *AIApp) Tasks() []scheduler.TaskDef {
	return nil
}
