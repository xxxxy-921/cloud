package itsm

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrEscalationRuleNotFound = errors.New("escalation rule not found")
	ErrEscalationLevelExists  = errors.New("escalation level already exists for this SLA and trigger type")
)

type EscalationRuleService struct {
	repo *EscalationRuleRepo
}

func NewEscalationRuleService(i do.Injector) (*EscalationRuleService, error) {
	repo := do.MustInvoke[*EscalationRuleRepo](i)
	return &EscalationRuleService{repo: repo}, nil
}

func (s *EscalationRuleService) Create(rule *EscalationRule) (*EscalationRule, error) {
	if _, err := s.repo.FindBySLATriggerLevel(rule.SLAID, rule.TriggerType, rule.Level); err == nil {
		return nil, ErrEscalationLevelExists
	}
	rule.IsActive = true
	if err := s.repo.Create(rule); err != nil {
		return nil, err
	}
	return s.repo.FindByID(rule.ID)
}

func (s *EscalationRuleService) Get(id uint) (*EscalationRule, error) {
	r, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEscalationRuleNotFound
		}
		return nil, err
	}
	return r, nil
}

func (s *EscalationRuleService) Update(id uint, updates map[string]any) (*EscalationRule, error) {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEscalationRuleNotFound
		}
		return nil, err
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *EscalationRuleService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEscalationRuleNotFound
		}
		return err
	}
	return s.repo.Delete(id)
}

func (s *EscalationRuleService) ListBySLA(slaID uint) ([]EscalationRule, error) {
	return s.repo.ListBySLA(slaID)
}
