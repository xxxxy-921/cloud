package license

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type LicenseRepo struct {
	db *database.DB
}

func NewLicenseRepo(i do.Injector) (*LicenseRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &LicenseRepo{db: db}, nil
}

func (r *LicenseRepo) CreateInTx(tx *gorm.DB, l *License) error {
	return tx.Create(l).Error
}

type LicenseDetail struct {
	License
	ProductName  string `gorm:"column:product_name"`
	ProductCode  string `gorm:"column:product_code"`
	LicenseeName string `gorm:"column:licensee_name"`
	LicenseeCode string `gorm:"column:licensee_code"`
}

func (r *LicenseRepo) FindByID(id uint) (*LicenseDetail, error) {
	var detail LicenseDetail
	err := r.db.Model(&License{}).
		Select("license_licenses.*, "+
			"license_products.name as product_name, "+
			"license_products.code as product_code, "+
			"license_licensees.name as licensee_name, "+
			"license_licensees.code as licensee_code").
		Joins("LEFT JOIN license_products ON license_products.id = license_licenses.product_id AND license_products.deleted_at IS NULL").
		Joins("LEFT JOIN license_licensees ON license_licensees.id = license_licenses.licensee_id AND license_licensees.deleted_at IS NULL").
		Where("license_licenses.id = ?", id).
		First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

type LicenseListParams struct {
	ProductID  uint
	LicenseeID uint
	Status     string
	Keyword    string
	Page       int
	PageSize   int
}

type LicenseListItem struct {
	License
	ProductName  string `gorm:"column:product_name"`
	LicenseeName string `gorm:"column:licensee_name"`
}

func (r *LicenseRepo) List(params LicenseListParams) ([]LicenseListItem, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&License{}).
		Select("license_licenses.*, "+
			"license_products.name as product_name, "+
			"license_licensees.name as licensee_name").
		Joins("LEFT JOIN license_products ON license_products.id = license_licenses.product_id AND license_products.deleted_at IS NULL").
		Joins("LEFT JOIN license_licensees ON license_licensees.id = license_licenses.licensee_id AND license_licensees.deleted_at IS NULL")

	if params.ProductID > 0 {
		query = query.Where("license_licenses.product_id = ?", params.ProductID)
	}
	if params.LicenseeID > 0 {
		query = query.Where("license_licenses.licensee_id = ?", params.LicenseeID)
	}
	if params.Status != "" {
		query = query.Where("license_licenses.status = ?", params.Status)
	}
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("(license_licenses.plan_name LIKE ? OR license_licenses.registration_code LIKE ?)", like, like)
	}

	var total int64
	countQuery := r.db.Model(&License{})
	if params.ProductID > 0 {
		countQuery = countQuery.Where("product_id = ?", params.ProductID)
	}
	if params.LicenseeID > 0 {
		countQuery = countQuery.Where("licensee_id = ?", params.LicenseeID)
	}
	if params.Status != "" {
		countQuery = countQuery.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		countQuery = countQuery.Where("(plan_name LIKE ? OR registration_code LIKE ?)", like, like)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []LicenseListItem
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).
		Order("license_licenses.created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *LicenseRepo) UpdateStatus(id uint, updates map[string]any) error {
	return r.db.Model(&License{}).Where("id = ?", id).Updates(updates).Error
}
