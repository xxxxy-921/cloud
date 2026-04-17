package engine

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// OrgService is an optional interface that the Org App can provide for participant resolution.
type OrgService interface {
	FindUsersByPositionID(positionID uint) ([]uint, error)
	FindUsersByDepartmentID(departmentID uint) ([]uint, error)
	FindManagerByUserID(userID uint) (uint, error)
	FindUsersByPositionCodeAndDepartmentCode(positionCode, departmentCode string) ([]uint, error)
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

	case "position_department":
		if r.orgService == nil {
			return nil, fmt.Errorf("参与人解析失败：position_department 类型需要安装组织架构模块")
		}
		if p.PositionCode == "" || p.DepartmentCode == "" {
			return nil, fmt.Errorf("position_department 类型需要同时指定 position_code 和 department_code")
		}
		return r.orgService.FindUsersByPositionCodeAndDepartmentCode(p.PositionCode, p.DepartmentCode)

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

// resolveForToolArgs is the JSON structure for decision.resolve_participant tool arguments.
type resolveForToolArgs struct {
	Type           string `json:"type"`
	Value          string `json:"value,omitempty"`
	PositionCode   string `json:"position_code,omitempty"`
	DepartmentCode string `json:"department_code,omitempty"`
}

// ResolveForTool resolves participants from JSON tool arguments.
// It wraps Resolve() with JSON parameter parsing for the decision tool.
func (r *ParticipantResolver) ResolveForTool(tx *gorm.DB, ticketID uint, toolArgs json.RawMessage) ([]uint, error) {
	var args resolveForToolArgs
	if err := json.Unmarshal(toolArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	if args.Type == "" {
		return nil, fmt.Errorf("participant type is required")
	}
	p := Participant{
		Type:           args.Type,
		Value:          args.Value,
		PositionCode:   args.PositionCode,
		DepartmentCode: args.DepartmentCode,
	}
	return r.Resolve(tx, ticketID, p)
}
