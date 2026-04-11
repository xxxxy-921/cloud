package ai

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
)

type ToolRepo struct {
	db *database.DB
}

func NewToolRepo(i do.Injector) (*ToolRepo, error) {
	return &ToolRepo{db: do.MustInvoke[*database.DB](i)}, nil
}

func (r *ToolRepo) List() ([]Tool, error) {
	var tools []Tool
	if err := r.db.Order("name ASC").Find(&tools).Error; err != nil {
		return nil, err
	}
	return tools, nil
}

func (r *ToolRepo) FindByID(id uint) (*Tool, error) {
	var t Tool
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ToolRepo) FindByName(name string) (*Tool, error) {
	var t Tool
	if err := r.db.Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ToolRepo) Create(t *Tool) error {
	return r.db.Create(t).Error
}

func (r *ToolRepo) Update(t *Tool) error {
	return r.db.Save(t).Error
}
