package ticket

import (
	"errors"
	"strings"
	"testing"

	. "metis/internal/app/itsm/definition"
	. "metis/internal/app/itsm/domain"
	. "metis/internal/app/itsm/sla"
	"metis/internal/app/itsm/testutil"

	"gorm.io/gorm"
)

func TestCreateCatalogRejectsSmartServiceAndCreateChecksPriorityAndActivation(t *testing.T) {
	t.Run("smart service cannot use catalog classic submit", func(t *testing.T) {
		db := newTestDB(t)
		ticketSvc := newSubmissionTicketService(t, db)
		smartService := testutil.SeedSmartSubmissionService(t, db)

		if _, err := ticketSvc.CreateCatalog(CreateTicketInput{
			Title:      "smart catalog submit",
			ServiceID:  smartService.ID,
			PriorityID: 1,
			FormData:   JSONField(`{"vpn_account":"smart@example.com"}`),
		}, 7); !errors.Is(err, ErrCatalogSubmissionClassic) {
			t.Fatalf("CreateCatalog smart error = %v, want %v", err, ErrCatalogSubmissionClassic)
		}
	})

	t.Run("missing service returns not found consistently", func(t *testing.T) {
		db := newTestDB(t)
		ticketSvc := newSubmissionTicketService(t, db)

		if _, err := ticketSvc.CreateCatalog(CreateTicketInput{
			Title:      "missing service",
			ServiceID:  999999,
			PriorityID: 1,
			FormData:   JSONField(`{}`),
		}, 7); !errors.Is(err, ErrServiceDefNotFound) {
			t.Fatalf("CreateCatalog missing service error = %v, want %v", err, ErrServiceDefNotFound)
		}
	})

	t.Run("inactive service and missing priority are rejected", func(t *testing.T) {
		db := newTestDB(t)
		ticketSvc := newSubmissionTicketService(t, db)
		classicService, priority := seedClassicSubmissionService(t, db)

		if err := db.Model(&ServiceDefinition{}).Where("id = ?", classicService.ID).Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate classic service: %v", err)
		}
		if _, err := ticketSvc.Create(CreateTicketInput{
			Title:      "inactive service",
			ServiceID:  classicService.ID,
			PriorityID: priority.ID,
			FormData:   JSONField(`{"vpn_account":"classic@example.com"}`),
		}, 7); !errors.Is(err, ErrServiceNotActive) {
			t.Fatalf("Create inactive service error = %v, want %v", err, ErrServiceNotActive)
		}

		if err := db.Model(&ServiceDefinition{}).Where("id = ?", classicService.ID).Update("is_active", true).Error; err != nil {
			t.Fatalf("reactivate classic service: %v", err)
		}
		if _, err := ticketSvc.Create(CreateTicketInput{
			Title:      "missing priority",
			ServiceID:  classicService.ID,
			PriorityID: 999,
			FormData:   JSONField(`{"vpn_account":"classic@example.com"}`),
		}, 7); !errors.Is(err, ErrPriorityNotFound) {
			t.Fatalf("Create missing priority error = %v, want %v", err, ErrPriorityNotFound)
		}
	})
}

func TestCreateRejectsInvalidClassicWorkflowDefinition(t *testing.T) {
	db := newTestDB(t)
	ticketSvc := newSubmissionTicketService(t, db)
	classicService, priority := seedClassicSubmissionService(t, db)

	if err := db.Model(&ServiceDefinition{}).Where("id = ?", classicService.ID).Update("workflow_json", JSONField(`{"nodes":[{"id":"start"}],"edges":[]}`)).Error; err != nil {
		t.Fatalf("break workflow json: %v", err)
	}

	_, err := ticketSvc.Create(CreateTicketInput{
		Title:      "invalid classic workflow",
		ServiceID:  classicService.ID,
		PriorityID: priority.ID,
		FormData:   JSONField(`{"vpn_account":"classic@example.com"}`),
	}, 7)
	if err == nil || !strings.Contains(err.Error(), "工作流校验失败") {
		t.Fatalf("expected workflow validation failure, got %v", err)
	}
}

func TestCreateRejectsInactiveSLATemplateSnapshot(t *testing.T) {
	db := newTestDB(t)
	ticketSvc := newSubmissionTicketService(t, db)
	classicService, priority := seedClassicSubmissionService(t, db)

	sla := SLATemplate{
		Name:              "Inactive SLA",
		Code:              "inactive-sla",
		ResponseMinutes:   5,
		ResolutionMinutes: 30,
		IsActive:          false,
	}
	if err := db.Create(&sla).Error; err != nil {
		t.Fatalf("create sla: %v", err)
	}
	version := ServiceDefinitionVersion{
		ServiceID:       classicService.ID,
		Version:         1,
		ContentHash:     "manual-empty-sla-snapshot",
		EngineType:      "classic",
		SLAID:           &sla.ID,
		WorkflowJSON:    classicService.WorkflowJSON,
		SLATemplateJSON: nil,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("create service version: %v", err)
	}

	if _, err := ticketSvc.Create(CreateTicketInput{
		Title:            "inactive sla",
		ServiceID:        classicService.ID,
		ServiceVersionID: &version.ID,
		PriorityID:       priority.ID,
		FormData:         JSONField(`{"vpn_account":"classic@example.com"}`),
	}, 7); !errors.Is(err, ErrSLATemplateNotFound) {
		t.Fatalf("Create inactive SLA error = %v, want %v", err, ErrSLATemplateNotFound)
	}
}

func TestCreateRejectsServiceVersionFromAnotherService(t *testing.T) {
	db := newTestDB(t)
	ticketSvc := newSubmissionTicketService(t, db)
	classicService, priority := seedClassicSubmissionService(t, db)
	otherCatalog := ServiceCatalog{Name: "其他目录", Code: "classic-account-other", IsActive: true}
	if err := db.Create(&otherCatalog).Error; err != nil {
		t.Fatalf("create other catalog: %v", err)
	}
	otherService := ServiceDefinition{
		Name:         "其他经典服务",
		Code:         "classic-vpn-access-other",
		CatalogID:    otherCatalog.ID,
		EngineType:   "classic",
		WorkflowJSON: classicService.WorkflowJSON,
		IsActive:     true,
	}
	if err := db.Create(&otherService).Error; err != nil {
		t.Fatalf("create other service: %v", err)
	}

	otherVersion, err := ticketSvc.serviceRepo.GetOrCreateRuntimeVersion(otherService.ID)
	if err != nil {
		t.Fatalf("GetOrCreateRuntimeVersion other service: %v", err)
	}

	_, err = ticketSvc.Create(CreateTicketInput{
		Title:            "cross service version should fail",
		ServiceID:        classicService.ID,
		ServiceVersionID: &otherVersion.ID,
		PriorityID:       priority.ID,
		FormData:         JSONField(`{"vpn_account":"classic@example.com"}`),
	}, 7)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Create cross-service version error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}
