package org

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
)

type DepartmentRepo struct {
	db *database.DB
}

func NewDepartmentRepo(i do.Injector) (*DepartmentRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &DepartmentRepo{db: db}, nil
}

func (r *DepartmentRepo) Create(dept *Department) error {
	return r.db.Create(dept).Error
}

func (r *DepartmentRepo) FindByID(id uint) (*Department, error) {
	var dept Department
	if err := r.db.First(&dept, id).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepo) FindByCode(code string) (*Department, error) {
	var dept Department
	if err := r.db.Where("code = ?", code).First(&dept).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepo) Update(id uint, updates map[string]any) error {
	return r.db.Model(&Department{}).Where("id = ?", id).Updates(updates).Error
}

func (r *DepartmentRepo) Delete(id uint) error {
	return r.db.Delete(&Department{}, id).Error
}

func (r *DepartmentRepo) ListAll() ([]Department, error) {
	var depts []Department
	if err := r.db.Order("sort ASC, id ASC").Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (r *DepartmentRepo) ListActive() ([]Department, error) {
	var depts []Department
	if err := r.db.Where("is_active = ?", true).Order("sort ASC, id ASC").Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (r *DepartmentRepo) HasChildren(parentID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&Department{}).Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DepartmentRepo) HasMembers(deptID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&UserPosition{}).Where("department_id = ?", deptID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
