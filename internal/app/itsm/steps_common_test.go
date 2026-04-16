package itsm

// steps_common_test.go — shared BDD test context, engine wiring, and common step definitions.
//
// bddContext holds the state shared across steps within a single scenario.
// Each scenario gets a fresh context via reset().

import (
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/app/ai"
	"metis/internal/app/itsm/engine"
	"metis/internal/app/org"
	"metis/internal/model"
)

// bddContext is the shared state container for BDD scenarios.
type bddContext struct {
	db      *gorm.DB
	lastErr error

	// Engine
	engine *engine.ClassicEngine

	// Participants (populated by Given steps)
	users       map[string]*model.User  // key = identity label (e.g. "申请人")
	usersByName map[string]*model.User  // key = username
	positions   map[string]*org.Position // key = position code
	departments map[string]*org.Department // key = department code

	// Ticket lifecycle (populated by When steps, asserted by Then steps)
	service *ServiceDefinition
	ticket  *Ticket
	tickets map[string]*Ticket // multi-ticket scenarios, key = alias
}

func newBDDContext() *bddContext {
	return &bddContext{}
}

// reset clears all state for a new scenario. Called in sc.Before.
func (bc *bddContext) reset() {
	bc.lastErr = nil
	bc.service = nil
	bc.ticket = nil
	bc.users = make(map[string]*model.User)
	bc.usersByName = make(map[string]*model.User)
	bc.positions = make(map[string]*org.Position)
	bc.departments = make(map[string]*org.Department)
	bc.tickets = make(map[string]*Ticket)

	// Fresh in-memory database per scenario.
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:bdd_%p?mode=memory&cache=shared", bc)), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("bdd: failed to open test db: %v", err))
	}

	// AutoMigrate all models needed for BDD scenarios.
	if err := db.AutoMigrate(
		// Kernel
		&model.User{},
		&model.Role{},
		// Org
		&org.Department{},
		&org.Position{},
		&org.UserPosition{},
		// AI
		&ai.Agent{},
		&ai.AgentSession{},
		// ITSM — configuration
		&ServiceCatalog{},
		&ServiceDefinition{},
		&ServiceAction{},
		&FormDefinition{},
		&Priority{},
		&SLATemplate{},
		&EscalationRule{},
		// ITSM — ticket lifecycle
		&Ticket{},
		&TicketActivity{},
		&TicketAssignment{},
		&TicketTimeline{},
		&TicketActionExecution{},
		// ITSM — incident
		&TicketLink{},
		&PostMortem{},
		// ITSM — process control
		&ProcessVariable{},
		&ExecutionToken{},
		// ITSM — knowledge
		&ServiceKnowledgeDocument{},
	); err != nil {
		panic(fmt.Sprintf("bdd: failed to migrate: %v", err))
	}

	bc.db = db

	// Build ClassicEngine with test dependencies.
	orgSvc := &testOrgService{db: db}
	resolver := engine.NewParticipantResolver(orgSvc)
	bc.engine = engine.NewClassicEngine(resolver, &noopSubmitter{})
}

// ---------------------------------------------------------------------------
// Test doubles for engine dependencies
// ---------------------------------------------------------------------------

// testOrgService implements engine.OrgService by querying the BDD in-memory DB.
type testOrgService struct {
	db *gorm.DB
}

func (s *testOrgService) FindUsersByPositionID(positionID uint) ([]uint, error) {
	var ups []org.UserPosition
	if err := s.db.Where("position_id = ?", positionID).Find(&ups).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(ups))
	for _, up := range ups {
		ids = append(ids, up.UserID)
	}
	return ids, nil
}

func (s *testOrgService) FindUsersByDepartmentID(departmentID uint) ([]uint, error) {
	var ups []org.UserPosition
	if err := s.db.Where("department_id = ?", departmentID).Find(&ups).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(ups))
	for _, up := range ups {
		ids = append(ids, up.UserID)
	}
	return ids, nil
}

func (s *testOrgService) FindManagerByUserID(userID uint) (uint, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return 0, err
	}
	if user.ManagerID == nil {
		return 0, nil
	}
	return *user.ManagerID, nil
}

// Compile-time check.
var _ engine.OrgService = (*testOrgService)(nil)

// noopSubmitter implements engine.TaskSubmitter as a no-op.
type noopSubmitter struct{}

func (n *noopSubmitter) SubmitTask(_ string, _ json.RawMessage) error { return nil }

var _ engine.TaskSubmitter = (*noopSubmitter)(nil)

// ---------------------------------------------------------------------------
// Common Given step definitions
// ---------------------------------------------------------------------------

// givenSystemInitialized is a no-op — reset() already handles initialization.
func (bc *bddContext) givenSystemInitialized() error {
	return nil
}

// givenParticipants parses a DataTable and creates User/Department/Position/UserPosition records.
//
// Expected DataTable format:
//
//	| 身份 | 用户名 | 部门 | 岗位 |
//	| 申请人 | vpn-requester | - | - |
//	| 网络管理员审批人 | network-operator | it | network_admin |
func (bc *bddContext) givenParticipants(table *godog.Table) error {
	if len(table.Rows) < 2 {
		return fmt.Errorf("participants table must have a header row and at least one data row")
	}

	// Parse header to find column indices.
	header := table.Rows[0]
	colIdx := make(map[string]int)
	for i, cell := range header.Cells {
		colIdx[cell.Value] = i
	}
	for _, required := range []string{"身份", "用户名", "部门", "岗位"} {
		if _, ok := colIdx[required]; !ok {
			return fmt.Errorf("participants table missing required column: %s", required)
		}
	}

	for _, row := range table.Rows[1:] {
		identity := row.Cells[colIdx["身份"]].Value
		username := row.Cells[colIdx["用户名"]].Value
		deptCode := row.Cells[colIdx["部门"]].Value
		posCode := row.Cells[colIdx["岗位"]].Value

		// Create or get User.
		user := &model.User{Username: username, IsActive: true}
		if err := bc.db.Where("username = ?", username).FirstOrCreate(user).Error; err != nil {
			return fmt.Errorf("create user %q: %w", username, err)
		}
		bc.users[identity] = user
		bc.usersByName[username] = user

		// Create Department if not "-".
		var dept *org.Department
		if deptCode != "-" && deptCode != "" {
			dept = &org.Department{Code: deptCode, Name: deptCode, IsActive: true}
			if err := bc.db.Where("code = ?", deptCode).FirstOrCreate(dept).Error; err != nil {
				return fmt.Errorf("create department %q: %w", deptCode, err)
			}
			bc.departments[deptCode] = dept
		}

		// Create Position + UserPosition if not "-".
		if posCode != "-" && posCode != "" {
			pos := &org.Position{Code: posCode, Name: posCode, IsActive: true}
			if err := bc.db.Where("code = ?", posCode).FirstOrCreate(pos).Error; err != nil {
				return fmt.Errorf("create position %q: %w", posCode, err)
			}
			bc.positions[posCode] = pos

			// UserPosition requires a department.
			if dept == nil {
				return fmt.Errorf("position %q specified without a department for user %q", posCode, username)
			}
			up := &org.UserPosition{
				UserID:       user.ID,
				DepartmentID: dept.ID,
				PositionID:   pos.ID,
				IsPrimary:    true,
			}
			if err := bc.db.Where("user_id = ? AND department_id = ?", user.ID, dept.ID).
				FirstOrCreate(up).Error; err != nil {
				return fmt.Errorf("create user_position for %q: %w", username, err)
			}
		}
	}

	return nil
}
