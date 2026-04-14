package engine

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// OrgService is an optional interface that the Org App can provide for participant resolution.
type OrgService interface {
	FindUsersByPositionID(positionID uint) ([]uint, error)
	FindUsersByDepartmentID(departmentID uint) ([]uint, error)
	FindManagerByUserID(userID uint) (uint, error)
}

// ParticipantResolver resolves participant configurations to user IDs.
type ParticipantResolver struct {
	orgService OrgService // nil when Org App is not installed
}

func NewParticipantResolver(orgService OrgService) *ParticipantResolver {
	return &ParticipantResolver{orgService: orgService}
}

// Resolve returns user IDs for a given participant configuration.
func (r *ParticipantResolver) Resolve(tx *gorm.DB, ticketID uint, p Participant) ([]uint, error) {
	switch p.Type {
	case "user":
		uid, err := strconv.ParseUint(p.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID %q: %w", p.Value, err)
		}
		return []uint{uint(uid)}, nil

	case "requester_manager":
		return r.resolveRequesterManager(tx, ticketID)

	case "position":
		if r.orgService == nil {
			return nil, fmt.Errorf("参与人解析失败：position 类型需要安装组织架构模块")
		}
		posID, err := strconv.ParseUint(p.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid position ID %q: %w", p.Value, err)
		}
		return r.orgService.FindUsersByPositionID(uint(posID))

	case "department":
		if r.orgService == nil {
			return nil, fmt.Errorf("参与人解析失败：department 类型需要安装组织架构模块")
		}
		deptID, err := strconv.ParseUint(p.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid department ID %q: %w", p.Value, err)
		}
		return r.orgService.FindUsersByDepartmentID(uint(deptID))

	default:
		return nil, fmt.Errorf("unsupported participant type: %s", p.Type)
	}
}

func (r *ParticipantResolver) resolveRequesterManager(tx *gorm.DB, ticketID uint) ([]uint, error) {
	var ticket ticketModel
	if err := tx.First(&ticket, ticketID).Error; err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if r.orgService == nil {
		return nil, fmt.Errorf("参与人解析失败：requester_manager 类型需要安装组织架构模块")
	}

	managerID, err := r.orgService.FindManagerByUserID(ticket.RequesterID)
	if err != nil {
		return nil, fmt.Errorf("failed to find manager for user %d: %w", ticket.RequesterID, err)
	}
	if managerID == 0 {
		return nil, nil
	}
	return []uint{managerID}, nil
}
