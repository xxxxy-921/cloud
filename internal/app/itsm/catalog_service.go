package itsm

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrCatalogNotFound    = errors.New("service catalog not found")
	ErrCatalogHasChildren = errors.New("catalog has sub-categories, cannot delete")
	ErrCatalogHasServices = errors.New("catalog has services, cannot delete")
)

type CatalogService struct {
	repo *CatalogRepo
}

func NewCatalogService(i do.Injector) (*CatalogService, error) {
	repo := do.MustInvoke[*CatalogRepo](i)
	return &CatalogService{repo: repo}, nil
}

func (s *CatalogService) Create(name, description, icon string, parentID *uint, sortOrder int) (*ServiceCatalog, error) {
	if parentID != nil {
		if _, err := s.repo.FindByID(*parentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCatalogNotFound
			}
			return nil, err
		}
	}

	catalog := &ServiceCatalog{
		Name:        name,
		Description: description,
		Icon:        icon,
		ParentID:    parentID,
		SortOrder:   sortOrder,
		IsActive:    true,
	}
	if err := s.repo.Create(catalog); err != nil {
		return nil, err
	}
	return s.repo.FindByID(catalog.ID)
}

func (s *CatalogService) Get(id uint) (*ServiceCatalog, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCatalogNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *CatalogService) Update(id uint, updates map[string]any) (*ServiceCatalog, error) {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCatalogNotFound
		}
		return nil, err
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *CatalogService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCatalogNotFound
		}
		return err
	}

	has, err := s.repo.HasChildren(id)
	if err != nil {
		return err
	}
	if has {
		return ErrCatalogHasChildren
	}

	hasSvc, err := s.repo.HasServices(id)
	if err != nil {
		return err
	}
	if hasSvc {
		return ErrCatalogHasServices
	}

	return s.repo.Delete(id)
}

// Tree returns the full catalog tree structure.
func (s *CatalogService) Tree() ([]ServiceCatalogResponse, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	return buildTree(all, nil), nil
}

func buildTree(catalogs []ServiceCatalog, parentID *uint) []ServiceCatalogResponse {
	var result []ServiceCatalogResponse
	for _, c := range catalogs {
		if ptrEq(c.ParentID, parentID) {
			resp := c.ToResponse()
			resp.Children = buildTree(catalogs, &c.ID)
			result = append(result, resp)
		}
	}
	return result
}

func ptrEq(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
