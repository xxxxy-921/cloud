package org

// OrgScopeResolverImpl implements app.OrgScopeResolver backed by AssignmentService.
type OrgScopeResolverImpl struct {
	svc *AssignmentService
}

// GetUserDeptScope returns department IDs visible to the user.
// If includeSubDepts is true, BFS-expands to active sub-departments.
func (r *OrgScopeResolverImpl) GetUserDeptScope(userID uint, includeSubDepts bool) ([]uint, error) {
	if includeSubDepts {
		return r.svc.GetUserDepartmentScope(userID)
	}
	return r.svc.GetUserDepartmentIDs(userID)
}

// OrgUserResolverImpl implements app.OrgUserResolver backed by AssignmentRepo.
type OrgUserResolverImpl struct {
	repo *AssignmentRepo
}

func (r *OrgUserResolverImpl) GetUserPositionIDs(userID uint) ([]uint, error) {
	return r.repo.GetUserPositionIDs(userID)
}

func (r *OrgUserResolverImpl) GetUserDepartmentIDs(userID uint) ([]uint, error) {
	return r.repo.GetUserDepartmentIDs(userID)
}
