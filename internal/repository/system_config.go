package repository

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
	"metis/internal/model"
)

type SysConfigRepo struct {
	db *database.DB
}

func NewSysConfig(i do.Injector) (*SysConfigRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &SysConfigRepo{db: db}, nil
}

func (r *SysConfigRepo) Get(key string) (*model.SystemConfig, error) {
	var cfg model.SystemConfig
	if err := r.db.Where("`key` = ?", key).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *SysConfigRepo) List() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	if err := r.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *SysConfigRepo) Set(cfg *model.SystemConfig) error {
	return r.db.Save(cfg).Error
}

func (r *SysConfigRepo) Delete(key string) error {
	result := r.db.Where("`key` = ?", key).Delete(&model.SystemConfig{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
