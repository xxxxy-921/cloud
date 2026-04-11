package node

import (
	"time"

	"github.com/samber/do/v2"

	"metis/internal/database"
)

type NodeCommandRepo struct {
	db *database.DB
}

func NewNodeCommandRepo(i do.Injector) (*NodeCommandRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &NodeCommandRepo{db: db}, nil
}

func (r *NodeCommandRepo) Create(cmd *NodeCommand) error {
	return r.db.Create(cmd).Error
}

func (r *NodeCommandRepo) FindByID(id uint) (*NodeCommand, error) {
	var cmd NodeCommand
	if err := r.db.First(&cmd, id).Error; err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (r *NodeCommandRepo) FindPendingByNodeID(nodeID uint) ([]NodeCommand, error) {
	var cmds []NodeCommand
	if err := r.db.Where("node_id = ? AND status = ?", nodeID, CommandStatusPending).
		Order("created_at ASC").
		Find(&cmds).Error; err != nil {
		return nil, err
	}
	return cmds, nil
}

func (r *NodeCommandRepo) ListByNodeID(nodeID uint, limit int) ([]NodeCommand, error) {
	var cmds []NodeCommand
	if limit < 1 {
		limit = 50
	}
	if err := r.db.Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(limit).
		Find(&cmds).Error; err != nil {
		return nil, err
	}
	return cmds, nil
}

func (r *NodeCommandRepo) Ack(id uint, result string) error {
	now := time.Now()
	return r.db.Model(&NodeCommand{}).Where("id = ?", id).Updates(map[string]any{
		"status":   CommandStatusAcked,
		"result":   result,
		"acked_at": &now,
	}).Error
}

func (r *NodeCommandRepo) Fail(id uint, result string) error {
	now := time.Now()
	return r.db.Model(&NodeCommand{}).Where("id = ?", id).Updates(map[string]any{
		"status":   CommandStatusFailed,
		"result":   result,
		"acked_at": &now,
	}).Error
}

func (r *NodeCommandRepo) CleanupExpired(timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := r.db.Model(&NodeCommand{}).
		Where("status = ? AND created_at < ?", CommandStatusPending, cutoff).
		Updates(map[string]any{
			"status": CommandStatusFailed,
			"result": "node_offline_timeout",
		})
	return result.RowsAffected, result.Error
}

func (r *NodeCommandRepo) FailPendingByNodeID(nodeID uint, reason string) error {
	now := time.Now()
	return r.db.Model(&NodeCommand{}).
		Where("node_id = ? AND status = ?", nodeID, CommandStatusPending).
		Updates(map[string]any{
			"status":   CommandStatusFailed,
			"result":   reason,
			"acked_at": &now,
		}).Error
}
