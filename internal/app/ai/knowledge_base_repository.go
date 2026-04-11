package ai

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type KnowledgeBaseRepo struct {
	db *database.DB
}

func NewKnowledgeBaseRepo(i do.Injector) (*KnowledgeBaseRepo, error) {
	return &KnowledgeBaseRepo{db: do.MustInvoke[*database.DB](i)}, nil
}

func (r *KnowledgeBaseRepo) Create(kb *KnowledgeBase) error {
	return r.db.Create(kb).Error
}

func (r *KnowledgeBaseRepo) FindByID(id uint) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	if err := r.db.First(&kb, id).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}

type KBListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func (r *KnowledgeBaseRepo) List(params KBListParams) ([]KnowledgeBase, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&KnowledgeBase{})
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []KnowledgeBase
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *KnowledgeBaseRepo) Update(kb *KnowledgeBase) error {
	return r.db.Save(kb).Error
}

func (r *KnowledgeBaseRepo) Delete(id uint) error {
	return r.db.Delete(&KnowledgeBase{}, id).Error
}

func (r *KnowledgeBaseRepo) UpdateCounts(id uint) error {
	return r.db.Exec(`
		UPDATE ai_knowledge_bases SET
			source_count = (SELECT COUNT(*) FROM ai_knowledge_sources WHERE kb_id = ? AND deleted_at IS NULL),
			node_count = (SELECT COUNT(*) FROM ai_knowledge_nodes WHERE kb_id = ? AND deleted_at IS NULL)
		WHERE id = ?
	`, id, id, id).Error
}

func (r *KnowledgeBaseRepo) DB() *gorm.DB {
	return r.db.DB
}
