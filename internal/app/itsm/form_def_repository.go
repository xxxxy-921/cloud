package itsm

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
)

type FormDefRepo struct {
	db *database.DB
}

func NewFormDefRepo(i do.Injector) (*FormDefRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &FormDefRepo{db: db}, nil
}

func (r *FormDefRepo) Create(fd *FormDefinition) error {
	return r.db.Create(fd).Error
}

func (r *FormDefRepo) FindByID(id uint) (*FormDefinition, error) {
	var fd FormDefinition
	if err := r.db.First(&fd, id).Error; err != nil {
		return nil, err
	}
	return &fd, nil
}

func (r *FormDefRepo) FindByCode(code string) (*FormDefinition, error) {
	var fd FormDefinition
	if err := r.db.Where("code = ?", code).First(&fd).Error; err != nil {
		return nil, err
	}
	return &fd, nil
}

func (r *FormDefRepo) Update(id uint, updates map[string]any) error {
	return r.db.Model(&FormDefinition{}).Where("id = ?", id).Updates(updates).Error
}

func (r *FormDefRepo) Delete(id uint) error {
	return r.db.Delete(&FormDefinition{}, id).Error
}

type FormDefListParams struct {
	Keyword  string
	IsActive *bool
	Page     int
	PageSize int
}

func (r *FormDefRepo) List(params FormDefListParams) ([]FormDefinition, int64, error) {
	query := r.db.Model(&FormDefinition{})

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", like, like, like)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	var items []FormDefinition
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CountByFormID returns the number of ServiceDefinitions referencing a given form ID.
func (r *FormDefRepo) CountServiceRefs(formID uint) (int64, error) {
	var count int64
	err := r.db.Model(&ServiceDefinition{}).Where("form_id = ?", formID).Count(&count).Error
	return count, err
}
