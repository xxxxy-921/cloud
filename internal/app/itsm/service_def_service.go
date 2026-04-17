package itsm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"metis/internal/app/ai"
	"metis/internal/database"

	"metis/internal/app/itsm/engine"
)

var (
	ErrServiceDefNotFound    = errors.New("service definition not found")
	ErrServiceCodeExists     = errors.New("service code already exists")
	ErrWorkflowValidation    = errors.New("workflow validation failed")
	ErrServiceEngineMismatch = errors.New("service engine field mismatch")
	ErrAgentNotAvailable     = errors.New("agent not available")
)

type ServiceDefService struct {
	repo     *ServiceDefRepo
	db       *database.DB
	catalogs *CatalogRepo
}

func NewServiceDefService(i do.Injector) (*ServiceDefService, error) {
	repo := do.MustInvoke[*ServiceDefRepo](i)
	db := do.MustInvoke[*database.DB](i)
	catalogs := do.MustInvoke[*CatalogRepo](i)
	return &ServiceDefService{repo: repo, db: db, catalogs: catalogs}, nil
}

func (s *ServiceDefService) Create(svc *ServiceDefinition) (*ServiceDefinition, error) {
	if _, err := s.repo.FindByCode(svc.Code); err == nil {
		return nil, ErrServiceCodeExists
	}
	if err := s.validateCatalogID(svc.CatalogID); err != nil {
		return nil, err
	}
	if err := s.validateEngineFields(svc.EngineType, svc.WorkflowJSON, svc.CollaborationSpec, svc.AgentID); err != nil {
		return nil, err
	}
	if err := s.validateAgent(svc.AgentID); err != nil {
		return nil, err
	}
	// Validate workflow_json for classic engine
	if svc.EngineType == "classic" && len(svc.WorkflowJSON) > 0 {
		if err := validateWorkflowJSON(json.RawMessage(svc.WorkflowJSON)); err != nil {
			return nil, err
		}
	}
	svc.IsActive = true
	if err := s.repo.Create(svc); err != nil {
		if isSQLiteUniqueError(err) {
			return nil, ErrServiceCodeExists
		}
		return nil, err
	}
	return s.repo.FindByID(svc.ID)
}

func (s *ServiceDefService) Get(id uint) (*ServiceDefinition, error) {
	svc, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceDefNotFound
		}
		return nil, err
	}
	return svc, nil
}

func (s *ServiceDefService) Update(id uint, updates map[string]any) (*ServiceDefinition, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceDefNotFound
		}
		return nil, err
	}
	if code, ok := updates["code"].(string); ok && code != existing.Code {
		if _, err := s.repo.FindByCode(code); err == nil {
			return nil, ErrServiceCodeExists
		}
	}
	if catalogID, ok := updates["catalog_id"].(uint); ok {
		if err := s.validateCatalogID(catalogID); err != nil {
			return nil, err
		}
	}
	engineType := existing.EngineType
	if et, ok := updates["engine_type"].(string); ok {
		engineType = et
	}
	workflowJSON := existing.WorkflowJSON
	if v, ok := updates["workflow_json"].(JSONField); ok {
		workflowJSON = v
	}
	collaborationSpec := existing.CollaborationSpec
	if v, ok := updates["collaboration_spec"].(string); ok {
		collaborationSpec = v
	}
	agentID := existing.AgentID
	if v, ok := updates["agent_id"].(uint); ok {
		agentID = &v
	}
	if err := s.validateEngineFields(engineType, workflowJSON, collaborationSpec, agentID); err != nil {
		return nil, err
	}
	if err := s.validateAgent(agentID); err != nil {
		return nil, err
	}
	// Validate workflow_json if being updated for classic engine
	if wfJSON, ok := updates["workflow_json"]; ok {
		if engineType == "classic" {
			if raw, ok := wfJSON.(JSONField); ok && len(raw) > 0 {
				if err := validateWorkflowJSON(json.RawMessage(raw)); err != nil {
					return nil, err
				}
			}
		}
	}

	if err := s.repo.Update(id, updates); err != nil {
		if isSQLiteUniqueError(err) {
			return nil, ErrServiceCodeExists
		}
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *ServiceDefService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServiceDefNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}

func (s *ServiceDefService) List(params ServiceDefListParams) ([]ServiceDefinition, int64, error) {
	return s.repo.List(params)
}

// validateWorkflowJSON runs the engine validator and wraps errors.
func validateWorkflowJSON(raw json.RawMessage) error {
	errs := engine.ValidateWorkflow(raw)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrWorkflowValidation, errs[0].Message)
}

func (s *ServiceDefService) validateCatalogID(catalogID uint) error {
	if _, err := s.catalogs.FindByID(catalogID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCatalogNotFound
		}
		return err
	}
	return nil
}

func (s *ServiceDefService) validateEngineFields(engineType string, workflowJSON JSONField, collaborationSpec string, agentID *uint) error {
	switch engineType {
	case "classic":
		if agentID != nil && *agentID != 0 {
			return ErrServiceEngineMismatch
		}
	}
	return nil
}

func (s *ServiceDefService) validateAgent(agentID *uint) error {
	if agentID == nil || *agentID == 0 {
		return nil
	}
	var agent ai.Agent
	if err := s.db.First(&agent, *agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotAvailable
		}
		return err
	}
	if !agent.IsActive {
		return ErrAgentNotAvailable
	}
	return nil
}
