package sla

import (
	"errors"
	. "metis/internal/app/itsm/domain"
	"strings"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

var (
	ErrSLATemplateNotFound        = errors.New("SLA template not found")
	ErrSLACodeExists              = errors.New("SLA code already exists")
	ErrSLATemplateInUse           = errors.New("SLA template is referenced by active services")
	ErrSLATemplateInvalidDuration = errors.New("SLA durations must be positive")
	ErrSLATemplateInvalidIdentifier = errors.New("SLA template name and code must not be blank")
)

type SLATemplateService struct {
	repo *SLATemplateRepo
	db   *database.DB
}

func NewSLATemplateService(i do.Injector) (*SLATemplateService, error) {
	repo := do.MustInvoke[*SLATemplateRepo](i)
	db := do.MustInvoke[*database.DB](i)
	return &SLATemplateService{repo: repo, db: db}, nil
}

func (s *SLATemplateService) Create(sla *SLATemplate) (*SLATemplate, error) {
	sla.Name = strings.TrimSpace(sla.Name)
	sla.Code = strings.TrimSpace(sla.Code)
	if sla.Name == "" || sla.Code == "" {
		return nil, ErrSLATemplateInvalidIdentifier
	}
	if err := validateSLADurations(sla.ResponseMinutes, sla.ResolutionMinutes); err != nil {
		return nil, err
	}
	if _, err := s.repo.FindByCode(sla.Code); err == nil {
		return nil, ErrSLACodeExists
	}
	sla.IsActive = true
	if err := s.repo.Create(sla); err != nil {
		return nil, err
	}
	return s.repo.FindByID(sla.ID)
}

func (s *SLATemplateService) Get(id uint) (*SLATemplate, error) {
	sla, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSLATemplateNotFound
		}
		return nil, err
	}
	return sla, nil
}

func (s *SLATemplateService) Update(id uint, updates map[string]any) (*SLATemplate, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSLATemplateNotFound
		}
		return nil, err
	}
	if code, ok := updates["code"].(string); ok && code != existing.Code {
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, ErrSLATemplateInvalidIdentifier
		}
		updates["code"] = code
		if _, err := s.repo.FindByCode(code); err == nil {
			return nil, ErrSLACodeExists
		}
	}
	if name, ok := updates["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrSLATemplateInvalidIdentifier
		}
		updates["name"] = name
	}
	responseMinutes := existing.ResponseMinutes
	if v, ok := updates["response_minutes"].(int); ok {
		responseMinutes = v
	}
	resolutionMinutes := existing.ResolutionMinutes
	if v, ok := updates["resolution_minutes"].(int); ok {
		resolutionMinutes = v
	}
	if err := validateSLADurations(responseMinutes, resolutionMinutes); err != nil {
		return nil, err
	}
	if isActive, ok := updates["is_active"].(bool); ok && !isActive && existing.IsActive {
		if err := s.ensureNotReferencedByActiveService(id); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *SLATemplateService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSLATemplateNotFound
		}
		return err
	}
	if err := s.ensureNotReferencedByActiveService(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *SLATemplateService) ListAll() ([]SLATemplate, error) {
	return s.repo.ListAll()
}

func (s *SLATemplateService) ensureNotReferencedByActiveService(id uint) error {
	var count int64
	if err := s.db.Model(&ServiceDefinition{}).
		Where("sla_id = ? AND is_active = ?", id, true).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrSLATemplateInUse
	}
	return nil
}

func validateSLADurations(responseMinutes, resolutionMinutes int) error {
	if responseMinutes <= 0 || resolutionMinutes <= 0 {
		return ErrSLATemplateInvalidDuration
	}
	return nil
}
