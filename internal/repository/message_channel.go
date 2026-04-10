package repository

import (
	"encoding/json"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
)

type MessageChannelRepo struct {
	db *database.DB
}

func NewMessageChannel(i do.Injector) (*MessageChannelRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &MessageChannelRepo{db: db}, nil
}

func (r *MessageChannelRepo) Create(ch *model.MessageChannel) error {
	return r.db.Create(ch).Error
}

func (r *MessageChannelRepo) FindByID(id uint) (*model.MessageChannel, error) {
	var ch model.MessageChannel
	if err := r.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *MessageChannelRepo) List(params ListParams) ([]model.MessageChannel, int64, error) {
	query := r.db.Model(&model.MessageChannel{})

	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ?", like)
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
	offset := (params.Page - 1) * params.PageSize

	var items []model.MessageChannel
	err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *MessageChannelRepo) Update(ch *model.MessageChannel) error {
	return r.db.Save(ch).Error
}

func (r *MessageChannelRepo) Delete(id uint) error {
	result := r.db.Delete(&model.MessageChannel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *MessageChannelRepo) ToggleEnabled(id uint) (*model.MessageChannel, error) {
	var ch model.MessageChannel
	if err := r.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	ch.Enabled = !ch.Enabled
	if err := r.db.Save(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// MaskConfig masks sensitive fields in the config JSON string.
func MaskConfig(configJSON string) string {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return configJSON
	}
	if _, ok := cfg["password"]; ok {
		cfg["password"] = "******"
	}
	masked, err := json.Marshal(cfg)
	if err != nil {
		return configJSON
	}
	return string(masked)
}
