package repository

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

type TwoFactorSecretRepo struct {
	db *database.DB
}

func NewTwoFactorSecret(i do.Injector) (*TwoFactorSecretRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &TwoFactorSecretRepo{db: db}, nil
}

func (r *TwoFactorSecretRepo) FindByUserID(userID uint) (*model.TwoFactorSecret, error) {
	var secret model.TwoFactorSecret
	if err := r.db.Where("user_id = ?", userID).First(&secret).Error; err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *TwoFactorSecretRepo) Create(secret *model.TwoFactorSecret) error {
	return r.db.Create(secret).Error
}

func (r *TwoFactorSecretRepo) Update(secret *model.TwoFactorSecret) error {
	return r.db.Save(secret).Error
}

func (r *TwoFactorSecretRepo) DeleteByUserID(userID uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&model.TwoFactorSecret{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
