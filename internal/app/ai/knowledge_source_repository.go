package ai

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type KnowledgeSourceRepo struct {
	db *database.DB
}

func NewKnowledgeSourceRepo(i do.Injector) (*KnowledgeSourceRepo, error) {
	return &KnowledgeSourceRepo{db: do.MustInvoke[*database.DB](i)}, nil
}

func (r *KnowledgeSourceRepo) Create(s *KnowledgeSource) error {
	return r.db.Create(s).Error
}

func (r *KnowledgeSourceRepo) FindByID(id uint) (*KnowledgeSource, error) {
	var s KnowledgeSource
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// FindByIDs returns sources matching the given IDs (batch fetch).
func (r *KnowledgeSourceRepo) FindByIDs(ids []uint) ([]KnowledgeSource, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var sources []KnowledgeSource
	if err := r.db.Where("id IN ?", ids).Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

type SourceListParams struct {
	KbID     uint
	Keyword  string
	Page     int
	PageSize int
}

func (r *KnowledgeSourceRepo) List(params SourceListParams) ([]KnowledgeSource, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&KnowledgeSource{}).Where("kb_id = ?", params.KbID)
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("title LIKE ?", like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []KnowledgeSource
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *KnowledgeSourceRepo) Update(s *KnowledgeSource) error {
	return r.db.Save(s).Error
}

func (r *KnowledgeSourceRepo) Delete(id uint) error {
	return r.db.Delete(&KnowledgeSource{}, id).Error
}

func (r *KnowledgeSourceRepo) DeleteByKbID(kbID uint) error {
	return r.db.Where("kb_id = ?", kbID).Delete(&KnowledgeSource{}).Error
}

func (r *KnowledgeSourceRepo) DeleteByParentID(parentID uint) error {
	return r.db.Where("parent_id = ?", parentID).Delete(&KnowledgeSource{}).Error
}

// FindChildIDs returns IDs of child sources (URL crawl children) for the given parent.
func (r *KnowledgeSourceRepo) FindChildIDs(parentID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&KnowledgeSource{}).Where("parent_id = ?", parentID).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *KnowledgeSourceRepo) FindByKbIDAndFormat(kbID uint, format string) ([]KnowledgeSource, error) {
	var items []KnowledgeSource
	if err := r.db.Where("kb_id = ? AND format = ?", kbID, format).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *KnowledgeSourceRepo) FindCompletedByKbID(kbID uint) ([]KnowledgeSource, error) {
	var items []KnowledgeSource
	if err := r.db.Where("kb_id = ? AND extract_status = ?", kbID, ExtractStatusCompleted).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *KnowledgeSourceRepo) FindURLSourcesByKbID(kbID uint) ([]KnowledgeSource, error) {
	var items []KnowledgeSource
	if err := r.db.Where("kb_id = ? AND format = ? AND parent_id IS NULL", kbID, SourceFormatURL).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *KnowledgeSourceRepo) FindCrawlEnabledSources() ([]KnowledgeSource, error) {
	var items []KnowledgeSource
	if err := r.db.Where("format = ? AND parent_id IS NULL AND crawl_enabled = ?", SourceFormatURL, true).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *KnowledgeSourceRepo) DB() *gorm.DB {
	return r.db.DB
}
