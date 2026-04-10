package service

import (
	"github.com/samber/do/v2"

	"metis/internal/model"
	"metis/internal/repository"
)

type SysConfigService struct {
	repo *repository.SysConfigRepo
}

func NewSysConfig(i do.Injector) (*SysConfigService, error) {
	repo := do.MustInvoke[*repository.SysConfigRepo](i)
	return &SysConfigService{repo: repo}, nil
}

func (s *SysConfigService) Get(key string) (*model.SystemConfig, error) {
	return s.repo.Get(key)
}

func (s *SysConfigService) List() ([]model.SystemConfig, error) {
	return s.repo.List()
}

func (s *SysConfigService) Set(cfg *model.SystemConfig) error {
	return s.repo.Set(cfg)
}

func (s *SysConfigService) Delete(key string) error {
	return s.repo.Delete(key)
}
