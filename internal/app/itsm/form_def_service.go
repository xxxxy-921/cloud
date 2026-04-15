package itsm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/app/itsm/form"
)

var (
	ErrFormDefNotFound   = errors.New("form definition not found")
	ErrFormDefCodeExists = errors.New("form definition code already exists")
	ErrFormDefInUse      = errors.New("form definition is in use by service definitions")
	ErrFormSchemaInvalid = errors.New("form schema validation failed")
)

type FormDefService struct {
	repo *FormDefRepo
}

func NewFormDefService(i do.Injector) (*FormDefService, error) {
	repo := do.MustInvoke[*FormDefRepo](i)
	return &FormDefService{repo: repo}, nil
}

func (s *FormDefService) Create(fd *FormDefinition) (*FormDefinition, error) {
	if _, err := s.repo.FindByCode(fd.Code); err == nil {
		return nil, ErrFormDefCodeExists
	}

	if err := s.validateSchema(fd.Schema); err != nil {
		return nil, err
	}

	fd.Version = 1
	fd.IsActive = true
	if fd.Scope == "" {
		fd.Scope = "global"
	}

	if err := s.repo.Create(fd); err != nil {
		return nil, err
	}
	return s.repo.FindByID(fd.ID)
}

func (s *FormDefService) Get(id uint) (*FormDefinition, error) {
	fd, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFormDefNotFound
		}
		return nil, err
	}
	return fd, nil
}

func (s *FormDefService) GetByCode(code string) (*FormDefinition, error) {
	fd, err := s.repo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFormDefNotFound
		}
		return nil, err
	}
	return fd, nil
}

func (s *FormDefService) Update(id uint, updates map[string]any) (*FormDefinition, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFormDefNotFound
		}
		return nil, err
	}

	// Check code uniqueness if being changed
	if code, ok := updates["code"].(string); ok && code != existing.Code {
		if _, err := s.repo.FindByCode(code); err == nil {
			return nil, ErrFormDefCodeExists
		}
	}

	// Validate schema if being updated
	if schemaStr, ok := updates["schema"].(string); ok {
		if err := s.validateSchema(schemaStr); err != nil {
			return nil, err
		}
	}

	// Bump version
	updates["version"] = existing.Version + 1

	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *FormDefService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFormDefNotFound
		}
		return err
	}

	// Check if any ServiceDefinition references this form
	count, err := s.repo.CountServiceRefs(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrFormDefInUse
	}

	return s.repo.Delete(id)
}

func (s *FormDefService) List(params FormDefListParams) ([]FormDefinition, int64, error) {
	return s.repo.List(params)
}

func (s *FormDefService) validateSchema(schemaStr string) error {
	var schema form.FormSchema
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", ErrFormSchemaInvalid, err)
	}
	errs := form.ValidateSchema(schema)
	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", ErrFormSchemaInvalid, errs[0].Error())
	}
	return nil
}
