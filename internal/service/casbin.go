package service

import (
	"github.com/casbin/casbin/v2"
	"github.com/samber/do/v2"
)

type CasbinService struct {
	enforcer *casbin.Enforcer
}

func NewCasbin(i do.Injector) (*CasbinService, error) {
	enforcer := do.MustInvoke[*casbin.Enforcer](i)
	return &CasbinService{enforcer: enforcer}, nil
}

func (s *CasbinService) CheckPermission(roleCode, obj, act string) (bool, error) {
	return s.enforcer.Enforce(roleCode, obj, act)
}

// GetPoliciesForRole returns all policies for a role code.
func (s *CasbinService) GetPoliciesForRole(roleCode string) [][]string {
	policies, _ := s.enforcer.GetFilteredPolicy(0, roleCode)
	return policies
}

// SetPoliciesForRole replaces all policies for a role with new ones.
func (s *CasbinService) SetPoliciesForRole(roleCode string, policies [][]string) error {
	// Remove existing policies for this role
	_, err := s.enforcer.RemoveFilteredPolicy(0, roleCode)
	if err != nil {
		return err
	}

	// Add new policies
	if len(policies) > 0 {
		_, err = s.enforcer.AddPolicies(policies)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetEnforcer returns the underlying enforcer for direct use.
func (s *CasbinService) GetEnforcer() *casbin.Enforcer {
	return s.enforcer
}
