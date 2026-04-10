package model

import "time"

type MessageChannel struct {
	BaseModel
	Name    string `json:"name" gorm:"size:100;not null"`
	Type    string `json:"type" gorm:"size:32;not null"`
	Config  string `json:"config" gorm:"type:text;not null"`
	Enabled bool   `json:"enabled" gorm:"not null;default:true"`
}

type MessageChannelResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (m *MessageChannel) ToResponse() MessageChannelResponse {
	return MessageChannelResponse{
		ID:        m.ID,
		Name:      m.Name,
		Type:      m.Type,
		Config:    m.Config,
		Enabled:   m.Enabled,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
