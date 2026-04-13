package org

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrDepartmentNotFound    = errors.New("department not found")
	ErrDepartmentCodeExists  = errors.New("department code already exists")
	ErrDepartmentHasChildren = errors.New("department has sub-departments")
	ErrDepartmentHasMembers  = errors.New("department has members")
)

type DepartmentService struct {
	repo       *DepartmentRepo
	assignRepo *AssignmentRepo
}

func NewDepartmentService(i do.Injector) (*DepartmentService, error) {
	repo := do.MustInvoke[*DepartmentRepo](i)
	assignRepo := do.MustInvoke[*AssignmentRepo](i)
	return &DepartmentService{repo: repo, assignRepo: assignRepo}, nil
}

func (s *DepartmentService) Create(name, code string, parentID, managerID *uint, sort int, description string) (*Department, error) {
	if _, err := s.repo.FindByCode(code); err == nil {
		return nil, ErrDepartmentCodeExists
	}

	dept := &Department{
		Name:        name,
		Code:        code,
		ParentID:    parentID,
		ManagerID:   managerID,
		Sort:        sort,
		Description: description,
		IsActive:    true,
	}
	if err := s.repo.Create(dept); err != nil {
		return nil, err
	}
	return s.repo.FindByID(dept.ID)
}

func (s *DepartmentService) Get(id uint) (*Department, error) {
	dept, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return dept, nil
}

func (s *DepartmentService) ListAll() ([]Department, error) {
	return s.repo.ListAll()
}

func (s *DepartmentService) Tree() ([]DepartmentTreeNode, error) {
	depts, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	counts, err := s.assignRepo.CountByDepartments()
	if err != nil {
		return nil, err
	}
	return buildDepartmentTree(depts, counts), nil
}

func (s *DepartmentService) Update(id uint, name, code *string, parentID, managerID *uint, sort *int, description *string, isActive *bool) (*Department, error) {
	dept, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}
	if code != nil {
		if existing, err := s.repo.FindByCode(*code); err == nil && existing.ID != id {
			return nil, ErrDepartmentCodeExists
		}
		updates["code"] = *code
	}
	if parentID != nil {
		updates["parent_id"] = *parentID
	}
	if managerID != nil {
		updates["manager_id"] = *managerID
	}
	if sort != nil {
		updates["sort"] = *sort
	}
	if description != nil {
		updates["description"] = *description
	}
	if isActive != nil {
		updates["is_active"] = *isActive
	}

	if len(updates) > 0 {
		if err := s.repo.Update(id, updates); err != nil {
			return nil, err
		}
		dept, _ = s.repo.FindByID(id)
	}
	return dept, nil
}

func (s *DepartmentService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDepartmentNotFound
		}
		return err
	}

	hasChildren, err := s.repo.HasChildren(id)
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrDepartmentHasChildren
	}

	hasMembers, err := s.repo.HasMembers(id)
	if err != nil {
		return err
	}
	if hasMembers {
		return ErrDepartmentHasMembers
	}

	return s.repo.Delete(id)
}

// Tree helpers

type DepartmentTreeNode struct {
	DepartmentResponse
	MemberCount int                  `json:"memberCount"`
	Children    []DepartmentTreeNode `json:"children,omitempty"`
}

func buildDepartmentTree(depts []Department, counts map[uint]int) []DepartmentTreeNode {
	byParent := make(map[uint][]Department)
	for _, d := range depts {
		pid := uint(0)
		if d.ParentID != nil {
			pid = *d.ParentID
		}
		byParent[pid] = append(byParent[pid], d)
	}

	var build func(parentID uint) []DepartmentTreeNode
	build = func(parentID uint) []DepartmentTreeNode {
		items := byParent[parentID]
		if len(items) == 0 {
			return nil
		}
		result := make([]DepartmentTreeNode, 0, len(items))
		for _, d := range items {
			result = append(result, DepartmentTreeNode{
				DepartmentResponse: d.ToResponse(),
				MemberCount:        counts[d.ID],
				Children:           build(d.ID),
			})
		}
		return result
	}

	return build(0)
}
