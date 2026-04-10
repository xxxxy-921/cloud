package repository

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
	"metis/internal/model"
)

type UserConnectionRepo struct {
	db *database.DB
}

func NewUserConnection(i do.Injector) (*UserConnectionRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &UserConnectionRepo{db: db}, nil
}

func (r *UserConnectionRepo) FindByUserID(userID uint) ([]model.UserConnection, error) {
	var conns []model.UserConnection
	if err := r.db.Where("user_id = ?", userID).Find(&conns).Error; err != nil {
		return nil, err
	}
	return conns, nil
}

func (r *UserConnectionRepo) FindByProviderAndExternalID(provider, externalID string) (*model.UserConnection, error) {
	var conn model.UserConnection
	if err := r.db.Where("provider = ? AND external_id = ?", provider, externalID).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *UserConnectionRepo) FindByUserAndProvider(userID uint, provider string) (*model.UserConnection, error) {
	var conn model.UserConnection
	if err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *UserConnectionRepo) Create(conn *model.UserConnection) error {
	return r.db.Create(conn).Error
}

func (r *UserConnectionRepo) Delete(id uint) error {
	return r.db.Delete(&model.UserConnection{}, id).Error
}

func (r *UserConnectionRepo) Update(conn *model.UserConnection) error {
	return r.db.Save(conn).Error
}

func (r *UserConnectionRepo) CountByUserID(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.UserConnection{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
