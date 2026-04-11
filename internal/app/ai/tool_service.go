package ai

import (
	"errors"

	"github.com/samber/do/v2"
)

var (
	ErrToolNotFound = errors.New("tool not found")
)

type ToolService struct {
	repo *ToolRepo
}

func NewToolService(i do.Injector) (*ToolService, error) {
	return &ToolService{
		repo: do.MustInvoke[*ToolRepo](i),
	}, nil
}

func (s *ToolService) List() ([]Tool, error) {
	return s.repo.List()
}

func (s *ToolService) ToggleActive(id uint, isActive bool) (*Tool, error) {
	t, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrToolNotFound
	}
	t.IsActive = isActive
	if err := s.repo.Update(t); err != nil {
		return nil, err
	}
	return t, nil
}
