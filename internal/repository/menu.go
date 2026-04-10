package repository

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

type MenuRepo struct {
	db *database.DB
}

func NewMenu(i do.Injector) (*MenuRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &MenuRepo{db: db}, nil
}

func (r *MenuRepo) FindByID(id uint) (*model.Menu, error) {
	var menu model.Menu
	if err := r.db.First(&menu, id).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepo) FindAll() ([]model.Menu, error) {
	var menus []model.Menu
	if err := r.db.Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

func (r *MenuRepo) FindByParentID(parentID *uint) ([]model.Menu, error) {
	var menus []model.Menu
	query := r.db.Order("sort ASC, id ASC")
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if err := query.Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

func (r *MenuRepo) FindByPermission(permission string) (*model.Menu, error) {
	var menu model.Menu
	if err := r.db.Where("permission = ?", permission).First(&menu).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepo) Create(menu *model.Menu) error {
	return r.db.Create(menu).Error
}

func (r *MenuRepo) Update(menu *model.Menu) error {
	return r.db.Save(menu).Error
}

func (r *MenuRepo) Delete(id uint) error {
	result := r.db.Delete(&model.Menu{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetTree returns all menus structured as a tree.
func (r *MenuRepo) GetTree() ([]model.Menu, error) {
	all, err := r.FindAll()
	if err != nil {
		return nil, err
	}
	return buildTree(all, nil), nil
}

func buildTree(all []model.Menu, parentID *uint) []model.Menu {
	var children []model.Menu
	for _, m := range all {
		if ptrEqual(m.ParentID, parentID) {
			m.Children = buildTree(all, &m.ID)
			children = append(children, m)
		}
	}
	return children
}

func ptrEqual(a *uint, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (r *MenuRepo) HasChildren(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Menu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type SortItem struct {
	ID   uint `json:"id"`
	Sort int  `json:"sort"`
}

func (r *MenuRepo) UpdateSorts(items []SortItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&model.Menu{}).Where("id = ?", item.ID).Update("sort", item.Sort).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindByIDs returns menus matching the given IDs.
func (r *MenuRepo) FindByIDs(ids []uint) ([]model.Menu, error) {
	var menus []model.Menu
	if len(ids) == 0 {
		return menus, nil
	}
	if err := r.db.Where("id IN ?", ids).Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}
