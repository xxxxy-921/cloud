package node

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrProcessDefNotFound   = errors.New("process definition not found")
	ErrProcessDefNameExists = errors.New("process definition name already exists")
)

type ProcessDefService struct {
	processDefRepo  *ProcessDefRepo
	nodeProcessRepo *NodeProcessRepo
	commandRepo     *NodeCommandRepo
	hub             *NodeHub
}

func NewProcessDefService(i do.Injector) (*ProcessDefService, error) {
	return &ProcessDefService{
		processDefRepo:  do.MustInvoke[*ProcessDefRepo](i),
		nodeProcessRepo: do.MustInvoke[*NodeProcessRepo](i),
		commandRepo:     do.MustInvoke[*NodeCommandRepo](i),
		hub:             do.MustInvoke[*NodeHub](i),
	}, nil
}

func (s *ProcessDefService) Create(pd *ProcessDef) error {
	if _, err := s.processDefRepo.FindByName(pd.Name); err == nil {
		return ErrProcessDefNameExists
	}
	return s.processDefRepo.Create(pd)
}

func (s *ProcessDefService) Get(id uint) (*ProcessDef, error) {
	pd, err := s.processDefRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessDefNotFound
		}
		return nil, err
	}
	return pd, nil
}

func (s *ProcessDefService) List(params ProcessDefListParams) ([]ProcessDef, int64, error) {
	return s.processDefRepo.List(params)
}

func (s *ProcessDefService) Update(id uint, updates map[string]any) (*ProcessDef, error) {
	pd, err := s.processDefRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessDefNotFound
		}
		return nil, err
	}

	if err := s.processDefRepo.Update(id, updates); err != nil {
		return nil, err
	}

	// Push config.update to all nodes running this process
	nodeProcesses, _ := s.nodeProcessRepo.ListByProcessDefID(id)
	payload, _ := json.Marshal(map[string]any{
		"process_def_id": id,
		"process_name":   pd.Name,
	})
	for _, np := range nodeProcesses {
		cmd := &NodeCommand{
			NodeID:  np.NodeID,
			Type:    CommandTypeConfigUpdate,
			Payload: JSONMap(payload),
			Status:  CommandStatusPending,
		}
		if err := s.commandRepo.Create(cmd); err != nil {
			slog.Warn("failed to enqueue config.update", "nodeId", np.NodeID, "processDef", pd.Name, "error", err)
			continue
		}
		// Push via SSE
		s.hub.SendCommand(np.NodeID, cmd)
	}

	return s.processDefRepo.FindByID(id)
}

func (s *ProcessDefService) Delete(id uint) error {
	_, err := s.processDefRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProcessDefNotFound
		}
		return err
	}

	// Enqueue stop commands for all nodes running this process
	nodeProcesses, _ := s.nodeProcessRepo.ListByProcessDefID(id)
	payload, _ := json.Marshal(map[string]any{"process_def_id": id})
	for _, np := range nodeProcesses {
		cmd := &NodeCommand{
			NodeID:  np.NodeID,
			Type:    CommandTypeProcessStop,
			Payload: JSONMap(payload),
			Status:  CommandStatusPending,
		}
		if err := s.commandRepo.Create(cmd); err != nil {
			slog.Warn("failed to enqueue stop on delete", "nodeId", np.NodeID, "error", err)
			continue
		}
		s.hub.SendCommand(np.NodeID, cmd)
	}

	return s.processDefRepo.Delete(id)
}
