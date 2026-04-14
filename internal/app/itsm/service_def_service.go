package itsm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app/itsm/engine"
)

var (
	ErrServiceDefNotFound    = errors.New("service definition not found")
	ErrServiceCodeExists     = errors.New("service code already exists")
	ErrWorkflowValidation    = errors.New("workflow validation failed")
)

type ServiceDefService struct {
	repo *ServiceDefRepo
}

func NewServiceDefService(i do.Injector) (*ServiceDefService, error) {
	repo := do.MustInvoke[*ServiceDefRepo](i)
	return &ServiceDefService{repo: repo}, nil
}

func (s *ServiceDefService) Create(svc *ServiceDefinition) (*ServiceDefinition, error) {
	if _, err := s.repo.FindByCode(svc.Code); err == nil {
		return nil, ErrServiceCodeExists
	}
	// Validate workflow_json for classic engine
	if svc.EngineType == "classic" && len(svc.WorkflowJSON) > 0 {
		if err := validateWorkflowJSON(json.RawMessage(svc.WorkflowJSON)); err != nil {
			return nil, err
		}
	}
	svc.IsActive = true
	if err := s.repo.Create(svc); err != nil {
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
	// Validate workflow_json if being updated for classic engine
	if wfJSON, ok := updates["workflow_json"]; ok {
		engineType := existing.EngineType
		if et, ok2 := updates["engine_type"].(string); ok2 {
			engineType = et
		}
		if engineType == "classic" {
			if raw, ok := wfJSON.(JSONField); ok && len(raw) > 0 {
				if err := validateWorkflowJSON(json.RawMessage(raw)); err != nil {
					return nil, err
				}
			}
		}
	}

	if err := s.repo.Update(id, updates); err != nil {
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
