package repository

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
	"metis/internal/model"
)

type AuthProviderRepo struct {
	db *database.DB
}

func NewAuthProvider(i do.Injector) (*AuthProviderRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &AuthProviderRepo{db: db}, nil
}

func (r *AuthProviderRepo) FindByKey(key string) (*model.AuthProvider, error) {
	var p model.AuthProvider
	if err := r.db.Where("provider_key = ?", key).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AuthProviderRepo) FindAllEnabled() ([]model.AuthProvider, error) {
	var providers []model.AuthProvider
	if err := r.db.Where("enabled = ?", true).Order("sort_order ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *AuthProviderRepo) FindAll() ([]model.AuthProvider, error) {
	var providers []model.AuthProvider
	if err := r.db.Order("sort_order ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *AuthProviderRepo) Update(p *model.AuthProvider) error {
	return r.db.Save(p).Error
}
