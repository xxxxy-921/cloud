package node

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrNodeProcessNotFound = errors.New("node process binding not found")
	ErrNodeProcessExists   = errors.New("process already bound to this node")
)

type NodeProcessService struct {
	nodeRepo        *NodeRepo
	processDefRepo  *ProcessDefRepo
	nodeProcessRepo *NodeProcessRepo
	commandRepo     *NodeCommandRepo
	hub             *NodeHub
}

func NewNodeProcessService(i do.Injector) (*NodeProcessService, error) {
	return &NodeProcessService{
		nodeRepo:        do.MustInvoke[*NodeRepo](i),
		processDefRepo:  do.MustInvoke[*ProcessDefRepo](i),
		nodeProcessRepo: do.MustInvoke[*NodeProcessRepo](i),
		commandRepo:     do.MustInvoke[*NodeCommandRepo](i),
		hub:             do.MustInvoke[*NodeHub](i),
	}, nil
}

// createAndPushCommand persists a command to DB and pushes it via SSE if the node is online.
func (s *NodeProcessService) createAndPushCommand(cmd *NodeCommand) error {
	if err := s.commandRepo.Create(cmd); err != nil {
		return err
	}
	// Best-effort push via SSE; if offline, command stays in DB for delivery on reconnect
	s.hub.SendCommand(cmd.NodeID, cmd)
	return nil
}

// buildStartPayload builds the full process.start command payload.
func (s *NodeProcessService) buildStartPayload(np *NodeProcess, pd *ProcessDef) JSONMap {
	payload, _ := json.Marshal(map[string]any{
		"process_def_id":  pd.ID,
		"node_process_id": np.ID,
		"override_vars":   json.RawMessage(np.OverrideVars),
		"process_def": map[string]any{
			"id":            pd.ID,
			"name":          pd.Name,
			"startCommand":  pd.StartCommand,
			"stopCommand":   pd.StopCommand,
			"reloadCommand": pd.ReloadCommand,
			"env":           json.RawMessage(pd.Env),
			"configFiles":   json.RawMessage(pd.ConfigFiles),
			"probeType":     pd.ProbeType,
			"probeConfig":   json.RawMessage(pd.ProbeConfig),
			"restartPolicy": pd.RestartPolicy,
			"maxRestarts":   pd.MaxRestarts,
		},
	})
	return JSONMap(payload)
}

func (s *NodeProcessService) Bind(nodeID, processDefID uint, overrideVars JSONMap) (*NodeProcess, error) {
	if _, err := s.nodeRepo.FindByID(nodeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	pd, err := s.processDefRepo.FindByID(processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProcessDefNotFound
		}
		return nil, err
	}

	// Check if already bound
	if _, err := s.nodeProcessRepo.FindByNodeAndProcessDef(nodeID, processDefID); err == nil {
		return nil, ErrNodeProcessExists
	}

	np := &NodeProcess{
		NodeID:       nodeID,
		ProcessDefID: processDefID,
		Status:       ProcessStatusPendingConfig,
		OverrideVars: overrideVars,
	}
	if err := s.nodeProcessRepo.Create(np); err != nil {
		return nil, err
	}

	cmd := &NodeCommand{
		NodeID:  nodeID,
		Type:    CommandTypeProcessStart,
		Payload: s.buildStartPayload(np, pd),
		Status:  CommandStatusPending,
	}
	if err := s.createAndPushCommand(cmd); err != nil {
		return np, fmt.Errorf("failed to enqueue start command: %w", err)
	}

	return np, nil
}

func (s *NodeProcessService) ListByNodeID(nodeID uint) ([]NodeProcessDetail, error) {
	return s.nodeProcessRepo.ListByNodeID(nodeID)
}

func (s *NodeProcessService) Unbind(nodeID, processDefID uint) error {
	np, err := s.nodeProcessRepo.FindByNodeAndProcessDef(nodeID, processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeProcessNotFound
		}
		return err
	}

	var processName string
	if pd, err := s.processDefRepo.FindByID(processDefID); err == nil {
		processName = pd.Name
	}

	payload, _ := json.Marshal(map[string]any{
		"process_def_id":  processDefID,
		"node_process_id": np.ID,
		"process_name":    processName,
	})
	cmd := &NodeCommand{
		NodeID:  nodeID,
		Type:    CommandTypeProcessStop,
		Payload: JSONMap(payload),
		Status:  CommandStatusPending,
	}
	if err := s.createAndPushCommand(cmd); err != nil {
		return fmt.Errorf("failed to enqueue stop command: %w", err)
	}

	return s.nodeProcessRepo.Delete(np.ID)
}

func (s *NodeProcessService) Start(nodeID, processDefID uint) error {
	np, err := s.nodeProcessRepo.FindByNodeAndProcessDef(nodeID, processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeProcessNotFound
		}
		return err
	}

	pd, err := s.processDefRepo.FindByID(processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProcessDefNotFound
		}
		return err
	}

	cmd := &NodeCommand{
		NodeID:  nodeID,
		Type:    CommandTypeProcessStart,
		Payload: s.buildStartPayload(np, pd),
		Status:  CommandStatusPending,
	}
	return s.createAndPushCommand(cmd)
}

func (s *NodeProcessService) Stop(nodeID, processDefID uint) error {
	np, err := s.nodeProcessRepo.FindByNodeAndProcessDef(nodeID, processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeProcessNotFound
		}
		return err
	}

	var processName string
	if pd, err := s.processDefRepo.FindByID(processDefID); err == nil {
		processName = pd.Name
	}

	payload, _ := json.Marshal(map[string]any{
		"process_def_id":  processDefID,
		"node_process_id": np.ID,
		"process_name":    processName,
	})
	cmd := &NodeCommand{
		NodeID:  nodeID,
		Type:    CommandTypeProcessStop,
		Payload: JSONMap(payload),
		Status:  CommandStatusPending,
	}
	return s.createAndPushCommand(cmd)
}

func (s *NodeProcessService) Restart(nodeID, processDefID uint) error {
	np, err := s.nodeProcessRepo.FindByNodeAndProcessDef(nodeID, processDefID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeProcessNotFound
		}
		return err
	}

	var processName string
	if pd, err := s.processDefRepo.FindByID(processDefID); err == nil {
		processName = pd.Name
	}

	payload, _ := json.Marshal(map[string]any{
		"process_def_id":  processDefID,
		"node_process_id": np.ID,
		"process_name":    processName,
	})
	cmd := &NodeCommand{
		NodeID:  nodeID,
		Type:    CommandTypeProcessRestart,
		Payload: JSONMap(payload),
		Status:  CommandStatusPending,
	}
	return s.createAndPushCommand(cmd)
}
