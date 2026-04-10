package license

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

// --- ProductRepo ---

type ProductRepo struct {
	db *database.DB
}

func NewProductRepo(i do.Injector) (*ProductRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &ProductRepo{db: db}, nil
}

func (r *ProductRepo) Create(p *Product) error {
	return r.db.Create(p).Error
}

func (r *ProductRepo) FindByID(id uint) (*Product, error) {
	var p Product
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) FindByIDWithPlans(id uint) (*Product, error) {
	var p Product
	if err := r.db.Preload("Plans", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, id ASC")
	}).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) FindByCode(code string) (*Product, error) {
	var p Product
	if err := r.db.Where("code = ?", code).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) ExistsByCode(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&Product{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type ProductListParams struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

type ProductListItem struct {
	Product
	PlanCount int `json:"planCount" gorm:"column:plan_count"`
}

func (r *ProductRepo) List(params ProductListParams) ([]ProductListItem, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&Product{})
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []Product
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&products).Error; err != nil {
		return nil, 0, err
	}

	// Get plan counts in a single query
	productIDs := make([]uint, len(products))
	for i, p := range products {
		productIDs[i] = p.ID
	}

	planCounts := make(map[uint]int)
	if len(productIDs) > 0 {
		type countResult struct {
			ProductID uint `gorm:"column:product_id"`
			Count     int  `gorm:"column:cnt"`
		}
		var counts []countResult
		r.db.Model(&Plan{}).
			Select("product_id, COUNT(*) as cnt").
			Where("product_id IN ?", productIDs).
			Group("product_id").
			Find(&counts)
		for _, c := range counts {
			planCounts[c.ProductID] = c.Count
		}
	}

	items := make([]ProductListItem, len(products))
	for i, p := range products {
		items[i] = ProductListItem{Product: p, PlanCount: planCounts[p.ID]}
	}

	return items, total, nil
}

func (r *ProductRepo) Update(p *Product) error {
	return r.db.Save(p).Error
}

func (r *ProductRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&Product{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ProductRepo) UpdateSchema(id uint, schema []byte) error {
	return r.db.Model(&Product{}).Where("id = ?", id).Update("constraint_schema", string(schema)).Error
}

// --- PlanRepo ---

type PlanRepo struct {
	db *database.DB
}

func NewPlanRepo(i do.Injector) (*PlanRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &PlanRepo{db: db}, nil
}

func (r *PlanRepo) Create(p *Plan) error {
	return r.db.Create(p).Error
}

func (r *PlanRepo) FindByID(id uint) (*Plan, error) {
	var p Plan
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlanRepo) ListByProductID(productID uint) ([]Plan, error) {
	var plans []Plan
	if err := r.db.Where("product_id = ?", productID).
		Order("sort_order ASC, id ASC").
		Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func (r *PlanRepo) ExistsByName(productID uint, name string, excludeID ...uint) (bool, error) {
	var count int64
	q := r.db.Model(&Plan{}).Where("product_id = ? AND name = ?", productID, name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		q = q.Where("id != ?", excludeID[0])
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PlanRepo) Update(p *Plan) error {
	return r.db.Save(p).Error
}

func (r *PlanRepo) Delete(id uint) error {
	return r.db.Delete(&Plan{}, id).Error
}

func (r *PlanRepo) ClearDefault(productID uint) error {
	return r.db.Model(&Plan{}).
		Where("product_id = ? AND is_default = ?", productID, true).
		Update("is_default", false).Error
}

func (r *PlanRepo) SetDefault(id uint, isDefault bool) error {
	return r.db.Model(&Plan{}).Where("id = ?", id).Update("is_default", isDefault).Error
}

// --- ProductKeyRepo ---

type ProductKeyRepo struct {
	db *database.DB
}

func NewProductKeyRepo(i do.Injector) (*ProductKeyRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &ProductKeyRepo{db: db}, nil
}

func (r *ProductKeyRepo) Create(k *ProductKey) error {
	return r.db.Create(k).Error
}

func (r *ProductKeyRepo) FindCurrentByProductID(productID uint) (*ProductKey, error) {
	var k ProductKey
	if err := r.db.Where("product_id = ? AND is_current = ?", productID, true).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *ProductKeyRepo) RevokeByProductID(tx *gorm.DB, productID uint) error {
	now := tx.NowFunc()
	return tx.Model(&ProductKey{}).
		Where("product_id = ? AND is_current = ?", productID, true).
		Updates(map[string]any{"is_current": false, "revoked_at": now}).Error
}

func (r *ProductKeyRepo) CreateInTx(tx *gorm.DB, k *ProductKey) error {
	return tx.Create(k).Error
}

func (r *ProductKeyRepo) FindByProductIDAndVersion(productID uint, version int) (*ProductKey, error) {
	var k ProductKey
	if err := r.db.Where("product_id = ? AND version = ?", productID, version).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}
