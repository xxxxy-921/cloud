package license

import (
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

type LicenseRegistrationRepo struct {
	db *database.DB
}

func NewLicenseRegistrationRepo(i do.Injector) (*LicenseRegistrationRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &LicenseRegistrationRepo{db: db}, nil
}

func (r *LicenseRegistrationRepo) Create(lr *LicenseRegistration) error {
	return r.db.Create(lr).Error
}

func (r *LicenseRegistrationRepo) CreateInTx(tx *gorm.DB, lr *LicenseRegistration) error {
	return tx.Create(lr).Error
}

func (r *LicenseRegistrationRepo) FindByID(id uint) (*LicenseRegistration, error) {
	var lr LicenseRegistration
	if err := r.db.First(&lr, id).Error; err != nil {
		return nil, err
	}
	return &lr, nil
}

func (r *LicenseRegistrationRepo) FindByCode(code string) (*LicenseRegistration, error) {
	var lr LicenseRegistration
	if err := r.db.Where("code = ?", code).First(&lr).Error; err != nil {
		return nil, err
	}
	return &lr, nil
}

func (r *LicenseRegistrationRepo) FindByCodeInTx(tx *gorm.DB, code string) (*LicenseRegistration, error) {
	var lr LicenseRegistration
	if err := tx.Where("code = ?", code).First(&lr).Error; err != nil {
		return nil, err
	}
	return &lr, nil
}

type LicenseRegistrationListParams struct {
	ProductID  uint
	LicenseeID uint
	Available  bool
	Page       int
	PageSize   int
}

func (r *LicenseRegistrationRepo) List(params LicenseRegistrationListParams) ([]LicenseRegistration, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&LicenseRegistration{})
	if params.ProductID > 0 {
		query = query.Where("product_id = ?", params.ProductID)
	}
	if params.LicenseeID > 0 {
		query = query.Where("licensee_id = ?", params.LicenseeID)
	}
	if params.Available {
		query = query.Where("bound_license_id IS NULL AND (expires_at IS NULL OR expires_at > ?)", time.Now())
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []LicenseRegistration
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *LicenseRegistrationRepo) UpdateBoundLicenseInTx(tx *gorm.DB, id uint, boundLicenseID uint) error {
	return tx.Model(&LicenseRegistration{}).Where("id = ?", id).Update("bound_license_id", boundLicenseID).Error
}

func (r *LicenseRegistrationRepo) UnbindLicenseInTx(tx *gorm.DB, code string) error {
	return tx.Model(&LicenseRegistration{}).Where("code = ?", code).Update("bound_license_id", nil).Error
}

func (r *LicenseRegistrationRepo) DeleteExpired(now time.Time) error {
	return r.db.Where("expires_at IS NOT NULL AND expires_at <= ? AND bound_license_id IS NULL", now).Delete(&LicenseRegistration{}).Error
}

func (r *LicenseRegistrationRepo) CodeExists(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&LicenseRegistration{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
