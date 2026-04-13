package org

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type AssignmentRepo struct {
	db *database.DB
}

func NewAssignmentRepo(i do.Injector) (*AssignmentRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &AssignmentRepo{db: db}, nil
}

func (r *AssignmentRepo) FindByID(id uint) (*UserPosition, error) {
	var up UserPosition
	if err := r.db.Preload("Department").Preload("Position").First(&up, id).Error; err != nil {
		return nil, err
	}
	return &up, nil
}

func (r *AssignmentRepo) FindByUserID(userID uint) ([]UserPosition, error) {
	var items []UserPosition
	if err := r.db.Where("user_id = ?", userID).
		Preload("Department").
		Preload("Position").
		Order("is_primary DESC, sort ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AssignmentRepo) FindByDepartmentID(deptID uint) ([]UserPosition, error) {
	var items []UserPosition
	if err := r.db.Where("department_id = ?", deptID).
		Preload("Position").
		Order("is_primary DESC, sort ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AssignmentRepo) AddPosition(up *UserPosition) error {
	return r.db.Create(up).Error
}

func (r *AssignmentRepo) ExistsByUserAndDept(userID, deptID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&UserPosition{}).
		Where("user_id = ? AND department_id = ?", userID, deptID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *AssignmentRepo) RemovePosition(assignmentID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var up UserPosition
		if err := tx.Where("id = ? AND user_id = ?", assignmentID, userID).First(&up).Error; err != nil {
			return err
		}
		if err := tx.Delete(&up).Error; err != nil {
			return err
		}
		// If removed was primary, auto-promote next
		if up.IsPrimary {
			var next UserPosition
			if err := tx.Where("user_id = ?", userID).Order("sort ASC, id ASC").First(&next).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil // no more assignments
				}
				return err
			}
			return tx.Model(&next).Update("is_primary", true).Error
		}
		return nil
	})
}

func (r *AssignmentRepo) UpdatePosition(assignmentID, userID uint, fields map[string]any) error {
	result := r.db.Model(&UserPosition{}).
		Where("id = ? AND user_id = ?", assignmentID, userID).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AssignmentRepo) SetPrimary(userID uint, assignmentID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserPosition{}).
			Where("user_id = ?", userID).
			Update("is_primary", false).Error; err != nil {
			return err
		}
		return tx.Model(&UserPosition{}).
			Where("id = ? AND user_id = ?", assignmentID, userID).
			Update("is_primary", true).Error
	})
}

func (r *AssignmentRepo) DeleteByID(id uint) error {
	return r.db.Delete(&UserPosition{}, id).Error
}

func (r *AssignmentRepo) ListUsersByDepartment(deptID uint, keyword string, page, pageSize int) ([]AssignmentItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	base := r.db.Table("user_positions").
		Select("user_positions.id as assignment_id, user_positions.user_id, users.username, users.email, users.avatar, user_positions.department_id, user_positions.position_id, user_positions.is_primary, user_positions.created_at").
		Joins("LEFT JOIN users ON users.id = user_positions.user_id").
		Where("user_positions.department_id = ? AND user_positions.deleted_at IS NULL", deptID)

	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("(users.username LIKE ? OR users.email LIKE ?)", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []AssignmentItem
	offset := (page - 1) * pageSize
	if err := base.Offset(offset).Limit(pageSize).Order("user_positions.is_primary DESC, users.username ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AssignmentRepo) CountByDepartments() (map[uint]int, error) {
	type countRow struct {
		DepartmentID uint
		Count        int
	}
	var rows []countRow
	if err := r.db.Model(&UserPosition{}).
		Select("department_id, COUNT(*) as count").
		Group("department_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]int, len(rows))
	for _, row := range rows {
		result[row.DepartmentID] = row.Count
	}
	return result, nil
}

func (r *AssignmentRepo) GetUserDepartmentIDs(userID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&UserPosition{}).
		Where("user_id = ?", userID).
		Distinct().
		Pluck("department_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *AssignmentRepo) GetSubDepartmentIDs(parentIDs []uint, activeOnly bool) ([]uint, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	query := r.db.Model(&Department{}).Where("parent_id IN ?", parentIDs)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var ids []uint
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *AssignmentRepo) CountAssignments(filters map[string]any) (int64, error) {
	var count int64
	query := r.db.Model(&UserPosition{})
	for k, v := range filters {
		query = query.Where(k+" = ?", v)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *AssignmentRepo) GetUserPrimaryPosition(userID uint) (*UserPosition, error) {
	var up UserPosition
	if err := r.db.Where("user_id = ? AND is_primary = ?", userID, true).
		Preload("Department").
		Preload("Position").
		First(&up).Error; err != nil {
		return nil, err
	}
	return &up, nil
}
