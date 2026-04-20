package ai

import (
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrAgentNotFound           = errors.New("agent not found")
	ErrAgentNameConflict       = errors.New("agent name already exists")
	ErrAgentCodeConflict       = errors.New("agent code already exists")
	ErrAgentHasRunningSessions = errors.New("agent has running sessions")
	ErrInvalidAgentType        = errors.New("invalid agent type")
	ErrNodeRequired            = errors.New("node_id is required for remote exec mode")
	ErrModelRequired           = errors.New("model_id is required for assistant agent")
	ErrRuntimeRequired         = errors.New("runtime is required for coding agent")
	ErrCodeRequired            = errors.New("code is required for internal agent")
	ErrInvalidBinding          = errors.New("invalid agent binding")
)

var ValidAgentTypes = map[string]bool{
	AgentTypeAssistant: true,
	AgentTypeCoding:    true,
	AgentTypeInternal:  true,
}

var ValidStrategies = map[string]bool{
	AgentStrategyReact:          true,
	AgentStrategyPlanAndExecute: true,
}

var ValidRuntimes = map[string]bool{
	AgentRuntimeClaudeCode: true,
	AgentRuntimeCodex:      true,
	AgentRuntimeOpenCode:   true,
	AgentRuntimeAider:      true,
}

type AgentService struct {
	repo *AgentRepo
}

type AgentBindings struct {
	ToolIDs           []uint
	SkillIDs          []uint
	MCPServerIDs      []uint
	KnowledgeBaseIDs  []uint
	KnowledgeGraphIDs []uint
}

func NewAgentService(i do.Injector) (*AgentService, error) {
	return &AgentService{
		repo: do.MustInvoke[*AgentRepo](i),
	}, nil
}

func (s *AgentService) Create(a *Agent) error {
	if err := s.validateForCreate(a); err != nil {
		return err
	}
	return s.repo.Create(a)
}

func (s *AgentService) CreateWithBindings(a *Agent, bindings AgentBindings) error {
	if err := s.validateForCreate(a); err != nil {
		return err
	}
	normalized, err := s.normalizeBindings(bindings)
	if err != nil {
		return err
	}

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		return s.repo.replaceBindingsInTx(tx, a.ID, normalized)
	})
}

func (s *AgentService) Get(id uint) (*Agent, error) {
	a, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *AgentService) GetAccessible(id, userID uint) (*Agent, error) {
	a, err := s.repo.FindAccessibleByID(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *AgentService) GetOwned(id, userID uint) (*Agent, error) {
	a, err := s.repo.FindOwnedByID(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *AgentService) GetByCode(code string) (*Agent, error) {
	a, err := s.repo.FindByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *AgentService) Update(a *Agent) error {
	if err := s.validateByType(a); err != nil {
		return err
	}
	return s.repo.Update(a)
}

func (s *AgentService) UpdateWithBindings(a *Agent, bindings AgentBindings) error {
	if err := s.validateByType(a); err != nil {
		return err
	}
	normalized, err := s.normalizeBindings(bindings)
	if err != nil {
		return err
	}

	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(a).Error; err != nil {
			return err
		}
		return s.repo.replaceBindingsInTx(tx, a.ID, normalized)
	})
}

func (s *AgentService) Delete(id uint) error {
	hasRunning, err := s.repo.HasRunningSessions(id)
	if err != nil {
		return err
	}
	if hasRunning {
		return ErrAgentHasRunningSessions
	}
	return s.repo.Delete(id)
}

func (s *AgentService) List(params AgentListParams) ([]Agent, int64, error) {
	return s.repo.List(params)
}

func (s *AgentService) validateByType(a *Agent) error {
	switch a.Type {
	case AgentTypeAssistant:
		if a.ModelID == nil {
			return ErrModelRequired
		}
		if a.Strategy == "" {
			a.Strategy = AgentStrategyReact
		}
		if !ValidStrategies[a.Strategy] {
			return errors.New("invalid strategy: " + a.Strategy)
		}
	case AgentTypeCoding:
		if a.Runtime == "" {
			return ErrRuntimeRequired
		}
		if !ValidRuntimes[a.Runtime] {
			return errors.New("invalid runtime: " + a.Runtime)
		}
		if a.ExecMode == "" {
			a.ExecMode = AgentExecModeLocal
		}
		if a.ExecMode == AgentExecModeRemote && a.NodeID == nil {
			return ErrNodeRequired
		}
	case AgentTypeInternal:
		if a.Code == nil || *a.Code == "" {
			return ErrCodeRequired
		}
	}
	return nil
}

// EnsureType checks that agent.Type matches expectedType; returns ErrAgentNotFound on mismatch.
func (s *AgentService) EnsureType(a *Agent, expectedType string) error {
	if a.Type != expectedType {
		return ErrAgentNotFound
	}
	return nil
}

// GetAccessibleByType loads an agent visible to the user and verifies its type.
func (s *AgentService) GetAccessibleByType(id, userID uint, expectedType string) (*Agent, error) {
	a, err := s.GetAccessible(id, userID)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureType(a, expectedType); err != nil {
		return nil, err
	}
	return a, nil
}

// GetOwnedByType loads an agent owned by the user and verifies its type.
func (s *AgentService) GetOwnedByType(id, userID uint, expectedType string) (*Agent, error) {
	a, err := s.GetOwned(id, userID)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureType(a, expectedType); err != nil {
		return nil, err
	}
	return a, nil
}

// ListTemplatesByType returns agent templates filtered by type.
func (s *AgentService) ListTemplatesByType(agentType string) ([]AgentTemplate, error) {
	return s.repo.ListTemplatesByType(agentType)
}

// UpdateBindings replaces all bindings for the given agent
func (s *AgentService) UpdateBindings(agentID uint, toolIDs, skillIDs, mcpIDs, kbIDs, kgIDs []uint) error {
	bindings, err := s.normalizeBindings(AgentBindings{
		ToolIDs:           toolIDs,
		SkillIDs:          skillIDs,
		MCPServerIDs:      mcpIDs,
		KnowledgeBaseIDs:  kbIDs,
		KnowledgeGraphIDs: kgIDs,
	})
	if err != nil {
		return err
	}
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		return s.repo.replaceBindingsInTx(tx, agentID, bindings)
	})
}

// GetBindings returns all binding IDs for an agent
func (s *AgentService) GetBindings(agentID uint) (toolIDs, skillIDs, mcpIDs, kbIDs, kgIDs []uint, err error) {
	toolIDs, err = s.repo.GetToolIDs(agentID)
	if err != nil {
		return
	}
	skillIDs, err = s.repo.GetSkillIDs(agentID)
	if err != nil {
		return
	}
	mcpIDs, err = s.repo.GetMCPServerIDs(agentID)
	if err != nil {
		return
	}
	kbIDs, err = s.repo.GetKnowledgeBaseIDs(agentID)
	if err != nil {
		return
	}
	kgIDs, err = s.repo.GetKnowledgeGraphIDs(agentID)
	return
}

// ListTemplates returns all agent templates
func (s *AgentService) ListTemplates() ([]AgentTemplate, error) {
	return s.repo.ListTemplates()
}

func (s *AgentService) validateForCreate(a *Agent) error {
	if !ValidAgentTypes[a.Type] {
		return ErrInvalidAgentType
	}
	if err := s.validateByType(a); err != nil {
		return err
	}
	if _, err := s.repo.FindByName(a.Name); err == nil {
		return ErrAgentNameConflict
	}
	if a.Code != nil && *a.Code != "" {
		if _, err := s.repo.FindByCode(*a.Code); err == nil {
			return ErrAgentCodeConflict
		}
	}
	return nil
}

func (s *AgentService) normalizeBindings(bindings AgentBindings) (AgentBindings, error) {
	var err error
	normalized := AgentBindings{}
	if normalized.ToolIDs, err = uniqueUintIDs(bindings.ToolIDs); err != nil {
		return AgentBindings{}, err
	}
	if normalized.SkillIDs, err = uniqueUintIDs(bindings.SkillIDs); err != nil {
		return AgentBindings{}, err
	}
	if normalized.MCPServerIDs, err = uniqueUintIDs(bindings.MCPServerIDs); err != nil {
		return AgentBindings{}, err
	}
	if normalized.KnowledgeBaseIDs, err = uniqueUintIDs(bindings.KnowledgeBaseIDs); err != nil {
		return AgentBindings{}, err
	}
	if normalized.KnowledgeGraphIDs, err = uniqueUintIDs(bindings.KnowledgeGraphIDs); err != nil {
		return AgentBindings{}, err
	}

	db := s.repo.DB()
	if err := ensureIDsExist(db, &Tool{}, normalized.ToolIDs, ""); err != nil {
		return AgentBindings{}, err
	}
	if err := ensureIDsExist(db, &Skill{}, normalized.SkillIDs, ""); err != nil {
		return AgentBindings{}, err
	}
	if err := ensureIDsExist(db, &MCPServer{}, normalized.MCPServerIDs, ""); err != nil {
		return AgentBindings{}, err
	}
	if err := ensureIDsExist(db, &KnowledgeAsset{}, normalized.KnowledgeBaseIDs, AssetCategoryKB); err != nil {
		return AgentBindings{}, err
	}
	if err := ensureIDsExist(db, &KnowledgeAsset{}, normalized.KnowledgeGraphIDs, AssetCategoryKG); err != nil {
		return AgentBindings{}, err
	}

	return normalized, nil
}

func uniqueUintIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	seen := make(map[uint]struct{}, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, ErrInvalidBinding
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func ensureIDsExist(db *gorm.DB, model any, ids []uint, category string) error {
	if len(ids) == 0 {
		return nil
	}

	query := db.Model(model).Where("id IN ?", ids)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return ErrInvalidBinding
	}
	return nil
}
