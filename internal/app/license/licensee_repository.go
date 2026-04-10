package license

import (
	"github.com/samber/do/v2"

	"metis/internal/database"
)

type LicenseeRepo struct {
	db *database.DB
}

func NewLicenseeRepo(i do.Injector) (*LicenseeRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &LicenseeRepo{db: db}, nil
}

func (r *LicenseeRepo) Create(l *Licensee) error {
	return r.db.Create(l).Error
}

func (r *LicenseeRepo) FindByID(id uint) (*Licensee, error) {
	var l Licensee
	if err := r.db.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

type LicenseeListParams struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

func (r *LicenseeRepo) List(params LicenseeListParams) ([]Licensee, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	query := r.db.Model(&Licensee{})
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if params.Status != "" && params.Status != "all" {
		query = query.Where("status = ?", params.Status)
	} else if params.Status == "" {
		// Default: exclude archived
		query = query.Where("status = ?", LicenseeStatusActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []Licensee
	offset := (params.Page - 1) * params.PageSize
	if err := query.Offset(offset).Limit(params.PageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *LicenseeRepo) Update(l *Licensee) error {
	return r.db.Save(l).Error
}

func (r *LicenseeRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&Licensee{}).Where("id = ?", id).Update("status", status).Error
}

func (r *LicenseeRepo) ExistsByName(name string, excludeID ...uint) (bool, error) {
	var count int64
	q := r.db.Model(&Licensee{}).Where("name = ?", name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		q = q.Where("id != ?", excludeID[0])
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *LicenseeRepo) ExistsByCode(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&Licensee{}).Where("code = ?", code).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
