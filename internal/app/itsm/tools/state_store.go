package tools

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// SessionStateStore implements StateStore by reading/writing the
// ai_agent_sessions.state JSON column.
type SessionStateStore struct {
	db *gorm.DB
}

// NewSessionStateStore creates a new SessionStateStore.
func NewSessionStateStore(db *gorm.DB) *SessionStateStore {
	return &SessionStateStore{db: db}
}

// GetState reads the session state from the database.
func (s *SessionStateStore) GetState(sessionID uint) (*ServiceDeskState, error) {
	var row struct {
		State string
	}
	if err := s.db.Table("ai_agent_sessions").
		Where("id = ?", sessionID).
		Select("state").First(&row).Error; err != nil {
		return nil, fmt.Errorf("session %d not found: %w", sessionID, err)
	}

	if row.State == "" || row.State == "null" {
		return defaultState(), nil
	}

	var state ServiceDeskState
	if err := json.Unmarshal([]byte(row.State), &state); err != nil {
		return defaultState(), nil
	}
	return &state, nil
}

// SaveState writes the session state to the database.
func (s *SessionStateStore) SaveState(sessionID uint, state *ServiceDeskState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := s.db.Table("ai_agent_sessions").
		Where("id = ?", sessionID).
		Update("state", string(b)).Error; err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	return nil
}
