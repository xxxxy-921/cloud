package node

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrNodeNotFound    = errors.New("node not found")
	ErrNodeNameExists  = errors.New("node name already exists")
)

type NodeService struct {
	nodeRepo       *NodeRepo
	processRepo    *NodeProcessRepo
	processDefRepo *ProcessDefRepo
	commandRepo    *NodeCommandRepo
}

func NewNodeService(i do.Injector) (*NodeService, error) {
	return &NodeService{
		nodeRepo:       do.MustInvoke[*NodeRepo](i),
		processRepo:    do.MustInvoke[*NodeProcessRepo](i),
		processDefRepo: do.MustInvoke[*ProcessDefRepo](i),
		commandRepo:    do.MustInvoke[*NodeCommandRepo](i),
	}, nil
}

type CreateNodeResult struct {
	Node  *Node
	Token string // raw token, display once
}

func (s *NodeService) Create(name string, labels JSONMap) (*CreateNodeResult, error) {
	if _, err := s.nodeRepo.FindByName(name); err == nil {
		return nil, ErrNodeNameExists
	}

	raw, hash, prefix, err := GenerateNodeToken()
	if err != nil {
		return nil, err
	}

	node := &Node{
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Status:      NodeStatusPending,
		Labels:      labels,
	}
	if err := s.nodeRepo.Create(node); err != nil {
		return nil, err
	}

	return &CreateNodeResult{Node: node, Token: raw}, nil
}

func (s *NodeService) Get(id uint) (*Node, error) {
	node, err := s.nodeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return node, nil
}

func (s *NodeService) List(params NodeListParams) ([]NodeListItem, int64, error) {
	return s.nodeRepo.List(params)
}

func (s *NodeService) Update(id uint, name *string, labels *JSONMap) (*Node, error) {
	node, err := s.nodeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}

	updates := map[string]any{}
	if name != nil {
		// Check uniqueness
		existing, err := s.nodeRepo.FindByName(*name)
		if err == nil && existing.ID != id {
			return nil, ErrNodeNameExists
		}
		updates["name"] = *name
	}
	if labels != nil {
		updates["labels"] = *labels
	}

	if len(updates) > 0 {
		if err := s.nodeRepo.Update(id, updates); err != nil {
			return nil, err
		}
		node, _ = s.nodeRepo.FindByID(id)
	}
	return node, nil
}

func (s *NodeService) Delete(id uint) error {
	_, err := s.nodeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeNotFound
		}
		return err
	}

	// Enqueue process.stop for all bound processes
	processes, _ := s.processRepo.ListByNodeID(id)
	for _, np := range processes {
		var processName string
		if pd, err := s.processDefRepo.FindByID(np.ProcessDefID); err == nil {
			processName = pd.Name
		}
		payload, _ := json.Marshal(map[string]any{
			"process_def_id":  np.ProcessDefID,
			"node_process_id": np.ID,
			"process_name":    processName,
		})
		cmd := &NodeCommand{
			NodeID:  id,
			Type:    CommandTypeProcessStop,
			Payload: JSONMap(payload),
			Status:  CommandStatusPending,
		}
		if err := s.commandRepo.Create(cmd); err != nil {
			slog.Warn("failed to enqueue stop command on delete", "nodeId", id, "error", err)
		}
	}

	// Mark all processes as stopped
	_ = s.processRepo.BatchUpdateStatusByNodeID(id, ProcessStatusStopped)

	// Cleanup pending commands (except the stop commands we just created)
	_ = s.commandRepo.FailPendingByNodeID(id, "node_deleted")

	return s.nodeRepo.Delete(id)
}

func (s *NodeService) RotateToken(id uint) (string, error) {
	_, err := s.nodeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNodeNotFound
		}
		return "", err
	}

	raw, hash, prefix, err := GenerateNodeToken()
	if err != nil {
		return "", err
	}

	if err := s.nodeRepo.UpdateToken(id, hash, prefix); err != nil {
		return "", err
	}
	return raw, nil
}
