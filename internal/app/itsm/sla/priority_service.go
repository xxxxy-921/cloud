package sla

import (
	"errors"
	. "metis/internal/app/itsm/domain"
	"strings"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrPriorityNotFound   = errors.New("priority not found")
	ErrPriorityCodeExists = errors.New("priority code already exists")
	ErrPriorityInvalidValue      = errors.New("priority value must be positive")
	ErrPriorityInvalidIdentifier = errors.New("priority name and code must not be blank")
)

type PriorityService struct {
	repo *PriorityRepo
}

func NewPriorityService(i do.Injector) (*PriorityService, error) {
	repo := do.MustInvoke[*PriorityRepo](i)
	return &PriorityService{repo: repo}, nil
}

func (s *PriorityService) Create(p *Priority) (*Priority, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Code = strings.TrimSpace(p.Code)
	if p.Name == "" || p.Code == "" {
		return nil, ErrPriorityInvalidIdentifier
	}
	if p.Value <= 0 {
		return nil, ErrPriorityInvalidValue
	}
	if _, err := s.repo.FindByCode(p.Code); err == nil {
		return nil, ErrPriorityCodeExists
	}
	p.IsActive = true
	if err := s.repo.Create(p); err != nil {
		if IsSQLiteUniqueError(err) {
			return nil, ErrPriorityCodeExists
		}
		return nil, err
	}
	return s.repo.FindByID(p.ID)
}

func (s *PriorityService) Get(id uint) (*Priority, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPriorityNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *PriorityService) Update(id uint, updates map[string]any) (*Priority, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPriorityNotFound
		}
		return nil, err
	}
	if code, ok := updates["code"].(string); ok && code != existing.Code {
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, ErrPriorityInvalidIdentifier
		}
		updates["code"] = code
		if _, err := s.repo.FindByCode(code); err == nil {
			return nil, ErrPriorityCodeExists
		}
	}
	if name, ok := updates["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, ErrPriorityInvalidIdentifier
		}
		updates["name"] = name
	}
	if value, ok := updates["value"].(int); ok && value <= 0 {
		return nil, ErrPriorityInvalidValue
	}
	if err := s.repo.Update(id, updates); err != nil {
		if IsSQLiteUniqueError(err) {
			return nil, ErrPriorityCodeExists
		}
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *PriorityService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPriorityNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}

func (s *PriorityService) ListAll() ([]Priority, error) {
	return s.repo.ListAll()
}
