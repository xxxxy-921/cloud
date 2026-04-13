package org

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrAssignmentNotFound    = errors.New("assignment not found")
	ErrAlreadyAssigned       = errors.New("user already assigned to this department")
)

type AssignmentService struct {
	repo *AssignmentRepo
}

func NewAssignmentService(i do.Injector) (*AssignmentService, error) {
	repo := do.MustInvoke[*AssignmentRepo](i)
	return &AssignmentService{repo: repo}, nil
}

func (s *AssignmentService) GetUserPositions(userID uint) ([]UserPositionResponse, error) {
	items, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	result := make([]UserPositionResponse, 0, len(items))
	for _, item := range items {
		resp := UserPositionResponse{
			ID:           item.ID,
			UserID:       item.UserID,
			DepartmentID: item.DepartmentID,
			PositionID:   item.PositionID,
			IsPrimary:    item.IsPrimary,
			Sort:         item.Sort,
		}
		if item.Department.ID > 0 {
			r := item.Department.ToResponse()
			resp.Department = &r
		}
		if item.Position.ID > 0 {
			r := item.Position.ToResponse()
			resp.Position = &r
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *AssignmentService) AddUserPosition(userID, deptID, posID uint, isPrimary bool) (*UserPosition, error) {
	exists, err := s.repo.ExistsByUserAndDept(userID, deptID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyAssigned
	}

	// If setting as primary, demote existing primary
	if isPrimary {
		_ = s.demoteCurrentPrimary(userID)
	} else {
		// Auto-set primary if this is the first assignment
		existing, err := s.repo.FindByUserID(userID)
		if err != nil {
			return nil, err
		}
		if len(existing) == 0 {
			isPrimary = true
		}
	}

	up := &UserPosition{
		UserID:       userID,
		DepartmentID: deptID,
		PositionID:   posID,
		IsPrimary:    isPrimary,
	}
	if err := s.repo.AddPosition(up); err != nil {
		return nil, err
	}
	return s.repo.FindByID(up.ID)
}

func (s *AssignmentService) RemoveUserPosition(userID, assignmentID uint) error {
	return s.repo.RemovePosition(assignmentID, userID)
}

func (s *AssignmentService) UpdateUserPosition(userID, assignmentID uint, positionID *uint, isPrimary *bool) error {
	fields := map[string]any{}
	if positionID != nil {
		fields["position_id"] = *positionID
	}
	if isPrimary != nil && *isPrimary {
		// Demote current primary first, then set this one
		_ = s.demoteCurrentPrimary(userID)
		fields["is_primary"] = true
	}
	if len(fields) == 0 {
		return nil
	}
	err := s.repo.UpdatePosition(assignmentID, userID, fields)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAssignmentNotFound
	}
	return err
}

func (s *AssignmentService) demoteCurrentPrimary(userID uint) error {
	primary, err := s.repo.GetUserPrimaryPosition(userID)
	if err != nil {
		return err // no primary exists, that's fine
	}
	return s.repo.UpdatePosition(primary.ID, userID, map[string]any{"is_primary": false})
}

func (s *AssignmentService) ListDepartmentMembers(deptID uint, keyword string, page, pageSize int) ([]AssignmentItem, int64, error) {
	return s.repo.ListUsersByDepartment(deptID, keyword, page, pageSize)
}

func (s *AssignmentService) SetPrimary(userID uint, assignmentID uint) error {
	return s.repo.SetPrimary(userID, assignmentID)
}

// Scope helpers for future data permission isolation

func (s *AssignmentService) GetUserDepartmentIDs(userID uint) ([]uint, error) {
	return s.repo.GetUserDepartmentIDs(userID)
}

func (s *AssignmentService) GetUserDepartmentScope(userID uint) ([]uint, error) {
	deptIDs, err := s.repo.GetUserDepartmentIDs(userID)
	if err != nil {
		return nil, err
	}
	if len(deptIDs) == 0 {
		return nil, nil
	}

	scope := make(map[uint]struct{})
	for _, id := range deptIDs {
		scope[id] = struct{}{}
	}

	// BFS to collect all active sub-departments
	queue := make([]uint, len(deptIDs))
	copy(queue, deptIDs)
	for len(queue) > 0 {
		subIDs, err := s.repo.GetSubDepartmentIDs(queue, true)
		if err != nil {
			return nil, err
		}
		queue = queue[:0]
		for _, sid := range subIDs {
			if _, ok := scope[sid]; !ok {
				scope[sid] = struct{}{}
				queue = append(queue, sid)
			}
		}
	}

	result := make([]uint, 0, len(scope))
	for id := range scope {
		result = append(result, id)
	}
	return result, nil
}
