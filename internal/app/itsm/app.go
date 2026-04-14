package itsm

import (
	"context"
	"encoding/json"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app"
	"metis/internal/app/itsm/engine"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/scheduler"
)

func init() {
	app.Register(&ITSMApp{})
}

type ITSMApp struct {
	injector do.Injector
}

func (a *ITSMApp) Name() string { return "itsm" }

func (a *ITSMApp) Models() []any {
	return []any{
		// Configuration models
		&ServiceCatalog{},
		&ServiceDefinition{},
		&ServiceAction{},
		&Priority{},
		&SLATemplate{},
		&EscalationRule{},
		// Ticket lifecycle models
		&Ticket{},
		&TicketActivity{},
		&TicketAssignment{},
		&TicketTimeline{},
		&TicketActionExecution{},
		// Incident models
		&TicketLink{},
		&PostMortem{},
	}
}

func (a *ITSMApp) Seed(db *gorm.DB, enforcer *casbin.Enforcer) error {
	return seedITSM(db, enforcer)
}

func (a *ITSMApp) Providers(i do.Injector) {
	a.injector = i
	// Repositories
	do.Provide(i, NewCatalogRepo)
	do.Provide(i, NewServiceDefRepo)
	do.Provide(i, NewServiceActionRepo)
	do.Provide(i, NewPriorityRepo)
	do.Provide(i, NewSLATemplateRepo)
	do.Provide(i, NewEscalationRuleRepo)
	do.Provide(i, NewTicketRepo)
	do.Provide(i, NewTimelineRepo)

	// Engine components
	do.Provide(i, func(i do.Injector) (*engine.ParticipantResolver, error) {
		// Try to resolve OrgService (optional — nil if Org App not installed)
		var orgSvc engine.OrgService
		// Org App provides OrgScopeResolver; we don't have a direct OrgService interface yet,
		// so for now the resolver starts with nil (user type and requester_manager basic support)
		return engine.NewParticipantResolver(orgSvc), nil
	})

	do.Provide(i, func(i do.Injector) (*engine.ClassicEngine, error) {
		resolver := do.MustInvoke[*engine.ParticipantResolver](i)
		db := do.MustInvoke[*database.DB](i)
		// Create a TaskSubmitter that uses the scheduler engine
		submitter := &schedulerSubmitter{db: db.DB}
		return engine.NewClassicEngine(resolver, submitter), nil
	})

	// Services
	do.Provide(i, NewCatalogService)
	do.Provide(i, NewServiceDefService)
	do.Provide(i, NewServiceActionService)
	do.Provide(i, NewPriorityService)
	do.Provide(i, NewSLATemplateService)
	do.Provide(i, NewEscalationRuleService)
	do.Provide(i, NewTicketService)
	do.Provide(i, NewTimelineService)
	// Handlers
	do.Provide(i, NewCatalogHandler)
	do.Provide(i, NewServiceDefHandler)
	do.Provide(i, NewServiceActionHandler)
	do.Provide(i, NewPriorityHandler)
	do.Provide(i, NewSLATemplateHandler)
	do.Provide(i, NewEscalationRuleHandler)
	do.Provide(i, NewTicketHandler)
}

func (a *ITSMApp) Routes(api *gin.RouterGroup) {
	catalogH := do.MustInvoke[*CatalogHandler](a.injector)
	serviceH := do.MustInvoke[*ServiceDefHandler](a.injector)
	actionH := do.MustInvoke[*ServiceActionHandler](a.injector)
	priorityH := do.MustInvoke[*PriorityHandler](a.injector)
	slaH := do.MustInvoke[*SLATemplateHandler](a.injector)
	escalationH := do.MustInvoke[*EscalationRuleHandler](a.injector)
	ticketH := do.MustInvoke[*TicketHandler](a.injector)

	g := api.Group("/itsm")
	{
		// Service Catalogs
		g.POST("/catalogs", catalogH.Create)
		g.GET("/catalogs/tree", catalogH.Tree)
		g.PUT("/catalogs/:id", catalogH.Update)
		g.DELETE("/catalogs/:id", catalogH.Delete)

		// Service Definitions
		g.POST("/services", serviceH.Create)
		g.GET("/services", serviceH.List)
		g.GET("/services/:id", serviceH.Get)
		g.PUT("/services/:id", serviceH.Update)
		g.DELETE("/services/:id", serviceH.Delete)

		// Service Actions
		g.POST("/services/:id/actions", actionH.Create)
		g.GET("/services/:id/actions", actionH.List)
		g.PUT("/services/:id/actions/:actionId", actionH.Update)
		g.DELETE("/services/:id/actions/:actionId", actionH.Delete)

		// Priorities
		g.POST("/priorities", priorityH.Create)
		g.GET("/priorities", priorityH.List)
		g.PUT("/priorities/:id", priorityH.Update)
		g.DELETE("/priorities/:id", priorityH.Delete)

		// SLA Templates
		g.POST("/sla", slaH.Create)
		g.GET("/sla", slaH.List)
		g.PUT("/sla/:id", slaH.Update)
		g.DELETE("/sla/:id", slaH.Delete)

		// Escalation Rules
		g.POST("/sla/:id/escalations", escalationH.Create)
		g.GET("/sla/:id/escalations", escalationH.List)
		g.PUT("/sla/:id/escalations/:escalationId", escalationH.Update)
		g.DELETE("/sla/:id/escalations/:escalationId", escalationH.Delete)

		// Tickets — special views must come before :id routes
		g.GET("/tickets/mine", ticketH.Mine)
		g.GET("/tickets/todo", ticketH.Todo)
		g.GET("/tickets/history", ticketH.History)
		g.POST("/tickets", ticketH.Create)
		g.GET("/tickets", ticketH.List)
		g.GET("/tickets/:id", ticketH.Get)
		g.PUT("/tickets/:id/assign", ticketH.Assign)
		g.PUT("/tickets/:id/complete", ticketH.Complete)
		g.PUT("/tickets/:id/cancel", ticketH.Cancel)
		g.GET("/tickets/:id/timeline", ticketH.Timeline)
		// Phase 2: Classic engine routes
		g.POST("/tickets/:id/progress", ticketH.Progress)
		g.POST("/tickets/:id/signal", ticketH.Signal)
		g.GET("/tickets/:id/activities", ticketH.Activities)
	}
}

func (a *ITSMApp) Tasks() []scheduler.TaskDef {
	db := do.MustInvoke[*database.DB](a.injector)
	classicEngine := do.MustInvoke[*engine.ClassicEngine](a.injector)

	return []scheduler.TaskDef{
		{
			Name:        "itsm-action-execute",
			Type:        scheduler.TypeAsync,
			Description: "Execute HTTP webhook for ITSM action nodes",
			Handler:     engine.HandleActionExecute(db.DB, classicEngine),
		},
		{
			Name:        "itsm-wait-timer",
			Type:        scheduler.TypeAsync,
			Description: "Check and trigger ITSM wait timer nodes",
			Handler:     engine.HandleWaitTimer(db.DB, classicEngine),
		},
	}
}

// schedulerSubmitter implements engine.TaskSubmitter by creating scheduler task records.
type schedulerSubmitter struct {
	db *gorm.DB
}

func (s *schedulerSubmitter) SubmitTask(name string, payload json.RawMessage) error {
	exec := &model.TaskExecution{
		TaskName: name,
		Trigger:  scheduler.TriggerAPI,
		Status:   scheduler.ExecPending,
		Payload:  string(payload),
	}
	return s.db.Create(exec).Error
}

// Ensure schedulerSubmitter implements engine.TaskSubmitter at compile time
var _ engine.TaskSubmitter = (*schedulerSubmitter)(nil)

// Ensure ClassicEngine implements engine.WorkflowEngine at compile time
var _ engine.WorkflowEngine = (*engine.ClassicEngine)(nil)

// Placeholder for background context usage
var _ = context.Background
